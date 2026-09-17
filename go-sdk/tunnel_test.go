/*
 * Copyright (c) Huawei Technologies Co., Ltd. 2026-2027. All rights reserved.
 */

package sdk

import (
	"errors"
	"strings"
	"testing"
)

func TestValidateTunnelDescription(t *testing.T) {
	cases := []struct {
		name    string
		desc    string
		wantErr bool
	}{
		{"空描述允许", "", false},
		{"中文", "开发联调环境", false},
		{"字母", "devbridge", false},
		{"数字", "20260911", false},
		{"中文字母数字混合", "联调env01", false},
		{"64字符边界", strings.Repeat("a", 64), false},
		{"65字符超限", strings.Repeat("a", 65), true},
		{"空格不允许", "dev bridge", true},
		{"连字符不允许", "dev-bridge", true},
		{"下划线不允许", "dev_bridge", true},
		{"标点不允许", "描述！", true},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			err := validateTunnelDescription(tc.desc)
			if (err != nil) != tc.wantErr {
				t.Fatalf("validateTunnelDescription(%q) err = %v, wantErr = %v", tc.desc, err, tc.wantErr)
			}
		})
	}
}

func TestValidateTunnelDescriptionSentinel(t *testing.T) {
	err := validateTunnelDescription("bad desc!")
	if !errors.Is(err, ErrInvalidTunnelDescription) {
		t.Fatalf("期望返回 ErrInvalidTunnelDescription，实际: %v", err)
	}
}
