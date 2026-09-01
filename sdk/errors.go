package devbridge

import (
	"errors"
	"fmt"
)

// ──────────────────────────────────────────────────────────────
// 自定义错误类型
// 让调用方能精确区分不同错误，而不是拿到一个模糊的 error
// ──────────────────────────────────────────────────────────────

// ErrMissingAPIKey 缺少 API Key
var ErrMissingAPIKey = errors.New("missing API key: set it via NewClient(WithAPIKey) or HW_API_KEY env var")

// ErrTunnelNotFound 隧道不存在
var ErrTunnelNotFound = errors.New("tunnel not found")

// ErrDuplicateHost 该隧道已有 Host 在运行
var ErrDuplicateHost = errors.New("host already connected for this tunnel")

// ErrQuotaExceeded 配额超限
var ErrQuotaExceeded = errors.New("account quota exceeded")

// ErrInvalidTunnelID 隧道 ID 格式无效
var ErrInvalidTunnelID = errors.New("invalid tunnel ID: must be 8 chars of lowercase letters and digits 2-7")

// ErrInvalidPort 端口号无效
var ErrInvalidPort = errors.New("invalid port number: must be 1-65535")

// ErrInvalidProtocol 协议无效
var ErrInvalidProtocol = errors.New("invalid protocol: must be http, https, or auto")

// ErrInvalidScope 令牌 scope 无效
var ErrInvalidScope = errors.New("invalid token scope: must be host or connect")

// APIError 表示服务端返回的业务错误
type APIError struct {
	Code    string // 错误码，如 "HD.98320078"
	Message string // 错误描述
}

func (e *APIError) Error() string {
	return fmt.Sprintf("api error [%s]: %s", e.Code, e.Message)
}

// IsAPIError 判断 error 是否为 APIError，并返回错误码
func IsAPIError(err error) (code string, ok bool) {
	var apiErr *APIError
	if errors.As(err, &apiErr) {
		return apiErr.Code, true
	}
	return "", false
}
