package main

import (
	"os"
	"testing"
	"time"
)

// --- EffectiveToken ---

func TestEffectiveToken_EnvVarTakesPriority(t *testing.T) {
	t.Setenv(EnvToken, "env-token")
	cfg := &Config{AccessToken: "stored-token"}

	if got := cfg.EffectiveToken(); got != "env-token" {
		t.Errorf("EffectiveToken() = %q, want env-token", got)
	}
}

func TestEffectiveToken_StoredTokenWhenNoEnv(t *testing.T) {
	t.Setenv(EnvToken, "")
	cfg := &Config{AccessToken: "stored-token"}

	if got := cfg.EffectiveToken(); got != "stored-token" {
		t.Errorf("EffectiveToken() = %q, want stored-token", got)
	}
}

func TestEffectiveToken_EmptyWhenNeitherSet(t *testing.T) {
	t.Setenv(EnvToken, "")
	cfg := &Config{}

	if got := cfg.EffectiveToken(); got != "" {
		t.Errorf("EffectiveToken() = %q, want empty", got)
	}
}

// --- HasToken ---

func TestHasToken_TrueWithEnvVar(t *testing.T) {
	t.Setenv(EnvToken, "env-token")
	cfg := &Config{}

	if !cfg.HasToken() {
		t.Error("HasToken() = false, want true when env var is set")
	}
}

func TestHasToken_TrueWithStoredToken(t *testing.T) {
	t.Setenv(EnvToken, "")
	cfg := &Config{AccessToken: "stored"}

	if !cfg.HasToken() {
		t.Error("HasToken() = false, want true with stored token")
	}
}

func TestHasToken_FalseWhenEmpty(t *testing.T) {
	t.Setenv(EnvToken, "")
	cfg := &Config{}

	if cfg.HasToken() {
		t.Error("HasToken() = true, want false when no token")
	}
}

// --- IsTokenExpired ---

func TestIsTokenExpired_ZeroTimeNotExpired(t *testing.T) {
	cfg := &Config{} // TokenExpiry is zero value

	if cfg.IsTokenExpired() {
		t.Error("IsTokenExpired() = true, want false for zero expiry")
	}
}

func TestIsTokenExpired_FutureNotExpired(t *testing.T) {
	cfg := &Config{TokenExpiry: time.Now().Add(time.Hour)}

	if cfg.IsTokenExpired() {
		t.Error("IsTokenExpired() = true, want false for token expiring in 1h")
	}
}

func TestIsTokenExpired_PastExpired(t *testing.T) {
	cfg := &Config{TokenExpiry: time.Now().Add(-time.Hour)}

	if !cfg.IsTokenExpired() {
		t.Error("IsTokenExpired() = false, want true for token expired 1h ago")
	}
}

func TestIsTokenExpired_WithinGracePeriodExpired(t *testing.T) {
	// Token expires in 3 minutes — within the 5-minute grace window
	cfg := &Config{TokenExpiry: time.Now().Add(3 * time.Minute)}

	if !cfg.IsTokenExpired() {
		t.Error("IsTokenExpired() = false, want true within 5-min grace period")
	}
}

func TestIsTokenExpired_OutsideGracePeriodNotExpired(t *testing.T) {
	// Token expires in 10 minutes — outside the 5-minute grace window
	cfg := &Config{TokenExpiry: time.Now().Add(10 * time.Minute)}

	if cfg.IsTokenExpired() {
		t.Error("IsTokenExpired() = true, want false with 10 minutes remaining")
	}
}

// --- CanRefresh ---

func TestCanRefresh_TrueWithRefreshTokenAndNoEnv(t *testing.T) {
	t.Setenv(EnvToken, "")
	cfg := &Config{RefreshToken: "refresh-token"}

	if !cfg.CanRefresh() {
		t.Error("CanRefresh() = false, want true")
	}
}

func TestCanRefresh_FalseWhenEnvVarSet(t *testing.T) {
	t.Setenv(EnvToken, "env-token")
	cfg := &Config{RefreshToken: "refresh-token"}

	// Env var tokens are external — we can't refresh them
	if cfg.CanRefresh() {
		t.Error("CanRefresh() = true, want false when env var token is set")
	}
}

func TestCanRefresh_FalseWithNoRefreshToken(t *testing.T) {
	t.Setenv(EnvToken, "")
	cfg := &Config{}

	if cfg.CanRefresh() {
		t.Error("CanRefresh() = true, want false with no refresh token")
	}
}

// --- toAuthConfig ---

func TestToAuthConfig_NilWhenBothEmpty(t *testing.T) {
	cfg := &Config{}

	if got := toAuthConfig(cfg); got != nil {
		t.Errorf("toAuthConfig() = %+v, want nil", got)
	}
}

func TestToAuthConfig_ReturnsConfigWhenClientIDSet(t *testing.T) {
	cfg := &Config{OAuth: OAuthConfig{ClientID: "my-id"}}

	got := toAuthConfig(cfg)
	if got == nil {
		t.Fatal("toAuthConfig() = nil, want non-nil")
	}
	if got.ClientID != "my-id" {
		t.Errorf("ClientID = %q, want my-id", got.ClientID)
	}
}

func TestToAuthConfig_ReturnsConfigWhenClientSecretSet(t *testing.T) {
	cfg := &Config{OAuth: OAuthConfig{ClientSecret: "my-secret"}}

	got := toAuthConfig(cfg)
	if got == nil {
		t.Fatal("toAuthConfig() = nil, want non-nil")
	}
	if got.ClientSecret != "my-secret" {
		t.Errorf("ClientSecret = %q, want my-secret", got.ClientSecret)
	}
}

func TestToAuthConfig_BothFieldsPreserved(t *testing.T) {
	cfg := &Config{OAuth: OAuthConfig{ClientID: "id", ClientSecret: "secret"}}

	got := toAuthConfig(cfg)
	if got == nil {
		t.Fatal("toAuthConfig() = nil, want non-nil")
	}
	if got.ClientID != "id" || got.ClientSecret != "secret" {
		t.Errorf("toAuthConfig() = {%q, %q}, want {id, secret}", got.ClientID, got.ClientSecret)
	}
}

// toAuthConfig is tested above; AuthConfig() method was removed as it was
// unused — yad.go calls toAuthConfig() directly.

// --- LoadConfig / SaveConfig (filesystem round-trip) ---

func TestLoadConfig_ReturnsEmptyWhenFileAbsent(t *testing.T) {
	// Point home to a temp dir with no config file
	tmp := t.TempDir()
	t.Setenv("HOME", tmp)
	t.Setenv("USERPROFILE", tmp) // Windows

	cfg, err := LoadConfig()
	if err != nil {
		t.Fatalf("LoadConfig() error = %v", err)
	}
	if cfg == nil {
		t.Fatal("LoadConfig() = nil, want empty Config")
	}
	if cfg.AccessToken != "" {
		t.Errorf("AccessToken = %q, want empty", cfg.AccessToken)
	}
}

func TestSaveAndLoadConfig_RoundTrip(t *testing.T) {
	tmp := t.TempDir()
	t.Setenv("HOME", tmp)
	t.Setenv("USERPROFILE", tmp)

	expiry := time.Now().Truncate(time.Second).UTC()
	original := &Config{
		AccessToken:  "access-abc",
		RefreshToken: "refresh-xyz",
		TokenExpiry:  expiry,
		OAuth:        OAuthConfig{ClientID: "cid", ClientSecret: "csecret"},
	}

	if err := SaveConfig(original); err != nil {
		t.Fatalf("SaveConfig() error = %v", err)
	}

	loaded, err := LoadConfig()
	if err != nil {
		t.Fatalf("LoadConfig() error = %v", err)
	}

	if loaded.AccessToken != original.AccessToken {
		t.Errorf("AccessToken = %q, want %q", loaded.AccessToken, original.AccessToken)
	}
	if loaded.RefreshToken != original.RefreshToken {
		t.Errorf("RefreshToken = %q, want %q", loaded.RefreshToken, original.RefreshToken)
	}
	if !loaded.TokenExpiry.Equal(original.TokenExpiry) {
		t.Errorf("TokenExpiry = %v, want %v", loaded.TokenExpiry, original.TokenExpiry)
	}
	if loaded.OAuth.ClientID != original.OAuth.ClientID {
		t.Errorf("OAuth.ClientID = %q, want %q", loaded.OAuth.ClientID, original.OAuth.ClientID)
	}
}

func TestSaveConfig_FileMode(t *testing.T) {
	if os.Getenv("GOOS") == "windows" || isWindows() {
		t.Skip("Unix file permission bits are not enforced on Windows")
	}

	tmp := t.TempDir()
	t.Setenv("HOME", tmp)
	t.Setenv("USERPROFILE", tmp)

	if err := SaveConfig(&Config{AccessToken: "tok"}); err != nil {
		t.Fatalf("SaveConfig() error = %v", err)
	}

	path, _ := configPath()
	info, err := os.Stat(path)
	if err != nil {
		t.Fatalf("Stat() error = %v", err)
	}
	if perm := info.Mode().Perm(); perm != 0600 {
		t.Errorf("file permissions = %o, want 0600", perm)
	}
}

func isWindows() bool {
	// runtime.GOOS is a compile-time constant; use a file write to detect
	// whether Unix permission bits are honoured at runtime.
	f, err := os.CreateTemp("", "permcheck-*")
	if err != nil {
		return false
	}
	name := f.Name()
	f.Close()
	defer os.Remove(name)

	_ = os.Chmod(name, 0600)
	info, err := os.Stat(name)
	if err != nil {
		return false
	}
	return info.Mode().Perm() != 0600
}
