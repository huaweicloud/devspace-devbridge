package sdk

// Tunnel 隧道
type Tunnel struct {
	ID               string `json:"tunnelId"` // 8 位小写 Base32
	Name             string `json:"name"`
	Description      string `json:"description"`
	ExpirationHours  int    `json:"expirationHours"`  // 有效期（小时）
	TunnelExpiration uint32 `json:"tunnelExpiration"` // 过期时间（Unix 秒）
	PortCount        int    `json:"portCount"`
}

// TunnelDetail 隧道详情（含状态）
type TunnelDetail struct {
	Name             string        `json:"name"`
	ID               string        `json:"tunnelId"`
	TunnelExpiration uint32        `json:"tunnelExpiration"` // Unix 秒
	Description      string        `json:"description"`
	Status           *TunnelStatus `json:"status,omitempty"`
}

// TunnelStatus 隧道运行状态
type TunnelStatus struct {
	ClientConnectionCount int   `json:"clientConnectionCount"`
	HostConnectionCount   int   `json:"hostConnectionCount"`
	TotalUploadBytes      int64 `json:"totalUploadBytes"`
	TotalDownloadBytes    int64 `json:"totalDownloadBytes"`
}

// Port 端口配置
type Port struct {
	TunnelID       string `json:"tunnelId"`
	Port           int32  `json:"port"`
	Protocol       string `json:"protocol"` // http, https, auto
	AllowAnonymous bool   `json:"allowAnonymous"`
}

// TunnelToken 隧道令牌
type TunnelToken struct {
	TunnelID string `json:"tunnelId"`
	Scope    string `json:"scope"` // host 或 connect
	Token    string `json:"token"` // JWT
}

// Limits 配额
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

// errorBody 是另一种错误格式（部分接口使用）
type errorBody struct {
	Error struct {
		Code    string `json:"code"`
		Message string `json:"message"`
		Target  string `json:"target"`
	} `json:"error"`
}
