package devbridge

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net"
	"net/http"
	"strconv"
	"sync"
	"time"

	"github.com/microsoft/dev-tunnels-ssh/src/go/ssh"
	"github.com/microsoft/dev-tunnels-ssh/src/go/tcp"
)

// ──────────────────────────────────────────────────────────────
// Connect — 连接远程服务
//
// Connect 运行在访问设备，连接同一条隧道，在本地建立端口映射。
// 连接建立后，通过 localhost:port 即可访问远端 Host 托管的服务。
//
// 工作流程：
//  1. WebSocket 连到 wss://<tunnelId>.<gatewayHost>/
//  2. 在 WebSocket 上建立 SSH 客户端会话
//  3. SSH 端口转发服务在本地创建 TCP listener
//  4. 远端 Host 的流量通过 SSH 隧道转到本地 listener
// ──────────────────────────────────────────────────────────────

// ConnectConfig Connect 连接配置
type ConnectConfig struct {
	TunnelID  string   // 隧道 ID
	Ports     []int    // 端口列表（为空时从 Host 端通过 SSH 下发）
	JWTToken  string   // JWT 令牌（与 APIKey 二选一）
	APIKey    string   // API Key（与 JWTToken 二选一）
	LocalIP   string   // 本地监听地址，默认 127.0.0.1
}

// Forwarding 端口转发映射信息
type Forwarding struct {
	LocalPort  int    // 本地端口
	RemotePort int    // 远端端口
	LocalIP    string // 本地监听地址
}

// Connect 启动 Connect 连接服务
//
// 这是一个阻塞方法，在 ctx 被取消或连接彻底断开时返回。
// 网络短暂中断会自动重连。
//
// 基本用法：
//
//	err := client.Connect(ctx, devbridge.ConnectConfig{
//	    TunnelID: "aaaadysa",
//	    Ports:    []int{8080},
//	})
//
// 使用 API Key 鉴权：
//
//	err := client.Connect(ctx, devbridge.ConnectConfig{
//	    TunnelID: "aaaadysa",
//	    APIKey:   "your-api-key",
//	})
//
// 使用已有 JWT 令牌：
//
//	err := client.Connect(ctx, devbridge.ConnectConfig{
//	    TunnelID: "aaaadysa",
//	    JWTToken: "your-jwt-token",
//	})
func (c *Client) Connect(ctx context.Context, cfg ConnectConfig) error {
	if err := validateTunnelID(cfg.TunnelID); err != nil {
		return err
	}
	if cfg.LocalIP == "" {
		cfg.LocalIP = "127.0.0.1"
	}

	header, subprotocols := buildWSHeader(cfg.JWTToken, cfg.APIKey)
	sniHost := cfg.TunnelID + "." + c.gatewayHost
	wsURL := "wss://" + sniHost + "/"

	factory := newListenerFactory(len(cfg.Ports), cfg.LocalIP, c.logger)

	const maxReconnectAttempts = 5
	const baseReconnectDelay = 3 * time.Second
	const maxReconnectDelay = 30 * time.Second

	consecutiveFailures := 0
	for consecutiveFailures < maxReconnectAttempts {
		connected, err := c.runConnectSession(ctx, wsURL, sniHost, header, subprotocols, cfg.TunnelID, cfg.Ports, factory)
		if ctx.Err() != nil {
			return nil
		}
		if err == nil {
			return nil
		}
		if errors.Is(err, ErrQuotaExceeded) || errors.Is(err, ErrTunnelNotFound) {
			return err
		}
		if connected {
			consecutiveFailures = 0
		} else {
			consecutiveFailures++
		}
		if consecutiveFailures >= maxReconnectAttempts {
			return fmt.Errorf("reconnect exhausted after %d attempts: %w", maxReconnectAttempts, err)
		}

		delay := baseReconnectDelay
		for i := 0; i < consecutiveFailures-1; i++ {
			delay *= 2
			if delay >= maxReconnectDelay {
				delay = maxReconnectDelay
				break
			}
		}
		c.logger.Info("connection lost, reconnecting...", "delay", delay, "err", err)
		select {
		case <-ctx.Done():
			return nil
		case <-time.After(delay):
		}
	}
	return nil
}

// runConnectSession 执行一次 Connect 会话
func (c *Client) runConnectSession(ctx context.Context, wsURL string, sniHost string, header http.Header, subprotocols []string, tunnelID string, ports []int, factory *listenerFactory) (connected bool, err error) {
	netConn, err := c.dialWebSocket(ctx, wsURL, sniHost, header, subprotocols, 5)
	if err != nil {
		return false, fmt.Errorf("WebSocket connection failed: %w", err)
	}

	config := ssh.NewNoSecurityConfig()
	config.KeepAliveIntervalSeconds = 10
	tcp.AddPortForwardingService(config)

	session := ssh.NewClientSession(config)
	session.Trace = sshTraceFunc(c.logger)
	defer func() { _ = session.Close() }()

	pfs := tcp.GetPortForwardingService(&session.Session)
	if pfs == nil {
		_ = netConn.Close()
		return false, fmt.Errorf("port forwarding service unavailable")
	}
	factory.reset()
	pfs.ListenerFactory = factory

	if err = session.Connect(ctx, netConn); err != nil {
		err = parseSSHCloseError(err)
		_ = netConn.Close()
		return false, fmt.Errorf("SSH client connect failed: %w", err)
	}
	connected = true

	c.logger.Info("connected to tunnel", "tunnelID", tunnelID)

	if len(ports) > 0 {
		c.logger.Debug("mode: active forwarding (ports from API)", "ports", ports)
	} else {
		c.logger.Debug("mode: passive forwarding (ports from host via SSH)")
	}

	// 等待转发建立
	if len(ports) > 0 {
		factory.waitForForwardings(3 * time.Second)
	} else {
		factory.waitForForwardings(2 * time.Second)
	}
	factory.printForwardings()

	select {
	case <-session.Session.Done():
		return true, fmt.Errorf("session closed")
	case <-ctx.Done():
		return true, nil
	}
}

// ──────────────────────────────────────────────────────────────
// listenerFactory — 创建本地 TCP listener 的工厂
// 实现 tcp.ListenerFactory 接口，在本地端口被占用时自动换随机端口
// ──────────────────────────────────────────────────────────────

type listenerFactory struct {
	mu                 sync.Mutex
	pendingForwardings []string
	portOverrides      map[int]int
	listeners          []net.Listener
	expectedCount      int
	allReceived        chan struct{}
	localIP            string
	logger             *slog.Logger
}

func newListenerFactory(expectedCount int, localIP string, logger *slog.Logger) *listenerFactory {
	return &listenerFactory{
		expectedCount: expectedCount,
		allReceived:   make(chan struct{}),
		portOverrides: make(map[int]int),
		localIP:       localIP,
		logger:        logger,
	}
}

// CreateTCPListener 实现 tcp.ListenerFactory 接口
func (f *listenerFactory) CreateTCPListener(
	remotePort int,
	localIPAddress string,
	localPort int,
	canChangeLocalPort bool,
) (net.Listener, error) {
	if override, ok := f.portOverrides[remotePort]; ok {
		localPort = override
	}

	listener, err := net.Listen("tcp", net.JoinHostPort(f.localIP, strconv.Itoa(localPort)))
	if err != nil {
		if canChangeLocalPort {
			return f.listenOnRandomPort(remotePort, localPort)
		}
		return nil, fmt.Errorf("port %d is already in use: %w", localPort, err)
	}
	f.portOverrides[remotePort] = localPort
	f.listeners = append(f.listeners, listener)
	f.addForwarding(fmt.Sprintf("Forwarding %s:%d -> tunnel port: %d", f.localIP, localPort, remotePort))
	return listener, nil
}

// listenOnRandomPort 本地端口被占用时，换一个随机端口
func (f *listenerFactory) listenOnRandomPort(remotePort, originalPort int) (net.Listener, error) {
	listener, err := net.Listen("tcp", net.JoinHostPort(f.localIP, "0"))
	if err != nil {
		return nil, err
	}
	actualPort := listener.Addr().(*net.TCPAddr).Port
	f.portOverrides[remotePort] = actualPort
	f.listeners = append(f.listeners, listener)
	f.addForwarding(fmt.Sprintf("Forwarding %s:%d -> tunnel port: %d (port %d in use)",
		f.localIP, actualPort, remotePort, originalPort))
	return listener, nil
}

func (f *listenerFactory) addForwarding(msg string) {
	f.mu.Lock()
	f.pendingForwardings = append(f.pendingForwardings, msg)
	count := len(f.pendingForwardings)
	f.mu.Unlock()

	if count >= f.expectedCount && f.expectedCount > 0 {
		select {
		case <-f.allReceived:
		default:
			close(f.allReceived)
		}
	}
}

func (f *listenerFactory) printForwardings() {
	f.mu.Lock()
	defer f.mu.Unlock()
	for _, msg := range f.pendingForwardings {
		f.logger.Info(msg)
	}
}

func (f *listenerFactory) reset() {
	f.mu.Lock()
	defer f.mu.Unlock()
	for _, l := range f.listeners {
		_ = l.Close()
	}
	f.listeners = nil
	f.pendingForwardings = nil
	f.allReceived = make(chan struct{})
}

func (f *listenerFactory) waitForForwardings(timeout time.Duration) {
	select {
	case <-f.allReceived:
	case <-time.After(timeout):
	}
}
