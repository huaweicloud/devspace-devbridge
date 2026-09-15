// Package sdk 提供 DevBridge 隧道服务的 Go 客户端：
// 隧道与端口的 REST API 管理，以及 Host 托管与 Connect 连接能力。
package sdk

import (
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"os"
	"regexp"

	"github.com/huaweicloud/devspace-devbridge/sdk/internal/httpclient"
)

const (
	DefaultAPIBaseURL  = "https://bridge.developer.myhuaweicloud.com/open-api-inner/v1/relay-controller"
	DefaultGatewayAddr = "gateway.cn-north-4-bridge.myhuaweicloud.com:443"
	DefaultGatewayHost = "cn-north-4-bridge.myhuaweicloud.com"
	DefaultClusterID   = "cn-north-4-bridge"
)

var (
	tunnelIDRegexp   = regexp.MustCompile(`^[a-z2-7]{8}$`)
	tunnelNameRegexp = regexp.MustCompile(`^[\x{4e00}-\x{9fa5}A-Za-z0-9]([\x{4e00}-\x{9fa5}A-Za-z0-9-]{0,62}[\x{4e00}-\x{9fa5}A-Za-z0-9])?$`)
	tunnelDescRegexp = regexp.MustCompile(`^[\x{4e00}-\x{9fa5}A-Za-z0-9]{0,64}$`)
)

// AllPortsSentinel 表示"所有端口"的哨兵值，对应后端存储的 -1。
// 它只对 visitor URL 访问有意义（网关按 SNI 动态路由任意端口），
// host/connect 的 SSH 主动端口转发不应转发该值。
const AllPortsSentinel = -1

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

	// HTTPClient optionally overrides the HTTP client used for REST API
	// requests. If nil, a default client with 30s timeout is used.
	HTTPClient *http.Client

	// StatusWriter receives user-facing status lines (connection progress,
	// hosted ports, forwarding info). Defaults to os.Stdout; set it to
	// io.Discard to silence these outputs.
	StatusWriter io.Writer
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
	if out.StatusWriter == nil {
		out.StatusWriter = os.Stdout
	}
	return out
}

// Devbridge DevBridge SDK 客户端
type Devbridge struct {
	apiKey       string
	gatewayAddr  string
	gatewayHost  string
	logger       *slog.Logger
	statusWriter io.Writer
	api          *httpclient.Client
}

// New creates a new SDK client from the given Config.
// A zero Config is valid; APIKey falls back to HW_API_KEY env var,
// and other fields fall back to sensible defaults.
func New(cfg Config) (*Devbridge, error) {
	resolved := cfg.resolve()
	return &Devbridge{
		apiKey:       resolved.APIKey,
		gatewayAddr:  resolved.GatewayAddr,
		gatewayHost:  resolved.GatewayHost,
		logger:       slog.Default(),
		statusWriter: resolved.StatusWriter,
		api:          httpclient.New(resolved.APIKey, resolved.APIBaseURL, resolved.HTTPClient, slog.Default()),
	}, nil
}

func (d *Devbridge) statusf(format string, args ...any) {
	fmt.Fprintf(d.statusWriter, format, args...)
}

func (d *Devbridge) statusln(args ...any) {
	fmt.Fprintln(d.statusWriter, args...)
}

func validateTunnelID(id string) error {
	if !tunnelIDRegexp.MatchString(id) {
		return fmt.Errorf("%w: got %q", ErrInvalidTunnelID, id)
	}
	return nil
}

func validateTunnelDescription(description string) error {
	if !tunnelDescRegexp.MatchString(description) {
		return fmt.Errorf("%w: got %q", ErrInvalidTunnelDescription, description)
	}
	return nil
}

func validatePortNumber(port int) error {
	if port == AllPortsSentinel {
		return nil
	}
	if port < 1 || port > 65535 {
		return fmt.Errorf("%w: got %d", ErrInvalidPort, port)
	}
	return nil
}

// filterForwardPorts 过滤掉"所有端口"哨兵值，返回仅包含真实端口（1-65535）的列表。
// host/connect 的 SSH 端口转发不应转发 -1（哨兵值对应 uint32 的 4294967295，
// 不是合法监听端口），它只对 visitor URL 访问有意义。
func filterForwardPorts(ports []int) []int {
	out := make([]int, 0, len(ports))
	for _, p := range ports {
		if p == AllPortsSentinel {
			continue
		}
		out = append(out, p)
	}
	return out
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
