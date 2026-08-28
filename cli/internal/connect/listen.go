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

type relayPortMessage struct {
	Ports []uint16 `json:"ports"`
}

func Listen(tunnelID string, ports []int, jwtToken string, apiKey string) {
	ctx := context.Background()

	header, subprotocols := buildWSHeader(jwtToken, apiKey)
	header.Set("Cookie", "APP_COOKIE=7")

	sniHost := tunnelID + "." + serverHost
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

		shift := consecutiveFailures - 1
		if shift > 4 {
			shift = 4
		}
		delay := baseReconnectDelay << uint(shift)
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

type portNotifier struct {
	ready    chan struct{}
	ports    []int
	received atomic.Bool
}

func newPortNotifier() *portNotifier {
	return &portNotifier{ready: make(chan struct{})}
}

func setupOuterSession(ctx context.Context, netConn net.Conn, tunnelID string) (*ssh.ClientSession, chan struct{}, error) {
	outerConfig := ssh.NewNoSecurityConfig()
	outerConfig.KeepAliveIntervalSeconds = 10
	outerConfig.KeyRotationThreshold = 0
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
		_ = netConn.Close()
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
			_ = outerSession.Close()
		}
	}

	return outerSession, disconnected, nil
}

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

					if len(ports) == 0 {

						pn.ports = readPortNotification(channel)
					} else {

						readPortNotification(channel)
					}
					pn.received.Store(true)
					close(pn.ready)
					continue
				}

				effectivePorts := ports
				if len(effectivePorts) == 0 {
					effectivePorts = pn.ports
				}

				go handleRelayChannel(ctx, channel, tunnelID, effectivePorts)
			default:

				slog.Debug("host: draining non-relay channel", "channelType", channel.ChannelType, "channelID", channel.ChannelID)
			}
		}
	}()
}

func printListenReady(ports []int, tunnelID string) {
	for _, p := range ports {
		fmt.Printf("Hosting port: %s%d%s\n", colorCyan, p, colorReset)
	}
	for _, p := range ports {
		fmt.Printf("Tunnel URL: https://%s-%d.%s\n", tunnelID, p, serverHost)
	}
	fmt.Println("Ready to accept connections")
	fmt.Println("Auto reconnect: enabled")
}

func runListenSession(ctx context.Context, wsURL string, sniHost string, header http.Header, subprotocols []string, tunnelID string, ports []int) (connected bool, err error) {
	netConn, err := dialWebSocket(ctx, wsURL, sniHost, header, subprotocols, 5)
	if err != nil {
		return false, err
	}
	defer func() { _ = netConn.Close() }()

	outerSession, disconnected, err := setupOuterSession(ctx, netConn, tunnelID)
	if err != nil {
		return false, err
	}
	connected = true

	pn := newPortNotifier()
	startAcceptLoop(ctx, outerSession, tunnelID, ports, pn)

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

func handleRelayChannel(ctx context.Context, channel *ssh.Channel, tunnelID string, ports []int) {
	innerSession := createInnerServerSession(ctx, channel)
	if innerSession == nil {
		return
	}
	sessionLookup[channel.ChannelID] = innerSession

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
