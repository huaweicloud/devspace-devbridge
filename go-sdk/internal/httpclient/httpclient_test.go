package httpclient

import (
	"context"
	"net/http"
	"net/http/httptest"
	"runtime"
	"strings"
	"testing"
)

// TestDo_NoServedByOnError 验证：非 2xx 且无 X-Served-By 时，报「请求未到达
// 隧道服务」，而不是泛泛的 server error。
func TestDo_NoServedByOnError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusForbidden)
		_, _ = w.Write([]byte("blocked"))
	}))
	defer srv.Close()

	c := New("key", srv.URL, nil)
	err := c.Do(context.Background(), http.MethodGet, "/x", nil, nil)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !strings.Contains(err.Error(), "tunnels") && !strings.Contains(err.Error(), "did not reach") && !strings.Contains(err.Error(), "未到达") {
		t.Fatalf("unexpected error text: %v", err)
	}
}

// TestDo_ServedByOnError 验证：有 X-Served-By 时，错误信息附带来源与请求 ID。
func TestDo_ServedByOnError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("X-Served-By", "tunnels-cn-north-4-bridge")
		w.Header().Set("X-Request-Id", "deadbeefdeadbeefdeadbeefdeadbeef")
		w.WriteHeader(http.StatusBadGateway)
		_, _ = w.Write([]byte("boom"))
	}))
	defer srv.Close()

	c := New("key", srv.URL, nil)
	err := c.Do(context.Background(), http.MethodGet, "/x", nil, nil)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !strings.Contains(err.Error(), "tunnels-cn-north-4-bridge") {
		t.Fatalf("error lacks X-Served-By value: %v", err)
	}
	if !strings.Contains(err.Error(), "deadbeefdeadbeefdeadbeefdeadbeef") {
		t.Fatalf("error lacks X-Request-Id value: %v", err)
	}
}

// TestDo_ServiceOriginOnError 验证：origin=service 的 4xx/5xx 报「隧道内服务
// 错误」而非「服务端错误」，且仍附带来源与请求 ID。
func TestDo_ServiceOriginOnError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("X-Served-By", "tunnels-cn-north-4-bridge")
		w.Header().Set("X-DevBridge-Error-Origin", "service")
		w.Header().Set("X-Request-Id", "deadbeefdeadbeefdeadbeefdeadbeef")
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = w.Write([]byte("boom"))
	}))
	defer srv.Close()

	c := New("key", srv.URL, nil)
	err := c.Do(context.Background(), http.MethodGet, "/x", nil, nil)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !strings.Contains(err.Error(), "tunnels-cn-north-4-bridge") {
		t.Fatalf("error lacks X-Served-By value: %v", err)
	}
	if !strings.Contains(err.Error(), "deadbeefdeadbeefdeadbeefdeadbeef") {
		t.Fatalf("error lacks X-Request-Id value: %v", err)
	}
}

// TestDo_BusinessErrorWins 验证：业务错误（{error:{code,message}}）仍优先判为
// APIError，不被响应头逻辑覆盖。
func TestDo_BusinessErrorWins(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		_, _ = w.Write([]byte(`{"error":{"code":"10002","message":"tunnel not found"}}`))
	}))
	defer srv.Close()

	c := New("key", srv.URL, nil)
	err := c.Do(context.Background(), http.MethodGet, "/x", nil, nil)
	apiErr, ok := err.(*APIError)
	if !ok {
		t.Fatalf("expected *APIError, got %T (%v)", err, err)
	}
	if apiErr.Code != "10002" {
		t.Fatalf("unexpected code %q", apiErr.Code)
	}
}

// TestDo_40400 验证：40400（老接口 /open-api-inner/v1/relay-controller/* 废弃）
// 的 APIError 附带可执行升级引导（GitCode/GitHub 双源安装命令），且仍以
// *APIError 返回。
func TestDo_40400(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		_, _ = w.Write([]byte(`{"error":{"code":"40400","message":"route not found; upgrade to the latest stable version"}}`))
	}))
	defer srv.Close()

	c := New("key", srv.URL, nil)
	err := c.Do(context.Background(), http.MethodGet, "/open-api-inner/v1/relay-controller/tunnels", nil, nil)
	apiErr, ok := err.(*APIError)
	if !ok {
		t.Fatalf("expected *APIError, got %T (%v)", err, err)
	}
	if apiErr.Code != "40400" {
		t.Fatalf("unexpected code %q", apiErr.Code)
	}

	msg := err.Error()
	if !strings.Contains(msg, "error code: 40400") {
		t.Fatalf("error lacks original code/message: %v", msg)
	}
	if runtime.GOOS == "windows" {
		if !strings.Contains(msg, "irm https://gitcode.com/CloudDeveloperDepartment/devbrige/releases/download/latest/install.ps1 | iex") {
			t.Fatalf("error lacks GitCode irm install command: %v", msg)
		}
		if !strings.Contains(msg, "irm https://github.com/huaweicloud/devspace-devbridge/releases/latest/download/install.ps1 | iex") {
			t.Fatalf("error lacks GitHub irm install command: %v", msg)
		}
	} else {
		if !strings.Contains(msg, "curl -fsSL https://gitcode.com/CloudDeveloperDepartment/devbrige/releases/download/latest/install.sh | bash") {
			t.Fatalf("error lacks GitCode curl install command: %v", msg)
		}
		if !strings.Contains(msg, "curl -fsSL https://github.com/huaweicloud/devspace-devbridge/releases/latest/download/install.sh | bash") {
			t.Fatalf("error lacks GitHub curl install command: %v", msg)
		}
	}
}

// TestDo_Non40400Unchanged 验证：非 40400 业务错误的输出不含升级引导。
func TestDo_Non40400Unchanged(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		_, _ = w.Write([]byte(`{"error":{"code":"10002","message":"tunnel not found"}}`))
	}))
	defer srv.Close()

	c := New("key", srv.URL, nil)
	err := c.Do(context.Background(), http.MethodGet, "/x", nil, nil)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if strings.Contains(err.Error(), "install.sh") || strings.Contains(err.Error(), "install.ps1") {
		t.Fatalf("non-40400 error must not contain upgrade guide: %v", err)
	}
	if err.Error() != "error code: 10002, error message: tunnel not found" {
		t.Fatalf("non-40400 error output changed: %q", err.Error())
	}
}
