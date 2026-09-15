// Package httpclient 实现 DevBridge REST API 的底层 HTTP 通信。
//
// 仅供 SDK 根包使用（Go internal 机制，外部无法导入），负责：
// 请求构造与认证头、错误状态码处理、业务错误解析、请求/响应日志。
package httpclient

import (
	"bytes"
	"context"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"time"
)

const (
	headerXAPIKey     = "X-API-Key"
	headerContentType = "content-type"
	headerJSON        = "application/json"
)

// APIError 表示服务端返回的业务错误
type APIError struct {
	Code    string // 错误码，如 "HD.98320078"
	Message string // 错误描述
}

func (e *APIError) Error() string {
	return fmt.Sprintf("error code: %s, error message: %s", e.Code, e.Message)
}

// errorBody 是另一种错误格式（部分接口使用）
type errorBody struct {
	Error struct {
		Code    string `json:"code"`
		Message string `json:"message"`
		Target  string `json:"target"`
	} `json:"error"`
}

// Client 发送 REST API 请求
type Client struct {
	APIKey  string
	BaseURL string
	HTTP    *http.Client
	Logger  *slog.Logger
}

// New 创建 HTTP 客户端，HTTP/Logger 为 nil 时使用默认值
func New(apiKey, baseURL string, httpClient *http.Client, logger *slog.Logger) *Client {
	if httpClient == nil {
		httpClient = &http.Client{
			Timeout: 30 * time.Second,
			Transport: &http.Transport{
				TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
			},
		}
	}
	if logger == nil {
		logger = slog.Default()
	}
	return &Client{APIKey: apiKey, BaseURL: baseURL, HTTP: httpClient, Logger: logger}
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

// Do 发送 HTTP 请求并处理响应
func (c *Client) Do(ctx context.Context, method, path string, body, result any) error {
	var bodyBytes []byte
	hasBody := method == http.MethodPost || method == http.MethodPut
	if hasBody && body != nil {
		b, err := json.Marshal(body)
		if err != nil {
			return fmt.Errorf("marshal request body: %w", err)
		}
		bodyBytes = b
	}

	req, err := http.NewRequestWithContext(ctx, method, c.BaseURL+path, bytes.NewReader(bodyBytes))
	if err != nil {
		return fmt.Errorf("create request: %w", err)
	}
	req.Header.Set(headerXAPIKey, c.APIKey)
	if hasBody {
		req.Header.Set(headerContentType, headerJSON)
	}

	c.logRequest(req, bodyBytes)

	start := time.Now()
	resp, err := c.HTTP.Do(req)
	if err != nil {
		return fmt.Errorf("http request failed: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("read response: %w", err)
	}

	c.logResponse(resp, respBody, time.Since(start))

	if resp.StatusCode < http.StatusOK || resp.StatusCode >= http.StatusMultipleChoices {
		if apiErr := parseAPIError(respBody); apiErr != nil {
			return apiErr
		}
		return fmt.Errorf("server error: HTTP %d: %s", resp.StatusCode, string(respBody))
	}

	if result != nil && len(respBody) > 0 {
		if err := json.Unmarshal(respBody, result); err != nil {
			return fmt.Errorf("unmarshal response: %w", err)
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

// parseAPIError 尝试从响应体解析 {error: {code, message}} 格式的业务错误。
func parseAPIError(body []byte) *APIError {
	var eb errorBody
	if json.Unmarshal(body, &eb) == nil && eb.Error.Code != "" {
		return &APIError{Code: eb.Error.Code, Message: eb.Error.Message}
	}
	return nil
}
