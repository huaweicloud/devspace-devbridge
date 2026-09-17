/*
 * Copyright (c) Huawei Technologies Co., Ltd. 2026-2027. All rights reserved.
 */

package sdk

import (
	"context"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net"
	"net/http"
	"strconv"
	"sync"
	"time"

	"github.com/microsoft/dev-tunnels-ssh/src/go/ssh"
	"github.com/microsoft/dev-tunnels-ssh/src/go/tcp"
)

// ConnectConfig is the configuration for connecting to a tunnel.
type ConnectConfig struct {
	TunnelID string // tunnel ID
	Ports    []int  // ports to forward (empty means delivered by the host via SSH)
	JWTToken string // JWT token (mutually exclusive with APIKey)
	APIKey   string // API key (mutually exclusive with JWTToken)
	LocalIP  string // local listen address, defaults to 127.0.0.1

	// OnReady is called once the session is ready (local port mappings established), with the list of mappings.
	// It fires once per established connection or reconnect. May be nil.
	OnReady func(forwardings []Forwarding)
}

// Forwarding describes a port forwarding mapping.
type Forwarding struct {
	LocalPort  int    // local port
	RemotePort int    // remote port
	LocalIP    string // local listen address
}

// Connect starts the Connect connection service.
//
// This is a blocking call that returns when ctx is canceled or the connection is fully lost;
// brief network interruptions are reconnected automatically.
func (d *Devbridge) Connect(ctx context.Context, cfg ConnectConfig) error {
	if err := validateTunnelID(cfg.TunnelID); err != nil {
		return err
	}
	if cfg.LocalIP == "" {
		cfg.LocalIP = "127.0.0.1"
	}

	// Auth fallback: use the Devbridge instance's API key when cfg does not specify one.
	apiKey := cfg.APIKey
	if apiKey == "" && cfg.JWTToken == "" {
		apiKey = d.apiKey
	}

	header, subprotocols := buildWSHeader(cfg.JWTToken, apiKey)
	sniHost := cfg.TunnelID + "." + d.gatewayHost
	wsURL := "wss://" + sniHost + "/"

	factory := newListenerFactory(len(cfg.Ports), cfg.LocalIP, d.outputWriter, d.logger)

	const maxReconnectAttempts = 5
	const baseReconnectDelay = 3 * time.Second
	const maxReconnectDelay = 30 * time.Second

	consecutiveFailures := 0
	for consecutiveFailures < maxReconnectAttempts {
		connected, err := d.runConnectSession(ctx, wsURL, sniHost, header, subprotocols, cfg.TunnelID, cfg.Ports, factory, cfg.OnReady)
		if ctx.Err() != nil {
			return nil
		}
		if errors.Is(err, ErrQuotaExceeded) || errors.Is(err, ErrTunnelNotFound) {
			d.logger.Debug("connection rejected by gateway", "tunnelID", cfg.TunnelID, "err", err)
			return err
		}
		if connected {
			consecutiveFailures = 0
		} else {
			consecutiveFailures++
		}
		if consecutiveFailures >= maxReconnectAttempts {
			d.logger.Debug("reconnect exhausted", "maxAttempts", maxReconnectAttempts, "err", err)
			return fmt.Errorf("reconnect failed after %d attempts: %w", maxReconnectAttempts, err)
		}

		delay := baseReconnectDelay
		for i := 0; i < consecutiveFailures-1; i++ {
			delay *= 2
			if delay >= maxReconnectDelay {
				delay = maxReconnectDelay
				break
			}
		}
		d.statusf("Connection lost, reconnecting... (%v)\n", err)
		select {
		case <-ctx.Done():
			return nil
		case <-time.After(delay):
		}
	}
	return nil
}

func (d *Devbridge) runConnectSession(ctx context.Context, wsURL string, sniHost string, header http.Header, subprotocols []string, tunnelID string, ports []int, factory *listenerFactory, onReady func([]Forwarding)) (connected bool, err error) {
	netConn, err := d.dialWebSocket(ctx, wsURL, sniHost, header, subprotocols, 5)
	if err != nil {
		return false, fmt.Errorf("WebSocket connection failed: %w", err)
	}

	config := ssh.NewNoSecurityConfig()
	config.KeepAliveIntervalSeconds = 10
	tcp.AddPortForwardingService(config)

	session := ssh.NewClientSession(config)
	session.Trace = sshTraceFunc(d.logger)
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

	d.statusf("Connected to tunnel: %s\n", tunnelID)

	// Filter out the "all ports" sentinel value (-1). Such a tunnel can only be reached by URL for any port,
	// so the connect side does not need to create a local listener for -1.
	realPorts := filterForwardPorts(ports)

	if len(realPorts) > 0 {
		d.statusln("Mode: active forwarding (ports from API)")
	} else if len(ports) > 0 {
		d.statusf("All ports mode: access via URL instead, e.g. https://%s-<port>.%s\n", tunnelID, d.gatewayHost)
	} else {
		d.statusln("Mode: passive forwarding (ports from host via SSH)")
	}

	if len(realPorts) > 0 {
		factory.waitForForwardings(3 * time.Second)
	} else {
		factory.waitForForwardings(2 * time.Second)
	}
	factory.printForwardings()

	d.statusln("Auto reconnect: enabled")

	if onReady != nil {
		onReady(factory.snapshotForwardings())
	}

	select {
	case <-session.Session.Done():
		return true, fmt.Errorf("session closed")
	case <-ctx.Done():
		return true, nil
	}
}

type listenerFactory struct {
	mu                 sync.Mutex
	pendingForwardings []string
	forwardings        []Forwarding
	portOverrides      map[int]int
	listeners          []net.Listener
	expectedCount      int
	allReceived        chan struct{}
	localIP            string
	outputWriter       io.Writer
	logger             *slog.Logger
}

func newListenerFactory(expectedCount int, localIP string, outputWriter io.Writer, logger *slog.Logger) *listenerFactory {
	return &listenerFactory{
		expectedCount: expectedCount,
		allReceived:   make(chan struct{}),
		portOverrides: make(map[int]int),
		localIP:       localIP,
		outputWriter:  outputWriter,
		logger:        logger,
	}
}

// CreateTCPListener implements the tcp.ListenerFactory interface.
func (f *listenerFactory) CreateTCPListener(
	remotePort int,
	localIPAddress string,
	localPort int,
	canChangeLocalPort bool,
) (net.Listener, error) {
	f.mu.Lock()
	defer f.mu.Unlock()

	if override, ok := f.portOverrides[remotePort]; ok {
		localPort = override
	}

	listener, err := net.Listen("tcp", net.JoinHostPort(f.localIP, strconv.Itoa(localPort)))
	if err != nil {
		if canChangeLocalPort {
			return f.listenOnRandomPortLocked(remotePort, localPort)
		}
		return nil, fmt.Errorf("port %d is already in use: %w", localPort, err)
	}
	f.portOverrides[remotePort] = localPort
	f.listeners = append(f.listeners, listener)
	f.recordForwarding(localPort, remotePort, fmt.Sprintf("Forwarding localhost: %s%d%s -> tunnel port: %s%d%s\n",
		colorCyan, localPort, colorReset, colorCyan, remotePort, colorReset))
	return listener, nil
}

// listenOnRandomPortLocked switches to a random port when the local port is in use. The caller must hold f.mu.
func (f *listenerFactory) listenOnRandomPortLocked(remotePort, originalPort int) (net.Listener, error) {
	listener, err := net.Listen("tcp", net.JoinHostPort(f.localIP, "0"))
	if err != nil {
		return nil, err
	}
	actualPort := listener.Addr().(*net.TCPAddr).Port
	f.portOverrides[remotePort] = actualPort
	f.listeners = append(f.listeners, listener)
	f.recordForwarding(actualPort, remotePort, fmt.Sprintf("Forwarding localhost: %s%d%s -> tunnel port: %s%d%s (port %s%d%s in use)\n",
		colorCyan, actualPort, colorReset, colorCyan, remotePort, colorReset, colorYellow, originalPort, colorReset))
	return listener, nil
}

// recordForwarding records a port mapping and wakes waiters. The caller must hold f.mu.
func (f *listenerFactory) recordForwarding(localPort, remotePort int, msg string) {
	f.forwardings = append(f.forwardings, Forwarding{LocalPort: localPort, RemotePort: remotePort, LocalIP: f.localIP})
	f.pendingForwardings = append(f.pendingForwardings, msg)
	if len(f.pendingForwardings) >= f.expectedCount && f.expectedCount > 0 {
		select {
		case <-f.allReceived:
		default:
			close(f.allReceived)
		}
	}
}

// snapshotForwardings returns a copy of the currently established port mappings.
func (f *listenerFactory) snapshotForwardings() []Forwarding {
	f.mu.Lock()
	defer f.mu.Unlock()
	return append([]Forwarding(nil), f.forwardings...)
}

func (f *listenerFactory) printForwardings() {
	f.mu.Lock()
	defer f.mu.Unlock()
	for _, msg := range f.pendingForwardings {
		fmt.Fprint(f.outputWriter, msg)
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
	f.forwardings = nil
	f.allReceived = make(chan struct{})
}

func (f *listenerFactory) waitForForwardings(timeout time.Duration) {
	select {
	case <-f.allReceived:
	case <-time.After(timeout):
	}
}
