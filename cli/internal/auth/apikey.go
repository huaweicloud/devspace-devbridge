package auth

import (
	"log/slog"

	"huawei.com/devbridge/internal/logging"
)

func maskSecret(s string) string {
	if len(s) <= 8 {
		return "****"
	}
	return s[:4] + "****" + s[len(s)-4:]
}

var overrideAPIKey string //nolint:gochecknoglobals

// SetOverrideAPIKey 设置命令行传入的 API Key，使后续 API 调用直接使用该 key.
func SetOverrideAPIKey(key string) {
	overrideAPIKey = key
}

func ReadValidAPIKey() *Credential {
	cred := readValidAPIKey()
	if cred != nil && logging.LogLevel() <= slog.LevelDebug {
		slog.Debug("read valid API key",
			"apiKey", maskSecret(cred.APIKey),
		)
	}
	return cred
}

func readValidAPIKey() *Credential {
	if overrideAPIKey != "" {
		return &Credential{APIKey: overrideAPIKey}
	}
	if cred := loadFromEnv(); cred != nil && isValidAPIKey(cred) {
		return cred
	}

	if cred, _, err := LoadCredential(CredentialName); err == nil && cred != nil && isValidAPIKey(cred) {
		return cred
	}
	return nil
}

func isValidAPIKey(cred *Credential) bool {
	return cred != nil && cred.APIKey != ""
}
