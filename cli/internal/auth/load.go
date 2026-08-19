package auth

import "os"

func loadFromEnv() *Credential {
	apiKey := os.Getenv(envHWAPIKey)
	if apiKey == "" {
		return nil
	}
	return &Credential{
		APIKey: apiKey,
	}
}

func loadFromConfigFile() *Credential {
	cred, _ := loadFromConfig()
	return cred
}
