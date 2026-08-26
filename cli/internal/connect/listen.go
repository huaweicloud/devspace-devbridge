package connect

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"net"
	"net/http"
	"sync/atomic"
	"time"

	"github.com/coder/websocket"
	"github.com/microsoft/dev-tunnels-ssh/src/go/ssh"
	"github.com/microsoft/dev-tunnels-ssh/src/go/tcp"
)

// relayPortMessage 网关在 host 连接后通过 relay channel 下发的端口列表.
type relayPortMessage struct {
	Ports []uint16 `json:"ports"`
}

func Listen(tunnelID string, ports []int, jwtToken string, apiKey string) {
	ctx := context.Background()

	header, subprotocols := buildWSHeader(jwtToken, apiKey)
	header.Set("Cookie", "APP_COOKIE=7")

	sniHost := tunnelID + "." + ServerHost
	wsURL := "wss://" + sniHost + "/" + tunnelID

	const maxReconnectAttempts = 5
	const baseReconnectDelay = 3 * time.Second
	const maxReconnectDelay = 30 * time.Second
	consecutiveFailures := 0
	everConnected := false
	for consecutiveFailures < maxReconnectAttempts {
		connected, err := runListenSession(ctx, wsURL, sniHost, header, subprotocols, tunnelID, ports)
		if ctx.Err() != nil {
			return
		}
		if errors.Is(err, errQuotaExceeded) || errors.Is(err, errTunnelNotFound) {
			slog.Error("connection rejected by gateway", "tunnelID", tunnelID, "err", err)
			return
		}
		if errors.Is(err, errDuplicateHost) && !everConnected {
			slog.Error("duplicate host, tunnel already has a listener", "tunnelID", tunnelID)
			return
		}
		if err == nil {
			return
		}
		if connected {
			consecutiveFailures = 0
			everConnected = true
		} else {
			consecutiveFailures++
		}
		if consecutiveFailures >= maxReconnectAttempts {
			slog.Error("reconnect exhausted", "maxAttempts", maxReconnectAttempts, "err", err)
			return
		}
		// 指数退避：用位移替代循环累乘.
		shift := consecutiveFailures - 1
		if shift > 4 { //nolint:gomnd // baseReconnectDelay=3s, maxReconnectDelay=30s, 最多移 4 位
			shift = 4
		}
		delay := baseReconnectDelay << uint(shift) //nolint:gosec // shift 已截断到 0-4，不会溢出
		if delay > maxReconnectDelay {
			delay = maxReconnectDelay
		}
		fmt.Println("Connection lost, reconnecting...")
		select {
		case <-ctx.Done():
			return
		case <-time.After(delay):
		}
	}
}

// portNotifier 封装网关端口通知的同步逻辑.
// accept loop 在收到第一个 relay channel 时写入 ports 并 close ready
type portNotifier struct {
	ready    chan struct{}
	ports    []int // 在 ready 关闭前由 accept loop 写入，主流程在 <-ready 后读取
	received atomic.Bool
}

func newPortNotifier() *portNotifier {
	return &portNotifier{ready: make(chan struct{})}
}

// setupOuterSession 建立外层 SSH 客户端会话并设置断连/保活回调.
func setupOuterSession(ctx context.Context, netConn net.Conn, tunnelID string) (*ssh.ClientSession, chan struct{}, error) {
	outerConfig := ssh.NewNoSecurityConfig()
	outerConfig.KeepAliveIntervalSeconds = 10
	outerConfig.KeyRotationThreshold = 0 // 禁用密钥轮换（no-security 模式无需轮换，且阈值不重置会导致频繁触发）.
	tcp.AddPortForwardingService(outerConfig)
	outerSession := ssh.NewClientSession(outerConfig)
	outerSession.Trace = traceFunc()

	if err := outerSession.Connect(ctx, netConn); err != nil {
		var ce websocket.CloseError
		if errors.As(err, &ce) && ce.Code == websocket.StatusPolicyViolation {
			switch ce.Reason {
			case "account quota exceeded":
				return nil, nil, errQuotaExceeded
			case "tunnel not found":
				return nil, nil, errTunnelNotFound
			default:
				return nil, nil, errDuplicateHost
			}
		}
		_ = netConn.Close() //nolint:errcheck // 错误路径中关闭连接失败不可操作
		return nil, nil, fmt.Errorf("outer SSH client connect failed: %w", err)
	}
	slog.Debug("host: outer SSH session established", "tunnelID", tunnelID)

	disconnected := make(chan struct{}, 1)
	outerSession.OnDisconnected = func() {
		select {
		case disconnected <- struct{}{}:
		default:
		}
	}
	outerSession.OnKeepAliveSucceeded = func(count int) {
		slog.Debug("keepalive ok", "count", count)
	}
	outerSession.OnKeepAliveFailed = func(count int) {
		slog.Debug("keepalive failed", "count", count)
		if count >= 5 {
			slog.Error("keepalive failed 5 times, forcing reconnect", "tunnelID", tunnelID)
			_ = outerSession.Close() //nolint:errcheck // 强制重连时关闭失败不可操作
		}
	}

	return outerSession, disconnected, nil
}

// startAcceptLoop 启动 relay channel 接收循环，每个 relay channel 在独立 goroutine 中处理.
func startAcceptLoop(ctx context.Context, outerSession *ssh.ClientSession, tunnelID string, ports []int, pn *portNotifier) {
	go func() {
		for {
			channel, err := outerSession.AcceptChannel(ctx)
			if err != nil {
				return
			}

			switch channel.ChannelType {
			case relayChannelType:
				if !pn.received.Load() {
					// 第一个 relay channel 是网关下发的端口列表通知.
					if len(ports) == 0 {
						// Passive 模式（--token）：从网关通知中获取端口列表
						pn.ports = readPortNotification(channel)
					} else {
						// Active 模式：已有端口，读取后丢弃
						readPortNotification(channel)
					}
					pn.received.Store(true)
					close(pn.ready)
					continue
				}
				// 后续 relay channel 是 send 端连接，创建内层 SSH 会话.
				effectivePorts := ports
				if len(effectivePorts) == 0 {
					effectivePorts = pn.ports
				}

				go handleRelayChannel(ctx, channel, tunnelID, effectivePorts)
			default:
				// Drain non-relay channels to prevent acceptQueue overflow
				slog.Debug("host: draining non-relay channel", "channelType", channel.ChannelType, "channelID", channel.ChannelID)
			}
		}
	}()
}

// printListenReady 打印监听就绪信息（端口、隧道 URL、提示）.
func printListenReady(ports []int, tunnelID string) {
	for _, p := range ports {
		fmt.Printf("Hosting port: %s%d%s\n", colorCyan, p, colorReset)
	}
	for _, p := range ports {
		fmt.Printf("Tunnel URL: https://%s-%d.%s\n", tunnelID, p, ServerHost)
	}
	fmt.Println("Ready to accept connections")
	fmt.Println("Auto reconnect: enabled")
}

func runListenSession(ctx context.Context, wsURL string, sniHost string, header http.Header, subprotocols []string, tunnelID string, ports []int) (connected bool, err error) {
	netConn, err := dialWebSocket(ctx, wsURL, sniHost, header, subprotocols, 5)
	if err != nil {
		return false, err
	}
	defer func() { _ = netConn.Close() }() //nolint:errcheck // 连接关闭失败不可操作

	outerSession, disconnected, err := setupOuterSession(ctx, netConn, tunnelID)
	if err != nil {
		return false, err
	}
	connected = true

	pn := newPortNotifier()
	startAcceptLoop(ctx, outerSession, tunnelID, ports, pn)

	// Passive 模式（--token）：等待网关下发端口列表后再打印
	printPorts := ports
	if len(ports) == 0 {
		select {
		case <-pn.ready:
			printPorts = pn.ports
		case <-time.After(5 * time.Second):
			slog.Warn("timeout waiting for port notification from gateway")
		case <-disconnected:
			return true, fmt.Errorf("disconnected")
		case <-outerSession.Session.Done():
			return true, fmt.Errorf("session closed")
		}
	}

	printListenReady(printPorts, tunnelID)

	for {
		select {
		case <-disconnected:
			return true, fmt.Errorf("disconnected")
		case <-outerSession.Session.Done():
			return true, fmt.Errorf("session closed")
		}
	}
}

// readPortNotification 读取网关下发的端口列表通知，返回端口列表.
// 网关在 host 连接后通过 sendRelayRequestMessage 主动开 relay channel 发送 {"ports":[...]}
func readPortNotification(channel *ssh.Channel) []int {
	stream := ssh.NewStream(channel)
	buf := make([]byte, 4096)
	n, err := stream.Read(buf)
	if err != nil {
		return nil
	}
	var msg relayPortMessage
	if err := json.Unmarshal(buf[:n], &msg); err != nil {
		slog.Warn("failed to parse port notification", "err", err, "data", string(buf[:n]))
		return nil
	}
	ports := make([]int, len(msg.Ports))
	for i, p := range msg.Ports {
		ports[i] = int(p)
	}
	return ports
}

// handleRelayChannel 处理 send 端连接的 relay channel，创建内层 SSH ServerSession.
// 创建完成后，通过 ForwardFromRemotePort 向 send 端发送 tcpip-forward 请求，
// 让 send 端自动在本地监听端口并建立转发
func handleRelayChannel(ctx context.Context, channel *ssh.Channel, tunnelID string, ports []int) {
	innerSession := createInnerServerSession(ctx, channel)
	if innerSession == nil {
		return
	}
	sessionLookup[channel.ChannelID] = innerSession

	// 通过 SSH 协议通知 send 端建立端口转发.
	// forwardPort 通过 ForwardFromRemotePort 让 send 端监听端口，转发目标为实际端口
	// 数据流：send端 → PFS channel → host inner ServerSession → PFS 连接 127.0.0.1:actualPort
	pfs := tcp.GetPortForwardingService(&innerSession.Session)
	if pfs != nil && len(ports) > 0 {
		for _, port := range ports {
			if _, err := pfs.ForwardFromRemotePort(ctx, "127.0.0.1", port, "127.0.0.1", port); err != nil {
				slog.Error("forward port failed", "port", port, "err", err)
			}
		}
	}

	go func(chID uint32) {
		for {
			_, err := innerSession.AcceptChannel(ctx)
			if err != nil {
				return
			}
		}
	}(channel.ChannelID)

	go func(chID uint32) {
		<-innerSession.Session.Done()
		delete(sessionLookup, chID)
	}(channel.ChannelID)
}

func createInnerServerSession(ctx context.Context, channel *ssh.Channel) *ssh.ServerSession {

	innerConfig := ssh.NewNoSecurityConfig()
	tcp.AddPortForwardingService(innerConfig)
	innerSession := ssh.NewServerSession(innerConfig)
	innerSession.Credentials = &ssh.ServerCredentials{PublicKeys: []ssh.KeyPair{persistentHostKey}}
	innerSession.Trace = traceFunc()

	if err := innerSession.Connect(ctx, ssh.NewStream(channel)); err != nil {
		slog.Error("inner SSH session failed", "channelID", channel.ChannelID, "err", err)
		return nil
	}

	slog.Debug("host: inner SSH server session established", "channelID", channel.ChannelID)
	return innerSession
}
