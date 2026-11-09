/*
 * Copyright (c) Huawei Technologies Co., Ltd. 2026-2027. All rights reserved.
 */

package sdk

import (
	"encoding/json"
	"testing"
)

func TestLimitsUnmarshalFieldPresence(t *testing.T) {
	const base = `{
		"resetAt": 1790784000,
		"quotaBytes": 53687091200,
		"remainingBytes": 53683945472,
		"activeTunnels": 1,
		"maxTunnels": 10,
		"maxPortsPerTunnel": 10,
		"maxHostsPerTunnel": 1`

	cases := []struct {
		name               string
		payload            string
		wantBandwidthSet   bool
		wantHTTPSet        bool
		wantConnectionsSet bool
		wantBandwidthB     int64
		wantHTTPPerMin     int32
		wantConnections    int32
	}{
		{
			name:    "fields omitted",
			payload: base + `}`,
		},
		{
			name:               "explicit zero",
			payload:            base + `,"maxTunnelBandwidthBytesPerSecond": 0, "maxHttpRequestsPerMinutePerPort": 0, "maxConnectionsPerPort": 0}`,
			wantBandwidthSet:   true,
			wantHTTPSet:        true,
			wantConnectionsSet: true,
		},
		{
			name:               "positive values",
			payload:            base + `,"maxTunnelBandwidthBytesPerSecond": 5242880, "maxHttpRequestsPerMinutePerPort": 500, "maxConnectionsPerPort": 100}`,
			wantBandwidthSet:   true,
			wantHTTPSet:        true,
			wantConnectionsSet: true,
			wantBandwidthB:     5242880,
			wantHTTPPerMin:     500,
			wantConnections:    100,
		},
		{
			name:    "null values count as omitted",
			payload: base + `,"maxTunnelBandwidthBytesPerSecond": null, "maxHttpRequestsPerMinutePerPort": null, "maxConnectionsPerPort": null}`,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			var limits Limits
			if err := json.Unmarshal([]byte(tc.payload), &limits); err != nil {
				t.Fatalf("unmarshal limits: %v", err)
			}

			if got := limits.HasMaxTunnelBandwidth(); got != tc.wantBandwidthSet {
				t.Errorf("HasMaxTunnelBandwidth() = %v, want %v", got, tc.wantBandwidthSet)
			}
			if got := limits.HasMaxHTTPRequestsPerMinutePerPort(); got != tc.wantHTTPSet {
				t.Errorf("HasMaxHTTPRequestsPerMinutePerPort() = %v, want %v", got, tc.wantHTTPSet)
			}
			if got := limits.HasMaxConnectionsPerPort(); got != tc.wantConnectionsSet {
				t.Errorf("HasMaxConnectionsPerPort() = %v, want %v", got, tc.wantConnectionsSet)
			}
			if limits.MaxTunnelBandwidthBytesPerSecond != tc.wantBandwidthB {
				t.Errorf("MaxTunnelBandwidthBytesPerSecond = %d, want %d", limits.MaxTunnelBandwidthBytesPerSecond, tc.wantBandwidthB)
			}
			if limits.MaxHTTPRequestsPerMinutePerPort != tc.wantHTTPPerMin {
				t.Errorf("MaxHTTPRequestsPerMinutePerPort = %d, want %d", limits.MaxHTTPRequestsPerMinutePerPort, tc.wantHTTPPerMin)
			}
			if limits.MaxConnectionsPerPort != tc.wantConnections {
				t.Errorf("MaxConnectionsPerPort = %d, want %d", limits.MaxConnectionsPerPort, tc.wantConnections)
			}
		})
	}
}

func TestLimitsUnmarshalKeepsExistingFields(t *testing.T) {
	const payload = `{
		"resetAt": 1790784000,
		"quotaBytes": 53687091200,
		"remainingBytes": 53683945472,
		"activeTunnels": 2,
		"maxTunnels": 10,
		"maxPortsPerTunnel": 10,
		"maxHostsPerTunnel": 1,
		"maxTunnelBandwidthBytesPerSecond": 8388608,
		"maxHttpRequestsPerMinutePerPort": 600,
		"maxConnectionsPerPort": 120
	}`

	var limits Limits
	if err := json.Unmarshal([]byte(payload), &limits); err != nil {
		t.Fatalf("unmarshal limits: %v", err)
	}

	if limits.QuotaBytes != 53687091200 || limits.RemainingBytes != 53683945472 {
		t.Errorf("traffic fields not decoded: %+v", limits)
	}
	if limits.ActiveTunnels != 2 || limits.MaxTunnels != 10 ||
		limits.MaxPortsPerTunnel != 10 || limits.MaxHostsPerTunnel != 1 {
		t.Errorf("count fields not decoded: %+v", limits)
	}
	if limits.MaxTunnelBandwidthBytesPerSecond != 8388608 ||
		limits.MaxHTTPRequestsPerMinutePerPort != 600 || limits.MaxConnectionsPerPort != 120 {
		t.Errorf("limit fields not decoded: %+v", limits)
	}
}

func TestLimitsUnmarshalRejectsMalformedPayload(t *testing.T) {
	var limits Limits
	if err := json.Unmarshal([]byte(`{"maxTunnels": "ten"}`), &limits); err == nil {
		t.Fatal("expected an error for a malformed payload")
	}
}
