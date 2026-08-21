//go:build windows

package i18n

import (
	"os"
	"strings"
	"syscall"
	"unsafe"
)

var (
	kernel32                     = syscall.NewLazyDLL("kernel32.dll")
	procGetUserDefaultUILanguage = kernel32.NewProc("GetUserDefaultUILanguage")
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
	r, _, _ := procGetUserDefaultUILanguage.Call()
	langID := uint16(r)
	if langID&0x3FF == 0x04 {
		return ZH
	}
	return EN
}

var _ = unsafe.Sizeof(0)
