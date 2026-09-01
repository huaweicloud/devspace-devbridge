package api

const limitsPath = "/limits"

type LimitsResult struct {
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

func GetLimits() (*LimitsResult, error) {
	var result LimitsResult
	if err := get(limitsPath, &result); err != nil {
		return nil, err
	}
	return &result, nil
}
