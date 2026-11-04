package auth

import (
	"context"
	"log/slog"
)

var overrideAPIKey string

// SetOverrideAPIKey sets the command-line API Key so subsequent API calls use it directly.
func SetOverrideAPIKey(key string) {
	overrideAPIKey = key
}

func ReadValidAPIKey() *Credential {
	cred := readValidAPIKey()
	if cred != nil && slog.Default().Enabled(context.Background(), slog.LevelDebug) {
		slog.Debug("read valid API key",
			"apiKey", maskAPIKey(cred.APIKey),
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
