/*
 * Copyright (c) Huawei Technologies Co., Ltd. 2026-2027. All rights reserved.
 */

package sdk

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestValidateTunnelDescription(t *testing.T) {
	cases := []struct {
		name    string
		desc    string
		wantErr bool
	}{
		{"empty description allowed", "", false},
		{"letters", "devbridge", false},
		{"digits", "20260911", false},
		{"letters and digits mixed", "env01", false},
		{"64-char boundary", strings.Repeat("a", 64), false},
		{"65-char over limit", strings.Repeat("a", 65), true},
		{"space not allowed", "dev bridge", true},
		{"hyphen not allowed", "dev-bridge", true},
		{"underscore not allowed", "dev_bridge", true},
		{"punctuation not allowed", "dev!", true},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			err := validateTunnelDescription(tc.desc)
			if (err != nil) != tc.wantErr {
				t.Fatalf("validateTunnelDescription(%q) err = %v, wantErr = %v", tc.desc, err, tc.wantErr)
			}
		})
	}
}

func TestValidateTunnelDescriptionSentinel(t *testing.T) {
	err := validateTunnelDescription("bad desc!")
	if !errors.Is(err, ErrInvalidTunnelDescription) {
		t.Fatalf("expected ErrInvalidTunnelDescription, got: %v", err)
	}
}

func TestValidateTunnelName(t *testing.T) {
	cases := []struct {
		name    string
		in      string
		wantErr bool
	}{
		{"letters", "devbridge", false},
		{"digits", "20260911", false},
		{"letters and digits mixed", "env01", false},
		{"hyphen in middle allowed", "dev-bridge", false},
		{"1-char lower bound", "a", false},
		{"64-char upper bound", strings.Repeat("a", 64), false},
		{"65-char over limit", strings.Repeat("a", 65), true},
		{"empty name", "", true},
		{"space not allowed", "dev bridge", true},
		{"underscore not allowed", "dev_bridge", true},
		{"leading hyphen not allowed", "-devbridge", true},
		{"trailing hyphen not allowed", "devbridge-", true},
		{"punctuation not allowed", "dev!", true},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			err := validateTunnelName(tc.in)
			if (err != nil) != tc.wantErr {
				t.Fatalf("validateTunnelName(%q) err = %v, wantErr = %v", tc.in, err, tc.wantErr)
			}
		})
	}
}

func TestValidateTunnelNameSentinel(t *testing.T) {
	err := validateTunnelName("bad name!")
	if !errors.Is(err, ErrInvalidTunnelName) {
		t.Fatalf("expected ErrInvalidTunnelName, got: %v", err)
	}
}

func newTunnelTestServer(t *testing.T, body string) *Devbridge {
	t.Helper()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(body))
	}))
	t.Cleanup(srv.Close)
	return New(Config{APIBaseURL: srv.URL, HTTPClient: srv.Client()})
}

func TestShowTunnelStatus(t *testing.T) {
	cases := []struct {
		name       string
		body       string
		wantStatus bool
		wantHost   int
		wantReport int64
	}{
		{
			name:       "status present with reportedAt",
			body:       `{"tunnelId":"aaaadysa","name":"demo","status":{"hostConnectionCount":1,"clientConnectionCount":2,"totalUploadBytes":1024,"totalDownloadBytes":2048,"reportedAt":1789516860}}`,
			wantStatus: true,
			wantHost:   1,
			wantReport: 1789516860,
		},
		{
			name:       "status null",
			body:       `{"tunnelId":"aaaadysa","name":"demo","status":null}`,
			wantStatus: false,
		},
		{
			name:       "reportedAt missing",
			body:       `{"tunnelId":"aaaadysa","name":"demo","status":{"hostConnectionCount":1,"clientConnectionCount":0,"totalUploadBytes":0,"totalDownloadBytes":0}}`,
			wantStatus: true,
			wantHost:   1,
			wantReport: 0,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			d := newTunnelTestServer(t, tc.body)
			detail, err := d.ShowTunnel(context.Background(), "aaaadysa")
			if err != nil {
				t.Fatalf("ShowTunnel() error = %v", err)
			}
			if (detail.Status != nil) != tc.wantStatus {
				t.Fatalf("status presence = %v, want %v", detail.Status != nil, tc.wantStatus)
			}
			if !tc.wantStatus {
				return
			}
			if detail.Status.HostConnectionCount != tc.wantHost {
				t.Fatalf("hostConnectionCount = %d, want %d", detail.Status.HostConnectionCount, tc.wantHost)
			}
			if detail.Status.ReportedAt != tc.wantReport {
				t.Fatalf("reportedAt = %d, want %d", detail.Status.ReportedAt, tc.wantReport)
			}
		})
	}
}
