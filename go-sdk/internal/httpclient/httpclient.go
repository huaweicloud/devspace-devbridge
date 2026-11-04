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
	"sort"
	"strings"
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

// APIError represents a business error returned by the server.
type APIError struct {
	Code    string // error code, e.g. "HD.98320078"
	Message string // error description
}

func (e *APIError) Error() string {
	return fmt.Sprintf("error code: %s, error message: %s", e.Code, e.Message)
}

// UpgradeRequiredError indicates the server rejected the request because the
// client version is below the server's minimum required version (HTTP 426).
type UpgradeRequiredError struct {
	Message string
}

func (e *UpgradeRequiredError) Error() string {
	return e.Message
}

// errorBody is the nested {error: {code, message}} business-error shape.
type errorBody struct {
	Error struct {
		Code    string `json:"code"`
		Message string `json:"message"`
		Target  string `json:"target"`
	} `json:"error"`
}

// flatErrorBody is the flat {code, message} business-error shape (e.g. the
// 426 upgrade-required response).
type flatErrorBody struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

// Client sends REST API requests.
type Client struct {
	APIKey  string
	BaseURL string
	HTTP    *http.Client
	Logger  *slog.Logger
}

// New creates an HTTP client. When httpClient is nil a default client with a
// 30s timeout is created; otherwise the caller-provided client is used.
func New(apiKey, baseURL string, httpClient *http.Client, logger *slog.Logger) *Client {
	if logger == nil {
		logger = slog.Default()
	}
	if httpClient == nil {
		httpClient = &http.Client{
			Timeout: 30 * time.Second,
		}
	}
	return &Client{
		APIKey:  apiKey,
		BaseURL: baseURL,
		HTTP:    httpClient,
		Logger:  logger,
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
	c.logRequestHeaders(req)
	if err != nil {
		return fmt.Errorf(i18n.T(i18n.MsgHTTPRequestFailed), err)
	}
	defer func() { _ = resp.Body.Close() }()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf(i18n.T(i18n.MsgReadResponseFailed), err)
	}

	c.logResponse(resp, respBody, time.Since(start))

	if resp.StatusCode == http.StatusUpgradeRequired {
		return &UpgradeRequiredError{Message: parseMessage(respBody)}
	}

	if resp.StatusCode < http.StatusOK || resp.StatusCode >= http.StatusMultipleChoices {
		if apiErr := parseAPIError(respBody); apiErr != nil {
			return apiErr
		}
		// No X-Served-By: the response did not come from the tunnel (likely
		// blocked by firewall/LB/CDN); only then attach origin/request ID.
		servedBy := resp.Header.Get(headerServedBy)
		if servedBy == "" {
			return fmt.Errorf(i18n.T(i18n.MsgNotReachedTunnel))
		}
		// X-DevBridge-Error-Origin is a custom header WAF/ELB won't rewrite, so
		// it beats Server; "service" marks the tunneled service's own response.
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

// logRequestHeaders dumps the final request headers after the transport has
// run (so headers injected by a caller-provided RoundTripper, such as the
// CLI's User-Agent, are included).
func (c *Client) logRequestHeaders(req *http.Request) {
	if !c.Logger.Enabled(context.Background(), slog.LevelDebug) {
		return
	}
	keys := make([]string, 0, len(req.Header))
	for k := range req.Header {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	attrs := make([]slog.Attr, 0, len(keys))
	for _, k := range keys {
		if strings.EqualFold(k, headerXAPIKey) {
			attrs = append(attrs, slog.String(k, "<redacted>"))
			continue
		}
		attrs = append(attrs, slog.String(k, strings.Join(req.Header[k], ", ")))
	}
	c.Logger.LogAttrs(context.Background(), slog.LevelDebug, "HTTP request headers", attrs...)
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

// parseMessage extracts the "message" field from a flat {"code","message"}
// business response, falling back to the raw body when it can't be parsed.
func parseMessage(body []byte) string {
	var fb flatErrorBody
	if json.Unmarshal(body, &fb) == nil && fb.Message != "" {
		return fb.Message
	}
	return string(body)
}
