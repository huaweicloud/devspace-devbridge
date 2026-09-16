/*
 * Copyright (c) Huawei Technologies Co., Ltd. 2026-2027. All rights reserved.
 */

package config

import (
	devbridge "github.com/huaweicloud/devspace-devbridge/sdk"
)

// NewClient creates an SDK client from the CLI configuration.
//
// The API Key is resolved by the caller (see cmd package) and passed in,
// keeping this package free of any dependency on the auth package.
// API base URL and gateway address come from CLI config and ldflags-injected values.
func NewClient(apiKey string) *devbridge.Devbridge {
	cfg := devbridge.Config{
		APIBaseURL:         DefaultServerDomain + "/open-api-inner/v1/relay-controller",
		GatewayAddr:        ResolveGatewayAddr(),
		GatewayHost:        ResolveGatewayHost(),
		InsecureSkipVerify: ResolveTLSSkipVerify(),
		APIKey:             apiKey,
	}

	return devbridge.New(cfg)
}
