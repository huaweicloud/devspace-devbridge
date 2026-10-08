/*
 * Copyright (c) Huawei Technologies Co., Ltd. 2026-2027. All rights reserved.
 */

package sdk

import (
	"context"
	"crypto/tls"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"math/rand/v2"
	"net"
	"net/http"
	"strings"
	"time"

	"github.com/coder/websocket"
	"github.com/microsoft/dev-tunnels-ssh/src/go/ssh"

	"github.com/huaweicloud/devspace-devbridge/go-sdk/internal/i18n"
)

const (
	subprotocolDevBridge = "devbridge-v1"
	relayChannelType     = "relay"

	// Application close codes (RFC 6455 private range 4000-4999) sent by the
	// gateway to signal business errors. Keep the numeric values in sync with
	// the relay-gateway side.
	closeCodeQuotaExceeded  websocket.StatusCode = 4001
	closeCodeTunnelNotFound websocket.StatusCode = 4002
	closeCodeDuplicateHost  websocket.StatusCode = 4003

	// ANSI color codes for user-facing terminal output
	colorCyan   = "\033[36m"
	colorYellow = "\033[33m"
	colorReset  = "\033[0m"
)

// sessionParams bundles the inputs shared by a host/connect session: the
// WebSocket dial target and handshake, plus the tunnel and ports context.
// Grouping them avoids an unwieldy per-session function signature.
type sessionParams struct {
	wsURL        string
	sniHost      string
	header       http.Header
	subprotocols []string
	tunnelID     string
	ports        []int
}

// buildWSHeader builds the WebSocket handshake header.
//
// Authentication is exclusive:
//   - JWT: passed via Sec-WebSocket-Protocol
//   - API Key: passed via the X-API-Key header
func buildWSHeader(jwtToken, apiKey string) (http.Header, []string) {
	header := http.Header{}
	subprotocols := []string{subprotocolDevBridge}
	if apiKey != "" {
		header.Set("X-API-Key", apiKey)
	}
	if jwtToken != "" {
		subprotocols = append(subprotocols, jwtToken)
	}
	return header, subprotocols
}

// dialWebSocket establishes a WebSocket connection, converts it to a net.Conn, and retries with backoff.
func (d *Devbridge) dialWebSocket(ctx context.Context, wsURL, sniHost string, header http.Header, subprotocols []string, maxRetries int) (net.Conn, error) {
	dialCtx, dialCancel := context.WithTimeout(ctx, 30*time.Second)
	defer dialCancel()

	conn, err := d.dialWithRetry(dialCtx, wsURL, &websocket.DialOptions{
		HTTPClient:   d.getWSHTTPClient(sniHost),
		HTTPHeader:   header,
		Subprotocols: subprotocols,
	}, maxRetries)
	if err != nil {
		return nil, err
	}
	return websocket.NetConn(ctx, conn, websocket.MessageBinary), nil
}

// getWSHTTPClient returns the HTTP client used for the WebSocket handshake.
// Key detail: DialContext is overridden to dial the gateway address, with TLS SNI set to sniHost.
func (d *Devbridge) getWSHTTPClient(sniHost string) *http.Client {
	dialer := &net.Dialer{}
	return &http.Client{
		Transport: &http.Transport{
			TLSClientConfig: &tls.Config{
				MinVersion:         tls.VersionTLS12,
				MaxVersion:         tls.VersionTLS13,
				ServerName:         sniHost,
				ClientSessionCache: d.tlsSessionCache,
			},
			DialContext: func(ctx context.Context, network, addr string) (net.Conn, error) {
				return dialer.DialContext(ctx, network, d.gatewayAddr)
			},
		},
	}
}

// dialWithRetry dials the WebSocket with exponential backoff and retries.
func (d *Devbridge) dialWithRetry(ctx context.Context, url string, opts *websocket.DialOptions, maxRetries int) (*websocket.Conn, error) {
	const baseDelay = 1 * time.Second
	const maxDelay = 30 * time.Second

	var lastErr error
	for attempt := 0; attempt <= maxRetries; attempt++ {
		d.logger.Debug("WebSocket handshake attempt",
			"attempt", attempt+1, "maxRetries", maxRetries, "url", url)

		conn, resp, err := websocket.Dial(ctx, url, opts)
		if err == nil {
			d.logger.Debug("WebSocket handshake succeeded", "attempt", attempt+1, "url", url)
			return conn, nil
		}
		lastErr = err

		// 409 Conflict: this tunnel already has a host
		if resp != nil && resp.StatusCode == http.StatusConflict {
			_ = resp.Body.Close()
			return nil, ErrDuplicateHost
		}

		// 429 Too Many Requests: rate limited
		if resp != nil && resp.StatusCode == http.StatusTooManyRequests {
			body, _ := io.ReadAll(resp.Body)
			_ = resp.Body.Close()
			reason := strings.TrimSpace(string(body))
			if attempt == 0 {
				d.statusf(i18n.T(i18n.MsgGatewayRejected), reason)
			}
			lastErr = fmt.Errorf(i18n.T(i18n.MsgGatewayRejected429), reason)
		} else if attempt == 0 {
			d.statusln(i18n.T(i18n.MsgConnectionFailedRetrying))
		}

		if attempt == maxRetries {
			break
		}

		// exponential backoff + random jitter
		delay := min(baseDelay*time.Duration(1<<uint(attempt)), maxDelay)
		jittered := time.Duration(rand.Int64N(int64(delay)))

		d.logger.Debug("WebSocket dial retry",
			"attempt", attempt+1, "retryAfter", jittered, "err", lastErr)

		select {
		case <-ctx.Done():
			return nil, fmt.Errorf(i18n.T(i18n.MsgDialCancelled), ctx.Err())
		case <-time.After(jittered):
		}
	}
	return nil, fmt.Errorf(i18n.T(i18n.MsgDialFailedRetries), maxRetries, lastErr)
}

// sshTraceFunc returns a trace function that logs SSH protocol events: errors
// at error level, everything else at debug level.
func sshTraceFunc(logger *slog.Logger) ssh.TraceFunc {
	if logger == nil {
		return nil
	}
	return func(level ssh.TraceLevel, eventID int, message string) {
		lvl := slog.LevelDebug
		if level == ssh.TraceLevelError {
			lvl = slog.LevelError
		}
		logger.LogAttrs(context.Background(), lvl, "ssh trace",
			slog.Int("eventID", eventID),
			slog.String("msg", message))
	}
}

// parseSSHCloseError extracts a typed business error from a WebSocket close
// error. Two wire formats are recognised:
//   - application close codes 4001/4002/4003 (designed protocol)
//   - close code 1008 (StatusPolicyViolation) carrying a reason text, which is
//     what the gateway currently sends before the application codes land
func parseSSHCloseError(err error) error {
	var ce websocket.CloseError
	if !errors.As(err, &ce) {
		return err
	}
	switch ce.Code {
	case closeCodeQuotaExceeded:
		return ErrQuotaExceeded
	case closeCodeTunnelNotFound:
		return ErrTunnelNotFound
	case closeCodeDuplicateHost:
		return ErrDuplicateHost
	case websocket.StatusPolicyViolation:
		switch ce.Reason {
		case "account quota exceeded":
			return ErrQuotaExceeded
		case "tunnel not found":
			return ErrTunnelNotFound
		case "tunnel already registered":
			return ErrDuplicateHost
		}
	}
	return err
}
