/*
 * Copyright (c) Huawei Technologies Co., Ltd. 2026-2027. All rights reserved.
 */

package sdk

import (
	"errors"
	"testing"
	"time"

	"github.com/microsoft/dev-tunnels-ssh/src/go/ssh"
	"github.com/microsoft/dev-tunnels-ssh/src/go/ssh/messages"
)

// TestAuthFailedFromClose covers the close-reason classification shared by the
// host and connect session loops: expired/invalid credentials are terminal,
// everything else stays a recoverable drop.
func TestAuthFailedFromClose(t *testing.T) {
	tests := []struct {
		name    string
		reason  string
		message string
		want    bool
	}{
		{name: "expired token message", reason: "2", message: "invalid token: expired", want: true},
		{name: "api key expired message", reason: "2", message: "API key expired", want: true},
		{name: "expired reason only", reason: "token expired", want: true},
		{name: "unauthorized message", reason: "2", message: "Unauthorized", want: true},
		{name: "network drop stays retriable", reason: "2", message: "connection reset by peer", want: false},
		{name: "quota close stays retriable", reason: "1008", message: "too many concurrent connections", want: false},
		{name: "empty close event", reason: "", message: "", want: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := authFailedFromClose(tt.reason, tt.message)
			if got := errors.Is(err, ErrAuthFailed); got != tt.want {
				t.Fatalf("authFailedFromClose(%q, %q) = %v, want auth failure = %v", tt.reason, tt.message, err, tt.want)
			}
		})
	}
}

// TestCloseReasonCaptureHandlerBeforeWait verifies the normal order where the
// SSH OnClosed callback already fired before the loop selects on Done().
func TestCloseReasonCaptureHandlerBeforeWait(t *testing.T) {
	c := newCloseReasonCapture()
	c.handler(&ssh.SessionClosedEventArgs{
		Reason:  messages.DisconnectByApplication,
		Message: "invalid token",
	})

	err := c.authError()
	if !errors.Is(err, ErrAuthFailed) {
		t.Fatalf("authError() = %v, want ErrAuthFailed", err)
	}
	var ae *AuthFailedError
	if !errors.As(err, &ae) || ae.Message != "invalid token" {
		t.Fatalf("authError() detail = %v, want message %q", err, "invalid token")
	}
}

// TestCloseReasonCaptureWaitsForLateHandler covers the real ordering: ssh closes
// Session.Done() just before firing OnClosed, so the loop may arrive first and
// must still classify the close reason.
func TestCloseReasonCaptureWaitsForLateHandler(t *testing.T) {
	c := newCloseReasonCapture()
	go func() {
		time.Sleep(50 * time.Millisecond)
		c.handler(&ssh.SessionClosedEventArgs{Reason: messages.DisconnectByApplication, Message: "token expired"})
	}()

	if err := c.authError(); !errors.Is(err, ErrAuthFailed) {
		t.Fatalf("authError() = %v, want ErrAuthFailed after late OnClosed", err)
	}
}

// TestCloseReasonCaptureWithoutCloseEvent verifies a missing close event yields
// no auth error, so a plain network drop keeps the existing retry behavior.
func TestCloseReasonCaptureWithoutCloseEvent(t *testing.T) {
	c := newCloseReasonCapture()
	if err := c.authError(); err != nil {
		t.Fatalf("authError() = %v, want nil when no close event was captured", err)
	}
}
