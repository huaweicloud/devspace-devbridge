/*
 * Copyright (c) Huawei Technologies Co., Ltd. 2026-2027. All rights reserved.
 */

package sdk

import (
	"errors"
	"fmt"
	"strings"

	"github.com/huaweicloud/devspace-devbridge/go-sdk/internal/httpclient"
	"github.com/huaweicloud/devspace-devbridge/go-sdk/internal/i18n"
)

var ErrMissingAPIKey = errors.New("missing API key: set it via Config.APIKey or HW_API_KEY env var")

// ErrAuthFailed is the sentinel for terminal authentication failures
// (expired/invalid token or API key). It is matched with errors.Is so callers
// can stop retrying immediately instead of treating it as a network flap.
var ErrAuthFailed = errors.New("authentication failed")

// AuthFailedError carries the gateway's authentication detail while still
// unwrapping to ErrAuthFailed. StatusCode is the handshake HTTP status when the
// failure happened during the WebSocket handshake (401/403), or 0 for a
// session-level close reason.
type AuthFailedError struct {
	StatusCode int
	Message    string
}

func (e *AuthFailedError) Error() string {
	base := i18n.T(i18n.MsgAuthFailed)
	if detail := strings.TrimSpace(e.Message); detail != "" {
		return fmt.Sprintf("%s: %s", base, detail)
	}
	return base
}

// Unwrap lets errors.Is(err, ErrAuthFailed) succeed.
func (e *AuthFailedError) Unwrap() error { return ErrAuthFailed }

var ErrTunnelNotFound = errors.New("tunnel not found")

var ErrDuplicateHost = errors.New("host already connected for this tunnel")

var ErrQuotaExceeded = errors.New("account quota exceeded")

var ErrInvalidTunnelID = errors.New("invalid tunnel id")

var ErrInvalidTunnelName = errors.New("invalid tunnel name: only Chinese characters, letters, digits, hyphens, length 1-64")

var ErrInvalidTunnelDescription = errors.New("invalid tunnel description: only Chinese characters, letters, digits, length 0-64")

var ErrInvalidPort = errors.New("invalid port number: must be 1-65535")

var ErrInvalidProtocol = errors.New("invalid protocol: must be http, https, or auto")

var ErrInvalidScope = errors.New("invalid token scope: must be host or connect")

// APIError represents a business error returned by the server.
type APIError = httpclient.APIError

// IsAPIError reports whether err is an APIError and returns its code.
func IsAPIError(err error) (code string, ok bool) {
	var apiErr *APIError
	if errors.As(err, &apiErr) {
		return apiErr.Code, true
	}
	return "", false
}

// UpgradeRequiredError indicates the server rejected a request because the
// client version is below the required minimum (HTTP 426 Upgrade Required).
type UpgradeRequiredError = httpclient.UpgradeRequiredError

// IsUpgradeRequired reports whether err indicates an upgrade-required response.
func IsUpgradeRequired(err error) bool {
	var ue *UpgradeRequiredError
	return errors.As(err, &ue)
}
