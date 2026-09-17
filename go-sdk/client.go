/*
 * Copyright (c) Huawei Technologies Co., Ltd. 2026-2027. All rights reserved.
 */

// Package sdk provides a Go client for the DevBridge tunnel service:
// REST API management for tunnels and ports, plus Host hosting and Connect connection capabilities.
package sdk

import (
	"fmt"
	"io"
	"log/slog"
	"os"
	"regexp"

	"github.com/huaweicloud/devspace-devbridge/go-sdk/internal/httpclient"
)

const (
	DefaultAPIBaseURL  = "https://bridge.developer.myhuaweicloud.com/open-api-inner/v1/relay-controller"
	DefaultGatewayAddr = "gateway.devbridge-s2.hwtunnel.com:443"
	DefaultGatewayHost = "devbridge-s2.hwtunnel.com"
	DefaultClusterID   = "devbridge-s2"
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

	// InsecureSkipVerify disables TLS certificate verification for the
	// WebSocket connection to the gateway. Defaults to false (verify).
	// Enable only for gateways whose certificate does not cover the tunnel
	// domain (e.g. multi-level domains beyond a single-segment wildcard).
	InsecureSkipVerify bool

	// StatusWriter receives user-facing status lines (connection progress,
	// hosted ports, forwarding info). Defaults to os.Stdout; set it to
	// io.Discard to silence these outputs.
	StatusWriter io.Writer
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
	if out.StatusWriter == nil {
		out.StatusWriter = os.Stdout
	}
	return out
}

// Devbridge is the DevBridge SDK client.
type Devbridge struct {
	apiKey             string
	gatewayAddr        string
	gatewayHost        string
	insecureSkipVerify bool
	logger             *slog.Logger
	statusWriter       io.Writer
	api                *httpclient.Client
}

// New creates a new SDK client from the given Config.
// A zero Config is valid; APIKey falls back to HW_API_KEY env var,
// and other fields fall back to sensible defaults.
func New(cfg Config) *Devbridge {
	resolved := cfg.resolve()
	return &Devbridge{
		apiKey:             resolved.APIKey,
		gatewayAddr:        resolved.GatewayAddr,
		gatewayHost:        resolved.GatewayHost,
		insecureSkipVerify: resolved.InsecureSkipVerify,
		logger:             slog.Default(),
		statusWriter:       resolved.StatusWriter,
		api:                httpclient.New(resolved.APIKey, resolved.APIBaseURL, slog.Default()),
	}
}

func (d *Devbridge) statusf(format string, args ...any) {
	fmt.Fprintf(d.statusWriter, format, args...)
}

func (d *Devbridge) statusln(args ...any) {
	fmt.Fprintln(d.statusWriter, args...)
}

func validateTunnelID(id string) error {
	if !tunnelIDRegexp.MatchString(id) {
		return fmt.Errorf("%w: %q (only lowercase letters and digits 2-7 allowed, length must be 8)", ErrInvalidTunnelID, id)
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
