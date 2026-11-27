package auth

import (
	"encoding/base64"
	"fmt"
	"log/slog"
	"sync"

	"huawei.com/devbridge/internal/config"
	"huawei.com/devbridge/internal/i18n"

	"github.com/zalando/go-keyring"
)

const CredentialName = "HWCLOUD"

const (
	credentialsKey = "credentials"
	userInfoKey    = "user-info"

	// keyringUser is the attribute that identifies the stored item inside the
	// Secret Service collection.
	keyringUser = "Credentials"
)

// secretBackend is the subset of go-keyring used by this package. It is a
// package variable so tests can substitute a recording fake and assert that the
// Secret Service is never touched when no session bus is available.
type secretBackend interface {
	Set(service, user, password string) error
	Get(service, user string) (string, error)
	Delete(service, user string) error
}

// keyringBackend is the production Secret Service backend.
type keyringBackend struct{}

func (keyringBackend) Set(service, user, password string) error {
	return keyring.Set(service, user, password)
}

func (keyringBackend) Get(service, user string) (string, error) {
	return keyring.Get(service, user)
}

func (keyringBackend) Delete(service, user string) error {
	return keyring.Delete(service, user)
}

var credentialsBackend secretBackend = keyringBackend{}

// fallbackNoticeOnce keeps the degradation notice to one message per process.
var fallbackNoticeOnce sync.Once

// useSecretService reports whether credentials may go through go-keyring.
//
// go-keyring reaches the D-Bus session bus on every call, and a bus-less host
// answers that call by forking an orphan dbus-daemon through dbus-launch. So the
// decision has to be made before the call, not by falling back after it fails.
// When no bus is available the caller persists credentials in the config file
// and the user is told once per process where they land and how they are stored.
func useSecretService() bool {
	available, _ := sessionBusAvailable()
	if !available {
		notifyCredentialFallback()
	}
	return available
}

// notifyCredentialFallback announces the config-file backend once per process.
func notifyCredentialFallback() {
	fallbackNoticeOnce.Do(func() {
		path := ""
		if filePath, err := config.ConfigPath(); err == nil {
			path = filePath
		}
		slog.Warn(fmt.Sprintf(i18n.T(i18n.Msg.Auth.CredentialBackendFallback), path))
	})
}

// resetFallbackNotice allows tests to observe the one-time notice. It exists for
// tests only.
func resetFallbackNotice() {
	fallbackNoticeOnce = sync.Once{}
}

func encodeCredential(cred *Credential) string {
	return base64.StdEncoding.EncodeToString([]byte(cred.APIKey))
}

func decodeCredential(blob string) (*Credential, bool) {
	data, err := base64.StdEncoding.DecodeString(blob)
	if err != nil {
		return nil, false
	}
	return &Credential{APIKey: string(data)}, true
}

func StoreCredential(name string, cred *Credential, userInfo *UserInfo) error {
	blob := encodeCredential(cred)

	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("%s: %w", i18n.T(i18n.Msg.Auth.LoadConfigFailed), err)
	}

	if storeInSecretService(name, blob) {
		delete(cfg, credentialsKey)
	} else {
		cfg[credentialsKey] = credToMap(cred)
	}

	if userInfo != nil {
		cfg[userInfoKey] = userInfoToMap(userInfo)
	}

	return config.Save(cfg)
}

// storeInSecretService writes the encoded credential to the Secret Service and
// reports whether it was accepted. A false result means the caller must keep the
// credential in the config file: either no session bus was detected, in which
// case the keyring is skipped outright, or the keyring write failed.
func storeInSecretService(name, blob string) bool {
	if !useSecretService() {
		return false
	}
	return credentialsBackend.Set(name, keyringUser, blob) == nil
}

func LoadCredential(name string) (*Credential, *UserInfo, error) {

	var cred *Credential
	if useSecretService() {
		if blob, err := credentialsBackend.Get(name, keyringUser); err == nil && blob != "" {
			if c, ok := decodeCredential(blob); ok {
				cred = c
			}
		}
	}

	cfgCred, cfgUserInfo := loadFromConfig()
	if cred == nil {
		cred = cfgCred
	}
	if cred == nil {
		return nil, nil, fmt.Errorf("%s", i18n.T(i18n.Msg.Auth.NoCredential))
	}

	return cred, cfgUserInfo, nil
}

func DeleteCredential(name string) error {
	if useSecretService() {
		_ = credentialsBackend.Delete(name, keyringUser)
	}
	cfg, err := config.Load()
	if err != nil {
		return err
	}
	delete(cfg, credentialsKey)
	delete(cfg, userInfoKey)
	return config.Save(cfg)
}

func credToMap(cred *Credential) map[string]any {
	return map[string]any{
		"api_key": cred.APIKey,
	}
}

func userInfoToMap(userInfo *UserInfo) map[string]any {
	return map[string]any{
		"user_name": userInfo.UserName,
		"user_id":   userInfo.UserID,
	}
}

func parseCredAndUserFromMap(cfg map[string]any) (*Credential, *UserInfo) {
	var cred *Credential
	if v, ok := cfg[credentialsKey]; ok {
		if m, ok := v.(map[string]any); ok {
			cred = &Credential{
				APIKey: getString(m, "api_key"),
			}
		}
	}
	var userInfo *UserInfo
	if v, ok := cfg[userInfoKey]; ok {
		if m, ok := v.(map[string]any); ok {
			userInfo = &UserInfo{
				UserName: getString(m, "user_name"),
				UserID:   getString(m, "user_id"),
			}
		}
	}
	return cred, userInfo
}

func getString(m map[string]any, key string) string {
	if v, ok := m[key]; ok {
		if s, ok := v.(string); ok {
			return s
		}
	}
	return ""
}

func loadFromConfig() (*Credential, *UserInfo) {
	cfg, err := config.Load()
	if err != nil {
		return nil, nil
	}
	return parseCredAndUserFromMap(cfg)
}
