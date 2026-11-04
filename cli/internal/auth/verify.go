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
	"huawei.com/devbridge/internal/i18n"
)

const (
	// authCheckPath points at the dedicated API-key validation endpoint on the
	// v1 backend. /v1/api-keys/check is implemented by mgmt-service and shared
	// with relay-gateway and bridge-service; it validates the X-API-Key header
	// and returns 200 when the key is valid and 401 when it is not.
	authCheckPath = config.RelayControllerPath + "/api-keys/check"
	headerXAPIKey = "X-API-Key"
)

// errAPIKeyInvalid indicates the API Key is invalid (401).
var errAPIKeyInvalid = errors.New(i18n.T(i18n.Msg.Auth.APIKeyInvalid))

var verifyClient = &http.Client{
	Timeout: 10 * time.Second,
}

// VerifyAPIKey validates the API Key; a 200 response means it is valid.
func VerifyAPIKey(apiKey string) error {
	if apiKey == "" {
		return errAPIKeyInvalid
	}

	url := strings.TrimRight(config.DefaultServerDomain, "/") + authCheckPath
	req, err := http.NewRequest(http.MethodPost, url, nil)
	if err != nil {
		return fmt.Errorf("%s: %w", i18n.T(i18n.Msg.Auth.VerifyRequestFailed), err)
	}
	req.Header.Set(headerXAPIKey, apiKey)

	logVerifyRequest(req, apiKey)

	start := time.Now()
	resp, err := verifyClient.Do(req)
	if err != nil {
		return fmt.Errorf("%s: %w", i18n.T(i18n.Msg.Auth.VerifyAPIKeyFailed), err)
	}
	defer func() { _ = resp.Body.Close() }()
	body, err := io.ReadAll(io.LimitReader(resp.Body, 4096))
	if err != nil {
		return fmt.Errorf("%s: %w", i18n.T(i18n.Msg.Auth.VerifyResponseRead), err)
	}

	logVerifyResponse(resp, body, time.Since(start))

	switch resp.StatusCode {
	case http.StatusOK, http.StatusNoContent:
		return nil
	case http.StatusUnauthorized:
		return errAPIKeyInvalid
	default:
		return fmt.Errorf(i18n.T(i18n.Msg.Auth.VerifyUnexpected), resp.StatusCode, string(body))
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
