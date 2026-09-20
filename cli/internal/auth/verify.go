package auth

import (
	"context"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"huawei.com/devbridge/internal/config"
)

const (
	authCheckPath = config.RelayControllerPath + "/auth/check"
	headerXAPIKey = "X-API-Key"
)

// errAPIKeyInvalid 表示 API Key 无效（401）。
var errAPIKeyInvalid = errors.New("api key is invalid or disabled")

var verifyClient = &http.Client{
	Timeout: 10 * time.Second,
}

// VerifyAPIKey 校验 API Key 有效性，返回 204 表示有效。
func VerifyAPIKey(apiKey string) error {
	if apiKey == "" {
		return errAPIKeyInvalid
	}

	url := strings.TrimRight(config.DefaultServerDomain, "/") + authCheckPath
	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		return fmt.Errorf("build verify request: %w", err)
	}
	req.Header.Set(headerXAPIKey, apiKey)

	logVerifyRequest(req, apiKey)

	start := time.Now()
	resp, err := verifyClient.Do(req)
	if err != nil {
		return fmt.Errorf("verify api key: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()
	body, err := io.ReadAll(io.LimitReader(resp.Body, 4096))
	if err != nil {
		return fmt.Errorf("read verify response: %w", err)
	}

	logVerifyResponse(resp, body, time.Since(start))

	switch resp.StatusCode {
	case http.StatusNoContent:
		return nil
	case http.StatusUnauthorized:
		return errAPIKeyInvalid
	default:
		return fmt.Errorf("verify api key: unexpected status %d, body=%s", resp.StatusCode, string(body))
	}
}

func maskAPIKey(key string) string {
	if len(key) <= 8 {
		return strings.Repeat("*", len(key))
	}
	return key[:4] + strings.Repeat("*", len(key)-8) + key[len(key)-4:]
}

func logVerifyRequest(req *http.Request, apiKey string) {
	if !slog.Default().Enabled(context.Background(), slog.LevelDebug) {
		return
	}
	slog.Debug("api key verify request",
		"method", req.Method,
		"url", req.URL.String(),
		"X-API-Key", maskAPIKey(apiKey),
	)
}

func logVerifyResponse(resp *http.Response, body []byte, elapsed time.Duration) {
	if !slog.Default().Enabled(context.Background(), slog.LevelDebug) {
		return
	}
	slog.Debug("api key verify response",
		"statusCode", resp.StatusCode,
		"elapsed", elapsed.Milliseconds(),
		"body", string(body),
	)
}
