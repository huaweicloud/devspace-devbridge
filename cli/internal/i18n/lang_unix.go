//go:build !windows

package i18n

import (
	"os"
	"strings"
)

func DetectSystemLang() Lang {
	for _, key := range []string{"LC_ALL", "LC_MESSAGES", "LANG"} {
		if val := os.Getenv(key); val != "" {
			lower := strings.ToLower(val)
			if strings.HasPrefix(lower, "zh") {
				return ZH
			}
			if strings.HasPrefix(lower, "en") {
				return EN
			}
		}
	}
	return EN
}
