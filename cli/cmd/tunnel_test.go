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
