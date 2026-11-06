/*
 * Copyright (c) Huawei Technologies Co., Ltd. 2026-2027. All rights reserved.
 */

package sdk

import "encoding/json"

const (
	maxTunnelBandwidthField              = "maxTunnelBandwidthBytesPerSecond"
	maxHTTPRequestsPerMinutePerPortField = "maxHttpRequestsPerMinutePerPort"
	maxConnectionsPerPortField           = "maxConnectionsPerPort"
)

// Tunnel represents a tunnel.
type Tunnel struct {
	ID               string `json:"tunnelId"` // 8-char lowercase Base32
	Name             string `json:"name"`
	Description      string `json:"description"`
	ExpirationHours  int    `json:"expirationHours"`  // validity in hours
	TunnelExpiration uint32 `json:"tunnelExpiration"` // expiration time (Unix seconds)
	PortCount        int    `json:"portCount"`
}

// TunnelDetail represents tunnel details including status.
type TunnelDetail struct {
	Name             string        `json:"name"`
	ID               string        `json:"tunnelId"`
	TunnelExpiration uint32        `json:"tunnelExpiration"` // Unix seconds
	Description      string        `json:"description"`
	Status           *TunnelStatus `json:"status,omitempty"`
}

// TunnelStatus represents the runtime status of a tunnel.
type TunnelStatus struct {
	ClientConnectionCount int   `json:"clientConnectionCount"`
	HostConnectionCount   int   `json:"hostConnectionCount"`
	TotalUploadBytes      int64 `json:"totalUploadBytes"`
	TotalDownloadBytes    int64 `json:"totalDownloadBytes"`
}

// Port represents a port configuration.
type Port struct {
	TunnelID       string `json:"tunnelId"`
	Port           int    `json:"port"`
	Protocol       string `json:"protocol"` // http, https, auto
	AllowAnonymous bool   `json:"allowAnonymous"`
}

// TunnelToken represents a tunnel token.
type TunnelToken struct {
	TunnelID string `json:"tunnelId"`
	Scope    string `json:"scope"` // "host" or "connect"
	Token    string `json:"token"` // JWT
}

// Limits represents the quota.
type Limits struct {
	ResetAt                          int64 `json:"resetAt"`
	QuotaBytes                       int64 `json:"quotaBytes"`
	RemainingBytes                   int64 `json:"remainingBytes"`
	ActiveTunnels                    int64 `json:"activeTunnels"`
	MaxTunnels                       int32 `json:"maxTunnels"`
	MaxPortsPerTunnel                int32 `json:"maxPortsPerTunnel"`
	MaxHostsPerTunnel                int32 `json:"maxHostsPerTunnel"`
	MaxTunnelBandwidthBytesPerSecond int64 `json:"maxTunnelBandwidthBytesPerSecond"`
	MaxHTTPRequestsPerMinutePerPort  int32 `json:"maxHttpRequestsPerMinutePerPort"`
	MaxConnectionsPerPort            int32 `json:"maxConnectionsPerPort"`

	present limitsFieldPresence
}

// limitsFieldPresence records which optional limits the server actually sent.
// An omitted field is not the same as a field explicitly set to zero.
type limitsFieldPresence struct {
	maxTunnelBandwidth              bool
	maxHTTPRequestsPerMinutePerPort bool
	maxConnectionsPerPort           bool
}

// UnmarshalJSON decodes a Limits value while recording whether the per-port and
// bandwidth limit fields were present in the response. The exported fields are
// unchanged, so existing callers keep working; use the Has* methods to tell an
// omitted field from an explicit zero.
func (l *Limits) UnmarshalJSON(data []byte) error {
	type limitsAlias Limits
	var decoded limitsAlias
	if err := json.Unmarshal(data, &decoded); err != nil {
		return err
	}

	var fields map[string]json.RawMessage
	if err := json.Unmarshal(data, &fields); err != nil {
		return err
	}

	*l = Limits(decoded)
	l.present = limitsFieldPresence{
		maxTunnelBandwidth:              fieldDelivered(fields, maxTunnelBandwidthField),
		maxHTTPRequestsPerMinutePerPort: fieldDelivered(fields, maxHTTPRequestsPerMinutePerPortField),
		maxConnectionsPerPort:           fieldDelivered(fields, maxConnectionsPerPortField),
	}
	return nil
}

// fieldDelivered reports whether key was present with a non-null value.
func fieldDelivered(fields map[string]json.RawMessage, key string) bool {
	value, ok := fields[key]
	return ok && string(value) != "null"
}

// HasMaxTunnelBandwidth reports whether the response carried
// maxTunnelBandwidthBytesPerSecond. When false, the field was omitted by the
// server and must not be rendered as an explicit zero.
func (l *Limits) HasMaxTunnelBandwidth() bool { return l.present.maxTunnelBandwidth }

// HasMaxHTTPRequestsPerMinutePerPort reports whether the response carried
// maxHttpRequestsPerMinutePerPort.
func (l *Limits) HasMaxHTTPRequestsPerMinutePerPort() bool {
	return l.present.maxHTTPRequestsPerMinutePerPort
}

// HasMaxConnectionsPerPort reports whether the response carried
// maxConnectionsPerPort.
func (l *Limits) HasMaxConnectionsPerPort() bool { return l.present.maxConnectionsPerPort }

type createTunnelRequest struct {
	Name        string `json:"name"`
	Description string `json:"description,omitempty"`
	ClusterID   string `json:"clusterId"`
	Expiration  int    `json:"expiration,omitempty"`
}

type updateTunnelRequest struct {
	Name        *string `json:"name,omitempty"`
	Description *string `json:"description,omitempty"`
	Expiration  *int    `json:"expiration,omitempty"`
}

type createPortRequest struct {
	Port           int    `json:"port"`
	Protocol       string `json:"protocol,omitempty"`
	AllowAnonymous *bool  `json:"allowAnonymous,omitempty"`
}

type updatePortRequest struct {
	AllowAnonymous *bool `json:"allowAnonymous,omitempty"`
}
