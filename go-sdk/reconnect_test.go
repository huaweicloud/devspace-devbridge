/*
 * Copyright (c) Huawei Technologies Co., Ltd. 2026-2027. All rights reserved.
 */

package sdk

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"testing"
	"time"
)

func TestReconnectDelay(t *testing.T) {
	tests := []struct {
		name  string
		count int
		want  time.Duration
	}{
		{"zero-never-zero", 0, 3 * time.Second},
		{"negative-never-zero", -3, 3 * time.Second},
		{"first-failure", 1, 3 * time.Second},
		{"second-failure", 2, 6 * time.Second},
		{"third-failure", 3, 12 * time.Second},
		{"fourth-failure", 4, 24 * time.Second},
		{"fifth-capped", 5, 30 * time.Second},
		{"capped", 100, 30 * time.Second},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := reconnectDelay(tt.count); got != tt.want {
				t.Fatalf("reconnectDelay(%d) = %v, want %v", tt.count, got, tt.want)
			}
		})
	}
}

func TestReconnectLoopStopsOnAuthFailed(t *testing.T) {
	d := &Devbridge{logger: slog.Default(), outputWriter: io.Discard}
	attempts := 0

	err := d.reconnectLoop(context.Background(), func() (bool, error) {
		attempts++
		return false, &AuthFailedError{Message: "API key expired"}
	}, reconnectDecision{
		shouldStop: func(err error) bool { return errors.Is(err, ErrAuthFailed) },
	})

	if !errors.Is(err, ErrAuthFailed) {
		t.Fatalf("reconnectLoop() err = %v, want ErrAuthFailed", err)
	}
	if attempts != 1 {
		t.Fatalf("session attempts = %d, want 1 (auth failure must not retry)", attempts)
	}
}

func TestReconnectLoopConnectedAuthFailedStopsWithoutReset(t *testing.T) {
	d := &Devbridge{logger: slog.Default(), outputWriter: io.Discard}
	attempts := 0

	err := d.reconnectLoop(context.Background(), func() (bool, error) {
		attempts++
		return true, &AuthFailedError{Message: "invalid token"}
	}, reconnectDecision{})

	if !errors.Is(err, ErrAuthFailed) {
		t.Fatalf("reconnectLoop() err = %v, want ErrAuthFailed", err)
	}
	if attempts != 1 {
		t.Fatalf("session attempts = %d, want 1 (connected auth close must abort)", attempts)
	}
}
