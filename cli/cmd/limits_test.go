package cmd

import (
	"encoding/json"
	"strings"
	"testing"

	devbridge "github.com/huaweicloud/devspace-devbridge/go-sdk"
	"huawei.com/devbridge/internal/i18n"
)

// limitsKV decodes a raw /v1/limits response and renders the detail rows.
func limitsKV(t *testing.T, payload string) [][2]string {
	t.Helper()
	var limits devbridge.Limits
	if err := json.Unmarshal([]byte(payload), &limits); err != nil {
		t.Fatalf("decode limits: %v", err)
	}
	return limitsDetailKV(&limits)
}

// limitValues returns the rendered values of the three optional limit rows.
func limitValues(rows [][2]string) (bandwidth, httpPerMinute, connections string) {
	return rows[7][1], rows[8][1], rows[9][1]
}

func TestLimitsDetailMissingFields(t *testing.T) {
	rows := limitsKV(t, `{
		"resetAt": 1790784000,
		"quotaBytes": 53687091200,
		"remainingBytes": 53683945472,
		"activeTunnels": 1,
		"maxTunnels": 10,
		"maxPortsPerTunnel": 10,
		"maxHostsPerTunnel": 1
	}`)

	bandwidth, httpPerMinute, connections := limitValues(rows)
	notProvided := i18n.T(i18n.Msg.Limits.NotProvided)
	for name, got := range map[string]string{
		"maxTunnelBandwidthBytesPerSecond": bandwidth,
		"maxHttpRequestsPerMinutePerPort":  httpPerMinute,
		"maxConnectionsPerPort":            connections,
	} {
		if got != notProvided {
			t.Errorf("%s = %q, want %q", name, got, notProvided)
		}
	}
	if strings.HasPrefix(bandwidth, "0") || httpPerMinute == "0" || connections == "0" {
		t.Errorf("missing limits must not render as zero: %q / %q / %q", bandwidth, httpPerMinute, connections)
	}
}

func TestLimitsDetailExplicitZero(t *testing.T) {
	rows := limitsKV(t, `{
		"resetAt": 1790784000,
		"quotaBytes": 53687091200,
		"remainingBytes": 53683945472,
		"activeTunnels": 1,
		"maxTunnels": 10,
		"maxPortsPerTunnel": 10,
		"maxHostsPerTunnel": 1,
		"maxTunnelBandwidthBytesPerSecond": 0,
		"maxHttpRequestsPerMinutePerPort": 0,
		"maxConnectionsPerPort": 0
	}`)

	bandwidth, httpPerMinute, connections := limitValues(rows)
	if bandwidth != "0 B/s" {
		t.Errorf("bandwidth = %q, want %q", bandwidth, "0 B/s")
	}
	if httpPerMinute != "0" || connections != "0" {
		t.Errorf("per-port limits = %q / %q, want %q / %q", httpPerMinute, connections, "0", "0")
	}
}

func TestLimitsDetailPositiveValues(t *testing.T) {
	rows := limitsKV(t, `{
		"resetAt": 1790784000,
		"quotaBytes": 53687091200,
		"remainingBytes": 53683945472,
		"activeTunnels": 1,
		"maxTunnels": 10,
		"maxPortsPerTunnel": 10,
		"maxHostsPerTunnel": 1,
		"maxTunnelBandwidthBytesPerSecond": 5242880,
		"maxHttpRequestsPerMinutePerPort": 500,
		"maxConnectionsPerPort": 100
	}`)

	bandwidth, httpPerMinute, connections := limitValues(rows)
	if bandwidth != "5 MB/s" {
		t.Errorf("bandwidth = %q, want %q", bandwidth, "5 MB/s")
	}
	if httpPerMinute != "500" {
		t.Errorf("httpPerMinute = %q, want %q", httpPerMinute, "500")
	}
	if connections != "100" {
		t.Errorf("connections = %q, want %q", connections, "100")
	}
}

func TestLimitsDetailUnlimited(t *testing.T) {
	rows := limitsKV(t, `{
		"resetAt": 1790784000,
		"quotaBytes": 53687091200,
		"remainingBytes": 53683945472,
		"activeTunnels": 1,
		"maxTunnels": 10,
		"maxPortsPerTunnel": 10,
		"maxHostsPerTunnel": 1,
		"maxTunnelBandwidthBytesPerSecond": -1,
		"maxHttpRequestsPerMinutePerPort": -1,
		"maxConnectionsPerPort": -1
	}`)

	bandwidth, httpPerMinute, connections := limitValues(rows)
	unlimited := i18n.T(i18n.Msg.Limits.Unlimited)
	if bandwidth != unlimited || httpPerMinute != unlimited || connections != unlimited {
		t.Errorf("unlimited limits = %q / %q / %q, want %q", bandwidth, httpPerMinute, connections, unlimited)
	}
}
