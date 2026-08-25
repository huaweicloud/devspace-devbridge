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

// overrideAPIKey 命令行 --api-key 传入的 key，优先级最高.
var overrideAPIKey string //nolint:gochecknoglobals // cobra CLI 惯用全局变量

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
	// LoadCredential tries keyring (password vault) first, then config file,
	// matching StoreCredential: keyring first, config file only as fallback when keyring is unavailable.
	if cred, _, err := LoadCredential(CredentialName); err == nil && cred != nil && isValidAPIKey(cred) {
		return cred
	}
	return nil
}

// isValidAPIKey 判断凭证是否有效。API Key 不会过期，因此只需检查非空。
func isValidAPIKey(cred *Credential) bool {
	return cred != nil && cred.APIKey != ""
}
