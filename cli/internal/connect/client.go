package connect

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"errors"
	"fmt"
	"io"
	"log"
	"log/slog"
	"math/rand"
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

// sshEventCategory 根据 eventID 返回 SSH 事件类别标签
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

// sshTraceFunc 将 dev-tunnels-ssh 的 Trace 事件桥接到 slog
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

// traceFunc 返回当前日志级别下的 TraceFunc，Debug 级别以上时返回 nil
func traceFunc() ssh.TraceFunc {
	if isDebugEnabled() {
		return sshTraceFunc
	}
	return nil
}

// 隧道服务连接地址配置（导出，供 cmd 包引用）
var (
	ServerAddr = "gateway.cn-north-4-bridge.myhuaweicloud.com:443"
	ServerHost = "cn-north-4-bridge.myhuaweicloud.com" // clusterId 域名，用于拼接 {tunnelId}.{ServerHost}
)

// relayChannelType 监听端外层 SSH ClientSession 接收的通道类型
// 网关在 send端连接时，会在 host 的外层 session 上 OpenChannelWithType("relay") 打开通道
// 监听端通过 AcceptChannelWithType("relay") 接收，每个 relay 通道承载一个内层 SSH 会话
const relayChannelType = "relay"

// sessionLookup 监听端内层 ServerSession 映射表
// key = relay 通道的 ChannelID，value = 该通道上的内层 SSH ServerSession
// 当 send端请求端口转发时，数据通过 relay 通道到达对应的内层 ServerSession
var sessionLookup = make(map[uint32]*ssh.ServerSession)

// persistentHostKey 隧道生命周期内共用的 host key，确保重连时 host key 不变
var persistentHostKey ssh.KeyPair

func init() {
	var err error
	persistentHostKey, err = ssh.GenerateKeyPair(ssh.AlgoPKEcdsaSha2P256)
	if err != nil {
		log.Fatalf("Failed to generate host key: %v", err)
	}
}

// buildWSHeader 构建 WebSocket 连接所需的 header 和 subprotocols
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

// dialWebSocket 建立 WebSocket 连接并包装为 net.Conn，供 SSH 使用
func dialWebSocket(ctx context.Context, wsURL, host string, header http.Header, subprotocols []string, maxRetries int) (net.Conn, error) {

	dialCtx, dialCancel := context.WithTimeout(ctx, 30*time.Second)
	defer dialCancel()

	conn, err := dialWithRetry(dialCtx, wsURL, &websocket.DialOptions{
		HTTPClient:   getHttpClient(host),
		HTTPHeader:   header,
		Subprotocols: subprotocols,
	}, maxRetries)
	if err != nil {
		return nil, err
	}
	return websocket.NetConn(ctx, conn, websocket.MessageBinary), nil
}

var caCert = []byte(`-----BEGIN CERTIFICATE-----
MIIEGzCCAoOgAwIBAgIUE5r3H5raO0s3FQGXcrrFm4yttkMwDQYJKoZIhvcNAQEL
BQAwHTEbMBkGA1UEAwwSTW9jayBMb2NhbCBSb290IENBMB4XDTI2MDcwNzA2Mjk1
NloXDTM2MDcwNDA2Mjk1NlowHTEbMBkGA1UEAwwSTW9jayBMb2NhbCBSb290IENB
MIIBojANBgkqhkiG9w0BAQEFAAOCAY8AMIIBigKCAYEAknEJQtH5SXOYOZ2GNd8S
bkGw1XZ3NhKNQZIg+Oa0PTpfwfrLw6H1Ce78JpeRUXuHiT3eXgk0MMUUiGD2Ta5H
tkAOj4x526/oeN6Ge9y07n8+zmpaLq2m295XJiUgYY4rjTScsGOJLSixqR5chMPF
KqE/jxfwJBFyFSyO1HXkHQAqmxBGH0uxyAZ3QacZynPdtAbZHgHm7nTld0olvD04
j8nyGGUixUnOH5zySqO3ch3V/hl6eruGv3M76buFUo8BL48U9oK4xlkz/K6ai2S0
TIIipeYyb9pFOQamBTEgEamJn35/Nij+ggR5SYgLE/8dWVUB+PtbDNS+2ZIHqBK5
wsHd+QnbhDOTg0Z4YIV0LDO8g+2jgmipUVA561Xxx6ahSc0PKCEE+DyEiKJsEuc6
i+1h8Jvtgxs8vlxM+FifNKluf78lyobWguK+autiKKuHqrpA2lypPsbf32SFOFx9
j5K5EQwFODVoTDmmy4iRgxiO24YhDFqOIgSr4MA9IYv1AgMBAAGjUzBRMB0GA1Ud
DgQWBBR+tk9hb1UjHc5WQrHq6qN5fPmvWTAfBgNVHSMEGDAWgBR+tk9hb1UjHc5W
QrHq6qN5fPmvWTAPBgNVHRMBAf8EBTADAQH/MA0GCSqGSIb3DQEBCwUAA4IBgQBQ
A8Rs0WikyMJqzWXBsBQtAbzvfnfACFHY4CjR5MjDwIZatdRoXaAJ9YAeG552/P0S
Zt0UpL+R7aac+p142AgItFFD5k4N5q88w1zqQ0kARf1ea1taweB1ZGPp0QeHd50H
3pvb0EYKNhNL0/vNyj0SarzfUCYSCHQf+eVFUiWuI0Oe2++O2zWzTBlkRVG3cXed
rds1CXtd4Chpz1uKLuMeF8IJO8Avo26FNUDiC9hxnIJzUwbeeJTVylKhiWp7/i/O
jpGhgXcwAsLTaXPzLKpcrZNn6990eCDCOqrzgpQN/rNc2kcTKFUDna6F9uMVC+1W
nkUC9heEYACEbsaRQK+PUMhE/0pjYR5mPyPjpRVjltRBQWnVXT1HqMfjHCl2NyJe
f3DKOEWdkHNVkzZA4L7qu5gTDsaUL2K1U3B21mZF4PVgMqqq4XuxEL+uAyxhkpi3
hbrxv7TeS3kQ4OgX0rorzWsq6eGhiAvwRQcqUSrG2nfkl9gAhkEDxULex9GqvqQ=
-----END CERTIFICATE-----
`)

func getHttpClient(serverHost string) *http.Client {
	dialer := &net.Dialer{}
	return &http.Client{
		Transport: &http.Transport{
			TLSClientConfig: createTLSConfig(serverHost),
			DialContext: func(ctx context.Context, network, addr string) (net.Conn, error) {
				// 域名不在公网 DNS，直接连已知 IP
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
	rootPool.AppendCertsFromPEM(caCert)
	return &tls.Config{
		MinVersion: tls.VersionTLS12,
		MaxVersion: tls.VersionTLS13,
		ServerName: serverHost,
		RootCAs:    rootPool,
	}
}

// dialWithRetry 带指数退避重试的 WebSocket 拨号，最多重试 maxRetries 次
func dialWithRetry(ctx context.Context, url string, opts *websocket.DialOptions, maxRetries int) (*websocket.Conn, error) {
	baseDelay := 1 * time.Second
	maxDelay := 30 * time.Second

	for attempt := 0; ; attempt++ {
		slog.Debug("WebSocket handshake attempt", "attempt", attempt+1, "maxRetries", maxRetries, "url", url)
		conn, resp, err := websocket.Dial(ctx, url, opts)
		if err == nil {
			slog.Debug("WebSocket handshake succeeded", "attempt", attempt+1, "url", url)
			return conn, nil
		}

		if resp != nil && resp.StatusCode == http.StatusConflict {
			return nil, errDuplicateHost
		}

		// 网关返回 429 表示被限流（请求频率或并发连接数），读取 body 显示具体原因
		if resp != nil && resp.StatusCode == http.StatusTooManyRequests {
			body, _ := io.ReadAll(resp.Body)
			resp.Body.Close()
			reason := strings.TrimSpace(string(body))
			if reason == "" {
				reason = "too many requests"
			}
			if attempt == 0 {
				fmt.Printf("Connection rejected by gateway: %s, retrying...\n", reason)
			}
			// 429 是临时限流，等待后重试而不是直接退出
			delay := baseDelay
			for i := 0; i < attempt; i++ {
				delay *= 2
				if delay >= maxDelay {
					delay = maxDelay
					break
				}
			}
			jittered := time.Duration(rand.Int63n(int64(delay)))

			if attempt >= maxRetries {
				return nil, fmt.Errorf("connection rejected by gateway (429): %s", reason)
			}
			slog.Debug("WebSocket dial rejected by 429, retrying", "attempt", attempt+1, "maxRetries", maxRetries, "retryAfter", jittered, "reason", reason)

			select {
			case <-ctx.Done():
				return nil, fmt.Errorf("websocket dial cancelled: %w", ctx.Err())
			case <-time.After(jittered):
			}
			continue
		}

		if attempt >= maxRetries {
			return nil, fmt.Errorf("websocket dial failed after %d retries: %w", maxRetries, err)
		}

		// 指数退避 + 全抖动
		delay := baseDelay
		for i := 0; i < attempt; i++ {
			delay *= 2
			if delay >= maxDelay {
				delay = maxDelay
				break
			}
		}
		jittered := time.Duration(rand.Int63n(int64(delay)))

		if attempt == 0 {
			fmt.Println("Connection failed, retrying...")
		}
		slog.Debug("WebSocket dial retry", "attempt", attempt+1, "maxRetries", maxRetries, "retryAfter", jittered, "err", err)

		select {
		case <-ctx.Done():
			return nil, fmt.Errorf("websocket dial cancelled: %w", ctx.Err())
		case <-time.After(jittered):
		}
	}
}
