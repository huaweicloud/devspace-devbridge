/*
 * Copyright (c) Huawei Technologies Co., Ltd. 2026-2027. All rights reserved.
 */

// Package i18n provides minimal bilingual (zh/en) message support for the
// SDK's user-facing terminal output. It mirrors the CLI's i18n package and
// reads the same DEVBRIDGE_LANG environment variable, so the CLI and the SDK
// always render in the same language.
package i18n

import (
	"os"
	"strings"
)

// Lang is the output language.
type Lang string

const (
	// ZH is Chinese.
	ZH Lang = "zh"
	// EN is English (default).
	EN Lang = "en"
)

// Message holds the zh/en variants of a user-facing string.
type Message struct {
	ZH string
	EN string
}

var currentLang = detectLang()

func detectLang() Lang {
	if env := os.Getenv("DEVBRIDGE_LANG"); env != "" {
		if strings.HasPrefix(strings.ToLower(env), "zh") {
			return ZH
		}
		return EN
	}
	return EN
}

// T returns the message in the current language.
func T(msg Message) string {
	if currentLang == ZH {
		return msg.ZH
	}
	return msg.EN
}
