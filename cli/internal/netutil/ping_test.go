package netutil

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestPingURI_NoServedByOnError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusForbidden)
	}))
	defer srv.Close()

	r := PingURI(context.Background(), srv.URL, 5*time.Second)
	if r.Err == nil {
		t.Fatal("expected Err on non-OK without X-Served-By, got nil")
	}
	if r.StatusText == "403 Forbidden" {
		t.Fatalf("should classify as not-served-by-tunnel, got raw status %q", r.StatusText)
	}
}

func TestPingURI_ServedByKeepsStatus(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("X-Served-By", "tunnels-cn-north-4-bridge")
		w.Header().Set("X-DevBridge-Error-Origin", "gateway")
		w.WriteHeader(http.StatusBadGateway)
	}))
	defer srv.Close()

	r := PingURI(context.Background(), srv.URL, 5*time.Second)
	if r.Err == nil {
		t.Fatal("expected Err for gateway-origin 4xx, got nil")
	}
	if r.StatusText == "502 Bad Gateway" {
		t.Fatalf("should classify as gateway error, got raw status %q", r.StatusText)
	}
}

func TestPingURI_ServiceOriginError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("X-Served-By", "tunnels-cn-north-4-bridge")
		w.Header().Set("X-DevBridge-Error-Origin", "service")
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer srv.Close()

	r := PingURI(context.Background(), srv.URL, 5*time.Second)
	if r.Err != nil {
		t.Fatalf("expected nil Err for service-origin 5xx (network is fine), got %v", r.Err)
	}
	if r.StatusText == "500 Internal Server Error" {
		t.Fatalf("should classify as service error, got raw status %q", r.StatusText)
	}
}

func TestPingURI_DisconnectedOnTransportError(t *testing.T) {
	r := PingURI(context.Background(), "http://127.0.0.1:1/", 2*time.Second)
	if r.Err == nil {
		t.Fatal("expected transport Err, got nil")
	}
	if r.StatusText == "Bad Gateway" || r.StatusText == "Gateway Timeout" {
		t.Fatalf("must not fake %q for transport error", r.StatusText)
	}
}
