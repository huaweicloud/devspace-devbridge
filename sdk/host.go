package devbridge

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

// ──────────────────────────────────────────────────────────────
// Host — 托管本地服务
//
// Host 运行在服务所在设备，通过 WebSocket 出站连接到 DevBridge 中继，
// 把本地端口转发出去，让远端 Connect 可以访问。
//
// 工作流程：
//  1. WebSocket 连到 wss://<tunnelId>.<gatewayHost>/<tunnelId>
//  2. 在 WebSocket 上建立 SSH 会话
//  3. 接受 relay channel，为每个 channel 创建内层 SSH 会话
//  4. 通过端口转发把流量从远端转到本地端口
// ──────────────────────────────────────────────────────────────

// HostConfig Host 托管配置
type HostConfig struct {
	TunnelID  string   // 隧道 ID
	Ports     []int    // 本地端口列表（为空时从网关下发）
	JWTToken  string   // JWT 令牌（与 APIKey 二选一）
	APIKey    string   // API Key（与 JWTToken 二选一）
}

// HostResult Host 运行结果信息
type HostResult struct {
	TunnelID  string   // 隧道 ID
	Ports     []int    // 实际托管的端口
	TunnelURL string   // 隧道访问地址
}

// relayPortMessage 网关下发的端口通知
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

// Host 启动 Host 托管服务
//
// 这是一个阻塞方法，在 ctx 被取消或连接彻底断开时返回。
// 网络短暂中断会自动重连。
//
// 基本用法（已有隧道和端口配置）：
//
//	err := client.Host(ctx, devbridge.HostConfig{
//	    TunnelID: "aaaadysa",
//	    Ports:    []int{8080},
//	})
//
// 使用 API Key 鉴权（跳过令牌签发）：
//
//	err := client.Host(ctx, devbridge.HostConfig{
//	    TunnelID: "aaaadysa",
//	    APIKey:   "your-api-key",
//	})
//
// 使用已有 JWT 令牌（跳过 API 调用）：
//
//	err := client.Host(ctx, devbridge.HostConfig{
//	    TunnelID: "aaaadysa",
//	    JWTToken: "your-jwt-token",
//	})
func (c *Client) Host(ctx context.Context, cfg HostConfig) error {
	if err := validateTunnelID(cfg.TunnelID); err != nil {
		return err
	}

	header, subprotocols := buildWSHeader(cfg.JWTToken, cfg.APIKey)
	header.Set("Cookie", "APP_COOKIE=7")

	sniHost := cfg.TunnelID + "." + c.gatewayHost
	wsURL := "wss://" + sniHost + "/" + cfg.TunnelID

	const maxReconnectAttempts = 5
	const baseReconnectDelay = 3 * time.Second
	const maxReconnectDelay = 30 * time.Second

	consecutiveFailures := 0
	everConnected := false

	for consecutiveFailures < maxReconnectAttempts {
		connected, err := c.runHostSession(ctx, wsURL, sniHost, header, subprotocols, cfg.TunnelID, cfg.Ports)
		if ctx.Err() != nil {
			return nil
		}
		if errors.Is(err, ErrQuotaExceeded) || errors.Is(err, ErrTunnelNotFound) {
			return err
		}
		if errors.Is(err, ErrDuplicateHost) && !everConnected {
			return err
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
			return fmt.Errorf("reconnect exhausted after %d attempts: %w", maxReconnectAttempts, err)
		}

		shift := consecutiveFailures - 1
		if shift > 4 {
			shift = 4
		}
		delay := baseReconnectDelay << uint(shift)
		if delay > maxReconnectDelay {
			delay = maxReconnectDelay
		}
		c.logger.Info("connection lost, reconnecting...", "delay", delay)
		select {
		case <-ctx.Done():
			return nil
		case <-time.After(delay):
		}
	}
	return nil
}

// runHostSession 执行一次 Host 会话
func (c *Client) runHostSession(ctx context.Context, wsURL string, sniHost string, header http.Header, subprotocols []string, tunnelID string, ports []int) (connected bool, err error) {
	netConn, err := c.dialWebSocket(ctx, wsURL, sniHost, header, subprotocols, 5)
	if err != nil {
		return false, err
	}
	defer func() { _ = netConn.Close() }()

	// 建立 SSH 客户端会话
	outerConfig := ssh.NewNoSecurityConfig()
	outerConfig.KeepAliveIntervalSeconds = 10
	outerConfig.KeyRotationThreshold = 0
	tcp.AddPortForwardingService(outerConfig)
	outerSession := ssh.NewClientSession(outerConfig)
	outerSession.Trace = sshTraceFunc(c.logger)

	if err := outerSession.Connect(ctx, netConn); err != nil {
		err = parseSSHCloseError(err)
		_ = netConn.Close()
		return false, fmt.Errorf("outer SSH connect failed: %w", err)
	}
	c.logger.Debug("host: outer SSH session established", "tunnelID", tunnelID)
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
			c.logger.Error("keepalive failed 5 times, forcing reconnect", "tunnelID", tunnelID)
			_ = outerSession.Close()
		}
	}

	// 端口通知
	pn := newPortNotifier()

	// 接受 channel 循环
	go c.startHostAcceptLoop(ctx, outerSession, tunnelID, ports, pn)

	// 等待端口通知（如果端口由网关下发）
	if len(ports) == 0 {
		select {
		case <-pn.ready:
		case <-time.After(5 * time.Second):
			c.logger.Warn("timeout waiting for port notification from gateway")
		case <-disconnected:
			return true, fmt.Errorf("disconnected")
		case <-outerSession.Session.Done():
			return true, fmt.Errorf("session closed")
		}
	}

	// 打印就绪信息
	printPorts := ports
	if len(ports) == 0 {
		printPorts = pn.ports
	}
	for _, p := range printPorts {
		c.logger.Info("hosting port", "port", p,
			"tunnelURL", fmt.Sprintf("https://%s-%d.%s", tunnelID, p, c.gatewayHost))
	}

	// 等待断开
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

// portNotifier 端口通知器
type portNotifier struct {
	ready    chan struct{}
	ports    []int
	received atomic.Bool
}

func newPortNotifier() *portNotifier {
	return &portNotifier{ready: make(chan struct{})}
}

// startHostAcceptLoop 接受 SSH channel 的循环
func (c *Client) startHostAcceptLoop(ctx context.Context, outerSession *ssh.ClientSession, tunnelID string, ports []int, pn *portNotifier) {
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
			go c.handleRelayChannel(ctx, channel, tunnelID, effectivePorts)
		default:
			c.logger.Debug("host: draining non-relay channel",
				"channelType", channel.ChannelType, "channelID", channel.ChannelID)
		}
	}
}

// readPortNotification 从 channel 读取端口通知
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

// handleRelayChannel 处理一个 relay channel
func (c *Client) handleRelayChannel(ctx context.Context, channel *ssh.Channel, tunnelID string, ports []int) {
	// 创建内层 SSH 服务端会话
	innerConfig := ssh.NewNoSecurityConfig()
	tcp.AddPortForwardingService(innerConfig)
	innerSession := ssh.NewServerSession(innerConfig)
	innerSession.Credentials = &ssh.ServerCredentials{PublicKeys: []ssh.KeyPair{persistentHostKey}}
	innerSession.Trace = sshTraceFunc(c.logger)

	if err := innerSession.Connect(ctx, ssh.NewStream(channel)); err != nil {
		c.logger.Error("inner SSH session failed", "channelID", channel.ChannelID, "err", err)
		return
	}
	c.logger.Debug("host: inner SSH server session established", "channelID", channel.ChannelID)
	hostSessionLookup[channel.ChannelID] = innerSession

	// 设置端口转发
	pfs := tcp.GetPortForwardingService(&innerSession.Session)
	if pfs != nil && len(ports) > 0 {
		for _, port := range ports {
			if _, err := pfs.ForwardFromRemotePort(ctx, "127.0.0.1", port, "127.0.0.1", port); err != nil {
				c.logger.Error("forward port failed", "port", port, "err", err)
			}
		}
	}

	// 清理
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
