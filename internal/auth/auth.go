// Package auth handles Yandex OAuth 2.0 for the YaD CLI.
//
// The client ID is embedded in the binary and is intentionally public -
// this is standard practice for open-source CLI tools (gh, heroku, etc.).
// The client secret is injected at build time via ldflags so it never
// appears in source code:
//
//	go build -ldflags "-X github.com/ilyabrin/yad/internal/auth.clientSecret=xxx" .
//
// Users who prefer to use their own registered application can set
// oauth.client_id and oauth.client_secret in ~/.yad/config.yaml.
package auth

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"
)

// ---- Build-time constants --------------------------------------------------

// clientID is the public identifier of the YaD application registered on
// https://oauth.yandex.ru. Safe to embed in open-source code.
var clientID = "d82340a941d148c4890a3d70297d639f"

// clientSecret is injected via ldflags at release build time.
// Empty in development builds - falls back to manual token entry.
var clientSecret = ""

// ---- OAuth endpoints -------------------------------------------------------

const (
	authorizeURL = "https://oauth.yandex.ru/authorize"
	tokenURL     = "https://oauth.yandex.ru/token"

	// redirectURI tells Yandex to display the auth code on screen instead of
	// redirecting to a URL - designed specifically for CLI / native apps.
	redirectURI = "https://oauth.yandex.ru/verification_code"
)

// ---- Public API ------------------------------------------------------------

// Config holds OAuth credentials. If non-empty values are provided they
// override the build-time defaults, allowing users to bring their own app.
type Config struct {
	ClientID     string // overrides build-time clientID when non-empty
	ClientSecret string // overrides build-time clientSecret when non-empty
}

// TokenResponse is the response from the Yandex token endpoint.
type TokenResponse struct {
	AccessToken  string `json:"access_token"`
	TokenType    string `json:"token_type"`
	ExpiresIn    int64  `json:"expires_in"` // seconds
	RefreshToken string `json:"refresh_token"`
	Scope        string `json:"scope"`
}

// ExpiresAt returns the absolute expiry time for the access token.
func (t *TokenResponse) ExpiresAt() time.Time {
	if t.ExpiresIn <= 0 {
		return time.Time{}
	}
	return time.Now().Add(time.Duration(t.ExpiresIn) * time.Second)
}

// AuthURL returns the URL the user must open to authorise the application.
// cfg may be nil to use the build-time defaults.
func AuthURL(cfg *Config) (string, error) {
	id := resolveClientID(cfg)
	if id == "" {
		return "", fmt.Errorf(
			"OAuth client ID is not configured.\n" +
				"Register your application at https://oauth.yandex.ru and either:\n" +
				"  • rebuild with -ldflags \"-X github.com/ilyabrin/yad/internal/auth.clientID=xxx\"\n" +
				"  • or set oauth.client_id in ~/.yad/config.yaml",
		)
	}

	// Build URL manually to keep it readable and avoid double-encoding
	u := fmt.Sprintf(
		"%s?response_type=code&client_id=%s&redirect_uri=%s",
		authorizeURL,
		url.QueryEscape(id),
		url.QueryEscape(redirectURI),
	)
	return u, nil
}

// ExchangeCode exchanges an authorization code for access + refresh tokens.
func ExchangeCode(ctx context.Context, code string, cfg *Config) (*TokenResponse, error) {
	id, secret, err := resolveCredentials(cfg)
	if err != nil {
		return nil, err
	}

	data := url.Values{}
	data.Set("grant_type", "authorization_code")
	data.Set("code", strings.TrimSpace(code))
	data.Set("client_id", id)
	data.Set("client_secret", secret)
	// redirect_uri is intentionally omitted for the verification_code flow:
	// Yandex treats it as an out-of-band display, not a real redirect,
	// and rejects the token request if redirect_uri is present.

	return postToken(ctx, data)
}

// RefreshAccessToken uses a refresh token to obtain a new access token.
func RefreshAccessToken(ctx context.Context, refreshToken string, cfg *Config) (*TokenResponse, error) {
	id, secret, err := resolveCredentials(cfg)
	if err != nil {
		return nil, err
	}

	data := url.Values{}
	data.Set("grant_type", "refresh_token")
	data.Set("refresh_token", refreshToken)
	data.Set("client_id", id)
	data.Set("client_secret", secret)

	return postToken(ctx, data)
}

// IsConfigured reports whether a client secret is available (either from
// the build-time ldflags injection or from a user-supplied config).
// When false, the app falls back to manual token paste.
func IsConfigured(cfg *Config) bool {
	_, secret, err := resolveCredentials(cfg)
	return err == nil && secret != ""
}

// ---- Internals -------------------------------------------------------------

func resolveClientID(cfg *Config) string {
	if cfg != nil && cfg.ClientID != "" {
		return cfg.ClientID
	}
	return clientID
}

func resolveCredentials(cfg *Config) (id, secret string, err error) {
	id = resolveClientID(cfg)
	secret = clientSecret
	if cfg != nil && cfg.ClientSecret != "" {
		secret = cfg.ClientSecret
	}

	if id == "" {
		return "", "", fmt.Errorf("OAuth client ID not configured (see internal/auth/auth.go)")
	}
	if secret == "" {
		return "", "", fmt.Errorf(
			"OAuth client secret not available.\n" +
				"Build with: -ldflags \"-X github.com/ilyabrin/yad/internal/auth.clientSecret=YOUR_SECRET\"\n" +
				"or set oauth.client_secret in ~/.yad/config.yaml",
		)
	}
	return id, secret, nil
}

func postToken(ctx context.Context, data url.Values) (*TokenResponse, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, tokenURL,
		strings.NewReader(data.Encode()))
	if err != nil {
		return nil, fmt.Errorf("building token request: %w", err)
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	client := &http.Client{Timeout: 15 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("token request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		var errBody struct {
			Error            string `json:"error"`
			ErrorDescription string `json:"error_description"`
		}
		_ = json.NewDecoder(resp.Body).Decode(&errBody)
		return nil, fmt.Errorf("token request rejected by Yandex (HTTP %d): %s - %s",
			resp.StatusCode, errBody.Error, errBody.ErrorDescription)
	}

	var token TokenResponse
	if err := json.NewDecoder(resp.Body).Decode(&token); err != nil {
		return nil, fmt.Errorf("decoding token response: %w", err)
	}
	return &token, nil
}
