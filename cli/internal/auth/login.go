package auth

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net"
	"net/http"
	"net/url"
	"time"

	"huawei.com/devbridge/internal/i18n"

	"github.com/cli/browser"
)

type Credential struct {
	APIKey string `json:"api_key" yaml:"api_key"`
}

type UserInfo struct {
	UserName string `json:"user_name" yaml:"user_name"`
	UserID   string `json:"user_id"   yaml:"user_id"`
}

type legacyCredential struct {
	APIKey string `json:"api_key"`
	Access string `json:"access"`
}

func (c *Credential) UnmarshalJSON(data []byte) error {
	type alias Credential
	if err := json.Unmarshal(data, (*alias)(c)); err == nil && c.APIKey != "" {
		return nil
	}
	var old legacyCredential
	if err := json.Unmarshal(data, &old); err != nil {
		return err
	}
	if old.APIKey != "" {
		c.APIKey = old.APIKey
	} else {
		c.APIKey = old.Access
	}
	return nil
}

type callbackResponse struct {
	APIKey   string `json:"apiKey"`
	UserName string `json:"userName"`
	UserID   string `json:"userId"`
}

// loginCallbackEnvelope 适配浏览器登录回调返回的包装格式：
//
//	{"error_code":"0000","error_msg":"","result":{"apiKey":"...","userName":"...","userId":"..."}}
//
// error_code 为 loginSuccessCode 表示成功，其他值为失败。
type loginCallbackEnvelope struct {
	ErrorCode string           `json:"error_code"`
	ErrorMsg  string           `json:"error_msg"`
	Result    callbackResponse `json:"result"`
}

// loginError 登录回调返回的错误，格式与 API 客户端 (api.apiError) 保持一致：
//
//	error code: %s, error message: %s
type loginError struct {
	Code    string
	Message string
}

func (e *loginError) Error() string {
	return fmt.Sprintf("error code: %s, error message: %s", e.Code, e.Message)
}

var (
	errMissingAPIKey = errors.New("missing api key")
	errLoginTimeout  = errors.New("login timeout")
)

const (
	loginOriginParam = "devbridge"
	loginPageURL     = "%s/space/devbridge/redirect?%s"
	envHWAPIKey      = "HW_API_KEY"

	// LoginSuccessCode 浏览器登录回调中 error_code 的成功值。
	loginSuccessCode = "0000"
)

var LoginURL = "https://devstation.ulanqab.huawei.com"

func hcBrowserLogin() (Credential, *UserInfo, error) {
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		return Credential{}, nil, err
	}
	defer func() { _ = listener.Close() }()     //nolint:errcheck // 监听器关闭失败不可操作
	port := listener.Addr().(*net.TCPAddr).Port //nolint:errcheck // TCP 监听器类型断言始终安全

	lang := "en-us"
	if i18n.DetectSystemLang() == i18n.ZH {
		lang = "zh-cn"
	}
	params := url.Values{}
	params.Set("origin", loginOriginParam)
	params.Set("language", lang)
	params.Set("callback", fmt.Sprintf("%d", port))
	redirectURL := fmt.Sprintf(loginPageURL, LoginURL, params.Encode())

	slog.Debug(i18n.T(i18n.Msg.Auth.OpenBrowser))
	if err := browser.OpenURL(redirectURL); err != nil {
		slog.Debug("open browser failed", "err", err)
		apikeyPageURL := LoginURL + "/space/devbridge/apikey"
		return Credential{}, nil, fmt.Errorf(i18n.T(i18n.Msg.Auth.NoBrowserHint), apikeyPageURL)
	}
	slog.Debug(i18n.T(i18n.Msg.Auth.BrowserOpened))

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()

	resultCh := make(chan callbackResponse, 1)
	errCh := make(chan error, 1)

	server := &http.Server{
		Handler: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			handleLoginCallback(w, r, LoginURL, resultCh, errCh)
		}),
	}
	go func() {
		if err := server.Serve(listener); err != nil &&
			!errors.Is(err, http.ErrServerClosed) && !errors.Is(err, net.ErrClosed) {
			slog.Debug("login callback server error", "err", err)
		}
	}()

	slog.Debug("waiting for browser login callback", "port", port)
	select {
	case resp := <-resultCh:
		cred := Credential{
			APIKey: resp.APIKey,
		}
		var userInfo *UserInfo
		if resp.UserName != "" || resp.UserID != "" {
			userInfo = &UserInfo{UserName: resp.UserName, UserID: resp.UserID}
		}
		return cred, userInfo, nil
	case err := <-errCh:
		return Credential{}, nil, err
	case <-ctx.Done():
		return Credential{}, nil, errLoginTimeout
	}
}

func handleLoginCallback(
	w http.ResponseWriter, r *http.Request, origin string,
	resultCh chan<- callbackResponse, errCh chan<- error,
) {
	w.Header().Set("Access-Control-Allow-Origin", origin)
	w.Header().Set("Access-Control-Allow-Methods", "POST, OPTIONS")
	w.Header().Set("Access-Control-Allow-Headers", "Content-Type")

	if r.Method == http.MethodOptions {
		w.WriteHeader(http.StatusOK)
		return
	}
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	body, err := io.ReadAll(http.MaxBytesReader(w, r.Body, 4096))
	if err != nil {
		errCh <- err
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}
	resp, err := parseLoginCallbackBody(body)
	if err != nil {
		errCh <- err
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}
	if resp.APIKey == "" {
		errCh <- errMissingAPIKey
		http.Error(w, "missing api key", http.StatusBadRequest)
		return
	}
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte("OK")) //nolint:errcheck // HTTP 回调响应写入失败不可操作
	resultCh <- resp
}

// parseLoginCallbackBody 解析浏览器登录回调的响应体。
//
// 按包装格式 {"error_code","error_msg","result"} 解析：
//   - error_code 为 "0000" 表示成功，返回 result 中的数据；
//   - error_code 为其他值表示失败，返回错误（格式与 API 客户端一致，前缀 Failed to login:）。
func parseLoginCallbackBody(body []byte) (callbackResponse, error) {
	var envelope loginCallbackEnvelope
	if err := json.Unmarshal(body, &envelope); err != nil {
		return callbackResponse{}, err
	}
	if envelope.ErrorCode != loginSuccessCode {
		return callbackResponse{}, fmt.Errorf("Failed to login: %w", &loginError{
			Code:    envelope.ErrorCode,
			Message: envelope.ErrorMsg,
		})
	}
	return envelope.Result, nil
}

func HCAuth(apiKey string) (Credential, *UserInfo, error) {
	if apiKey != "" {
		return Credential{
			APIKey: apiKey,
		}, nil, nil
	}
	return hcBrowserLogin()
}
