package main

import (
	"context"
	"fmt"
	"os"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/ilyabrin/disk"
	"github.com/ilyabrin/yad/internal/auth"
	"github.com/ilyabrin/yad/tui"
)

// version is injected at build time via:
//
//	go build -ldflags "-X main.version=v0.1.0"
var version = "dev"

const helpText = `yad — Yandex.Disk terminal UI

Usage:
  yad [flag]

Flags:
  -v, --version   print version and exit
  -h, --help      print this help and exit

Config:
  ~/.yad/config.yaml   tokens and UI preferences

Environment:
  YANDEX_DISK_TOKEN   override stored token (useful for CI/scripting)

Keybindings (inside the app):
  ↑/k  ↓/j          navigate        u  upload file
  ↵/→/l              open dir        U  upload from URL
  ←/h/Backspace      go up           d  download
  Space              select          D  delete
  Ctrl+A             select all      n  new directory
  /                  filter by name  s  cycle sort
  Esc                clear filter    r  rename
  p                  publish         i  disk info
  c                  copy URL        t  trash
  o                  open in browser q  quit

Source: https://github.com/ilyabrin/yad`

func main() {
	if len(os.Args) == 2 {
		switch os.Args[1] {
		case "--version", "-v":
			fmt.Println("yad", version)
			return
		case "--help", "-h":
			fmt.Println(helpText)
			return
		}
	}
	if err := run(); err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}
}

func run() error {
	cfg, err := LoadConfig()
	if err != nil {
		return fmt.Errorf("loading config: %w", err)
	}

	// Build auth.Config from any user-supplied OAuth credentials in config.
	oauthCfg := toAuthConfig(cfg)

	// Try to get a ready-to-use disk client without showing the setup screen.
	client, err := resolveClient(cfg, oauthCfg)
	if err != nil {
		// Non-fatal: log and proceed to setup screen
		fmt.Fprintf(os.Stderr, "warning: %v\n", err)
	}

	var refreshFn func() (string, error)
	if cfg.CanRefresh() {
		refreshFn = func() (string, error) {
			return refreshToken(cfg, oauthCfg)
		}
	}

	app := tui.New(client, oauthCfg, cfg.EffectiveSort(), cfg.LastPath, refreshFn)
	p := tea.NewProgram(app, tea.WithAltScreen())

	finalModel, err := p.Run()
	if err != nil {
		return fmt.Errorf("program error: %w", err)
	}

	// Persist session state (tokens + last visited path)
	if a, ok := finalModel.(*tui.App); ok {
		changed := false
		if result := a.TokenResult(); result != nil {
			cfg.AccessToken = result.AccessToken
			cfg.RefreshToken = result.RefreshToken
			cfg.TokenExpiry = result.Expiry
			changed = true
		}
		if p := a.LastPath(); p != "" && p != cfg.LastPath {
			cfg.LastPath = p
			changed = true
		}
		if changed {
			if saveErr := SaveConfig(cfg); saveErr != nil {
				fmt.Fprintf(os.Stderr, "warning: could not save config: %v\n", saveErr)
			}
		}
	}

	return nil
}

// resolveClient attempts to build a disk.Client from stored credentials,
// refreshing the access token if it has expired.
// Returns (nil, nil) when no token is available - the TUI setup screen handles it.
func resolveClient(cfg *Config, oauthCfg *auth.Config) (*disk.Client, error) {
	token := cfg.EffectiveToken()
	if token == "" {
		return nil, nil // no token yet → show setup
	}

	// Refresh expired token silently if we have a refresh token
	if cfg.IsTokenExpired() && cfg.CanRefresh() {
		refreshed, err := refreshToken(cfg, oauthCfg)
		if err != nil {
			// Refresh failed - fall through with the old token and let
			// the API call fail with a proper error message in the TUI
			fmt.Fprintf(os.Stderr, "warning: token refresh failed: %v\n", err)
		} else {
			token = refreshed
		}
	}

	client, err := disk.New(token)
	if err != nil {
		return nil, fmt.Errorf("creating disk client: %w", err)
	}
	return client, nil
}

// refreshToken exchanges the stored refresh token for a new access token
// and persists the updated credentials to config immediately.
func refreshToken(cfg *Config, oauthCfg *auth.Config) (string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	resp, err := auth.RefreshAccessToken(ctx, cfg.RefreshToken, oauthCfg)
	if err != nil {
		return "", err
	}

	cfg.AccessToken = resp.AccessToken
	cfg.TokenExpiry = resp.ExpiresAt()
	if resp.RefreshToken != "" {
		cfg.RefreshToken = resp.RefreshToken
	}

	if saveErr := SaveConfig(cfg); saveErr != nil {
		fmt.Fprintf(os.Stderr, "warning: could not persist refreshed token: %v\n", saveErr)
	}

	return resp.AccessToken, nil
}

// toAuthConfig converts user-supplied config credentials to auth.Config.
// Returns nil when the config has no OAuth overrides.
func toAuthConfig(cfg *Config) *auth.Config {
	if cfg.OAuth.ClientID == "" && cfg.OAuth.ClientSecret == "" {
		return nil
	}
	return &auth.Config{
		ClientID:     cfg.OAuth.ClientID,
		ClientSecret: cfg.OAuth.ClientSecret,
	}
}
