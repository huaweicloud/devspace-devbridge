/*
 * Copyright (c) Huawei Technologies Co., Ltd. 2026-2027. All rights reserved.
 */

package config

// ServerAddr is the WebSocket gateway address (host:port), injected via ldflags.
var ServerAddr = "gateway.devbridge-s2.hwtunnel.com"

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

// ResolveGatewayHost returns the WebSocket gateway SNI host.
// Config file gateway-host takes precedence over the ldflags-injected default.
func ResolveGatewayHost() string {
	if v := LoadGatewayHost(); v != "" {
		return v
	}
	return ServerHost
}

// ResolveTLSSkipVerify returns whether to skip TLS certificate verification
// for the WebSocket gateway connection. Reads from the config file only;
// unset defaults to false (verify).
func ResolveTLSSkipVerify() bool {
	return LoadTLSSkipVerify()
}
