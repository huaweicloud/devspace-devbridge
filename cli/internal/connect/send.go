package connect

import (
	"context"
	"fmt"
	"log/slog"
	"net"
	"net/http"
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
	portOverrides      map[int]int
	listeners          []net.Listener
	expectedCount      int
	allReceived        chan struct{}
}

func newListenerFactory(expectedCount int) *listenerFactory {
	return &listenerFactory{
		expectedCount: expectedCount,
		allReceived:   make(chan struct{}),
		portOverrides: make(map[int]int),
	}
}

func (f *listenerFactory) listenOnRandomPort(localIPAddress string, remotePort, originalPort int) (net.Listener, error) {
	listener, err := net.Listen("tcp", net.JoinHostPort(localIPAddress, "0"))
	if err != nil {
		return nil, err
	}
	actualPort := listener.Addr().(*net.TCPAddr).Port
	f.portOverrides[remotePort] = actualPort
	f.listeners = append(f.listeners, listener)
	f.addForwarding(fmt.Sprintf("Forwarding localhost: %s%d%s -> tunnel port: %s%d%s (port %s%d%s in use)\n",
		colorCyan, actualPort, colorReset, colorCyan, remotePort, colorReset, colorYellow, originalPort, colorReset))
	return listener, nil
}

func (f *listenerFactory) CreateTCPListener(
	remotePort int,
	localIPAddress string,
	localPort int,
	canChangeLocalPort bool,
) (net.Listener, error) {

	if override, ok := f.portOverrides[remotePort]; ok {
		localPort = override
	}

	listener, err := net.Listen("tcp", net.JoinHostPort(localIPAddress, strconv.Itoa(localPort)))
	if err != nil {
		if canChangeLocalPort {
			return f.listenOnRandomPort(localIPAddress, remotePort, localPort)
		}
		return nil, fmt.Errorf("port %d is already in use", localPort)
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
		fmt.Print(msg)
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

// Connect 启动发送端连接到网关，等待 host 端下发端口转发。
func Connect(tunnelID string, jwtToken string, ports []int, apiKey string) {
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	header, subprotocols := buildWSHeader(jwtToken, apiKey)

	sniHost := tunnelID + "." + ServerHost
	wsURL := "wss://" + sniHost + "/"

	factory := newListenerFactory(len(ports))
	const maxReconnectAttempts = 5
	const baseReconnectDelay = 3 * time.Second
	const maxReconnectDelay = 30 * time.Second
	consecutiveFailures := 0
	for consecutiveFailures < maxReconnectAttempts {
		connected, err := runSendSession(ctx, wsURL, sniHost, header, subprotocols, tunnelID, ports, factory)
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

func runSendSession(ctx context.Context, wsURL string, sniHost string, header http.Header, subprotocols []string, tunnelID string, ports []int, factory *listenerFactory) (connected bool, err error) {
	netConn, err := dialWebSocket(ctx, wsURL, sniHost, header, subprotocols, 5)
	if err != nil {
		return false, fmt.Errorf("WebSocket connection failed: %w", err)
	}

	config := ssh.NewNoSecurityConfig()
	config.KeepAliveIntervalSeconds = 10
	tcp.AddPortForwardingService(config)

	session := ssh.NewClientSession(config)
	session.Trace = traceFunc()
	defer func() { _ = session.Close() }()

	pfs := tcp.GetPortForwardingService(&session.Session)
	if pfs == nil {
		_ = netConn.Close()
		return false, fmt.Errorf("port forwarding service unavailable")
	}
	factory.reset()
	pfs.ListenerFactory = factory

	if err = session.Connect(ctx, netConn); err != nil {
		_ = netConn.Close()
		return false, fmt.Errorf("SSH client connect failed: %w", err)
	}
	connected = true

	fmt.Printf("Connected to tunnel: %s\n", tunnelID)

	if len(ports) > 0 {
		fmt.Println("Mode: active forwarding (ports from API)")
	} else {
		fmt.Println("Mode: passive forwarding (ports from host via SSH)")
	}

	if len(ports) > 0 {
		factory.waitForForwardings(3 * time.Second)
	} else {
		factory.waitForForwardings(2 * time.Second)
	}

	factory.printForwardings()

	fmt.Println("Auto reconnect: enabled")

	select {
	case <-session.Session.Done():
		return true, fmt.Errorf("session closed")
	case <-ctx.Done():
		return false, ctx.Err()
	}
}
