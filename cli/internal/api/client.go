package api

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"huawei.com/devbridge/internal/auth"
	"huawei.com/devbridge/internal/config"
	"huawei.com/devbridge/internal/i18n"
	"huawei.com/devbridge/internal/logging"
)

const (
	TunnelNotFoundCode = "10002"

	headerXAPIKey         = "X-API-Key"
	headerContentType     = "content-type"
	headerApplicationJSON = "application/json"
)

var errMissingAPIKey = errors.New("missing api key")

func signRequest(cred *auth.Credential, req *http.Request) error {
	if cred == nil || cred.APIKey == "" {
		return errMissingAPIKey
	}

	req.Header.Set(headerXAPIKey, cred.APIKey)
	if req.Header.Get(headerContentType) == "" {
		req.Header.Set(headerContentType, headerApplicationJSON)
	}
	return nil
}

type apiError struct {
	Code    string
	Message string
}

func (e *apiError) Error() string {
	return fmt.Sprintf("error code: %s, error message: %s", e.Code, e.Message)
}

func GetApiErrorCode(err error) string {
	var apiErr *apiError
	if errors.As(err, &apiErr) {
		return apiErr.Code
	}
	return ""
}

// errorBody 适配接口错误返回结构：{"error":{"code":"10007","message":"...","target":"name"}}
type errorBody struct {
	Error struct {
		Code    string `json:"code"`
		Message string `json:"message"`
		Target  string `json:"target"`
	} `json:"error"`
}

type restClientType struct {
	httpClient *http.Client
	BaseURL    string
}

var restClient *restClientType //nolint:gochecknoglobals // cobra CLI 惯用全局变量

func InitClient(baseURL string) {
	if baseURL == "" {
		// /open-api-inner/v1/relay-controller/tunnels.
		baseURL = config.DefaultServerDomain + "/open-api-inner/v1/relay-controller"
	}
	restClient = &restClientType{
		httpClient: &http.Client{Timeout: 30 * time.Second},
		BaseURL:    baseURL,
	}
}

func getClient() *restClientType {
	if restClient == nil {
		InitClient("")
	}
	return restClient
}

func isDebugEnabled() bool {
	return logging.LogLevel() <= slog.LevelDebug
}

func logHTTPRequest(req *http.Request, body []byte) {
	if !isDebugEnabled() {
		return
	}
	attrs := []slog.Attr{
		slog.String("method", req.Method),
		slog.String("url", req.URL.String()),
	}
	if len(body) > 0 {
		attrs = append(attrs, slog.String("body", string(body)))
	}
	slog.LogAttrs(context.Background(), slog.LevelDebug, "HTTP request", attrs...)
}

func logHTTPResponse(resp *http.Response, body []byte, elapsed time.Duration) {
	if !isDebugEnabled() {
		return
	}
	slog.Debug("HTTP response",
		"statusCode", resp.StatusCode,
		"status", resp.Status,
		"elapsed", elapsed.Milliseconds(),
	)
	slog.Debug("HTTP response body",
		"data", string(body),
		"size", fmt.Sprintf("%d bytes total", len(body)),
	)
}

func doRequest(req *http.Request) (*http.Response, error) {
	cred := auth.ReadValidAPIKey()
	if cred == nil {
		return nil, errors.New(i18n.T(i18n.Msg.API.APIKeyExpired))
	}
	if err := signRequest(cred, req); err != nil {
		return nil, err
	}
	start := time.Now()
	resp, err := getClient().httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode == http.StatusUnauthorized {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 4096)) //nolint:errcheck // 错误路径中读取 body 失败不影响错误返回
		_ = resp.Body.Close()                                  //nolint:errcheck // 错误路径中关闭 body 失败不可操作
		logHTTPResponse(resp, body, time.Since(start))
		if strings.Contains(string(body), "APIGW.0301") {
			return nil, fmt.Errorf("%w: %s", errors.New(i18n.T(i18n.Msg.API.APIKeyExpired)), string(body))
		}
		// 尝试解析 {"error":{"code","message","target"}} 错误结构.
		if apiErr := parseErrorBody(body); apiErr != nil {
			return nil, fmt.Errorf("%w: %w", errors.New(i18n.T(i18n.Msg.API.Unauthorized)), apiErr) //nolint:errorlint // 两个 %w 均需包装
		}
		return nil, fmt.Errorf("%w: %s", errors.New(i18n.T(i18n.Msg.API.Unauthorized)), string(body))
	}
	if resp.StatusCode < http.StatusOK || resp.StatusCode >= http.StatusMultipleChoices {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 4096)) //nolint:errcheck // 错误路径中读取 body 失败不影响错误返回
		_ = resp.Body.Close()                                  //nolint:errcheck // 错误路径中关闭 body 失败不可操作
		logHTTPResponse(resp, body, time.Since(start))
		// 尝试解析 {"error":{"code","message","target"}} 错误结构.
		if apiErr := parseErrorBody(body); apiErr != nil {
			return nil, apiErr
		}
		return nil, fmt.Errorf("%w: status=%d body=%s",
			errors.New(i18n.T(i18n.Msg.API.ServerError)), resp.StatusCode, string(body))
	}
	return resp, nil
}

// parseErrorBody 从响应体中解析 {"error":{"code","message","target"}} 错误结构。
// 解析成功且 code 非空时返回 *apiError，否则返回 nil。
func parseErrorBody(body []byte) *apiError {
	var eb errorBody
	if json.Unmarshal(body, &eb) != nil {
		return nil // 解析失败，返回 nil 让调用方使用通用错误信息
	}
	if eb.Error.Code == "" {
		return nil
	}
	return &apiError{Code: eb.Error.Code, Message: eb.Error.Message}
}

func request(method, path string, body interface{}, result interface{}) error {
	client := getClient()
	url := client.BaseURL + path

	bodyBytes, err := json.Marshal(body)
	if err != nil {
		return err
	}
	if body == nil {
		bodyBytes = nil // GET/DELETE 无 body，避免发送 "null"
	}

	req, err := http.NewRequestWithContext(context.Background(), method, url, bytes.NewReader(bodyBytes))
	if err != nil {
		return err
	}
	if body != nil {
		req.Header.Set(headerContentType, headerApplicationJSON)
	}
	logHTTPRequest(req, bodyBytes)

	start := time.Now()
	resp, err := doRequest(req)
	if err != nil {
		return err
	}
	respBody, err := io.ReadAll(resp.Body)
	_ = resp.Body.Close() //nolint:errcheck // 响应已读完，关闭失败不可操作
	if err != nil {
		return err
	}
	logHTTPResponse(resp, respBody, time.Since(start))

	// 后端统一返回裸数据，直接反序列化给调用方.
	if result != nil && len(respBody) > 0 {
		if err := json.Unmarshal(respBody, result); err != nil {
			return fmt.Errorf("%w: %w", errors.New(i18n.T(i18n.Msg.API.InvalidResponse)), err) //nolint:errorlint // 两个 %w 均需包装
		}
	}
	return nil
}

func get(path string, result interface{}) error {
	return request(http.MethodGet, path, nil, result)
}

func post(path string, body, result interface{}) error {
	return request(http.MethodPost, path, body, result)
}

func put(path string, body, result interface{}) error {
	return request(http.MethodPut, path, body, result)
}

func deleteReq(path string, result interface{}) error {
	return request(http.MethodDelete, path, nil, result)
}
