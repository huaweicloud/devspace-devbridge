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

// sshEventCategory 根据 eventID 返回 SSH 事件类别标签.
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

// sshTraceFunc 将 dev-tunnels-ssh 的 Trace 事件桥接到 slog.
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

// traceFunc 返回当前日志级别下的 TraceFunc，非 Debug 级别时返回 nil.
func traceFunc() ssh.TraceFunc {
	if isDebugEnabled() {
		return sshTraceFunc
	}
	return nil
}

// 隧道服务连接地址配置（导出，供 cmd 包引用）.
var (
	ServerAddr = "gateway.cn-north-4-bridge.myhuaweicloud.com:443" //nolint:gochecknoglobals // cobra CLI 惯用全局变量
	ServerHost = "cn-north-4-bridge.myhuaweicloud.com"             // clusterId 域名，用于拼接 {tunnelID}.{ServerHost} //nolint:gochecknoglobals // cobra CLI 惯用全局变量
)

// relayChannelType 监听端外层 SSH ClientSession 接收的通道类型.
// 网关在 send端连接时，会在 host 的外层 session 上 OpenChannelWithType("relay") 打开通道
// 监听端通过 AcceptChannel 接收后按 ChannelType 匹配，每个 relay 通道承载一个内层 SSH 会话
const relayChannelType = "relay"

// sessionLookup 监听端内层 ServerSession 映射表.
// key = relay 通道的 ChannelID，value = 该通道上的内层 SSH ServerSession
// 当 send端请求端口转发时，数据通过 relay 通道到达对应的内层 ServerSession
var sessionLookup = make(map[uint32]*ssh.ServerSession) //nolint:gochecknoglobals // cobra CLI 惯用全局变量

// persistentHostKey 隧道生命周期内共用的 host key，确保重连时 host key 不变.
var persistentHostKey ssh.KeyPair //nolint:gochecknoglobals // cobra CLI 惯用全局变量

func init() { //nolint:gochecknoinits // cobra CLI 惯用 init 函数
	var err error
	persistentHostKey, err = ssh.GenerateKeyPair(ssh.AlgoPKEcdsaSha2P256)
	if err != nil {
		log.Fatalf("Failed to generate host key: %v", err)
	}
}

// buildWSHeader 构建 WebSocket 连接所需的 header 和 subprotocols.
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

// dialWebSocket 建立 WebSocket 连接并包装为 net.Conn，供 SSH 使用.
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
				// 域名不在公网 DNS，直接连已知 IP.
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

// dialWithRetry 带指数退避重试的 WebSocket 拨号，最多重试 maxRetries 次.
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

		// 网关返回 429 表示被限流（请求频率或并发连接数），读取 body 显示具体原因.
		if resp != nil && resp.StatusCode == http.StatusTooManyRequests {
			body, _ := io.ReadAll(resp.Body) //nolint:errcheck // 错误路径中读取 body 失败不影响错误返回
			_ = resp.Body.Close()            //nolint:errcheck // 错误路径中关闭 body 失败不可操作
			reason := strings.TrimSpace(string(body))
			if attempt == 0 {
				fmt.Printf("Connection rejected by gateway: %s, retrying...\n", reason)
			}
			lastErr = fmt.Errorf("connection rejected by gateway (429): %s", reason)
		} else if attempt == 0 {
			fmt.Println("Connection failed, retrying...")
		}

		// 最后一次尝试失败，不再退避等待.
		if attempt == maxRetries {
			break
		}

		// 统一指数退避 + 全抖动：用位移替代循环累乘.
		delay := baseDelay * time.Duration(1<<uint(attempt)) //nolint:gosec // attempt 是非负重试计数器，不会溢出
		if delay > maxDelay {
			delay = maxDelay
		}
		// 使用 crypto/rand 生成安全随机数用于退避抖动.
		var jittered time.Duration
		bigN, err := rand.Int(rand.Reader, big.NewInt(int64(delay)))
		if err != nil {
			jittered = delay / 2 // crypto/rand 失败时回退到半延迟.
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
