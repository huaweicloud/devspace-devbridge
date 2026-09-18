/*
 * Copyright (c) Huawei Technologies Co., Ltd. 2026-2027. All rights reserved.
 */

package sdk

import (
	"errors"
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
			name: "policy violation no longer mapped to duplicate host",
			err:  websocket.CloseError{Code: websocket.StatusPolicyViolation, Reason: "too many concurrent connections"},
			want: websocket.CloseError{Code: websocket.StatusPolicyViolation, Reason: "too many concurrent connections"},
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

			// Sentinel error cases: compare by identity.
			if tt.want == ErrQuotaExceeded || tt.want == ErrTunnelNotFound || tt.want == ErrDuplicateHost {
				if !errors.Is(got, tt.want) {
					t.Fatalf("parseSSHCloseError() = %v, want %v", got, tt.want)
				}
				return
			}

			// Pass-through cases: compare by concrete value.
			var gotCE, wantCE websocket.CloseError
			gotOK := errors.As(got, &gotCE)
			wantOK := errors.As(tt.want, &wantCE)
			if gotOK != wantOK || (gotOK && gotCE != wantCE) {
				if !gotOK && !wantOK {
					if got.Error() != tt.want.Error() {
						t.Fatalf("parseSSHCloseError() = %q, want %q", got, tt.want)
					}
					return
				}
				t.Fatalf("parseSSHCloseError() = %v (AsCloseErr=%v), want %v (AsCloseErr=%v)", got, gotOK, tt.want, wantOK)
			}
		})
	}
}
