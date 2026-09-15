/*
 * Copyright (c) Huawei Technologies Co., Ltd. 2026-2027. All rights reserved.
 */

package sdk

import (
	devbridge "github.com/huaweicloud/devspace-devbridge/sdk"
	"huawei.com/devbridge/internal/auth"
	"huawei.com/devbridge/internal/config"
)

// NewClient creates an SDK client from the CLI auth system.
//
// Read order: override API Key → env var → keyring/config storage.
// API base URL and gateway address come from CLI config and ldflags-injected values.
func NewClient() (*devbridge.Devbridge, error) {
	cfg := devbridge.Config{
		APIBaseURL:  config.DefaultServerDomain + "/open-api-inner/v1/relay-controller",
		GatewayAddr: config.ResolveGatewayAddr(),
		GatewayHost: config.ResolveGatewayHost(),
	}

	if cred := auth.ReadValidAPIKey(); cred != nil && cred.APIKey != "" {
		cfg.APIKey = cred.APIKey
	}

	return devbridge.New(cfg)
}
