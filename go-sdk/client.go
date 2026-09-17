/*
 * Copyright (c) Huawei Technologies Co., Ltd. 2026-2027. All rights reserved.
 */

// Package sdk provides a Go client for the DevBridge tunnel service:
// REST API management for tunnels and ports, plus Host hosting and Connect connection capabilities.
package sdk

import (
	"crypto/tls"
	"fmt"
	"io"
	"log/slog"
	"net"
	"os"
	"regexp"

	"github.com/huaweicloud/devspace-devbridge/go-sdk/internal/httpclient"
)

const (
	DefaultAPIBaseURL  = "https://bridge.developer.myhuaweicloud.com/open-api-inner/v1/relay-controller"
	DefaultGatewayAddr = "gateway.cn-north-4-bridge.myhuaweicloud.com:443" //"gateway.devbridge-s2.hwtunnel.com:443"
	DefaultGatewayHost = "cn-north-4-bridge.myhuaweicloud.com"             //"devbridge-s2.hwtunnel.com"
	DefaultClusterID   = "cn-north-4-bridge"
)

var (
	tunnelIDRegexp   = regexp.MustCompile(`^[a-z2-7]{8}$`)
	tunnelNameRegexp = regexp.MustCompile(`^[\x{4e00}-\x{9fa5}A-Za-z0-9]([\x{4e00}-\x{9fa5}A-Za-z0-9-]{0,62}[\x{4e00}-\x{9fa5}A-Za-z0-9])?$`)
	tunnelDescRegexp = regexp.MustCompile(`^[\x{4e00}-\x{9fa5}A-Za-z0-9]{0,64}$`)
)

// AllPortsSentinel is the "all ports" sentinel value, corresponding to -1 in the backend.
// It only matters for visitor URL access (the gateway routes any port by SNI);
// host/connect SSH port forwarding must not forward this value.
const AllPortsSentinel = -1

// Config holds the SDK client configuration. A zero Config is valid —
// missing fields fall back to environment variables and sensible defaults.
type Config struct {
	// APIKey authenticates REST API requests (X-API-Key header).
	// Defaults to HW_API_KEY environment variable.
	APIKey string

	// APIBaseURL is the REST API base URL.
	// Defaults to DefaultAPIBaseURL.
	APIBaseURL string

	// GatewayAddr is the WebSocket gateway address (host:port).
	// Defaults to DefaultGatewayAddr.
	GatewayAddr string

	// GatewayHost is the WebSocket gateway SNI host.
	// Defaults to DefaultGatewayHost.
	GatewayHost string

	// OutputWriter receives user-facing output (connection progress,
	// hosted ports, forwarding info). Defaults to os.Stdout; set it to
	// io.Discard to silence these outputs.
	OutputWriter io.Writer
}

// resolve returns a copy with defaults and env-var fallbacks applied.
func (cfg Config) resolve() Config {
	out := cfg
	if out.APIBaseURL == "" {
		out.APIBaseURL = DefaultAPIBaseURL
	}
	if out.GatewayAddr == "" {
		out.GatewayAddr = DefaultGatewayAddr
	}
	if out.GatewayHost == "" {
		out.GatewayHost = DefaultGatewayHost
	}
	if out.APIKey == "" {
		out.APIKey = os.Getenv("HW_API_KEY")
	}
	if out.OutputWriter == nil {
		out.OutputWriter = os.Stdout
	}
	return out
}

// Devbridge is the DevBridge SDK client.
type Devbridge struct {
	apiKey          string
	gatewayAddr     string
	gatewayHost     string
	logger          *slog.Logger
	outputWriter    io.Writer
	api             *httpclient.Client
	tlsSessionCache tls.ClientSessionCache
}

// New creates a new SDK client from the given Config.
// A zero Config is valid; APIKey falls back to HW_API_KEY env var,
// and other fields fall back to sensible defaults.
//
// As part of initialization it starts a background DNS lookup for the gateway
// address to warm the OS resolver cache, and installs a shared TLS session
// cache so reconnects reuse TLS session tickets instead of re-negotiating
// from scratch.
func New(cfg Config) *Devbridge {
	resolved := cfg.resolve()
	d := &Devbridge{
		apiKey:          resolved.APIKey,
		gatewayAddr:     resolved.GatewayAddr,
		gatewayHost:     resolved.GatewayHost,
		logger:          slog.Default(),
		outputWriter:    resolved.OutputWriter,
		api:             httpclient.New(resolved.APIKey, resolved.APIBaseURL, slog.Default()),
		tlsSessionCache: tls.NewLRUClientSessionCache(32),
	}
	go warmDNS(d.gatewayAddr)
	return d
}

// warmDNS primes the OS DNS resolver cache for the gateway address so the
// first WebSocket dial does not pay the full resolution cost. Best-effort:
// failures are ignored and the dial falls back to its own resolution.
func warmDNS(gatewayAddr string) {
	host, _, err := net.SplitHostPort(gatewayAddr)
	if err != nil {
		host = gatewayAddr
	}
	if net.ParseIP(host) != nil {
		return
	}
	_, _ = net.LookupHost(host)
}

func (d *Devbridge) statusf(format string, args ...any) {
	fmt.Fprintf(d.outputWriter, format, args...)
}

func (d *Devbridge) statusln(args ...any) {
	fmt.Fprintln(d.outputWriter, args...)
}

func validateTunnelID(id string) error {
	if !tunnelIDRegexp.MatchString(id) {
		return fmt.Errorf("%w: %q (only lowercase letters and digits 2-7 allowed, length must be 8)", ErrInvalidTunnelID, id)
	}
	return nil
}

func validateTunnelName(name string) error {
	if !tunnelNameRegexp.MatchString(name) {
		return fmt.Errorf("%w: got %q", ErrInvalidTunnelName, name)
	}
	return nil
}

func validateTunnelDescription(description string) error {
	if !tunnelDescRegexp.MatchString(description) {
		return fmt.Errorf("%w: got %q", ErrInvalidTunnelDescription, description)
	}
	return nil
}

func validatePortNumber(port int) error {
	if port == AllPortsSentinel {
		return nil
	}
	if port < 1 || port > 65535 {
		return fmt.Errorf("%w: got %d", ErrInvalidPort, port)
	}
	return nil
}

// filterForwardPorts filters out the "all ports" sentinel value and returns only real ports (1-65535).
// host/connect SSH port forwarding must not forward -1 (the sentinel maps to uint32 4294967295,
// which is not a valid listening port); it only matters for visitor URL access.
func filterForwardPorts(ports []int) []int {
	out := make([]int, 0, len(ports))
	for _, p := range ports {
		if p == AllPortsSentinel {
			continue
		}
		out = append(out, p)
	}
	return out
}

func validateProtocol(protocol string) error {
	switch protocol {
	case "http", "https", "auto", "":
		return nil
	default:
		return fmt.Errorf("%w: got %s", ErrInvalidProtocol, protocol)
	}
}

func validateScope(scope string) error {
	if scope != "host" && scope != "connect" {
		return fmt.Errorf("%w: got %s", ErrInvalidScope, scope)
	}
	return nil
}
