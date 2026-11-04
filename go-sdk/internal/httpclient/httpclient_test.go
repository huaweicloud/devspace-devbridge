package httpclient

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestDo_NoServedByOnError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusForbidden)
		_, _ = w.Write([]byte("blocked"))
	}))
	defer srv.Close()

	c := New("key", srv.URL, nil, nil)
	err := c.Do(context.Background(), http.MethodGet, "/x", nil, nil)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !strings.Contains(err.Error(), "tunnels") && !strings.Contains(err.Error(), "did not reach") && !strings.Contains(err.Error(), "未到达") {
		t.Fatalf("unexpected error text: %v", err)
	}
}

func TestDo_ServedByOnError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("X-Served-By", "tunnels-cn-north-4-bridge")
		w.Header().Set("X-Request-Id", "deadbeefdeadbeefdeadbeefdeadbeef")
		w.WriteHeader(http.StatusBadGateway)
		_, _ = w.Write([]byte("boom"))
	}))
	defer srv.Close()

	c := New("key", srv.URL, nil, nil)
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

func TestDo_ServiceOriginOnError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("X-Served-By", "tunnels-cn-north-4-bridge")
		w.Header().Set("X-DevBridge-Error-Origin", "service")
		w.Header().Set("X-Request-Id", "deadbeefdeadbeefdeadbeefdeadbeef")
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = w.Write([]byte("boom"))
	}))
	defer srv.Close()

	c := New("key", srv.URL, nil, nil)
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

func TestDo_BusinessErrorWins(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		_, _ = w.Write([]byte(`{"error":{"code":"10002","message":"tunnel not found"}}`))
	}))
	defer srv.Close()

	c := New("key", srv.URL, nil, nil)
	err := c.Do(context.Background(), http.MethodGet, "/x", nil, nil)
	apiErr, ok := err.(*APIError)
	if !ok {
		t.Fatalf("expected *APIError, got %T (%v)", err, err)
	}
	if apiErr.Code != "10002" {
		t.Fatalf("unexpected code %q", apiErr.Code)
	}
}
