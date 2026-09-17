package config

import (
	"errors"
	"fmt"
)

const (
	defaultTunnelKey = "default-tunnel-id"
	gatewayAddrKey   = "gateway-addr"
	gatewayHostKey   = "gateway-host"
)

// StoreDefaultTunnel 将默认隧道 ID 直接写入配置文件.
func StoreDefaultTunnel(tunnelID string) error {
	return set(defaultTunnelKey, tunnelID)
}

// LoadDefaultTunnel 从配置文件读取默认隧道 ID.
func LoadDefaultTunnel() (string, error) {
	if v, ok := get(defaultTunnelKey); ok {
		if s, ok := v.(string); ok {
			return s, nil
		}
	}
	return "", fmt.Errorf("tunnel ID not specified and no default tunnel set, " +
		"please specify via argument or use 'devbridge set' to set default")
}

// DeleteDefaultTunnel 删除默认隧道 ID，key 不存在视为已删除。
func DeleteDefaultTunnel() error {
	if err := deleteKey(defaultTunnelKey); err != nil && !errors.Is(err, errKeyNotFound) {
		return err
	}
	return nil
}

// getStringOpt 从配置文件读取字符串配置项；不存在或类型不匹配时返回空串。
func getStringOpt(key string) string {
	if v, ok := get(key); ok {
		if s, ok := v.(string); ok {
			return s
		}
	}
	return ""
}

// LoadGatewayAddr 从配置文件读取 WebSocket 网关地址（host:port）。
// 未配置时返回空串。
func LoadGatewayAddr() string {
	return getStringOpt(gatewayAddrKey)
}

// LoadGatewayHost 从配置文件读取 WebSocket 网关 SNI host。
// 未配置时返回空串。
func LoadGatewayHost() string {
	return getStringOpt(gatewayHostKey)
}

// StoreGatewayAddr 写入 WebSocket 网关地址（host:port）。
func StoreGatewayAddr(addr string) error {
	return set(gatewayAddrKey, addr)
}

// StoreGatewayHost 写入 WebSocket 网关 SNI host。
func StoreGatewayHost(host string) error {
	return set(gatewayHostKey, host)
}

// DeleteGatewayAddr 删除 WebSocket 网关地址配置（恢复编译时默认值）。
func DeleteGatewayAddr() error {
	if err := deleteKey(gatewayAddrKey); err != nil && !errors.Is(err, errKeyNotFound) {
		return err
	}
	return nil
}

// DeleteGatewayHost 删除 WebSocket 网关 SNI host 配置（恢复编译时默认值）。
func DeleteGatewayHost() error {
	if err := deleteKey(gatewayHostKey); err != nil && !errors.Is(err, errKeyNotFound) {
		return err
	}
	return nil
}
