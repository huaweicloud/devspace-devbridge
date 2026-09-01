package auth

import (
	"encoding/base64"
	"fmt"

	"huawei.com/devbridge/internal/config"

	"github.com/zalando/go-keyring"
)

const CredentialName = "HWCLOUD"

const (
	credentialsKey = "credentials"
	userInfoKey    = "user-info"
)

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

	cfg, _ := config.Load()
	if cfg == nil {
		cfg = make(map[string]any)
	}

	if err := keyring.Set(name, "Credentials", blob); err == nil {
		delete(cfg, credentialsKey)
	} else {
		cfg[credentialsKey] = credToMap(cred)
	}

	if userInfo != nil {
		cfg[userInfoKey] = userInfoToMap(userInfo)
	}

	return config.Save(cfg)
}

func LoadCredential(name string) (*Credential, *UserInfo, error) {

	var cred *Credential
	if blob, err := keyring.Get(name, "Credentials"); err == nil && blob != "" {
		if c, ok := decodeCredential(blob); ok {
			cred = c
		}
	}

	cfgCred, cfgUserInfo := loadFromConfig()
	if cred == nil {
		cred = cfgCred
	}
	if cred == nil {
		return nil, nil, fmt.Errorf("no credential found")
	}

	return cred, cfgUserInfo, nil
}

func DeleteCredential(name string) error {
	_ = keyring.Delete(name, "Credentials")
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
