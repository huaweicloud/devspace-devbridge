/*
 * Copyright (c) Huawei Technologies Co., Ltd. 2026-2027. All rights reserved.
 */

package config

import (
	"strings"
)

// ServerAddr is the WebSocket gateway address (host:port), injected via ldflags.
var ServerAddr = "gateway.devbridge-s2.hwtunnel.com:443"

// ServerHost is the WebSocket gateway SNI host, injected via ldflags.
var ServerHost = "devbridge-s2.hwtunnel.com"

// ResolveGatewayAddr returns the WebSocket gateway address.
// Config file gateway-addr takes precedence over the ldflags-injected default.
func ResolveGatewayAddr() string {
	if v := LoadGatewayAddr(); v != "" {
		return v
	}
	return ServerAddr
}

// ResolveGatewayHost returns the WebSocket gateway SNI host for the given API
// key. Precedence: explicit config (gateway-host) > API key scope inference
// (devbox_ -> devbox-s2.hwtunnel.com, devbridge_ -> devbridge-s2.hwtunnel.com)
// > ldflags-injected build default. The SNI cluster label must match the JWT
// cluster claim, which the backend derives from the API key scope, so inferring
// the host from the key keeps host/connect sessions accepted by the gateway.
func ResolveGatewayHost(apiKey string) string {
	if v := LoadGatewayHost(); v != "" {
		return v
	}
	switch {
	case strings.HasPrefix(apiKey, "devbox_"):
		return "devbox-s2.hwtunnel.com"
	case strings.HasPrefix(apiKey, "devbridge_"):
		return "devbridge-s2.hwtunnel.com"
	}
	return ServerHost
}
