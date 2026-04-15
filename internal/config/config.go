package config

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/viper"
)

var configKeys = []string{
	"dokploy_url",
	"dokploy_api_key",
	"hetzner_dns_token",
	"github_org",
	"domain",
	"dokploy_server_ip",
}

func Init() error {
	home, err := os.UserHomeDir()
	if err != nil {
		return fmt.Errorf("get home dir: %w", err)
	}

	configDir := filepath.Join(home, ".monolinie")
	if err := os.MkdirAll(configDir, 0755); err != nil {
		return fmt.Errorf("create config dir: %w", err)
	}

	configFile := filepath.Join(configDir, "config.yaml")
	if _, err := os.Stat(configFile); os.IsNotExist(err) {
		if err := os.WriteFile(configFile, []byte(""), 0644); err != nil {
			return fmt.Errorf("create config file: %w", err)
		}
	}

	viper.SetConfigFile(configFile)
	viper.SetDefault("github_org", "monolinie")
	viper.SetDefault("domain", "monolinie.com")

	return viper.ReadInConfig()
}

func Set(key, value string) error {
	if !isValidKey(key) {
		return fmt.Errorf("unknown config key: %s\nValid keys: %v", key, configKeys)
	}
	viper.Set(key, value)
	return viper.WriteConfig()
}

func Get(key string) string {
	return viper.GetString(key)
}

func AllSettings() map[string]any {
	return viper.AllSettings()
}

func ValidKeys() []string {
	return configKeys
}

func isValidKey(key string) bool {
	for _, k := range configKeys {
		if k == key {
			return true
		}
	}
	return false
}

// Validate checks that all required config keys are set.
func Validate() error {
	required := []string{"dokploy_url", "dokploy_api_key", "hetzner_dns_token", "dokploy_server_ip"}
	for _, key := range required {
		if Get(key) == "" {
			return fmt.Errorf("missing required config: %s\nRun: monolinie config set %s <value>", key, key)
		}
	}
	return nil
}
