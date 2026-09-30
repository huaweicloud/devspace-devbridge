package config

import (
	"errors"

	"huawei.com/devbridge/internal/i18n"
)

const (
	defaultTunnelKey = "default-tunnel-id"
	gatewayAddrKey   = "gateway-addr"
	gatewayHostKey   = "gateway-host"
)

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

// getStringOpt reads a string config item; returns "" when the key is missing or the type mismatches.
func getStringOpt(key string) string {
	if v, ok := get(key); ok {
		if s, ok := v.(string); ok {
			return s
		}
	}
	return ""
}

// LoadGatewayAddr reads the WebSocket gateway address (host:port).
// Returns "" when not configured.
func LoadGatewayAddr() string {
	return getStringOpt(gatewayAddrKey)
}

// LoadGatewayHost reads the WebSocket gateway SNI host.
// Returns "" when not configured.
func LoadGatewayHost() string {
	return getStringOpt(gatewayHostKey)
}

// StoreGatewayAddr writes the WebSocket gateway address (host:port).
func StoreGatewayAddr(addr string) error {
	return set(gatewayAddrKey, addr)
}

// StoreGatewayHost writes the WebSocket gateway SNI host.
func StoreGatewayHost(host string) error {
	return set(gatewayHostKey, host)
}

// DeleteGatewayAddr deletes the WebSocket gateway address config (restores the build-time default).
func DeleteGatewayAddr() error {
	if err := deleteKey(gatewayAddrKey); err != nil && !errors.Is(err, errKeyNotFound) {
		return err
	}
	return nil
}

// DeleteGatewayHost deletes the WebSocket gateway SNI host config (restores the build-time default).
func DeleteGatewayHost() error {
	if err := deleteKey(gatewayHostKey); err != nil && !errors.Is(err, errKeyNotFound) {
		return err
	}
	return nil
}
