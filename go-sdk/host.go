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

	const maxReconnectAttempts = 5
	const baseReconnectDelay = 3 * time.Second
	const maxReconnectDelay = 30 * time.Second

	consecutiveFailures := 0
	everConnected := false

	for consecutiveFailures < maxReconnectAttempts {
		connected, err := d.runHostSession(ctx, wsURL, sniHost, header, subprotocols, cfg.TunnelID, cfg.Ports, cfg.OnReady)
		if ctx.Err() != nil {
			return nil
		}
		if errors.Is(err, ErrQuotaExceeded) || errors.Is(err, ErrTunnelNotFound) {
			d.logger.Error("connection rejected by gateway", "tunnelID", cfg.TunnelID, "err", err)
			return err
		}
		if errors.Is(err, ErrDuplicateHost) && !everConnected {
			d.logger.Error("duplicate host, tunnel already has a listener", "tunnelID", cfg.TunnelID)
			return err
		}
		if connected {
			consecutiveFailures = 0
			everConnected = true
		} else {
			consecutiveFailures++
		}
		if consecutiveFailures >= maxReconnectAttempts {
			d.logger.Error("reconnect exhausted", "maxAttempts", maxReconnectAttempts, "err", err)
			return fmt.Errorf("reconnect failed after %d attempts: %w", maxReconnectAttempts, err)
		}

		shift := consecutiveFailures - 1
		if shift > 4 {
			shift = 4
		}
		delay := baseReconnectDelay << uint(shift)
		if delay > maxReconnectDelay {
			delay = maxReconnectDelay
		}
		d.statusln("Connection lost, reconnecting...")
		select {
		case <-ctx.Done():
			return nil
		case <-time.After(delay):
		}
	}
	return nil
}

func (d *Devbridge) runHostSession(ctx context.Context, wsURL string, sniHost string, header http.Header, subprotocols []string, tunnelID string, ports []int, onReady func([]int)) (connected bool, err error) {
	netConn, err := d.dialWebSocket(ctx, wsURL, sniHost, header, subprotocols, 5)
	if err != nil {
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
			return true, fmt.Errorf("disconnected")
		case <-outerSession.Session.Done():
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
			return true, fmt.Errorf("disconnected")
		case <-outerSession.Session.Done():
			return true, fmt.Errorf("session closed")
		case <-ctx.Done():
			return true, nil
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

func (d *Devbridge) startHostAcceptLoop(ctx context.Context, outerSession *ssh.ClientSession, tunnelID string, ports []int, pn *portNotifier) {
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
			go d.handleRelayChannel(ctx, channel, tunnelID, effectivePorts)
		default:
			d.logger.Debug("host: draining non-relay channel",
				"channelType", channel.ChannelType, "channelID", channel.ChannelID)
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
		return nil
	}
	ports := make([]int, len(msg.Ports))
	for i, p := range msg.Ports {
		ports[i] = int(p)
	}
	return ports
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
