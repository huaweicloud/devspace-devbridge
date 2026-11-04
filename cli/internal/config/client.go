/*
 * Copyright (c) Huawei Technologies Co., Ltd. 2026-2027. All rights reserved.
 */

package config

import (
	"fmt"
	"net/http"
	"runtime"
	"strings"
	"time"

	devbridge "github.com/huaweicloud/devspace-devbridge/go-sdk"
)

// versionTransport injects the CLI version into the User-Agent header of
// every REST request, so the backend can decide whether to prompt the client
// to upgrade. It only affects requests that flow through this transport.
type versionTransport struct {
	base      http.RoundTripper
	userAgent string
}

func (t versionTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	req.Header.Set("User-Agent", t.userAgent)
	return t.base.RoundTrip(req)
}

// userAgent builds the User-Agent value: devbridge-cli/<version> (<OS>/<arch>; stable).
func userAgent(version string) string {
	osName := runtime.GOOS
	if osName != "" {
		osName = strings.ToUpper(osName[:1]) + osName[1:]
	}
	return fmt.Sprintf("devbridge-cli/%s (%s/%s; stable)", version, osName, runtime.GOARCH)
}

// NewClient creates an SDK client from the CLI configuration.
//
// The API Key is resolved by the caller (see cmd package) and passed in,
// keeping this package free of any dependency on the auth package.
// version is the CLI version, sent in the User-Agent header of REST requests.
// API base URL and gateway address come from CLI config and ldflags-injected values.
func NewClient(apiKey, version string) *devbridge.Devbridge {
	cfg := devbridge.Config{
		APIBaseURL:  DefaultServerDomain,
		GatewayAddr: ResolveGatewayAddr(),
		GatewayHost: ResolveGatewayHost(apiKey),
		APIKey:      apiKey,
		Version:     version,
		HTTPClient: &http.Client{
			Timeout: 30 * time.Second,
			Transport: versionTransport{
				base:      http.DefaultTransport,
				userAgent: userAgent(version),
			},
		},
	}

	return devbridge.New(cfg)
}
