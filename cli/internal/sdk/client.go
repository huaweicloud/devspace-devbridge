package sdk

import (
	"huawei.com/devbridge/internal/auth"
	"huawei.com/devbridge/internal/config"
	devbridge "huawei.com/devbridge/sdk"
)

// ServerAddr WebSocket 网关地址（host:port），供 ldflags 注入。
var ServerAddr = "gateway.cn-north-4-bridge.myhuaweicloud.com:443"

// ServerHost WebSocket 网关 SNI host，供 ldflags 注入。
var ServerHost = "cn-north-4-bridge.myhuaweicloud.com"

// NewClient 从 CLI 的认证体系创建 SDK 客户端。
//
// 读取顺序：override API Key → 环境变量 → keyring/config 存储。
// API base URL 和网关地址从 CLI 配置和 ldflags 注入值获取。
func NewClient() (*devbridge.Client, error) {
	cfg := devbridge.Config{
		APIBaseURL:  config.DefaultServerDomain + "/open-api-inner/v1/relay-controller",
		GatewayAddr: ServerAddr,
		GatewayHost: ServerHost,
	}

	if cred := auth.ReadValidAPIKey(); cred != nil && cred.APIKey != "" {
		cfg.APIKey = cred.APIKey
	}

	return devbridge.NewClient(cfg)
}
