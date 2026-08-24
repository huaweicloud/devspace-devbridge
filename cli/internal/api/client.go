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
	successCode        = "0000"
	TunnelNotFoundCode = "10002"
)

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

type apiResponse struct {
	Result    json.RawMessage `json:"result"`
	ErrorCode string          `json:"error_code"`
	ErrorMsg  string          `json:"error_msg"`
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

var restClient *restClientType

func InitClient(baseURL string) {
	if baseURL == "" {
		// /open-api-inner/v1/relay-controller/tunnels
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
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 4096))
		resp.Body.Close()
		logHTTPResponse(resp, body, time.Since(start))
		if strings.Contains(string(body), "APIGW.0301") {
			return nil, fmt.Errorf("%w: %s", errors.New(i18n.T(i18n.Msg.API.APIKeyExpired)), string(body))
		}
		// 尝试解析 {"error":{"code","message","target"}} 错误结构
		if apiErr := parseErrorBody(body); apiErr != nil {
			return nil, apiErr
		}
		return nil, fmt.Errorf("%w: %s", errors.New(i18n.T(i18n.Msg.API.Unauthorized)), string(body))
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 4096))
		resp.Body.Close()
		logHTTPResponse(resp, body, time.Since(start))
		// 尝试解析 {"error":{"code","message","target"}} 错误结构
		if apiErr := parseErrorBody(body); apiErr != nil {
			return nil, apiErr
		}
		return nil, fmt.Errorf("%w: status=%d body=%s", errors.New(i18n.T(i18n.Msg.API.ServerError)), resp.StatusCode, string(body))
	}
	return resp, nil
}

// parseErrorBody 从响应体中解析 {"error":{"code","message","target"}} 错误结构。
// 解析成功且 code 非空时返回 *apiError，否则返回 nil。
func parseErrorBody(body []byte) *apiError {
	var eb errorBody
	if err := json.Unmarshal(body, &eb); err != nil {
		return nil
	}
	if eb.Error.Code == "" {
		return nil
	}
	return &apiError{Code: eb.Error.Code, Message: eb.Error.Message}
}

func request(method, path string, body interface{}, result interface{}) error {
	client := getClient()
	url := client.BaseURL + path

	var bodyBytes []byte
	if body != nil {
		var err error
		bodyBytes, err = json.Marshal(body)
		if err != nil {
			return err
		}
	}

	req, err := http.NewRequest(method, url, bytes.NewReader(bodyBytes))
	if err != nil {
		return err
	}
	if body != nil {
		req.Header.Set(headerContentType, headerApplicationJson)
	}
	logHTTPRequest(req, bodyBytes)

	start := time.Now()
	resp, err := doRequest(req)
	if err != nil {
		return err
	}
	respBody, err := io.ReadAll(resp.Body)
	resp.Body.Close()
	if err != nil {
		return err
	}
	logHTTPResponse(resp, respBody, time.Since(start))

	// 兼容两种响应格式：
	// 1. 封装体：{"result": ..., "error_code": "0000", "error_msg": ""}
	// 2. 裸数据：直接返回 result（数组或对象），无 error_code/error_msg 外层封装
	if isWrappedResponse(respBody) {
		var apiResp apiResponse
		if err := json.Unmarshal(respBody, &apiResp); err != nil {
			return fmt.Errorf("%w: %v", errors.New(i18n.T(i18n.Msg.API.InvalidResponse)), err)
		}
		if apiResp.ErrorCode != "" && apiResp.ErrorCode != successCode {
			return &apiError{Code: apiResp.ErrorCode, Message: apiResp.ErrorMsg}
		}
		if result != nil && len(apiResp.Result) > 0 {
			if err := json.Unmarshal(apiResp.Result, result); err != nil {
				return err
			}
		}
		return nil
	}
	// 裸响应体：直接反序列化给调用方
	if result != nil && len(respBody) > 0 {
		if err := json.Unmarshal(respBody, result); err != nil {
			return fmt.Errorf("%w: %v", errors.New(i18n.T(i18n.Msg.API.InvalidResponse)), err)
		}
	}
	return nil
}

// isWrappedResponse 判断响应体是否为 {result, error_code, error_msg} 封装格式。
// 通过检测 JSON 顶层对象是否包含 "error_code" 或 "result" 字段来区分封装体与裸数据。
func isWrappedResponse(body []byte) bool {
	var probe struct {
		ErrorCode *string          `json:"error_code"`
		Result    *json.RawMessage `json:"result"`
	}
	if err := json.Unmarshal(body, &probe); err != nil {
		return false // 非 JSON 对象（如数组）→ 裸数据
	}
	return probe.ErrorCode != nil || probe.Result != nil
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
