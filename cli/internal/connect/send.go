package connect

import (
	"context"
	"fmt"
	"log/slog"
	"net"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"sync"
	"syscall"
	"time"

	"github.com/microsoft/dev-tunnels-ssh/src/go/ssh"
	"github.com/microsoft/dev-tunnels-ssh/src/go/tcp"
)

const (
	colorCyan   = "\033[36m"
	colorYellow = "\033[33m"
	colorReset  = "\033[0m"
)

type listenerFactory struct {
	mu                 sync.Mutex
	pendingForwardings []string
	expectedCount      int
	allReceived        chan struct{}
	portOverrides      map[int]int    // remotePort -> 实际使用的 localPort，重连时复用
	listeners          []net.Listener // 已创建的监听器，重连前关闭以释放端口
}

func newListenerFactory(expectedCount int) *listenerFactory {
	return &listenerFactory{
		expectedCount: expectedCount,
		allReceived:   make(chan struct{}),
		portOverrides: make(map[int]int),
	}
}

func (f *listenerFactory) CreateTCPListener(
	remotePort int,
	localIPAddress string,
	localPort int,
	canChangeLocalPort bool,
) (net.Listener, error) {
	// 重连时复用上次实际使用的端口
	if override, ok := f.portOverrides[remotePort]; ok {
		localPort = override
	}
	// 先尝试连接目标端口，如果能连上说明已被其他进程占用
	// Windows 上 0.0.0.0:port 和 127.0.0.1:port 可以共存，net.Listen 不会报错，
	// 但实际端口已被占用，需要通过 net.Dial 预检测发现
	if localPort != 0 {
		conn, err := net.Dial("tcp", net.JoinHostPort(localIPAddress, strconv.Itoa(localPort)))
		if err == nil {
			conn.Close()
			if canChangeLocalPort {
				randomListener, listenErr := net.Listen("tcp", net.JoinHostPort(localIPAddress, strconv.Itoa(0)))
				if listenErr != nil {
					return nil, listenErr
				}
				actualPort := randomListener.Addr().(*net.TCPAddr).Port
				f.portOverrides[remotePort] = actualPort
				f.listeners = append(f.listeners, randomListener)
				f.addForwarding(fmt.Sprintf("Forwarding localhost: %s%d%s -> tunnel port: %s%d%s (port %s%d%s in use)\n",
					colorCyan, actualPort, colorReset, colorCyan, remotePort, colorReset, colorYellow, localPort, colorReset))
				return randomListener, nil
			}
			return nil, fmt.Errorf("port %d is already in use", localPort)
		}
	}

	listener, err := net.Listen("tcp", net.JoinHostPort(localIPAddress, strconv.Itoa(localPort)))
	if err != nil && canChangeLocalPort {
		randomListener, listenErr := net.Listen("tcp", net.JoinHostPort(localIPAddress, strconv.Itoa(0)))
		if listenErr != nil {
			return nil, listenErr
		}
		actualPort := randomListener.Addr().(*net.TCPAddr).Port
		f.portOverrides[remotePort] = actualPort
		f.listeners = append(f.listeners, randomListener)
		f.addForwarding(fmt.Sprintf("Forwarding localhost: %s%d%s -> tunnel port: %s%d%s (port %s%d%s in use)\n",
			colorCyan, actualPort, colorReset, colorCyan, remotePort, colorReset, colorYellow, localPort, colorReset))
		return randomListener, nil
	}
	if err != nil {
		return nil, err
	}
	f.portOverrides[remotePort] = localPort
	f.listeners = append(f.listeners, listener)
	f.addForwarding(fmt.Sprintf("Forwarding localhost: %s%d%s -> tunnel port: %s%d%s\n",
		colorCyan, localPort, colorReset, colorCyan, remotePort, colorReset))
	return listener, nil
}

func (f *listenerFactory) addForwarding(msg string) {
	f.mu.Lock()
	f.pendingForwardings = append(f.pendingForwardings, msg)
	count := len(f.pendingForwardings)
	f.mu.Unlock()
	// 所有期望端口都到齐，通知等待方
	if count >= f.expectedCount && f.expectedCount > 0 {
		select {
		case <-f.allReceived:
		default:
			close(f.allReceived)
		}
	}
}

func (f *listenerFactory) PrintForwardings() {
	f.mu.Lock()
	defer f.mu.Unlock()
	for _, msg := range f.pendingForwardings {
		fmt.Print(msg)
	}
}

// reset 重连前重置状态：关闭上一轮的监听器释放端口，保留 portOverrides（端口复用）
func (f *listenerFactory) reset() {
	f.mu.Lock()
	defer f.mu.Unlock()
	for _, l := range f.listeners {
		l.Close()
	}
	f.listeners = nil
	f.pendingForwardings = nil
	f.allReceived = make(chan struct{})
}

// waitForForwardings 等待所有期望端口到齐或超时
func (f *listenerFactory) waitForForwardings(timeout time.Duration) {
	select {
	case <-f.allReceived:
	case <-time.After(timeout):
	}
}

// Connect 启动发送端连接到网关，等待 host 端的端口转发请求
// tunnelId: 隧道ID
// jwtToken: JWT 认证 token
// ports: 端口列表（仅用于模式提示日志，实际端口转发由 host 端通过 ForwardFromRemotePort 下发）
func Connect(tunnelId string, jwtToken string, ports []int, apiKey string) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		<-sigCh
		cancel()
	}()

	header, subprotocols := buildWSHeader(jwtToken, apiKey)

	sniHost := tunnelId + "." + ServerHost
	wsURL := "wss://" + sniHost + "/"

	factory := newListenerFactory(len(ports))
	const maxReconnectAttempts = 5
	const baseReconnectDelay = 3 * time.Second
	const maxReconnectDelay = 30 * time.Second
	consecutiveFailures := 0
	for consecutiveFailures < maxReconnectAttempts {
		connected, err := runSendSession(ctx, wsURL, sniHost, header, subprotocols, tunnelId, ports, factory)
		if ctx.Err() != nil {
			return
		}
		if err == nil {
			return
		}
		if connected {
			consecutiveFailures = 0
		} else {
			consecutiveFailures++
		}
		if consecutiveFailures >= maxReconnectAttempts {
			slog.Error("reconnect exhausted", "maxAttempts", maxReconnectAttempts, "err", err)
			return
		}
		delay := baseReconnectDelay
		for i := 0; i < consecutiveFailures-1; i++ {
			delay *= 2
			if delay >= maxReconnectDelay {
				delay = maxReconnectDelay
				break
			}
		}
		fmt.Printf("Connection lost, reconnecting... (%v)\n", err)
		select {
		case <-ctx.Done():
			return
		case <-time.After(delay):
		}
	}
}

func runSendSession(ctx context.Context, wsURL string, sniHost string, header http.Header, subprotocols []string, tunnelId string, ports []int, factory *listenerFactory) (connected bool, err error) {
	netConn, err := dialWebSocket(ctx, wsURL, sniHost, header, subprotocols, 5)
	if err != nil {
		return false, fmt.Errorf("WebSocket connection failed: %w", err)
	}

	config := ssh.NewNoSecurityConfig()
	config.KeepAliveIntervalSeconds = 10
	tcp.AddPortForwardingService(config)

	session := ssh.NewClientSession(config)
	session.Trace = traceFunc()
	defer session.Close()

	// 在 Connect 之前设置 ListenerFactory
	// Host 端在 SSH 握手完成后会立即发 ForwardFromRemotePort，
	// 如果在 Connect/Authenticate 之后才设置，PFS 会用默认 factory 静默处理
	pfs := tcp.GetPortForwardingService(&session.Session)
	if pfs == nil {
		netConn.Close()
		return false, fmt.Errorf("port forwarding service unavailable")
	}
	factory.reset()
	pfs.ListenerFactory = factory

	if err = session.Connect(ctx, netConn); err != nil {
		netConn.Close()
		return false, fmt.Errorf("SSH client connect failed: %w", err)
	}
	connected = true

	fmt.Printf("Connected to tunnel: %s\n", tunnelId)

	if len(ports) > 0 {
		fmt.Println("Mode: active forwarding (ports from API)")
	} else {
		fmt.Println("Mode: passive forwarding (ports from host via SSH)")
	}

	// 等待端口转发到齐后统一打印
	// 传统模式：知道端口数，到齐就提前打印，3 秒超时兜底
	// -token 模式：不知道端口数，等 1 秒收集后打印
	if len(ports) > 0 {
		factory.waitForForwardings(3 * time.Second)
	} else {
		factory.waitForForwardings(2 * time.Second)
	}

	// 统一打印端口转发日志，避免和主流程日志交叉
	factory.PrintForwardings()

	fmt.Println("Auto reconnect: enabled")

	select {
	case <-session.Session.Done():
		return true, fmt.Errorf("session closed")
	case <-ctx.Done():
		return false, ctx.Err()
	}
}
