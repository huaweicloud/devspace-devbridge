package sdk

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"sync/atomic"
	"time"

	"github.com/microsoft/dev-tunnels-ssh/src/go/ssh"
	"github.com/microsoft/dev-tunnels-ssh/src/go/tcp"
)

// HostConfig Host 托管配置
type HostConfig struct {
	TunnelID string // 隧道 ID
	Ports    []int  // 本地端口列表（为空时从网关下发）
	JWTToken string // JWT 令牌（与 APIKey 二选一）
	APIKey   string // API Key（与 JWTToken 二选一）

	// OnReady 在会话就绪（端口开始转发）后调用，参数为本次托管的端口列表。
	// 每次连接或重连成功都会触发一次。可为 nil。
	OnReady func(ports []int)
}

// HostResult Host 运行结果信息
type HostResult struct {
	TunnelID  string
	Ports     []int
	TunnelURL string
}

type relayPortMessage struct {
	Ports []uint16 `json:"ports"`
}

var (
	hostSessionLookup = make(map[uint32]*ssh.ServerSession)
	persistentHostKey ssh.KeyPair
)

func init() {
	var err error
	persistentHostKey, err = ssh.GenerateKeyPair(ssh.AlgoPKEcdsaSha2P256)
	if err != nil {
		slog.Error("Failed to generate host key", "err", err)
	}
}

// Host 启动 Host 托管服务。
//
// 这是一个阻塞方法，在 ctx 被取消或连接彻底断开时返回；网络短暂中断会自动重连。
//
// 工作流程：
//  1. WebSocket 连到 wss://<tunnelId>.<gatewayHost>/<tunnelId>
//  2. 在 WebSocket 上建立 SSH 会话
//  3. 接受 relay channel，为每个 channel 创建内层 SSH 会话
//  4. 通过端口转发把流量从远端转到本地端口
func (d *Devbridge) Host(ctx context.Context, cfg HostConfig) error {
	if err := validateTunnelID(cfg.TunnelID); err != nil {
		return err
	}

	// 认证回退：cfg 未显式指定时，使用 Devbridge 实例上的 API Key
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
			return nil
		}
		if errors.Is(err, ErrDuplicateHost) && !everConnected {
			d.logger.Error("duplicate host, tunnel already has a listener", "tunnelID", cfg.TunnelID)
			return nil
		}
		if err == nil {
			return nil
		}
		if connected {
			consecutiveFailures = 0
			everConnected = true
		} else {
			consecutiveFailures++
		}
		if consecutiveFailures >= maxReconnectAttempts {
			d.logger.Error("reconnect exhausted", "maxAttempts", maxReconnectAttempts, "err", err)
			return nil
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

	// 过滤掉"所有端口"哨兵值：它只对 visitor URL 访问有意义，
	// host 端不需要为 -1 发起本地端口转发。
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
	hostSessionLookup[channel.ChannelID] = innerSession

	pfs := tcp.GetPortForwardingService(&innerSession.Session)
	if pfs != nil && len(ports) > 0 {
		// 过滤"所有端口"哨兵值（-1），不被转发（不是合法监听端口）。
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

	go func(chID uint32) {
		<-innerSession.Session.Done()
		delete(hostSessionLookup, chID)
	}(channel.ChannelID)
}
