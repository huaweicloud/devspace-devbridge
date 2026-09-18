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

func TestValidateTunnelName(t *testing.T) {
	cases := []struct {
		name    string
		in      string
		wantErr bool
	}{
		{"中文", "开发联调", false},
		{"字母", "devbridge", false},
		{"数字", "20260911", false},
		{"中文字母数字混合", "联调env01", false},
		{"中间连字符允许", "dev-bridge", false},
		{"1字符下界", "a", false},
		{"64字符上界", strings.Repeat("a", 64), false},
		{"65字符超限", strings.Repeat("a", 65), true},
		{"空名称", "", true},
		{"空格不允许", "dev bridge", true},
		{"下划线不允许", "dev_bridge", true},
		{"连字符开头不允许", "-devbridge", true},
		{"连字符结尾不允许", "devbridge-", true},
		{"标点不允许", "隧道！", true},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			err := validateTunnelName(tc.in)
			if (err != nil) != tc.wantErr {
				t.Fatalf("validateTunnelName(%q) err = %v, wantErr = %v", tc.in, err, tc.wantErr)
			}
		})
	}
}

func TestValidateTunnelNameSentinel(t *testing.T) {
	err := validateTunnelName("bad name!")
	if !errors.Is(err, ErrInvalidTunnelName) {
		t.Fatalf("期望返回 ErrInvalidTunnelName，实际: %v", err)
	}
}
