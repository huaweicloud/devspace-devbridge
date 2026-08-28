package connect

import (
	"context"
	"crypto/rand"
	"crypto/tls"
	"crypto/x509"
	"errors"
	"fmt"
	"io"
	"log"
	"log/slog"
	"math/big"
	"net"
	"net/http"
	"strings"
	"time"

	"github.com/coder/websocket"
	"github.com/microsoft/dev-tunnels-ssh/src/go/ssh"
)

var (
	errDuplicateHost  = errors.New("host already connected for this tunnel")
	errQuotaExceeded  = errors.New("account quota exceeded")
	errTunnelNotFound = errors.New("tunnel not found")
)

func isDebugEnabled() bool {
	return slog.Default().Enabled(context.Background(), slog.LevelDebug)
}

func sshEventCategory(eventID int) string {
	switch {
	case eventID >= 1 && eventID <= 10:
		return "connection"
	case eventID >= 11 && eventID <= 20:
		return "channel"
	case eventID >= 21 && eventID <= 30:
		return "data-frame"
	case eventID >= 31 && eventID <= 40:
		return "auth"
	case eventID >= 41 && eventID <= 50:
		return "keepalive"
	default:
		return "other"
	}
}

func sshTraceFunc(level ssh.TraceLevel, eventID int, message string) {
	category := sshEventCategory(eventID)
	switch level {
	case ssh.TraceLevelError:
		slog.Error("ssh trace", "category", category, "eventID", eventID, "msg", message)
	case ssh.TraceLevelWarning:
		slog.Warn("ssh trace", "category", category, "eventID", eventID, "msg", message)
	case ssh.TraceLevelInfo:
		slog.Info("ssh trace", "category", category, "eventID", eventID, "msg", message)
	case ssh.TraceLevelVerbose:
		slog.Debug("ssh trace", "category", category, "eventID", eventID, "msg", message)
	}
}

func traceFunc() ssh.TraceFunc {
	if isDebugEnabled() {
		return sshTraceFunc
	}
	return nil
}

// 隧道服务连接地址配置（导出，供 ldflags 注入）.
var (
	ServerAddr = "gateway.cn-north-4-bridge.myhuaweicloud.com:443"
	ServerHost = "cn-north-4-bridge.myhuaweicloud.com"
)

const relayChannelType = "relay"

var sessionLookup = make(map[uint32]*ssh.ServerSession)

var persistentHostKey ssh.KeyPair

func init() {
	var err error
	persistentHostKey, err = ssh.GenerateKeyPair(ssh.AlgoPKEcdsaSha2P256)
	if err != nil {
		log.Fatalf("Failed to generate host key: %v", err)
	}
}

func buildWSHeader(jwtToken string, apiKey string) (http.Header, []string) {
	header := http.Header{}
	subprotocols := []string{"devbridge-v1"}
	if apiKey != "" {
		header.Set("X-API-Key", apiKey)
	}
	if jwtToken != "" {
		header.Set("Sec-WebSocket-Protocol", "devbridge-v1, "+jwtToken)
		subprotocols = append(subprotocols, jwtToken)
	}
	return header, subprotocols
}

func dialWebSocket(ctx context.Context, wsURL, host string, header http.Header, subprotocols []string, maxRetries int) (net.Conn, error) {

	dialCtx, dialCancel := context.WithTimeout(ctx, 30*time.Second)
	defer dialCancel()

	conn, err := dialWithRetry(dialCtx, wsURL, &websocket.DialOptions{
		HTTPClient:   getHTTPClient(host),
		HTTPHeader:   header,
		Subprotocols: subprotocols,
	}, maxRetries)
	if err != nil {
		return nil, err
	}
	return websocket.NetConn(ctx, conn, websocket.MessageBinary), nil
}

func getHTTPClient(serverHost string) *http.Client {
	dialer := &net.Dialer{}
	return &http.Client{
		Transport: &http.Transport{
			TLSClientConfig: createTLSConfig(serverHost),
			DialContext: func(ctx context.Context, network, addr string) (net.Conn, error) {

				return dialer.DialContext(ctx, network, ServerAddr)
			},
		},
	}
}

func createTLSConfig(serverHost string) *tls.Config {
	rootPool, err := x509.SystemCertPool()
	if err != nil {
		rootPool = x509.NewCertPool()
	}
	return &tls.Config{
		MinVersion: tls.VersionTLS12,
		MaxVersion: tls.VersionTLS13,
		ServerName: serverHost,
		RootCAs:    rootPool,
	}
}

func dialWithRetry(ctx context.Context, url string, opts *websocket.DialOptions, maxRetries int) (*websocket.Conn, error) {
	const baseDelay = 1 * time.Second
	const maxDelay = 30 * time.Second

	var lastErr error
	for attempt := 0; attempt <= maxRetries; attempt++ {
		slog.Debug("WebSocket handshake attempt", "attempt", attempt+1, "maxRetries", maxRetries, "url", url)
		conn, resp, err := websocket.Dial(ctx, url, opts)
		if err == nil {
			slog.Debug("WebSocket handshake succeeded", "attempt", attempt+1, "url", url)
			return conn, nil
		}
		lastErr = err

		if resp != nil && resp.StatusCode == http.StatusConflict {
			return nil, errDuplicateHost
		}

		if resp != nil && resp.StatusCode == http.StatusTooManyRequests {
			body, _ := io.ReadAll(resp.Body)
			_ = resp.Body.Close()
			reason := strings.TrimSpace(string(body))
			if attempt == 0 {
				fmt.Printf("Connection rejected by gateway: %s, retrying...\n", reason)
			}
			lastErr = fmt.Errorf("connection rejected by gateway (429): %s", reason)
		} else if attempt == 0 {
			fmt.Println("Connection failed, retrying...")
		}

		if attempt == maxRetries {
			break
		}

		delay := baseDelay * time.Duration(1<<uint(attempt))
		if delay > maxDelay {
			delay = maxDelay
		}

		var jittered time.Duration
		bigN, err := rand.Int(rand.Reader, big.NewInt(int64(delay)))
		if err != nil {
			jittered = delay / 2
		} else {
			jittered = time.Duration(bigN.Int64())
		}

		slog.Debug("WebSocket dial retry", "attempt", attempt+1,
			"maxRetries", maxRetries, "retryAfter", jittered, "err", lastErr)

		select {
		case <-ctx.Done():
			return nil, fmt.Errorf("websocket dial cancelled: %w", ctx.Err())
		case <-time.After(jittered):
		}
	}
	return nil, fmt.Errorf("websocket dial failed after %d retries: %w", maxRetries, lastErr)
}
