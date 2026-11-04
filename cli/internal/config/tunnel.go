package config

import (
	"errors"

	"huawei.com/devbridge/internal/i18n"
)

const defaultTunnelKey = "default-tunnel-id"

// StoreDefaultTunnel writes the default tunnel ID directly to the config file.
func StoreDefaultTunnel(tunnelID string) error {
	return set(defaultTunnelKey, tunnelID)
}

// LoadDefaultTunnel reads the default tunnel ID from the config file.
func LoadDefaultTunnel() (string, error) {
	if v, ok := get(defaultTunnelKey); ok {
		if s, ok := v.(string); ok {
			return s, nil
		}
	}
	return "", errors.New(i18n.T(i18n.Msg.Config.NoDefaultTunnel))
}

// DeleteDefaultTunnel deletes the default tunnel ID; a missing key is treated as already deleted.
func DeleteDefaultTunnel() error {
	if err := deleteKey(defaultTunnelKey); err != nil && !errors.Is(err, errKeyNotFound) {
		return err
	}
	return nil
}