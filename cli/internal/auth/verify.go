package auth

import (
	"errors"
	"crypto/tls"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"huawei.com/devbridge/internal/config"
	"huawei.com/devbridge/internal/logging"
)

const (
	authCheckPath = "/open-api-inner/v1/relay-controller/auth/check"
	headerXAPIKey = "X-API-Key"
)

// ErrAPIKeyInvalid 表示 API Key 无效（401）。
var ErrAPIKeyInvalid = errors.New("api key is invalid or disabled")

var verifyClient = &http.Client{
	Timeout: 10 * time.Second,
	Transport: &http.Transport{
		TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
	},
}

// Identity 是 check 接口返回的身份信息。
type Identity struct {
	DomainID string `json:"domainId"`
	UserID   string `json:"userId"`
}

// VerifyAPIKey 校验 API Key 有效性，返回 204 表示有效。
func VerifyAPIKey(apiKey string) (*Identity, error) {
	if apiKey == "" {
		return nil, ErrAPIKeyInvalid
	}

	url := strings.TrimRight(config.DefaultServerDomain, "/") + authCheckPath
	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		return nil, fmt.Errorf("build verify request: %w", err)
	}
	req.Header.Set(headerXAPIKey, apiKey)

	logVerifyRequest(req, apiKey)

	start := time.Now()
	resp, err := verifyClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("verify api key: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()
	body, err := io.ReadAll(io.LimitReader(resp.Body, 4096))
	if err != nil {
		return nil, fmt.Errorf("read verify response: %w", err)
	}

	logVerifyResponse(resp, body, time.Since(start))

	switch resp.StatusCode {
	case http.StatusNoContent:
		return &Identity{}, nil
	case http.StatusUnauthorized:
		return nil, ErrAPIKeyInvalid
	default:
		return nil, fmt.Errorf("verify api key: unexpected status %d, body=%s", resp.StatusCode, string(body))
	}
}

func maskAPIKey(key string) string {
	if len(key) <= 8 {
		return strings.Repeat("*", len(key))
	}
	return key[:4] + strings.Repeat("*", len(key)-8) + key[len(key)-4:]
}

func logVerifyRequest(req *http.Request, apiKey string) {
	if logging.LogLevel() > slog.LevelDebug {
		return
	}
	slog.Debug("api key verify request",
		"method", req.Method,
		"url", req.URL.String(),
		"X-API-Key", maskAPIKey(apiKey),
	)
}

func logVerifyResponse(resp *http.Response, body []byte, elapsed time.Duration) {
	if logging.LogLevel() > slog.LevelDebug {
		return
	}
	slog.Debug("api key verify response",
		"statusCode", resp.StatusCode,
		"elapsed", elapsed.Milliseconds(),
		"body", string(body),
	)
}
