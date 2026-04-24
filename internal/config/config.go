package config

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"time"

	"github.com/BurntSushi/toml"
)

type Config struct {
	BaseURL   string `toml:"base_url"`
	ClientID  string `toml:"client_id"`
	SecretKey string `toml:"secret_key"`
}

type TokenCache struct {
	Token     string    `toml:"token"`
	ExpiresAt time.Time `toml:"expires_at"`
}

func Dir() string {
	if xdg := os.Getenv("XDG_CONFIG_HOME"); xdg != "" {
		return filepath.Join(xdg, "telex")
	}
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".config", "telex")
}

func ConfigPath() string { return filepath.Join(Dir(), "config.toml") }

func TokenPath() string { return filepath.Join(Dir(), "token.toml") }

func Paths(override string) (string, string) {
	if override == "" {
		return ConfigPath(), TokenPath()
	}
	cleaned := filepath.Clean(override)
	return cleaned, filepath.Join(filepath.Dir(cleaned), "token.toml")
}

func LoadFrom(path string) (*Config, error) {
	var cfg Config
	if _, err := toml.DecodeFile(path, &cfg); err != nil {
		return nil, fmt.Errorf("loading config from %s: %w", path, err)
	}
	return &cfg, nil
}

func (c *Config) Validate() error {
	if c.BaseURL == "" {
		return fmt.Errorf("base_url is required")
	}
	if c.ClientID == "" {
		return fmt.Errorf("client_id is required")
	}
	if c.SecretKey == "" {
		return fmt.Errorf("secret_key is required")
	}
	return nil
}

func (c *Config) SaveTo(path string) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return err
	}
	f, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0o600)
	if err != nil {
		return err
	}
	defer closeSilently(f)
	return toml.NewEncoder(f).Encode(c)
}

func LoadTokenFrom(path string) (*TokenCache, error) {
	var tc TokenCache
	if _, err := toml.DecodeFile(path, &tc); err != nil {
		return nil, err
	}
	return &tc, nil
}

func SaveTokenTo(path string, tc *TokenCache) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return err
	}
	f, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0o600)
	if err != nil {
		return err
	}
	defer closeSilently(f)
	return toml.NewEncoder(f).Encode(tc)
}

func (tc *TokenCache) Valid() bool {
	return tc != nil && tc.Token != "" && time.Until(tc.ExpiresAt) > 60*time.Second
}

func closeSilently(closer io.Closer) { _ = closer.Close() }
