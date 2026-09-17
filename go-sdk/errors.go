/*
 * Copyright (c) Huawei Technologies Co., Ltd. 2026-2027. All rights reserved.
 */

package sdk

import (
	"errors"

	"github.com/huaweicloud/devspace-devbridge/go-sdk/internal/httpclient"
)

var ErrMissingAPIKey = errors.New("missing API key: set it via Config.APIKey or HW_API_KEY env var")

var ErrTunnelNotFound = errors.New("tunnel not found")

var ErrDuplicateHost = errors.New("host already connected for this tunnel")

var ErrQuotaExceeded = errors.New("account quota exceeded")

var ErrInvalidTunnelID = errors.New("invalid tunnel id")

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
