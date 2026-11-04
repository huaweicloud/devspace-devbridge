/*
 * Copyright (c) Huawei Technologies Co., Ltd. 2026-2027. All rights reserved.
 */

package sdk

import (
	"errors"
	"net/http"
	"testing"

	"github.com/coder/websocket"
)

func TestParseSSHCloseError(t *testing.T) {
	tests := []struct {
		name string
		err  error
		want error
	}{
		{
			name: "quota exceeded",
			err:  websocket.CloseError{Code: closeCodeQuotaExceeded, Reason: "quota exceeded for account 1"},
			want: ErrQuotaExceeded,
		},
		{
			name: "tunnel not found",
			err:  websocket.CloseError{Code: closeCodeTunnelNotFound, Reason: "tunnel abc not found"},
			want: ErrTunnelNotFound,
		},
		{
			name: "duplicate host",
			err:  websocket.CloseError{Code: closeCodeDuplicateHost, Reason: "tunnel abc already registered"},
			want: ErrDuplicateHost,
		},
		{
			name: "unknown close code passes through",
			err:  websocket.CloseError{Code: websocket.StatusInternalError, Reason: "connect session failed"},
			want: websocket.CloseError{Code: websocket.StatusInternalError, Reason: "connect session failed"},
		},
		{
			name: "policy violation mapping for concurrent connections over quota",
			err:  websocket.CloseError{Code: websocket.StatusPolicyViolation, Reason: "too many concurrent connections"},
			want: ErrQuotaExceeded,
		},
		{
			name: "non close error passes through",
			err:  errors.New("some other error"),
			want: errors.New("some other error"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := parseSSHCloseError(tt.err)

			if errors.Is(tt.want, ErrQuotaExceeded) ||
				errors.Is(tt.want, ErrTunnelNotFound) ||
				errors.Is(tt.want, ErrDuplicateHost) {
				if !errors.Is(got, tt.want) {
					t.Fatalf("parseSSHCloseError() = %v, want %v", got, tt.want)
				}
				return
			}

			var gotCE, wantCE websocket.CloseError
			gotOK := errors.As(got, &gotCE)
			wantOK := errors.As(tt.want, &wantCE)
			if !gotOK && !wantOK {
				if got.Error() != tt.want.Error() {
					t.Fatalf("parseSSHCloseError() = %q, want %q", got, tt.want)
				}
				return
			}
			if gotOK != wantOK || gotCE != wantCE {
				t.Fatalf("parseSSHCloseError() = %v (AsCloseErr=%v), want %v (AsCloseErr=%v)", got, gotOK, tt.want, wantOK)
			}
		})
	}
}

func TestBuildWSHeader(t *testing.T) {
	hdr, sub := buildWSHeader("jwt-token", "api-key", http.Header{
		"User-Agent": []string{"devbridge-cli/test (Linux/amd64; stable)"},
	})

	if got := hdr.Get("X-API-Key"); got != "api-key" {
		t.Fatalf("X-API-Key = %q, want %q", got, "api-key")
	}
	if got := hdr.Get("User-Agent"); got != "devbridge-cli/test (Linux/amd64; stable)" {
		t.Fatalf("User-Agent = %q, want version header injected", got)
	}
	if len(sub) != 2 || sub[0] != subprotocolDevBridge || sub[1] != "jwt-token" {
		t.Fatalf("subprotocols = %v, want [%q jwt-token]", sub, subprotocolDevBridge)
	}

	// A nil extra map adds nothing and keeps the header minimal.
	hdr2, _ := buildWSHeader("", "api-key", nil)
	if _, ok := hdr2["User-Agent"]; ok {
		t.Fatalf("nil extra headers must not add User-Agent, got %v", hdr2)
	}
	if hdr2.Get("X-API-Key") != "api-key" {
		t.Fatalf("X-API-Key = %q, want %q", hdr2.Get("X-API-Key"), "api-key")
	}
}
