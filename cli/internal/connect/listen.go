package connect

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"sync/atomic"
	"time"

	"github.com/coder/websocket"
	"github.com/microsoft/dev-tunnels-ssh/src/go/ssh"
	"github.com/microsoft/dev-tunnels-ssh/src/go/tcp"
)

// relayPortMessage 网关在 host 连接后通过 relay channel 下发的端口列表
type relayPortMessage struct {
	Ports []uint16 `json:"ports"`
}

func Listen(tunnelId string, ports []int, jwtToken string, apiKey string) {
	ctx := context.Background()

	header, subprotocols := buildWSHeader(jwtToken, apiKey)
	header.Set("Cookie", "APP_COOKIE=7")

	sniHost := tunnelId + "." + ServerHost
	wsURL := "wss://" + sniHost + "/" + tunnelId

	const maxReconnectAttempts = 5
	const baseReconnectDelay = 3 * time.Second
	const maxReconnectDelay = 30 * time.Second
	consecutiveFailures := 0
	everConnected := false
	for consecutiveFailures < maxReconnectAttempts {
		connected, err := runListenSession(ctx, wsURL, sniHost, header, subprotocols, tunnelId, ports)
		if ctx.Err() != nil {
			return
		}
		if errors.Is(err, errQuotaExceeded) || errors.Is(err, errTunnelNotFound) {
			slog.Error("connection rejected by gateway", "tunnelId", tunnelId, "err", err)
			return
		}
		if errors.Is(err, errDuplicateHost) && !everConnected {
			slog.Error("duplicate host, tunnel already has a listener", "tunnelId", tunnelId)
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
		delay := baseReconnectDelay
		for i := 0; i < consecutiveFailures-1; i++ {
			delay *= 2
			if delay >= maxReconnectDelay {
				delay = maxReconnectDelay
				break
			}
		}
		fmt.Println("Connection lost, reconnecting...")
		select {
		case <-ctx.Done():
			return
		case <-time.After(delay):
		}
	}
}

func runListenSession(ctx context.Context, wsURL string, sniHost string, header http.Header, subprotocols []string, tunnelId string, ports []int) (connected bool, err error) {
	netConn, err := dialWebSocket(ctx, wsURL, sniHost, header, subprotocols, 5)
	if err != nil {
		return false, err
	}
	defer netConn.Close()

	outerConfig := ssh.NewNoSecurityConfig()
	outerConfig.KeepAliveIntervalSeconds = 10
	outerConfig.KeyRotationThreshold = 0 // 禁用密钥轮换（no-security 模式无需轮换，且阈值不重置会导致频繁触发）
	tcp.AddPortForwardingService(outerConfig)
	outerSession := ssh.NewClientSession(outerConfig)
	outerSession.Trace = traceFunc()

	if err = outerSession.Connect(ctx, netConn); err != nil {
		var ce websocket.CloseError
		if errors.As(err, &ce) && ce.Code == websocket.StatusPolicyViolation {
			switch ce.Reason {
			case "account quota exceeded":
				return false, errQuotaExceeded
			case "tunnel not found":
				return false, errTunnelNotFound
			default:
				return false, errDuplicateHost
			}
		}
		netConn.Close()
		return false, fmt.Errorf("outer SSH client connect failed: %w", err)
	}
	slog.Debug("host: outer SSH session established", "tunnelId", tunnelId)
	connected = true

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
			slog.Error("keepalive failed 5 times, forcing reconnect", "tunnelId", tunnelId)
			outerSession.Close()
		}
	}

	// portNotificationReceived 标记是否已收到网关下发的端口列表通知
	// 网关在 host 连接后通过 sendRelayRequestMessage 主动开一个 relay channel 下发端口列表，
	// 这是第一个 relay channel；后续 send 端连接时网关再开 relay channel 用于数据转发。
	var portNotificationReceived atomic.Bool
	portsReady := make(chan struct{})
	// gatewayPorts: passive 模式下由 accept loop goroutine 写入（close(portsReady) 之前），
	// 主流程在 <-portsReady 之后读取，避免 data race
	var gatewayPorts []int

	// accept loop: 每个 relay channel 在独立 goroutine 中处理，避免阻塞后续 channel
	go func() {
		for {
			channel, err := outerSession.AcceptChannel(ctx)
			if err != nil {
				return
			}

			switch channel.ChannelType {
			case relayChannelType:
				if !portNotificationReceived.Load() {
					// 第一个 relay channel 是网关下发的端口列表通知
					if len(ports) == 0 {
						// passive 模式（--token）：从网关通知中获取端口列表
						gatewayPorts = readPortNotification(channel)
					} else {
						// active 模式：已有端口，读取后丢弃
						readPortNotification(channel)
					}
					// 第一个 relay channel 是网关下发的端口列表通知，读取后丢弃
					portNotificationReceived.Store(true)
					close(portsReady)
					continue
				}
				// 后续 relay channel 是 send 端连接，创建内层 SSH 会话
				effectivePorts := ports
				if len(effectivePorts) == 0 {
					effectivePorts = gatewayPorts
				}

				go handleRealRelayChannel(ctx, channel, tunnelId, effectivePorts)
			default:
				// drain non-relay channels to prevent acceptQueue overflow
				slog.Debug("host: draining non-relay channel", "channelType", channel.ChannelType, "channelID", channel.ChannelID)
			}
		}
	}()

	// passive 模式（--token）：等待网关下发端口列表后再打印
	printPorts := ports
	if len(ports) == 0 {
		select {
		case <-portsReady:
			printPorts = gatewayPorts
		case <-time.After(5 * time.Second):
			slog.Warn("timeout waiting for port notification from gateway")
		case <-disconnected:
			return true, fmt.Errorf("disconnected")
		case <-outerSession.Session.Done():
			return true, fmt.Errorf("session closed")
		}
	}

	for _, p := range printPorts {
		fmt.Printf("Hosting port: %s%d%s\n", colorCyan, p, colorReset)
	}
	for _, p := range printPorts {
		fmt.Printf("Tunnel URL: https://%s-%d.%s\n", tunnelId, p, ServerHost)
	}
	fmt.Println("Ready to accept connections")
	fmt.Println("Auto reconnect: enabled")

	for {
		select {
		case <-disconnected:
			return true, fmt.Errorf("disconnected")
		case <-outerSession.Session.Done():
			return true, fmt.Errorf("session closed")
		}
	}
}

// readPortNotification 读取网关下发的端口列表通知，返回端口列表
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

// handleRealRelayChannel 处理 send 端连接的 relay channel，创建内层 SSH ServerSession
// 创建完成后，通过 ForwardFromRemotePort 向 send 端发送 tcpip-forward 请求，
// 让 send 端自动在本地监听端口并建立转发
func handleRealRelayChannel(ctx context.Context, channel *ssh.Channel, tunnelId string, ports []int) {
	innerSession := createInnerServerSession(ctx, channel)
	if innerSession == nil {
		return
	}
	sessionLookup[channel.ChannelID] = innerSession

	// 通过 SSH 协议通知 send 端建立端口转发
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
