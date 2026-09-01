package devbridge

import (
	"context"
	"crypto/tls"
	"errors"
	"fmt"
	"io"
	"math/rand/v2"
	"net"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"github.com/coder/websocket"
	"github.com/microsoft/dev-tunnels-ssh/src/go/ssh"
)

// ──────────────────────────────────────────────────────────────
// WebSocket 连接逻辑 — Host 和 Connect 共用
// ──────────────────────────────────────────────────────────────

const (
	subprotocolDevBridge = "devbridge-v1"
	relayChannelType     = "relay"
)

// buildWSHeader 构建 WebSocket 握手头
//
// 认证方式二选一：
//   - JWT 令牌：通过 Sec-WebSocket-Protocol 传递
//   - API Key：通过 X-API-Key 头传递
func buildWSHeader(jwtToken, apiKey string) (http.Header, []string) {
	header := http.Header{}
	subprotocols := []string{subprotocolDevBridge}
	if apiKey != "" {
		header.Set(headerXAPIKey, apiKey)
	}
	if jwtToken != "" {
		header.Set("Sec-WebSocket-Protocol", subprotocolDevBridge+", "+jwtToken)
		subprotocols = append(subprotocols, jwtToken)
	}
	return header, subprotocols
}

// dialWebSocket 建立 WebSocket 连接，转换为 net.Conn，带重试
func (c *Client) dialWebSocket(ctx context.Context, wsURL, sniHost string, header http.Header, subprotocols []string, maxRetries int) (net.Conn, error) {
	dialCtx, dialCancel := context.WithTimeout(ctx, 30*time.Second)
	defer dialCancel()

	conn, err := c.dialWithRetry(dialCtx, wsURL, &websocket.DialOptions{
		HTTPClient:   c.getWSHTTPClient(sniHost),
		HTTPHeader:   header,
		Subprotocols: subprotocols,
	}, maxRetries)
	if err != nil {
		return nil, err
	}
	return websocket.NetConn(ctx, conn, websocket.MessageBinary), nil
}

// getWSHTTPClient 创建用于 WebSocket 握手的 HTTP 客户端
// 关键：DialContext 被替换为拨号到网关地址，TLS SNI 设为 sniHost
func (c *Client) getWSHTTPClient(sniHost string) *http.Client {
	dialer := &net.Dialer{}
	return &http.Client{
		Transport: &http.Transport{
			TLSClientConfig: &tls.Config{
				MinVersion: tls.VersionTLS12,
				MaxVersion: tls.VersionTLS13,
				ServerName: sniHost,
			},
			DialContext: func(ctx context.Context, network, addr string) (net.Conn, error) {
				return dialer.DialContext(ctx, network, c.gatewayAddr)
			},
		},
	}
}

// dialWithRetry 带指数退避重试的 WebSocket 拨号
func (c *Client) dialWithRetry(ctx context.Context, url string, opts *websocket.DialOptions, maxRetries int) (*websocket.Conn, error) {
	const baseDelay = 1 * time.Second
	const maxDelay = 30 * time.Second

	var lastErr error
	for attempt := 0; attempt <= maxRetries; attempt++ {
		c.logger.Debug("WebSocket handshake attempt",
			"attempt", attempt+1, "maxRetries", maxRetries, "url", url)

		conn, resp, err := websocket.Dial(ctx, url, opts)
		if err == nil {
			c.logger.Debug("WebSocket handshake succeeded", "attempt", attempt+1, "url", url)
			return conn, nil
		}
		lastErr = err

		// 409 Conflict: 该隧道已有 Host
		if resp != nil && resp.StatusCode == http.StatusConflict {
			return nil, ErrDuplicateHost
		}

		// 429 Too Many Requests: 被限流
		if resp != nil && resp.StatusCode == http.StatusTooManyRequests {
			body, _ := io.ReadAll(resp.Body)
			_ = resp.Body.Close()
			lastErr = fmt.Errorf("connection rejected by gateway (429): %s", strings.TrimSpace(string(body)))
		}

		if attempt == maxRetries {
			break
		}

		// 指数退避 + 随机抖动
		delay := baseDelay * time.Duration(1<<uint(attempt))
		if delay > maxDelay {
			delay = maxDelay
		}
		jittered := time.Duration(rand.Int64N(int64(delay)))

		c.logger.Debug("WebSocket dial retry",
			"attempt", attempt+1, "retryAfter", jittered, "err", lastErr)

		select {
		case <-ctx.Done():
			return nil, fmt.Errorf("websocket dial cancelled: %w", ctx.Err())
		case <-time.After(jittered):
		}
	}
	return nil, fmt.Errorf("websocket dial failed after %d retries: %w", maxRetries, lastErr)
}

// sshTraceFunc SSH 协议层日志
func sshTraceFunc(logger *slog.Logger) ssh.TraceFunc {
	if logger == nil {
		return nil
	}
	return func(level ssh.TraceLevel, eventID int, message string) {
		attrs := []slog.Attr{
			slog.Int("eventID", eventID),
			slog.String("msg", message),
		}
		switch level {
		case ssh.TraceLevelError:
			logger.LogAttrs(context.Background(), slog.LevelError, "ssh trace", attrs...)
		case ssh.TraceLevelWarning:
			logger.LogAttrs(context.Background(), slog.LevelWarn, "ssh trace", attrs...)
		case ssh.TraceLevelInfo:
			logger.LogAttrs(context.Background(), slog.LevelInfo, "ssh trace", attrs...)
		case ssh.TraceLevelVerbose:
			logger.LogAttrs(context.Background(), slog.LevelDebug, "ssh trace", attrs...)
		}
	}
}

// parseSSHCloseError 从 SSH 关闭错误中提取业务错误
func parseSSHCloseError(err error) error {
	var ce websocket.CloseError
	if errors.As(err, &ce) && ce.Code == websocket.StatusPolicyViolation {
		switch ce.Reason {
		case "account quota exceeded":
			return ErrQuotaExceeded
		case "tunnel not found":
			return ErrTunnelNotFound
		default:
			return ErrDuplicateHost
		}
	}
	return err
}
