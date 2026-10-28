/*
 * Copyright (c) Huawei Technologies Co., Ltd. 2026-2027. All rights reserved.
 */

// Package httpclient implements the low-level HTTP communication for the DevBridge REST API.
//
// It is only for the SDK root package (Go internal mechanism, not importable externally), and handles:
// request construction and auth headers, error status codes, business error parsing, and request/response logging.
package httpclient

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"runtime"
	"time"

	"github.com/huaweicloud/devspace-devbridge/go-sdk/internal/i18n"
)

const (
	headerXAPIKey     = "X-API-Key"
	headerContentType = "content-type"
	headerJSON        = "application/json"
	headerServedBy    = "X-Served-By"
	headerRequestID   = "X-Request-Id"
	headerErrorOrigin = "X-DevBridge-Error-Origin"
)

// legacyRouteNotFoundCode is the business error code the gateway returns for
// requests hitting the deprecated /open-api-inner/v1/relay-controller/* routes.
const legacyRouteNotFoundCode = "40400"

// Install script URLs, kept verbatim in sync with the constants in
// cli/internal/updater/updater.go (go-sdk and cli are separate Go modules,
// so the strings are duplicated here on purpose).
const (
	gitCodeInstallSh = "https://gitcode.com/CloudDeveloperDepartment/devbrige/releases/download/latest/install.sh"
	gitCodeInstallPs = "https://gitcode.com/CloudDeveloperDepartment/devbrige/releases/download/latest/install.ps1"
	githubInstallSh  = "https://github.com/huaweicloud/devspace-devbridge/releases/latest/download/install.sh"
	githubInstallPs  = "https://github.com/huaweicloud/devspace-devbridge/releases/latest/download/install.ps1"
)

// APIError represents a business error returned by the server.
type APIError struct {
	Code    string // error code, e.g. "HD.98320078"
	Message string // error description

	// upgrade, when non-empty, appends an actionable upgrade guide to Error().
	// It is only set for the deprecated-route error code 40400; all other
	// errors render exactly as before.
	upgrade string
}

func (e *APIError) Error() string {
	msg := fmt.Sprintf("error code: %s, error message: %s", e.Code, e.Message)
	if e.upgrade != "" {
		msg += "\n" + e.upgrade
	}
	return msg
}

// upgradeGuide returns the bilingual upgrade notice plus the dual-source
// (GitCode/GitHub) install commands for the current platform, mirroring
// cli/internal/updater/updater.go InstallCommand().
func upgradeGuide() string {
	var cmds string
	if runtime.GOOS == "windows" {
		cmds = fmt.Sprintf("Update:\n  # GitCode\n  irm %s | iex\n\n  # GitHub\n  irm %s | iex",
			gitCodeInstallPs, githubInstallPs)
	} else {
		cmds = fmt.Sprintf("Update:\n  # GitCode\n  curl -fsSL %s | bash\n\n  # GitHub\n  curl -fsSL %s | bash",
			gitCodeInstallSh, githubInstallSh)
	}
	return i18n.T(i18n.MsgLegacyAPIUpgrade) + "\n" + cmds
}

// errorBody is an alternative error format (used by some endpoints).
type errorBody struct {
	Error struct {
		Code    string `json:"code"`
		Message string `json:"message"`
		Target  string `json:"target"`
	} `json:"error"`
}

// Client sends REST API requests.
type Client struct {
	APIKey  string
	BaseURL string
	HTTP    *http.Client
	Logger  *slog.Logger
}

// New creates an HTTP client, using the default logger when logger is nil.
func New(apiKey, baseURL string, logger *slog.Logger) *Client {
	if logger == nil {
		logger = slog.Default()
	}
	return &Client{
		APIKey:  apiKey,
		BaseURL: baseURL,
		HTTP: &http.Client{
			Timeout: 30 * time.Second,
		},
		Logger: logger,
	}
}

func (c *Client) Get(ctx context.Context, path string, result any) error {
	return c.Do(ctx, http.MethodGet, path, nil, result)
}

func (c *Client) Post(ctx context.Context, path string, body, result any) error {
	return c.Do(ctx, http.MethodPost, path, body, result)
}

func (c *Client) Put(ctx context.Context, path string, body, result any) error {
	return c.Do(ctx, http.MethodPut, path, body, result)
}

func (c *Client) Delete(ctx context.Context, path string, result any) error {
	return c.Do(ctx, http.MethodDelete, path, nil, result)
}

// Do sends an HTTP request and handles the response.
func (c *Client) Do(ctx context.Context, method, path string, body, result any) error {
	var bodyBytes []byte
	hasBody := method == http.MethodPost || method == http.MethodPut
	if hasBody && body != nil {
		b, err := json.Marshal(body)
		if err != nil {
			return fmt.Errorf(i18n.T(i18n.MsgMarshalRequestFailed), err)
		}
		bodyBytes = b
	}

	req, err := http.NewRequestWithContext(ctx, method, c.BaseURL+path, bytes.NewReader(bodyBytes))
	if err != nil {
		return fmt.Errorf(i18n.T(i18n.MsgCreateRequestFailed), err)
	}
	req.Header.Set(headerXAPIKey, c.APIKey)
	if hasBody {
		req.Header.Set(headerContentType, headerJSON)
	}

	c.logRequest(req, bodyBytes)

	start := time.Now()
	resp, err := c.HTTP.Do(req)
	if err != nil {
		return fmt.Errorf(i18n.T(i18n.MsgHTTPRequestFailed), err)
	}
	defer func() { _ = resp.Body.Close() }()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf(i18n.T(i18n.MsgReadResponseFailed), err)
	}

	c.logResponse(resp, respBody, time.Since(start))

	if resp.StatusCode < http.StatusOK || resp.StatusCode >= http.StatusMultipleChoices {
		if apiErr := parseAPIError(respBody); apiErr != nil {
			if apiErr.Code == legacyRouteNotFoundCode {
				apiErr.upgrade = upgradeGuide()
			}
			return apiErr
		}
		// X-Served-By 为空表示响应不是经过隧道基础设施产生的：请求可能
		// 被防火墙/LB/CDN 拦截，根本没到达隧道服务。有该头时才附带
		// 来源集群与请求 ID，便于回查服务端日志。
		servedBy := resp.Header.Get(headerServedBy)
		if servedBy == "" {
			return fmt.Errorf(i18n.T(i18n.MsgNotReachedTunnel))
		}
		// X-DevBridge-Error-Origin 区分错误产生方（gateway vs service），
		// 是自定义头、WAF/ELB 不改写，故比 Server 可靠。service 表示隧道内
		// 服务自己返回的响应。
		if resp.Header.Get(headerErrorOrigin) == "service" {
			return fmt.Errorf(i18n.T(i18n.MsgServiceHTTPErrorWithMeta),
				resp.StatusCode, string(respBody), servedBy, resp.Header.Get(headerRequestID))
		}
		return fmt.Errorf(i18n.T(i18n.MsgServerHTTPErrorWithMeta),
			resp.StatusCode, string(respBody), servedBy, resp.Header.Get(headerRequestID))
	}

	if result != nil && len(respBody) > 0 {
		if err := json.Unmarshal(respBody, result); err != nil {
			return fmt.Errorf(i18n.T(i18n.MsgUnmarshalResponseFailed), err)
		}
	}
	return nil
}

func (c *Client) logRequest(req *http.Request, body []byte) {
	if !c.Logger.Enabled(context.Background(), slog.LevelDebug) {
		return
	}
	attrs := []slog.Attr{
		slog.String("method", req.Method),
		slog.String("url", req.URL.String()),
	}
	if len(body) > 0 {
		attrs = append(attrs, slog.String("body", string(body)))
	}
	c.Logger.LogAttrs(context.Background(), slog.LevelDebug, "HTTP request", attrs...)
}

func (c *Client) logResponse(resp *http.Response, body []byte, elapsed time.Duration) {
	if !c.Logger.Enabled(context.Background(), slog.LevelDebug) {
		return
	}
	attrs := []slog.Attr{
		slog.Int("statusCode", resp.StatusCode),
		slog.String("status", resp.Status),
		slog.Int64("elapsed", elapsed.Milliseconds()),
	}
	c.Logger.LogAttrs(context.Background(), slog.LevelDebug, "HTTP response", attrs...)
	if len(body) > 0 {
		c.Logger.LogAttrs(context.Background(), slog.LevelDebug, "HTTP response body",
			slog.String("data", string(body)),
			slog.String("size", fmt.Sprintf("%d bytes total", len(body))),
		)
	}
}

// parseAPIError tries to parse a business error of the form {error: {code, message}} from the response body.
func parseAPIError(body []byte) *APIError {
	var eb errorBody
	if json.Unmarshal(body, &eb) == nil && eb.Error.Code != "" {
		return &APIError{Code: eb.Error.Code, Message: eb.Error.Message}
	}
	return nil
}
