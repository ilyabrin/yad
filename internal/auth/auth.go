// Package auth handles Yandex OAuth 2.0 for the YaD CLI.
//
// YaD authenticates as a public client using PKCE (RFC 7636), so no client
// secret is needed or stored anywhere. Each authorization generates a random
// verifier; only its SHA-256 challenge is sent to Yandex, and the verifier is
// presented when the code is redeemed. A code intercepted on its way back is
// useless without the verifier, which never leaves this process.
//
// That means every build authenticates identically, whether it came from a
// release archive or from "go install". The client ID is embedded and is
// intentionally public, as it is for other open-source CLI tools.
//
// Users who prefer their own registered application can set oauth.client_id
// (and oauth.client_secret, if their application requires one) in
// ~/.yad/config.yaml.
package auth

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/ilyabrin/yad/internal/i18n"
)

// ---- Build-time constants --------------------------------------------------

// clientID is the public identifier of the YaD application registered on
// https://oauth.yandex.ru. Safe to embed in open-source code.
var clientID = "d82340a941d148c4890a3d70297d639f"

// clientSecret is empty by default: PKCE replaces it. It remains settable so
// that users bringing their own confidential Yandex application can supply
// one through ~/.yad/config.yaml.
var clientSecret = ""

// ---- OAuth endpoints -------------------------------------------------------

// tokenURL is a var rather than a const so tests can point it at a stub
// server. Nothing outside this package changes it.
var tokenURL = "https://oauth.yandex.ru/token"

const (
	authorizeURL = "https://oauth.yandex.ru/authorize"

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
// The challenge comes from a PKCE pair whose verifier must be kept until the
// code is exchanged. cfg may be nil to use the built-in client ID.
func AuthURL(cfg *Config, challenge string) (string, error) {
	id := resolveClientID(cfg)
	if id == "" {
		return "", fmt.Errorf(
			"OAuth client ID is not configured.\n" +
				"Register your application at https://oauth.yandex.ru and either:\n" +
				"  • rebuild with -ldflags \"-X github.com/ilyabrin/yad/internal/auth.clientID=xxx\"\n" +
				"  • or set oauth.client_id in ~/.yad/config.yaml",
		)
	}

	if challenge == "" {
		return "", fmt.Errorf("PKCE challenge is required")
	}

	// Build URL manually to keep it readable and avoid double-encoding
	u := fmt.Sprintf(
		"%s?response_type=code&client_id=%s&redirect_uri=%s&code_challenge=%s&code_challenge_method=%s",
		authorizeURL,
		url.QueryEscape(id),
		url.QueryEscape(redirectURI),
		url.QueryEscape(challenge),
		challengeMethod,
	)
	return u, nil
}

// ExchangeCode exchanges an authorization code for access + refresh tokens.
//
// verifier is the secret half of the PKCE pair whose challenge was used to
// build the authorization URL. Yandex accepts the exchange without a client
// secret when a verifier is present, which is what makes this work for a
// public client.
func ExchangeCode(ctx context.Context, code, verifier string, cfg *Config) (*TokenResponse, error) {
	id := resolveClientID(cfg)
	if id == "" {
		return nil, errNoClientID
	}
	if verifier == "" {
		return nil, fmt.Errorf(
			"this authorization has expired.\n" +
				"Open the sign-in link again to start over",
		)
	}

	data := url.Values{}
	data.Set("grant_type", "authorization_code")
	data.Set("code", strings.TrimSpace(code))
	data.Set("client_id", id)
	data.Set("code_verifier", verifier)
	// A client secret is not required alongside code_verifier, but users who
	// registered a confidential application of their own may still have one.
	if secret := resolveClientSecret(cfg); secret != "" {
		data.Set("client_secret", secret)
	}
	// redirect_uri is intentionally omitted for the verification_code flow:
	// Yandex treats it as an out-of-band display, not a real redirect,
	// and rejects the token request if redirect_uri is present.

	return postToken(ctx, data)
}

// RefreshAccessToken uses a refresh token to obtain a new access token.
//
// Yandex documents the secret as optional only for the code exchange, so a
// refresh may still be refused for a public client. Callers should treat a
// failure here as "ask the user to sign in again" rather than as fatal;
// Yandex access tokens are valid for a year, so this is rare.
func RefreshAccessToken(ctx context.Context, refreshToken string, cfg *Config) (*TokenResponse, error) {
	id := resolveClientID(cfg)
	if id == "" {
		return nil, errNoClientID
	}

	data := url.Values{}
	data.Set("grant_type", "refresh_token")
	data.Set("refresh_token", refreshToken)
	data.Set("client_id", id)
	if secret := resolveClientSecret(cfg); secret != "" {
		data.Set("client_secret", secret)
	}

	return postToken(ctx, data)
}

// IsConfigured reports whether guided sign-in is available, which now needs
// only a client ID. It is true for every ordinary build; it goes false only
// if someone strips the built-in client ID and supplies none of their own,
// in which case the app falls back to pasting a token by hand.
func IsConfigured(cfg *Config) bool {
	return resolveClientID(cfg) != ""
}

// ---- Internals -------------------------------------------------------------

// errNoClientID is returned when no client ID is available at all, which can
// only happen in a build that stripped the built-in one.
var errNoClientID = errors.New(
	"OAuth client ID is not configured.\n" +
		"Register an application at https://oauth.yandex.ru and set " +
		"oauth.client_id in ~/.yad/config.yaml")

func resolveClientID(cfg *Config) string {
	if cfg != nil && cfg.ClientID != "" {
		return cfg.ClientID
	}
	return clientID
}

// resolveClientSecret returns the optional secret for users who registered a
// confidential application of their own. Empty for ordinary builds, where
// PKCE takes its place.
func resolveClientSecret(cfg *Config) string {
	if cfg != nil && cfg.ClientSecret != "" {
		return cfg.ClientSecret
	}
	return clientSecret
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
		return nil, fmt.Errorf(i18n.T("auth.token_request_rejected_by_yandex"),
			resp.StatusCode, errBody.Error, errBody.ErrorDescription)
	}

	var token TokenResponse
	if err := json.NewDecoder(resp.Body).Decode(&token); err != nil {
		return nil, fmt.Errorf("decoding token response: %w", err)
	}
	return &token, nil
}
