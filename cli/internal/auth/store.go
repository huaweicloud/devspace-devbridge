package auth

import (
	"encoding/base64"
	"fmt"
	"strings"

	"huawei.com/devbridge/internal/config"

	"github.com/zalando/go-keyring"
)

const CredentialName = "HWCLOUD"

const (
	defaultTunnelKey = "default-tunnel-id"
	credPartsNum     = 2 // apikey:login_type
	credentialsKey   = "credentials"
	userInfoKey      = "user-info"
)

func encodeCredential(cred *Credential) string {
	parts := make([]string, credPartsNum)
	parts[0] = base64.StdEncoding.EncodeToString([]byte(cred.APIKey))
	parts[1] = base64.StdEncoding.EncodeToString([]byte(cred.LoginType))
	return strings.Join(parts, ":")
}

func decodeCredential(blob string) (*Credential, bool) {
	parts := strings.Split(blob, ":")
	// 兼容旧格式：4 段（apikey:login_type:account_namespace:namespace）
	// 5 段（apikey:expires_at:login_type:account_namespace:namespace）
	// 3 段（apikey:expires_at:login_type）或 2 段（apikey:expires_at / apikey:login_type）
	if len(parts) != credPartsNum && len(parts) != 5 && len(parts) != 4 && len(parts) != 3 && len(parts) != 2 {
		return nil, false
	}
	cred := &Credential{}
	var err error
	cred.APIKey, err = decodeBase64(parts[0])
	if err != nil {
		return nil, false
	}
	if len(parts) == credPartsNum {
		// 新格式：apikey:login_type
		cred.LoginType, err = decodeBase64(parts[1])
		if err != nil {
			return nil, false
		}
	} else {
		// 旧格式：expiresAt 位于 parts[1]
		cred.ExpiresAt, err = decodeBase64(parts[1])
		if err != nil {
			return nil, false
		}
		if len(parts) >= 3 {
			cred.LoginType, err = decodeBase64(parts[2])
			if err != nil {
				return nil, false
			}
		}
	}
	return cred, true
}

func decodeBase64(s string) (string, error) {
	data, err := base64.StdEncoding.DecodeString(s)
	if err != nil {
		return "", err
	}
	return string(data), nil
}

func StoreCredential(name string, cred *Credential, userInfo *UserInfo) error {
	blob := encodeCredential(cred)
	_ = keyring.Set(name, "Credentials", blob)

	cfg, _ := config.Load()
	if cfg == nil {
		cfg = make(map[string]any)
	}
	cfg[credentialsKey] = credToMap(cred)
	if userInfo != nil {
		cfg[userInfoKey] = userInfoToMap(userInfo)
	}
	return config.Save(cfg)
}

func LoadCredential(name string) (*Credential, *UserInfo, error) {
	blob, err := keyring.Get(name, "Credentials")
	if err == nil && blob != "" {
		if cred, ok := decodeCredential(blob); ok {
			userInfo := loadUserInfoFromConfig()
			return cred, userInfo, nil
		}
	}
	cred, userInfo := loadFromConfig()
	if cred == nil {
		return nil, nil, fmt.Errorf("no credential found")
	}
	return cred, userInfo, nil
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

func StoreDefaultTunnel(tunnelId string) error {
	_ = keyring.Set(CredentialName, defaultTunnelKey, tunnelId)
	return config.Set(defaultTunnelKey, tunnelId)
}

func LoadDefaultTunnel() (string, error) {
	if v, err := keyring.Get(CredentialName, defaultTunnelKey); err == nil && v != "" {
		return v, nil
	}
	if v, ok := config.Get(defaultTunnelKey); ok {
		if s, ok := v.(string); ok {
			return s, nil
		}
	}
	return "", fmt.Errorf("tunnel ID not specified and no default tunnel set, please specify via argument or use 'devbridge set' to set default")
}

func DeleteDefaultTunnel() error {
	_ = keyring.Delete(CredentialName, defaultTunnelKey)
	return config.Delete(defaultTunnelKey)
}

func credToMap(cred *Credential) map[string]any {
	return map[string]any{
		"api_key":    cred.APIKey,
		"login_type": cred.LoginType,
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
				APIKey:    getString(m, "api_key"),
				ExpiresAt: getString(m, "expires_at"),
				LoginType: getString(m, "login_type"),
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

func loadUserInfoFromConfig() *UserInfo {
	_, userInfo := loadFromConfig()
	return userInfo
}
