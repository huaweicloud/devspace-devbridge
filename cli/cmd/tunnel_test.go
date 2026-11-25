package cmd

import (
	"testing"
	"time"

	devbridge "github.com/huaweicloud/devspace-devbridge/go-sdk"
)

func TestTunnelStatusWarning(t *testing.T) {
	now := time.Unix(1789516860, 0)
	cases := []struct {
		name    string
		status  *devbridge.TunnelStatus
		wantMsg bool
	}{
		{"status nil", nil, true},
		{"reportedAt missing", &devbridge.TunnelStatus{HostConnectionCount: 1}, true},
		{"fresh status", &devbridge.TunnelStatus{ReportedAt: now.Unix() - 10}, false},
		{"boundary exactly 120s", &devbridge.TunnelStatus{ReportedAt: now.Unix() - 120}, false},
		{"stale beyond 120s", &devbridge.TunnelStatus{ReportedAt: now.Unix() - 121}, true},
		{"future timestamp", &devbridge.TunnelStatus{ReportedAt: now.Unix() + 30}, false},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := tunnelStatusWarning(tc.status, now)
			if (got != "") != tc.wantMsg {
				t.Fatalf("tunnelStatusWarning() = %q, want warning = %v", got, tc.wantMsg)
			}
		})
	}
}

func TestTunnelHostCountHint(t *testing.T) {
	now := time.Unix(1789516860, 0)
	cases := []struct {
		name       string
		status     *devbridge.TunnelStatus
		expectHost bool
		wantHint   bool
	}{
		{
			name:       "fresh zero with expect-host",
			status:     &devbridge.TunnelStatus{ReportedAt: now.Unix() - 10, HostConnectionCount: 0},
			expectHost: true,
			wantHint:   true,
		},
		{
			name:       "fresh zero without expect-host",
			status:     &devbridge.TunnelStatus{ReportedAt: now.Unix() - 10, HostConnectionCount: 0},
			expectHost: false,
			wantHint:   false,
		},
		{
			name:       "fresh non-zero with expect-host",
			status:     &devbridge.TunnelStatus{ReportedAt: now.Unix() - 10, HostConnectionCount: 2},
			expectHost: true,
			wantHint:   false,
		},
		{
			name:       "boundary exactly 120s is fresh",
			status:     &devbridge.TunnelStatus{ReportedAt: now.Unix() - 120, HostConnectionCount: 0},
			expectHost: true,
			wantHint:   true,
		},
		{
			name:       "stale zero with expect-host",
			status:     &devbridge.TunnelStatus{ReportedAt: now.Unix() - 121, HostConnectionCount: 0},
			expectHost: true,
			wantHint:   false,
		},
		{
			name:       "reportedAt missing zero with expect-host",
			status:     &devbridge.TunnelStatus{HostConnectionCount: 0},
			expectHost: true,
			wantHint:   false,
		},
		{
			name:       "status nil with expect-host",
			status:     nil,
			expectHost: true,
			wantHint:   false,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := tunnelHostCountHint(tc.status, tc.expectHost, now)
			if (got != "") != tc.wantHint {
				t.Fatalf("tunnelHostCountHint() = %q, want hint = %v", got, tc.wantHint)
			}
			if got != "" && tunnelStatusWarning(tc.status, now) != "" {
				t.Fatalf("hint must not be emitted together with a status warning")
			}
		})
	}
}
