package config

import (
	"errors"
	"fmt"
)

const defaultTunnelKey = "default-tunnel-id"

// StoreDefaultTunnel 将默认隧道 ID 直接写入配置文件.
func StoreDefaultTunnel(tunnelID string) error {
	return Set(defaultTunnelKey, tunnelID)
}

// LoadDefaultTunnel 从配置文件读取默认隧道 ID.
func LoadDefaultTunnel() (string, error) {
	if v, ok := Get(defaultTunnelKey); ok {
		if s, ok := v.(string); ok {
			return s, nil
		}
	}
	return "", fmt.Errorf("tunnel ID not specified and no default tunnel set, " +
		"please specify via argument or use 'devbridge set' to set default") //nolint:lll
}

// DeleteDefaultTunnel 从配置文件删除默认隧道 ID.
// 没设过默认隧道（key 不存在）视为已删除，不返回错误.
func DeleteDefaultTunnel() error {
	if err := Delete(defaultTunnelKey); err != nil && !errors.Is(err, ErrKeyNotFound) {
		return err
	}
	return nil
}
