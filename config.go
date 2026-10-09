package main

import (
	"fmt"
	"os"
	"path/filepath"
	"time"

	"gopkg.in/yaml.v3"
)

const (
	configDir  = ".yad"
	configFile = "config.yaml"

	// EnvToken is the environment variable that overrides stored credentials.
	// Useful for CI/CD and scripting: YANDEX_DISK_TOKEN=xxx yad ls /
	EnvToken = "YANDEX_DISK_TOKEN"
)

// OAuthConfig holds optional user-supplied OAuth credentials that override
// the build-time defaults. Set these in ~/.yad/config.yaml under "oauth:"
// if you want to use your own registered Yandex application.
type OAuthConfig struct {
	ClientID     string `yaml:"client_id,omitempty"`
	ClientSecret string `yaml:"client_secret,omitempty"`
}

// Config is persisted to ~/.yad/config.yaml.
type Config struct {
	// OAuth credentials (optional - only needed for your own application)
	OAuth OAuthConfig `yaml:"oauth,omitempty"`

	// Tokens obtained via OAuth flow
	AccessToken  string    `yaml:"access_token,omitempty"`
	RefreshToken string    `yaml:"refresh_token,omitempty"`
	TokenExpiry  time.Time `yaml:"token_expiry,omitempty"`

	// UI preferences
	DefaultSort string `yaml:"default_sort,omitempty"` // name, -name, modified, -modified, size, -size
	LastPath    string `yaml:"last_path,omitempty"`    // last visited directory; restored on next launch
	Language    string `yaml:"language,omitempty"`     // ru, en or auto (the default: from the system)
}

// validSorts is the set of values accepted by the Yandex Disk API sort parameter.
var validSorts = map[string]bool{
	"name": true, "-name": true,
	"modified": true, "-modified": true,
	"size": true, "-size": true,
}

// EffectiveSort returns the sort to use in the browser, falling back to "name".
func (c *Config) EffectiveSort() string {
	if validSorts[c.DefaultSort] {
		return c.DefaultSort
	}
	return "name"
}

// EffectiveToken returns the token to use when creating a disk.Client,
// in priority order:
//  1. YANDEX_DISK_TOKEN env var (highest - for CI/scripting)
//  2. Stored access token from OAuth flow
func (c *Config) EffectiveToken() string {
	if t := os.Getenv(EnvToken); t != "" {
		return t
	}
	return c.AccessToken
}

// IsTokenExpired reports whether the stored access token has expired.
// Returns false (not expired) when no expiry time is recorded - Yandex
// tokens are valid for one year by default.
func (c *Config) IsTokenExpired() bool {
	if c.TokenExpiry.IsZero() {
		return false
	}
	// Treat as expired 5 minutes early to avoid edge-case failures
	return time.Now().After(c.TokenExpiry.Add(-5 * time.Minute))
}

// HasToken reports whether any token is available (env var or stored).
func (c *Config) HasToken() bool {
	return c.EffectiveToken() != ""
}

// CanRefresh reports whether a refresh token is available to renew
// the access token without user interaction.
func (c *Config) CanRefresh() bool {
	return c.RefreshToken != "" && os.Getenv(EnvToken) == ""
}

// ---- Persistence -----------------------------------------------------------

func configPath() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("cannot determine home directory: %w", err)
	}
	return filepath.Join(home, configDir, configFile), nil
}

// LoadConfig reads ~/.yad/config.yaml.
// Returns an empty Config (not an error) when the file does not exist yet.
func LoadConfig() (*Config, error) {
	path, err := configPath()
	if err != nil {
		return nil, err
	}

	data, err := os.ReadFile(path)
	if os.IsNotExist(err) {
		return &Config{}, nil
	}
	if err != nil {
		return nil, fmt.Errorf("cannot read config: %w", err)
	}

	var cfg Config
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("cannot parse config: %w", err)
	}
	return &cfg, nil
}

// SaveConfig writes the config to ~/.yad/config.yaml (mode 0600).
func SaveConfig(cfg *Config) error {
	path, err := configPath()
	if err != nil {
		return err
	}

	if err := os.MkdirAll(filepath.Dir(path), 0700); err != nil {
		return fmt.Errorf("cannot create config directory: %w", err)
	}

	data, err := yaml.Marshal(cfg)
	if err != nil {
		return fmt.Errorf("cannot marshal config: %w", err)
	}

	// 0600 - readable only by the owner
	if err := os.WriteFile(path, data, 0600); err != nil {
		return fmt.Errorf("cannot write config: %w", err)
	}
	return nil
}
