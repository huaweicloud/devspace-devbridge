package auth

import (
	"encoding/base64"
	"errors"
	"fmt"
	"strings"

	"huawei.com/devbridge/internal/config"

	"github.com/zalando/go-keyring"
)

const CredentialName = "HWCLOUD"

const (
	defaultTunnelKey = "default-tunnel-id"
	credentialsKey   = "credentials"
	userInfoKey      = "user-info"
)

func encodeCredential(cred *Credential) string {
	return base64.StdEncoding.EncodeToString([]byte(cred.APIKey))
}

func decodeCredential(blob string) (*Credential, bool) {
	parts := strings.Split(blob, ":")
	// 兼容旧格式：多段（apikey:login_type:...），取第一段作为 API Key.
	apiKey, err := decodeBase64(parts[0])
	if err != nil {
		return nil, false
	}
	return &Credential{APIKey: apiKey}, true
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

	cfg, _ := config.Load() //nolint:errcheck // 加载失败时使用空 map 兜底
	if cfg == nil {
		cfg = make(map[string]any)
	}

	// 凭证：keyring 优先，失败降级配置文件.
	if err := keyring.Set(name, "Credentials", blob); err == nil {
		delete(cfg, credentialsKey) // 清掉残留旧明文.
	} else {
		cfg[credentialsKey] = credToMap(cred)
	}

	// User_info：直接写入配置文件（不走 keyring/保险箱）
	if userInfo != nil {
		cfg[userInfoKey] = userInfoToMap(userInfo)
	}

	return config.Save(cfg)
}

func LoadCredential(name string) (*Credential, *UserInfo, error) {
	// 凭证：keyring 优先，配置文件兜底.
	var cred *Credential
	if blob, err := keyring.Get(name, "Credentials"); err == nil && blob != "" {
		if c, ok := decodeCredential(blob); ok {
			cred = c
		}
	}
	// 配置文件兜底（cred 和 userInfo 一起读）.
	cfgCred, cfgUserInfo := loadFromConfig()
	if cred == nil {
		cred = cfgCred
	}
	if cred == nil {
		return nil, nil, fmt.Errorf("no credential found")
	}

	// User_info：仅从配置文件读取
	return cred, cfgUserInfo, nil
}

func DeleteCredential(name string) error {
	_ = keyring.Delete(name, "Credentials") //nolint:errcheck // keyring 中不存在时删除失败属正常
	cfg, err := config.Load()
	if err != nil {
		return err
	}
	delete(cfg, credentialsKey)
	delete(cfg, userInfoKey)
	return config.Save(cfg)
}

// StoreDefaultTunnel 将默认隧道 ID 直接写入配置文件（不走 keyring/保险箱）。
func StoreDefaultTunnel(tunnelID string) error {
	return config.Set(defaultTunnelKey, tunnelID)
}

// LoadDefaultTunnel 从配置文件读取默认隧道 ID。
func LoadDefaultTunnel() (string, error) {
	if v, ok := config.Get(defaultTunnelKey); ok {
		if s, ok := v.(string); ok {
			return s, nil
		}
	}
	return "", fmt.Errorf("tunnel ID not specified and no default tunnel set, " +
		"please specify via argument or use 'devbridge set' to set default") //nolint:lll // 错误信息拆行后可读性更差
}

// DeleteDefaultTunnel 从配置文件删除默认隧道 ID。
// 没设过默认隧道（key 不存在）视为已删除，不返回错误。
func DeleteDefaultTunnel() error {
	if err := config.Delete(defaultTunnelKey); err != nil && !errors.Is(err, config.ErrKeyNotFound) {
		return err
	}
	return nil
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
