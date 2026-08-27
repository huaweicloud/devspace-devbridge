package config

import (
	"errors"
	"os"
	"path/filepath"
	"sync"

	"gopkg.in/yaml.v3"
)

var DefaultServerDomain = "https://relay-dev-local.tailb4159e.ts.net:8443"

var ErrKeyNotFound = errors.New("key not found")

var (
	configMu sync.Mutex
)

func configPath() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, ".huawei", "devbridge", "config.yaml"), nil
}

func Load() (map[string]any, error) {
	path, err := configPath()
	if err != nil {
		return nil, err
	}
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return make(map[string]any), nil
		}
		return nil, err
	}
	var cfg map[string]any
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, err
	}
	if cfg == nil {
		cfg = make(map[string]any)
	}
	return cfg, nil
}

func Save(cfg map[string]any) error {
	configMu.Lock()
	defer configMu.Unlock()

	path, err := configPath()
	if err != nil {
		return err
	}
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return err
	}
	if err := os.Chmod(dir, 0o700); err != nil {
		return err
	}

	if len(cfg) == 0 {
		return os.WriteFile(path, nil, 0o600)
	}
	data, err := yaml.Marshal(cfg)
	if err != nil {
		return err
	}
	if err := os.WriteFile(path, data, 0o600); err != nil {
		return err
	}
	return os.Chmod(path, 0o600)
}

func Get(key string) (any, bool) {
	cfg, err := Load()
	if err != nil {
		return nil, false
	}
	v, ok := cfg[key]
	return v, ok
}

func Set(key string, value any) error {
	cfg, err := Load()
	if err != nil {
		return err
	}
	cfg[key] = value
	return Save(cfg)
}

func Delete(key string) error {
	cfg, err := Load()
	if err != nil {
		return err
	}
	if _, ok := cfg[key]; !ok {
		return ErrKeyNotFound
	}
	delete(cfg, key)
	return Save(cfg)
}
