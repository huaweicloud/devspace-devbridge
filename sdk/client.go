package devbridge

import (
	"bytes"
	"context"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"os"
	"regexp"
	"time"
)

// ──────────────────────────────────────────────────────────────
// Client — SDK 主入口
// 一个 Client 持有配置（API Key、服务地址），提供 REST API 方法和 Host/Connect 方法
// ──────────────────────────────────────────────────────────────

// 默认服务地址
const (
	// DefaultAPIBaseURL REST API 基础地址（与 CLI 生产环境对齐）
	// 对应 cli/.goreleaser.yaml 中注入的 DefaultServerDomain + cli/internal/api/client.go 的路径后缀
	DefaultAPIBaseURL = "https://bridge.developer.myhuaweicloud.com/open-api-inner/v1/relay-controller"

	// DefaultGatewayAddr WebSocket 网关地址
	DefaultGatewayAddr = "gateway.cn-north-4-bridge.myhuaweicloud.com:443"

	// DefaultGatewayHost WebSocket 网关 SNI host
	DefaultGatewayHost = "cn-north-4-bridge.myhuaweicloud.com"

	// DefaultClusterID 默认集群
	DefaultClusterID = "cn-north-4-bridge"
)

var (
	tunnelIDRegexp   = regexp.MustCompile(`^[a-z2-7]{8}$`)
	tunnelNameRegexp = regexp.MustCompile(`^[\x{4e00}-\x{9fa5}A-Za-z0-9]([\x{4e00}-\x{9fa5}A-Za-z0-9-]{0,62}[\x{4e00}-\x{9fa5}A-Za-z0-9])?$`)
)

// Config holds the SDK client configuration. A zero Config is valid —
// missing fields fall back to environment variables and sensible defaults.
type Config struct {
	// APIKey authenticates REST API requests (X-API-Key header).
	// Defaults to HW_API_KEY environment variable.
	APIKey string

	// APIBaseURL is the REST API base URL.
	// Defaults to DefaultAPIBaseURL.
	APIBaseURL string

	// GatewayAddr is the WebSocket gateway address (host:port).
	// Defaults to DefaultGatewayAddr.
	GatewayAddr string

	// GatewayHost is the WebSocket gateway SNI host.
	// Defaults to DefaultGatewayHost.
	GatewayHost string

	// HTTPClient optionally overrides the HTTP client.
	// If nil, a default client with 30s timeout is used.
	HTTPClient *http.Client

	// Logger is the structured logger.
	// Defaults to slog.Default().
	Logger *slog.Logger
}

// resolve returns a copy with defaults and env-var fallbacks applied.
func (cfg Config) resolve() Config {
	out := cfg
	if out.APIBaseURL == "" {
		out.APIBaseURL = DefaultAPIBaseURL
	}
	if out.GatewayAddr == "" {
		out.GatewayAddr = DefaultGatewayAddr
	}
	if out.GatewayHost == "" {
		out.GatewayHost = DefaultGatewayHost
	}
	if out.APIKey == "" {
		out.APIKey = os.Getenv("HW_API_KEY")
	}
	if out.Logger == nil {
		out.Logger = slog.Default()
	}
	if out.HTTPClient == nil {
		out.HTTPClient = &http.Client{
			Timeout: 30 * time.Second,
			Transport: &http.Transport{
				TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
			},
		}
	}
	return out
}

// Client DevBridge SDK 客户端
type Client struct {
	apiKey      string       // API Key，用于 REST API 认证
	apiBaseURL  string       // REST API 基础地址
	gatewayAddr string       // WebSocket 网关地址（host:port）
	gatewayHost string       // WebSocket 网关 SNI host
	httpClient  *http.Client // HTTP 客户端
	logger      *slog.Logger // 日志
}

// NewClient creates a new SDK client from the given Config.
// A zero Config is valid; APIKey falls back to HW_API_KEY env var,
// and other fields fall back to sensible defaults.
//
// 最少只需要 API Key：
//
//	client, err := devbridge.NewClient(devbridge.Config{
//	    APIKey: "your-api-key",
//	})
//
// 也会自动读取 HW_API_KEY 环境变量：
//
//	os.Setenv("HW_API_KEY", "your-key")
//	client, err := devbridge.NewClient(devbridge.Config{})
func NewClient(cfg Config) (*Client, error) {
	resolved := cfg.resolve()
	return &Client{
		apiKey:      resolved.APIKey,
		apiBaseURL:  resolved.APIBaseURL,
		gatewayAddr: resolved.GatewayAddr,
		gatewayHost: resolved.GatewayHost,
		httpClient:  resolved.HTTPClient,
		logger:      resolved.Logger,
	}, nil
}

// ──────────────────────────────────────────────────────────────
// 内部 HTTP 请求方法
// ──────────────────────────────────────────────────────────────

const (
	headerXAPIKey     = "X-API-Key"
	headerContentType = "content-type"
	headerJSON        = "application/json"
)

func (c *Client) resolveAPIKey() (string, error) {
	if c.apiKey == "" {
		return "", ErrMissingAPIKey
	}
	return c.apiKey, nil
}

// isDebugEnabled 检查当前 logger 是否启用了 Debug 级别
func (c *Client) isDebugEnabled() bool {
	return c.logger.Enabled(context.Background(), slog.LevelDebug)
}

// logHTTPRequest 记录 HTTP 请求日志（Debug 级别）
func (c *Client) logHTTPRequest(req *http.Request, body []byte) {
	if !c.isDebugEnabled() {
		return
	}
	attrs := []slog.Attr{
		slog.String("method", req.Method),
		slog.String("url", req.URL.String()),
	}
	if len(body) > 0 {
		attrs = append(attrs, slog.String("body", string(body)))
	}
	c.logger.LogAttrs(context.Background(), slog.LevelDebug, "HTTP request", attrs...)
}

// logHTTPResponse 记录 HTTP 响应日志（Debug 级别）
func (c *Client) logHTTPResponse(resp *http.Response, body []byte, elapsed time.Duration) {
	if !c.isDebugEnabled() {
		return
	}
	attrs := []slog.Attr{
		slog.Int("statusCode", resp.StatusCode),
		slog.String("status", resp.Status),
		slog.Int64("elapsed", elapsed.Milliseconds()),
	}
	c.logger.LogAttrs(context.Background(), slog.LevelDebug, "HTTP response", attrs...)
	if len(body) > 0 {
		c.logger.LogAttrs(context.Background(), slog.LevelDebug, "HTTP response body",
			slog.String("data", string(body)),
			slog.String("size", fmt.Sprintf("%d bytes total", len(body))),
		)
	}
}

// doRequest 发送 HTTP 请求并处理响应
func (c *Client) doRequest(ctx context.Context, method, path string, body any, result any) error {
	apiKey, err := c.resolveAPIKey()
	if err != nil {
		return err
	}

	url := c.apiBaseURL + path

	var bodyBytes []byte
	hasBody := method == http.MethodPost || method == http.MethodPut
	if hasBody && body != nil {
		bodyBytes, err = json.Marshal(body)
		if err != nil {
			return fmt.Errorf("marshal request body: %w", err)
		}
	}

	req, err := http.NewRequestWithContext(ctx, method, url, bytes.NewReader(bodyBytes))
	if err != nil {
		return fmt.Errorf("create request: %w", err)
	}
	req.Header.Set(headerXAPIKey, apiKey)
	if hasBody {
		req.Header.Set(headerContentType, headerJSON)
	}

	c.logHTTPRequest(req, bodyBytes)

	start := time.Now()
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("http request failed: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("read response: %w", err)
	}

	c.logHTTPResponse(resp, respBody, time.Since(start))

	// 处理 HTTP 错误状态码
	if resp.StatusCode < http.StatusOK || resp.StatusCode >= http.StatusMultipleChoices {
		if apiErr := parseAPIError(respBody); apiErr != nil {
			return apiErr
		}
		return fmt.Errorf("server error: HTTP %d: %s", resp.StatusCode, string(respBody))
	}

	// 解析响应
	if result != nil && len(respBody) > 0 {
		if err := json.Unmarshal(respBody, result); err != nil {
			return fmt.Errorf("unmarshal response: %w", err)
		}
	}
	return nil
}

// parseAPIError 从响应体解析错误
func parseAPIError(body []byte) *APIError {
	// 尝试 {error: {code, message}} 格式
	var eb errorBody
	if json.Unmarshal(body, &eb) == nil && eb.Error.Code != "" {
		return &APIError{Code: eb.Error.Code, Message: eb.Error.Message}
	}
	return nil
}

func (c *Client) get(ctx context.Context, path string, result any) error {
	return c.doRequest(ctx, http.MethodGet, path, nil, result)
}

func (c *Client) post(ctx context.Context, path string, body, result any) error {
	return c.doRequest(ctx, http.MethodPost, path, body, result)
}

func (c *Client) put(ctx context.Context, path string, body, result any) error {
	return c.doRequest(ctx, http.MethodPut, path, body, result)
}

func (c *Client) delete(ctx context.Context, path string, result any) error {
	return c.doRequest(ctx, http.MethodDelete, path, nil, result)
}

// ──────────────────────────────────────────────────────────────
// 校验方法
// ──────────────────────────────────────────────────────────────

func validateTunnelID(id string) error {
	if !tunnelIDRegexp.MatchString(id) {
		return fmt.Errorf("%w: got %q", ErrInvalidTunnelID, id)
	}
	return nil
}

func validatePortNumber(port int) error {
	if port < 1 || port > 65535 {
		return fmt.Errorf("%w: got %d", ErrInvalidPort, port)
	}
	return nil
}

func validateProtocol(protocol string) error {
	switch protocol {
	case "http", "https", "auto", "":
		return nil
	default:
		return fmt.Errorf("%w: got %s", ErrInvalidProtocol, protocol)
	}
}

func validateScope(scope string) error {
	if scope != "host" && scope != "connect" {
		return fmt.Errorf("%w: got %s", ErrInvalidScope, scope)
	}
	return nil
}
