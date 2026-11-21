/*
 * Copyright (c) Huawei Technologies Co., Ltd. 2026-2027. All rights reserved.
 */

package sdk

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"sort"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/microsoft/dev-tunnels-ssh/src/go/ssh"
	"github.com/microsoft/dev-tunnels-ssh/src/go/tcp"

	"github.com/huaweicloud/devspace-devbridge/go-sdk/internal/i18n"
)

// Session tuning constants shared by Host and Connect sessions. Kept as
// package-level constants so Host and Connect cannot drift apart.
const (
	keepAliveIntervalSeconds = 10              // SSH keepalive interval (seconds)
	keyRotationThreshold     = 0               // disable key rotation (host key is process-wide)
	keepAliveFailThreshold   = 5               // consecutive keepalive failures before forcing reconnect
	gatewayPortTimeout       = 5 * time.Second // wait for the gateway port notification
	// portNotificationReadTimeout bounds the single read of a potential port
	// notification. Kept below gatewayPortTimeout so a stalled notification still
	// leaves time for the caller's 5s path to react.
	portNotificationReadTimeout = 2 * time.Second
	// portRefreshMinInterval is the debounce window for refreshing the hosted
	// port set through the REST API. A relay channel accepted within the window
	// reuses the last fetched set instead of issuing another API call.
	portRefreshMinInterval = 1 * time.Second
	// portRefreshTimeout bounds a single ListPorts refresh so a slow API never
	// blocks channel forwarding for longer than the acceptance target (<=2s).
	portRefreshTimeout = 2 * time.Second
)

// HostConfig is the configuration for hosting a tunnel.
type HostConfig struct {
	TunnelID string // tunnel ID
	Ports    []int  // local ports to forward (empty means issued by the gateway)
	JWTToken string // JWT token (mutually exclusive with APIKey)
	APIKey   string // API key (mutually exclusive with JWTToken)

	// OnReady runs once the session is ready, with the hosted ports. May be nil.
	OnReady func(ports []int)

	// OnPortsChanged is invoked whenever the hosted port set changes while the
	// session is running, reporting added ports, removed ports, and the
	// resulting full port set. Only fires when the Host did not pin ports
	// (Ports is empty). May be nil.
	OnPortsChanged func(added, removed, current []int)
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
		return fmt.Errorf("%s: %w", i18n.T(i18n.MsgHostKeyFailed), err)
	}
	if err := validateTunnelID(cfg.TunnelID); err != nil {
		return err
	}

	// Auth fallback: use the Devbridge instance's API key when cfg does not specify one.
	apiKey := cfg.APIKey
	if apiKey == "" && cfg.JWTToken == "" {
		apiKey = d.apiKey
		if apiKey == "" {
			return ErrMissingAPIKey
		}
	}

	header, subprotocols := buildWSHeader(cfg.JWTToken, apiKey, d.wsHeaders)
	header.Set("Cookie", "APP_COOKIE=7")

	sniHost := cfg.TunnelID + "." + d.gatewayHost
	wsURL := "wss://" + sniHost + "/" + cfg.TunnelID

	params := sessionParams{
		wsURL:        wsURL,
		sniHost:      sniHost,
		header:       header,
		subprotocols: subprotocols,
		tunnelID:     cfg.TunnelID,
		ports:        cfg.Ports,
	}

	everConnected := false
	return d.reconnectLoop(ctx, func() (bool, error) {
		return d.runHostSession(ctx, params, cfg.OnReady, cfg.OnPortsChanged)
	}, reconnectDecision{
		// Abort on gateway rejection (quota/tunnel) or a duplicate host before the first
		// connection. onConnected runs after shouldStop, so everConnected is the prior value.
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
			d.statusln(i18n.T(i18n.MsgReconnecting))
		},
		onConnected: func() {
			everConnected = true
		},
	})
}

// hostSessionDiagnostics holds a host session's shutdown state: a one-shot
// disconnect signal and the last close reason captured for logging.
type hostSessionDiagnostics struct {
	disconnected chan struct{}
	closedMu     sync.Mutex
	closedArgs   *ssh.SessionClosedEventArgs
}

func newHostSessionDiagnostics() *hostSessionDiagnostics {
	return &hostSessionDiagnostics{disconnected: make(chan struct{}, 1)}
}

func (d *Devbridge) runHostSession(ctx context.Context, params sessionParams, onReady func([]int), onPortsChanged func(added, removed, current []int)) (connected bool, err error) {
	outerSession, netConn, err := d.connectOuterSession(ctx, params)
	if err != nil {
		return false, err
	}
	defer func() {
		_ = outerSession.Close()
		_ = netConn.Close()
	}()

	d.logger.Debug("host: outer SSH session established", "tunnelID", params.tunnelID)

	diag := newHostSessionDiagnostics()
	d.installSessionCallbacks(outerSession, params.tunnelID, diag)

	pn := newPortNotifier()
	refresher := newPortSetRefresher(d, params.tunnelID, params.ports, pn, onPortsChanged)
	go d.startHostAcceptLoop(ctx, outerSession, params, pn, refresher)

	if len(params.ports) == 0 {
		// Preferred path: refresh the port set through the REST API. Unlike the
		// gateway push notification, ListPorts is a documented interface that
		// this repository can verify end to end, so runtime port changes do not
		// depend on unverified gateway behaviour.
		if _, ok := refresher.Refresh(ctx, true); !ok {
			// Fallback: the gateway may deliver the port list over the first
			// relay channel (used by --token mode, where no API key is needed).
			if err := d.waitForGatewayPorts(ctx, outerSession, params.tunnelID, pn, diag); err != nil {
				return true, err
			}
		}
	}

	printPorts := params.ports
	if len(params.ports) == 0 {
		printPorts = pn.currentPorts()
	}
	d.reportReady(params.tunnelID, printPorts, onReady)

	return d.waitSessionEnd(ctx, outerSession, params.tunnelID, diag)
}

func (d *Devbridge) connectOuterSession(ctx context.Context, params sessionParams) (*ssh.ClientSession, net.Conn, error) {
	netConn, err := d.dialWebSocket(ctx, params.wsURL, params.sniHost, params.header, params.subprotocols, 5)
	if err != nil {
		if errors.Is(err, ErrDuplicateHost) {
			return nil, nil, fmt.Errorf("%s: %w", i18n.T(i18n.MsgOuterSSHFailed), err)
		}
		return nil, nil, err
	}

	outerConfig := ssh.NewNoSecurityConfig()
	outerConfig.KeepAliveIntervalSeconds = keepAliveIntervalSeconds
	outerConfig.KeyRotationThreshold = keyRotationThreshold
	tcp.AddPortForwardingService(outerConfig)
	outerSession := ssh.NewClientSession(outerConfig)
	outerSession.Trace = sshTraceFunc(d.logger)

	if err := outerSession.Connect(ctx, netConn); err != nil {
		_ = outerSession.Close()
		_ = netConn.Close()
		return nil, nil, fmt.Errorf("%s: %w", i18n.T(i18n.MsgOuterSSHFailed), parseSSHCloseError(err))
	}
	return outerSession, netConn, nil
}

func (d *Devbridge) installSessionCallbacks(outerSession *ssh.ClientSession, tunnelID string, diag *hostSessionDiagnostics) {
	outerSession.OnDisconnected = func() {
		select {
		case diag.disconnected <- struct{}{}:
		default:
		}
	}
	outerSession.OnClosed = func(args *ssh.SessionClosedEventArgs) {
		diag.closedMu.Lock()
		diag.closedArgs = args
		diag.closedMu.Unlock()
		d.logger.Debug("host: session closed (diagnostic)",
			"tunnelID", tunnelID,
			"reason", args.Reason,
			"message", args.Message,
			"err", args.Err)
	}
	outerSession.OnKeepAliveFailed = func(count int) {
		if count >= keepAliveFailThreshold {
			d.logger.Error(fmt.Sprintf("keepalive failed %d times, forcing reconnect", keepAliveFailThreshold), "tunnelID", tunnelID)
			_ = outerSession.Close()
		}
	}
}

func (d *Devbridge) waitForGatewayPorts(ctx context.Context, outerSession *ssh.ClientSession, tunnelID string, pn *portNotifier, diag *hostSessionDiagnostics) error {
	select {
	case <-pn.ready:
	case <-ctx.Done():
		return ctx.Err()
	case <-time.After(gatewayPortTimeout):
		d.logger.Warn("timeout waiting for port notification from gateway")
	case <-diag.disconnected:
		d.logDiagnosticClose(tunnelID, diag)
		return fmt.Errorf(i18n.T(i18n.MsgSessionDisconnected))
	case <-outerSession.Session.Done():
		d.logDiagnosticClose(tunnelID, diag)
		return fmt.Errorf(i18n.T(i18n.MsgSessionClosed))
	}
	return nil
}

func (d *Devbridge) reportReady(tunnelID string, ports []int, onReady func([]int)) {
	realPorts := filterForwardPorts(ports)
	if len(realPorts) == 0 && len(ports) > 0 {
		d.statusln(i18n.T(i18n.MsgAllPortsAnyPort))
		d.statusf(i18n.T(i18n.MsgAccessServiceAt), tunnelID, d.gatewayHost)
	}
	for _, p := range realPorts {
		d.statusf(i18n.T(i18n.MsgHostingPort), colorCyan, p, colorReset)
	}
	for _, p := range realPorts {
		d.statusf(i18n.T(i18n.MsgTunnelURL), tunnelID, p, d.gatewayHost)
	}
	d.statusln(i18n.T(i18n.MsgReadyToAccept))
	d.statusln(i18n.T(i18n.MsgAutoReconnectEnabled))
	if onReady != nil {
		onReady(realPorts)
	}
}

func (d *Devbridge) waitSessionEnd(ctx context.Context, outerSession *ssh.ClientSession, tunnelID string, diag *hostSessionDiagnostics) (bool, error) {
	for {
		select {
		case <-diag.disconnected:
			d.logDiagnosticClose(tunnelID, diag)
			return true, fmt.Errorf(i18n.T(i18n.MsgSessionDisconnected))
		case <-outerSession.Session.Done():
			d.logDiagnosticClose(tunnelID, diag)
			return true, fmt.Errorf(i18n.T(i18n.MsgSessionClosed))
		case <-ctx.Done():
			return true, nil
		}
	}
}

// logDiagnosticClose logs why the host connection was lost (diagnostic only).
func (d *Devbridge) logDiagnosticClose(tunnelID string, diag *hostSessionDiagnostics) {
	diag.closedMu.Lock()
	a := diag.closedArgs
	diag.closedMu.Unlock()
	if a == nil {
		d.logger.Debug("host connection lost (no close event captured)", "tunnelID", tunnelID)
		return
	}
	d.logger.Debug("host connection lost",
		"tunnelID", tunnelID,
		"reason", a.Reason,
		"message", a.Message,
		"err", a.Err)
}

type portNotifier struct {
	ready    chan struct{}
	mu       sync.Mutex
	ports    []int
	received atomic.Bool
}

func newPortNotifier() *portNotifier {
	return &portNotifier{ready: make(chan struct{})}
}

// update applies a full port set (from the API refresh or the gateway's
// startup notification) and returns the diff against the previous set
// (added, removed, changed). The first update also closes ready so
// waitForGatewayPorts can proceed.
func (pn *portNotifier) update(ports []int) (added, removed []int, changed bool) {
	pn.mu.Lock()
	defer pn.mu.Unlock()
	prev := pn.ports
	pn.ports = ports
	if !pn.received.Swap(true) {
		close(pn.ready)
		return nil, nil, false
	}
	added, removed = diffPorts(prev, ports)
	changed = len(added) > 0 || len(removed) > 0
	return added, removed, changed
}

// currentPorts returns a copy of the latest known port set.
func (pn *portNotifier) currentPorts() []int {
	pn.mu.Lock()
	defer pn.mu.Unlock()
	out := make([]int, len(pn.ports))
	copy(out, pn.ports)
	return out
}

// portSetRefresher keeps the hosted port set up to date at runtime by
// querying the REST API (ListPorts) instead of relying on an unverified
// gateway push notification. Each relay channel is forwarded with the latest
// port set, so a port added with `devbridge port create` becomes reachable on
// the running Host without a restart.
//
// A pinned port list (HostConfig.Ports non-empty) disables refreshing: the
// operator explicitly chose the static mode, where a port change requires
// restarting the Host.
type portSetRefresher struct {
	devbridge   *Devbridge
	tunnelID    string
	staticPorts []int
	pn          *portNotifier
	onChange    func(added, removed, current []int)
	listPorts   func(ctx context.Context, tunnelID string) ([]Port, error)
	// apiAvailable is false when the Host has no REST credential (--token
	// mode). Refreshing would only add latency, so the port set stays whatever
	// the gateway delivered over the first relay channel.
	apiAvailable bool
	minInterval  time.Duration
	timeout      time.Duration
	now          func() time.Time

	mu     sync.Mutex
	lastAt time.Time
}

func newPortSetRefresher(d *Devbridge, tunnelID string, staticPorts []int, pn *portNotifier, onChange func(added, removed, current []int)) *portSetRefresher {
	return &portSetRefresher{
		devbridge:    d,
		tunnelID:     tunnelID,
		staticPorts:  staticPorts,
		pn:           pn,
		onChange:     onChange,
		listPorts:    d.ListPorts,
		apiAvailable: d.apiKey != "",
		minInterval:  portRefreshMinInterval,
		timeout:      portRefreshTimeout,
		now:          time.Now,
	}
}

// Refresh fetches the tunnel's current port set and applies it to the running
// session, invoking OnPortsChanged when the set actually changed.
//
// It returns the effective port set for the channel being handled and whether
// that set comes from the API (fresh), so the caller can decide whether to wait
// for the gateway notification fallback. In static ports mode it returns the
// pinned ports without calling the API and always reports fresh=false.
func (r *portSetRefresher) Refresh(ctx context.Context, force bool) ([]int, bool) {
	if len(r.staticPorts) > 0 {
		return r.staticPorts, false
	}
	if !r.apiAvailable {
		return r.pn.currentPorts(), false
	}

	r.mu.Lock()
	defer r.mu.Unlock()

	now := r.now()
	if !force && now.Sub(r.lastAt) < r.minInterval {
		return r.pn.currentPorts(), true
	}
	r.lastAt = now

	reqCtx, cancel := context.WithTimeout(ctx, r.timeout)
	defer cancel()
	ports, err := r.listPorts(reqCtx, r.tunnelID)
	if err != nil {
		// Fail soft: keep forwarding the last known port set. A transient API
		// failure must not tear down a working Host session.
		r.devbridge.logger.Debug("host: port refresh failed, keeping current port set",
			"tunnelID", r.tunnelID, "err", err)
		return r.pn.currentPorts(), false
	}

	added, removed, changed := r.pn.update(portNumbers(ports))
	if !changed {
		return r.pn.currentPorts(), true
	}
	current := r.pn.currentPorts()
	r.devbridge.statusf(i18n.T(i18n.MsgPortsHotReloaded),
		formatPortList(added), formatPortList(removed), formatPortList(current))
	if r.onChange != nil {
		r.onChange(added, removed, current)
	}
	return current, true
}

// portNumbers projects API port records onto plain port numbers.
func portNumbers(ports []Port) []int {
	out := make([]int, 0, len(ports))
	for _, p := range ports {
		out = append(out, p.Port)
	}
	return out
}

// startHostAcceptLoop accepts relay channels and forwards each of them with
// the latest known port set. In dynamic ports mode the first relay channel is
// probed for the gateway's startup port notification (existing handshake,
// also the only port source in --token mode); every later channel refreshes
// the port set through the REST API instead of assuming push notifications.
func (d *Devbridge) startHostAcceptLoop(ctx context.Context, outerSession *ssh.ClientSession, params sessionParams, pn *portNotifier, refresher *portSetRefresher) {
	probedFirst := false
	for {
		channel, err := outerSession.AcceptChannel(ctx)
		if err != nil {
			return
		}

		switch channel.ChannelType {
		case relayChannelType:
			var stream *ssh.Stream
			if len(params.ports) == 0 && !probedFirst {
				probedFirst = true
				// The gateway may deliver the tunnel's port list over the first
				// relay channel. Runtime port changes are NOT assumed to be
				// pushed: they are picked up by the REST API refresh below.
				var notifPorts []int
				var isNotif bool
				stream, notifPorts, isNotif = readPortNotification(channel)
				if isNotif {
					pn.update(notifPorts)
					continue
				}
			}

			effectivePorts := params.ports
			if len(effectivePorts) == 0 {
				// Refresh in the channel goroutine so a slow API call never
				// blocks acceptance of subsequent relay channels.
				go func(ch *ssh.Channel, st *ssh.Stream) {
					ports, _ := refresher.Refresh(ctx, false)
					d.handleRelayChannel(ctx, ch, st, ports)
				}(channel, stream)
				continue
			}
			go d.handleRelayChannel(ctx, channel, stream, effectivePorts)
		default:
			d.logger.Debug("host: draining non-relay channel",
				"channelType", channel.ChannelType, "channelID", channel.ChannelID)
		}
	}
}

func formatPortList(ports []int) string {
	if len(ports) == 0 {
		return "-"
	}
	sort.Ints(ports)
	parts := make([]string, len(ports))
	for i, p := range ports {
		parts[i] = strconv.Itoa(p)
	}
	return strings.Join(parts, ",")
}

// diffPorts computes which ports were added and removed going from old to new.
func diffPorts(old, newPorts []int) (added, removed []int) {
	oldSet := make(map[int]struct{}, len(old))
	for _, p := range old {
		oldSet[p] = struct{}{}
	}
	newSet := make(map[int]struct{}, len(newPorts))
	for _, p := range newPorts {
		newSet[p] = struct{}{}
		if _, ok := oldSet[p]; !ok {
			added = append(added, p)
		}
	}
	for _, p := range old {
		if _, ok := newSet[p]; !ok {
			removed = append(removed, p)
		}
	}
	return added, removed
}

// readPortNotification reports whether a relay channel is the gateway's port
// notification. When isNotif is false it returns the stream already wrapping the
// channel so the caller can forward any buffered bytes as data without losing them.
// When isNotif is true the notification has been consumed and the stream closed.
func readPortNotification(channel *ssh.Channel) (stream *ssh.Stream, ports []int, isNotif bool) {
	stream = ssh.NewStream(channel)
	buf := make([]byte, 4096)
	n, err := readWithTimeout(stream, buf, portNotificationReadTimeout)
	if err != nil {
		// Timeout or read failure means this is not a notification channel (the
		// gateway sends it right after opening). Close it to unblock the pending
		// read goroutine (Stream has no cancel-read API); the caller reconnects.
		_ = stream.Close()
		return nil, nil, false
	}
	ports, isNotif = parsePortNotification(buf[:n])
	if isNotif {
		_ = stream.Close()
		return nil, ports, true
	}
	return stream, nil, false
}

// readWithTimeout reads up to len(buf) bytes from r, bounding the wait with a
// timeout. It never blocks forever on a stream that stays open without data.
// The caller must close the underlying stream on timeout to unblock the read
// goroutine.
func readWithTimeout(r io.Reader, buf []byte, timeout time.Duration) (int, error) {
	type result struct {
		n   int
		err error
	}
	ch := make(chan result, 1)
	go func() {
		n, err := r.Read(buf)
		ch <- result{n, err}
	}()
	select {
	case res := <-ch:
		return res.n, res.err
	case <-time.After(timeout):
		return 0, fmt.Errorf("read timed out after %s", timeout)
	}
}

// parsePortNotification returns the ports of a RelayRequest payload, or false when
// the payload is not a valid port notification.
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

func (d *Devbridge) handleRelayChannel(ctx context.Context, channel *ssh.Channel, stream *ssh.Stream, ports []int) {
	innerConfig := ssh.NewNoSecurityConfig()
	tcp.AddPortForwardingService(innerConfig)
	innerSession := ssh.NewServerSession(innerConfig)
	innerSession.Credentials = &ssh.ServerCredentials{PublicKeys: []ssh.KeyPair{persistentHostKey}}
	innerSession.Trace = sshTraceFunc(d.logger)

	if stream == nil {
		stream = ssh.NewStream(channel)
	}
	if err := innerSession.Connect(ctx, stream); err != nil {
		d.logger.Error("inner SSH session failed", "channelID", channel.ChannelID, "err", err)
		_ = innerSession.Close()
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

	go func() {
		defer func() { _ = innerSession.Close() }()
		for {
			if _, err := innerSession.AcceptChannel(ctx); err != nil {
				return
			}
		}
	}()
}
