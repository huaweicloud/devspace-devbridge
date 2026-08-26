package i18n

import (
	"os"
	"strings"
)

type Lang string

const (
	ZH Lang = "zh"
	EN Lang = "en"
)

type Message struct {
	ZH string
	EN string
}

var currentLang = detectLang() //nolint:gochecknoglobals

func detectLang() Lang {
	if env := os.Getenv("DEVBRIDGE_LANG"); env != "" {
		if strings.HasPrefix(strings.ToLower(env), "zh") {
			return ZH
		}
		return EN
	}
	return EN
}

func T(msg Message) string {
	if currentLang == ZH {
		return msg.ZH
	}
	return msg.EN
}

func GetLang() Lang {
	return currentLang
}
