/*
 * Copyright (c) Huawei Technologies Co., Ltd. 2026-2027. All rights reserved.
 */

package sdk

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"sync"
	"sync/atomic"
	"time"

	"github.com/microsoft/dev-tunnels-ssh/src/go/ssh"
	"github.com/microsoft/dev-tunnels-ssh/src/go/tcp"
)

// HostConfig is the configuration for hosting a tunnel.
type HostConfig struct {
	TunnelID string // tunnel ID
	Ports    []int  // local ports to forward (empty means issued by the gateway)
	JWTToken string // JWT token (mutually exclusive with APIKey)
	APIKey   string // API key (mutually exclusive with JWTToken)

	// OnReady is called once the session is ready (ports start forwarding), with the list of hosted ports.
	// It fires once per established connection or reconnect. May be nil.
	OnReady func(ports []int)
}

type relayPortMessage struct {
	Type  string   `json:"@type"`
	Ports []uint16 `json:"ports"`
}

var (
	hostKeyOnce       sync.Once
	persistentHostKey ssh.KeyPair
	hostKeyErr        error
)

// ensureHostKey generates a process-wide SSH host key.
// If generation fails, the error is cached and returned on subsequent calls, avoiding silent use of a nil key.
func ensureHostKey() error {
	hostKeyOnce.Do(func() {
		persistentHostKey, hostKeyErr = ssh.GenerateKeyPair(ssh.AlgoPKEcdsaSha2P256)
	})
	return hostKeyErr
}

// Host starts the Host hosting service.
//
// This is a blocking call that returns when ctx is canceled or the connection is fully lost;
// brief network interruptions are reconnected automatically.
//
// Workflow:
//  1. Connect WebSocket to wss://<tunnelId>.<gatewayHost>/<tunnelId>
//  2. Establish an SSH session over the WebSocket
//  3. Accept relay channels and create an inner SSH session for each
//  4. Forward traffic from the remote side to the local port
func (d *Devbridge) Host(ctx context.Context, cfg HostConfig) error {
	if err := ensureHostKey(); err != nil {
		return fmt.Errorf("generate host key: %w", err)
	}
	if err := validateTunnelID(cfg.TunnelID); err != nil {
		return err
	}

	// Auth fallback: use the Devbridge instance's API key when cfg does not specify one.
	apiKey := cfg.APIKey
	if apiKey == "" && cfg.JWTToken == "" {
		apiKey = d.apiKey
	}

	header, subprotocols := buildWSHeader(cfg.JWTToken, apiKey)
	header.Set("Cookie", "APP_COOKIE=7")

	sniHost := cfg.TunnelID + "." + d.gatewayHost
	wsURL := "wss://" + sniHost + "/" + cfg.TunnelID

	everConnected := false
	return d.reconnectLoop(ctx, func() (bool, error) {
		return d.runHostSession(ctx, wsURL, sniHost, header, subprotocols, cfg.TunnelID, cfg.Ports, cfg.OnReady)
	}, reconnectDecision{
		// 网关明确拒绝（额度超限/隧道不存在）或首次连接前就报重复 host：直接终止，不重试。
		// 注意：onConnected 在 shouldStop 之后才执行，因此此处的 everConnected 是上一次会话的值，
		// 与原实现“先判断 duplicate、后更新 everConnected”的语义完全一致。
		shouldStop: func(err error) bool {
			if errors.Is(err, ErrDuplicateHost) && !everConnected {
				d.logger.Debug("duplicate host, tunnel already has a listener", "tunnelID", cfg.TunnelID)
				return true
			}
			if gatewayRejectedError(err) {
				d.logger.Debug("connection rejected by gateway", "tunnelID", cfg.TunnelID, "err", err)
				return true
			}
			return false
		},
		onReconnect: func(err error) {
			d.statusln("Connection lost, reconnecting...")
		},
		onConnected: func() {
			everConnected = true
		},
	})
}

func (d *Devbridge) runHostSession(ctx context.Context, wsURL string, sniHost string, header http.Header, subprotocols []string, tunnelID string, ports []int, onReady func([]int)) (connected bool, err error) {
	netConn, err := d.dialWebSocket(ctx, wsURL, sniHost, header, subprotocols, 5)
	if err != nil {
		if errors.Is(err, ErrDuplicateHost) {
			return false, fmt.Errorf("outer SSH connect failed: %w", err)
		}
		return false, err
	}
	defer func() { _ = netConn.Close() }()

	outerConfig := ssh.NewNoSecurityConfig()
	outerConfig.KeepAliveIntervalSeconds = 10
	outerConfig.KeyRotationThreshold = 0
	tcp.AddPortForwardingService(outerConfig)
	outerSession := ssh.NewClientSession(outerConfig)
	outerSession.Trace = sshTraceFunc(d.logger)

	if err := outerSession.Connect(ctx, netConn); err != nil {
		err = parseSSHCloseError(err)
		_ = netConn.Close()
		return false, fmt.Errorf("outer SSH connect failed: %w", err)
	}
	d.logger.Debug("host: outer SSH session established", "tunnelID", tunnelID)
	connected = true

	disconnected := make(chan struct{}, 1)
	outerSession.OnDisconnected = func() {
		select {
		case disconnected <- struct{}{}:
		default:
		}
	}

	// 诊断：捕获底层会话关闭原因，区分"网关主动掐断"（DisconnectByApplication）
	// 与"网络断开/keepalive 超时"（DisconnectConnectionLost）。
	var (
		closedArgsMu sync.Mutex
		closedArgs   *ssh.SessionClosedEventArgs
	)
	outerSession.OnClosed = func(args *ssh.SessionClosedEventArgs) {
		closedArgsMu.Lock()
		closedArgs = args
		closedArgsMu.Unlock()
		d.logger.Warn("host: session closed (diagnostic)",
			"tunnelID", tunnelID,
			"reason", args.Reason,
			"message", args.Message,
			"err", args.Err)
	}
	outerSession.OnKeepAliveFailed = func(count int) {
		if count >= 5 {
			d.logger.Error("keepalive failed 5 times, forcing reconnect", "tunnelID", tunnelID)
			_ = outerSession.Close()
		}
	}

	pn := newPortNotifier()

	go d.startHostAcceptLoop(ctx, outerSession, tunnelID, ports, pn)

	if len(ports) == 0 {
		select {
		case <-pn.ready:
		case <-time.After(5 * time.Second):
			d.logger.Warn("timeout waiting for port notification from gateway")
		case <-disconnected:
			d.logDiagnosticClose(tunnelID, &closedArgsMu, &closedArgs)
			return true, fmt.Errorf("disconnected")
		case <-outerSession.Session.Done():
			d.logDiagnosticClose(tunnelID, &closedArgsMu, &closedArgs)
			return true, fmt.Errorf("session closed")
		}
	}

	printPorts := ports
	if len(ports) == 0 {
		printPorts = pn.ports
	}

	// Filter out the "all ports" sentinel value: it only matters for visitor URL access,
	// and host does not need to start a local port forward for -1.
	realPorts := filterForwardPorts(printPorts)
	if len(realPorts) == 0 && len(printPorts) > 0 {
		d.statusln("All ports mode: this tunnel accepts any port via URL")
		d.statusf("Access your service at: https://%s-<port>.%s\n", tunnelID, d.gatewayHost)
	}
	for _, p := range realPorts {
		d.statusf("Hosting port: %s%d%s\n", colorCyan, p, colorReset)
	}
	for _, p := range realPorts {
		d.statusf("Tunnel URL: https://%s-%d.%s\n", tunnelID, p, d.gatewayHost)
	}
	d.statusln("Ready to accept connections")
	d.statusln("Auto reconnect: enabled")

	if onReady != nil {
		onReady(realPorts)
	}

	for {
		select {
		case <-disconnected:
			d.logDiagnosticClose(tunnelID, &closedArgsMu, &closedArgs)
			return true, fmt.Errorf("disconnected")
		case <-outerSession.Session.Done():
			d.logDiagnosticClose(tunnelID, &closedArgsMu, &closedArgs)
			return true, fmt.Errorf("session closed")
		case <-ctx.Done():
			return true, nil
		}
	}
}

// logDiagnosticClose 打印 host 连接丢失时的底层关闭原因（诊断用）。
// reason=DisconnectByApplication 说明对端(网关)主动关；DisconnectConnectionLost 说明网络层断开。
func (d *Devbridge) logDiagnosticClose(tunnelID string, mu *sync.Mutex, args **ssh.SessionClosedEventArgs) {
	mu.Lock()
	a := *args
	mu.Unlock()
	if a == nil {
		d.logger.Error("host connection lost (no close event captured)", "tunnelID", tunnelID)
		return
	}
	d.logger.Error("host connection lost",
		"tunnelID", tunnelID,
		"reason", a.Reason,
		"message", a.Message,
		"err", a.Err)
}

type portNotifier struct {
	ready    chan struct{}
	ports    []int
	received atomic.Bool
}

func newPortNotifier() *portNotifier {
	return &portNotifier{ready: make(chan struct{})}
}

func (d *Devbridge) startHostAcceptLoop(ctx context.Context, outerSession *ssh.ClientSession, tunnelID string, ports []int, pn *portNotifier) {
	for {
		channel, err := outerSession.AcceptChannel(ctx)
		if err != nil {
			return
		}

		switch channel.ChannelType {
		case relayChannelType:
			if !pn.received.Load() {
				// 第一个 relay channel 按协议是网关的端口通知（RelayRequest）。
				// 但以防协议演进，用内容校验而非位置约定：解析出 RelayRequest 才当通知消费，
				// 否则视为转发请求走正常转发，避免"无条件吞掉第一个非通知 channel"。
				notifPorts, isNotif := readPortNotification(channel)
				if isNotif {
					if len(ports) == 0 {
						pn.ports = notifPorts
					}
					pn.received.Store(true)
					close(pn.ready)
					continue
				}
			}

			effectivePorts := ports
			if len(effectivePorts) == 0 {
				effectivePorts = pn.ports
			}
			go d.handleRelayChannel(ctx, channel, tunnelID, effectivePorts)
		default:
			d.logger.Debug("host: draining non-relay channel",
				"channelType", channel.ChannelType, "channelID", channel.ChannelID)
		}
	}
}

// readPortNotification reads one relay channel and reports whether it is the gateway's
// port notification (RelayRequest). It returns false when the payload is not a valid
// RelayRequest message, so the caller must treat the channel as a data-forwarding
// channel instead of consuming/discarding it.
func readPortNotification(channel *ssh.Channel) ([]int, bool) {
	stream := ssh.NewStream(channel)
	buf := make([]byte, 4096)
	n, err := stream.Read(buf)
	if err != nil {
		return nil, false
	}
	return parsePortNotification(buf[:n])
}

// parsePortNotification parses a gateway port-notification payload.
// A valid notification is a RelayRequest message (@type="RelayRequest"); anything
// else (garbage, empty, or a different message type) is not a port notification.
func parsePortNotification(data []byte) ([]int, bool) {
	var msg relayPortMessage
	if err := json.Unmarshal(data, &msg); err != nil {
		return nil, false
	}
	if msg.Type != "RelayRequest" {
		return nil, false
	}
	ports := make([]int, len(msg.Ports))
	for i, p := range msg.Ports {
		ports[i] = int(p)
	}
	return ports, true
}

func (d *Devbridge) handleRelayChannel(ctx context.Context, channel *ssh.Channel, tunnelID string, ports []int) {
	innerConfig := ssh.NewNoSecurityConfig()
	tcp.AddPortForwardingService(innerConfig)
	innerSession := ssh.NewServerSession(innerConfig)
	innerSession.Credentials = &ssh.ServerCredentials{PublicKeys: []ssh.KeyPair{persistentHostKey}}
	innerSession.Trace = sshTraceFunc(d.logger)

	if err := innerSession.Connect(ctx, ssh.NewStream(channel)); err != nil {
		d.logger.Error("inner SSH session failed", "channelID", channel.ChannelID, "err", err)
		return
	}
	d.logger.Debug("host: inner SSH server session established", "channelID", channel.ChannelID)

	pfs := tcp.GetPortForwardingService(&innerSession.Session)
	if pfs != nil && len(ports) > 0 {
		// Filter out the "all ports" sentinel value (-1): it is not a valid listening port and should not be forwarded.
		for _, port := range filterForwardPorts(ports) {
			if _, err := pfs.ForwardFromRemotePort(ctx, "127.0.0.1", port, "127.0.0.1", port); err != nil {
				d.logger.Error("forward port failed", "port", port, "err", err)
			}
		}
	}

	go func(chID uint32) {
		for {
			if _, err := innerSession.AcceptChannel(ctx); err != nil {
				return
			}
		}
	}(channel.ChannelID)
}
