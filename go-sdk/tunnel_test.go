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
		{"empty description allowed", "", false},
		{"letters", "devbridge", false},
		{"digits", "20260911", false},
		{"letters and digits mixed", "env01", false},
		{"64-char boundary", strings.Repeat("a", 64), false},
		{"65-char over limit", strings.Repeat("a", 65), true},
		{"space not allowed", "dev bridge", true},
		{"hyphen not allowed", "dev-bridge", true},
		{"underscore not allowed", "dev_bridge", true},
		{"punctuation not allowed", "dev!", true},
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
		t.Fatalf("expected ErrInvalidTunnelDescription, got: %v", err)
	}
}

func TestValidateTunnelName(t *testing.T) {
	cases := []struct {
		name    string
		in      string
		wantErr bool
	}{
		{"letters", "devbridge", false},
		{"digits", "20260911", false},
		{"letters and digits mixed", "env01", false},
		{"hyphen in middle allowed", "dev-bridge", false},
		{"1-char lower bound", "a", false},
		{"64-char upper bound", strings.Repeat("a", 64), false},
		{"65-char over limit", strings.Repeat("a", 65), true},
		{"empty name", "", true},
		{"space not allowed", "dev bridge", true},
		{"underscore not allowed", "dev_bridge", true},
		{"leading hyphen not allowed", "-devbridge", true},
		{"trailing hyphen not allowed", "devbridge-", true},
		{"punctuation not allowed", "dev!", true},
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
		t.Fatalf("expected ErrInvalidTunnelName, got: %v", err)
	}
}
