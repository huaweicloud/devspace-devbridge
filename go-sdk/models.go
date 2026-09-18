/*
 * Copyright (c) Huawei Technologies Co., Ltd. 2026-2027. All rights reserved.
 */

package sdk

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
}

type createTunnelRequest struct {
	Name        string `json:"name"`
	Description string `json:"description,omitempty"`
	ClusterID   string `json:"ClusterId"`
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
