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

type loginCallbackEnvelope struct {
	ErrorCode string           `json:"error_code"`
	ErrorMsg  string           `json:"error_msg"`
	Result    callbackResponse `json:"result"`
}

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

	loginSuccessCode = "0000"
)

var LoginURL = "https://devstation.ulanqab.huawei.com"

func hcBrowserLogin() (Credential, *UserInfo, error) {
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		return Credential{}, nil, err
	}
	defer func() { _ = listener.Close() }()     //nolint:errcheck
	port := listener.Addr().(*net.TCPAddr).Port //nolint:errcheck

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
	_, _ = w.Write([]byte("OK")) //nolint:errcheck
	resultCh <- resp
}

func parseLoginCallbackBody(body []byte) (callbackResponse, error) {
	var envelope loginCallbackEnvelope
	if err := json.Unmarshal(body, &envelope); err != nil {
		return callbackResponse{}, err
	}
	if envelope.ErrorCode != loginSuccessCode {
		apikeyPageURL := LoginURL + "/space/devbridge/apikey"
		return callbackResponse{}, fmt.Errorf("Failed to login: %w\n%s", &loginError{
			Code:    envelope.ErrorCode,
			Message: envelope.ErrorMsg,
		}, fmt.Sprintf(i18n.T(i18n.Msg.Auth.LoginErrorHint), apikeyPageURL))
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
