package auth

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"
)

// withTokenServer points the package at a stub token endpoint for one test and
// hands back the form values it received.
func withTokenServer(t *testing.T, handler http.HandlerFunc) *url.Values {
	t.Helper()

	got := &url.Values{}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if err := r.ParseForm(); err != nil {
			t.Errorf("parsing request form: %v", err)
		}
		*got = r.PostForm
		handler(w, r)
	}))
	t.Cleanup(srv.Close)

	original := tokenURL
	tokenURL = srv.URL
	t.Cleanup(func() { tokenURL = original })

	return got
}

func okToken(w http.ResponseWriter, _ *http.Request) {
	_ = json.NewEncoder(w).Encode(TokenResponse{
		AccessToken:  "access-123",
		RefreshToken: "refresh-456",
		ExpiresIn:    31536000,
		TokenType:    "bearer",
	})
}

func TestAuthURLCarriesTheChallenge(t *testing.T) {
	u, err := AuthURL(nil, "test-challenge")
	if err != nil {
		t.Fatal(err)
	}

	parsed, err := url.Parse(u)
	if err != nil {
		t.Fatal(err)
	}
	q := parsed.Query()

	for field, want := range map[string]string{
		"response_type":         "code",
		"client_id":             clientID,
		"redirect_uri":          redirectURI,
		"code_challenge":        "test-challenge",
		"code_challenge_method": "S256",
	} {
		if got := q.Get(field); got != want {
			t.Errorf("%s = %q, want %q", field, got, want)
		}
	}
}

func TestAuthURLRequiresAChallenge(t *testing.T) {
	if _, err := AuthURL(nil, ""); err == nil {
		t.Error("expected a missing challenge to be rejected")
	}
}

func TestAuthURLHonoursACustomClientID(t *testing.T) {
	u, err := AuthURL(&Config{ClientID: "my-app"}, "c")
	if err != nil {
		t.Fatal(err)
	}
	if got := mustQuery(t, u).Get("client_id"); got != "my-app" {
		t.Errorf("client_id = %q, want the configured one", got)
	}
}

func TestExchangeCodeSendsVerifierAndNoSecret(t *testing.T) {
	form := withTokenServer(t, okToken)

	resp, err := ExchangeCode(context.Background(), "  the-code  ", "the-verifier", nil)
	if err != nil {
		t.Fatal(err)
	}
	if resp.AccessToken != "access-123" {
		t.Errorf("access token = %q", resp.AccessToken)
	}

	if got := form.Get("grant_type"); got != "authorization_code" {
		t.Errorf("grant_type = %q", got)
	}
	if got := form.Get("code"); got != "the-code" {
		t.Errorf("code = %q, want it trimmed", got)
	}
	if got := form.Get("code_verifier"); got != "the-verifier" {
		t.Errorf("code_verifier = %q", got)
	}
	if _, present := (*form)["client_secret"]; present {
		t.Error("a public client must not send client_secret")
	}
}

// Users who registered a confidential application of their own keep working.
func TestExchangeCodeSendsAConfiguredSecret(t *testing.T) {
	form := withTokenServer(t, okToken)

	if _, err := ExchangeCode(context.Background(), "c", "v", &Config{ClientSecret: "shh"}); err != nil {
		t.Fatal(err)
	}
	if got := form.Get("client_secret"); got != "shh" {
		t.Errorf("client_secret = %q, want the configured one", got)
	}
}

func TestExchangeCodeRequiresAVerifier(t *testing.T) {
	err := mustFail(t, func() error {
		_, e := ExchangeCode(context.Background(), "code", "", nil)
		return e
	})
	if !strings.Contains(err.Error(), "expired") {
		t.Errorf("expected a message about starting over, got %q", err)
	}
}

func TestRefreshAccessTokenSendsNoSecret(t *testing.T) {
	form := withTokenServer(t, okToken)

	if _, err := RefreshAccessToken(context.Background(), "refresh-456", nil); err != nil {
		t.Fatal(err)
	}
	if got := form.Get("grant_type"); got != "refresh_token" {
		t.Errorf("grant_type = %q", got)
	}
	if got := form.Get("refresh_token"); got != "refresh-456" {
		t.Errorf("refresh_token = %q", got)
	}
	if _, present := (*form)["client_secret"]; present {
		t.Error("a public client must not send client_secret")
	}
}

func TestPostTokenSurfacesTheServerError(t *testing.T) {
	withTokenServer(t, func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(map[string]string{
			"error":             "invalid_grant",
			"error_description": "Code has expired",
		})
	})

	err := mustFail(t, func() error {
		_, e := ExchangeCode(context.Background(), "code", "verifier", nil)
		return e
	})
	for _, want := range []string{"invalid_grant", "Code has expired"} {
		if !strings.Contains(err.Error(), want) {
			t.Errorf("expected %q in %q", want, err)
		}
	}
	// ST1005: the message is shown in the TUI and must read as a sentence.
	if first := err.Error()[:1]; first == strings.ToUpper(first) && first != strings.ToLower(first) {
		t.Errorf("error should not start capitalised: %q", err)
	}
}

func TestIsConfiguredNeedsOnlyAClientID(t *testing.T) {
	if !IsConfigured(nil) {
		t.Error("the built-in client ID should be enough for guided sign-in")
	}

	original := clientID
	clientID = ""
	t.Cleanup(func() { clientID = original })

	if IsConfigured(nil) {
		t.Error("without any client ID, guided sign-in is unavailable")
	}
	if !IsConfigured(&Config{ClientID: "mine"}) {
		t.Error("a user-supplied client ID should re-enable guided sign-in")
	}
}

func TestExpiresAt(t *testing.T) {
	if got := (&TokenResponse{ExpiresIn: 0}).ExpiresAt(); !got.IsZero() {
		t.Errorf("no expiry information should give the zero time, got %v", got)
	}

	got := (&TokenResponse{ExpiresIn: 3600}).ExpiresAt()
	if delta := time.Until(got); delta < 59*time.Minute || delta > 61*time.Minute {
		t.Errorf("expiry is %v away, want about an hour", delta)
	}
}

// --- helpers ---

func mustQuery(t *testing.T, raw string) url.Values {
	t.Helper()
	parsed, err := url.Parse(raw)
	if err != nil {
		t.Fatal(err)
	}
	return parsed.Query()
}

func mustFail(t *testing.T, fn func() error) error {
	t.Helper()
	err := fn()
	if err == nil {
		t.Fatal("expected an error")
	}
	return err
}
