package auth

import (
	"crypto/sha256"
	"encoding/base64"
	"regexp"
	"testing"
)

// RFC 7636 section 4.1: the verifier is 43 to 128 characters drawn from
// [A-Za-z0-9-._~].
var verifierCharset = regexp.MustCompile(`^[A-Za-z0-9\-._~]+$`)

func TestNewPKCEProducesAValidVerifier(t *testing.T) {
	p, err := NewPKCE()
	if err != nil {
		t.Fatal(err)
	}

	if n := len(p.Verifier); n < 43 || n > 128 {
		t.Errorf("verifier length %d is outside the 43-128 range RFC 7636 requires", n)
	}
	if !verifierCharset.MatchString(p.Verifier) {
		t.Errorf("verifier contains characters outside the allowed set: %q", p.Verifier)
	}
}

func TestNewPKCEChallengeIsSHA256OfVerifier(t *testing.T) {
	p, err := NewPKCE()
	if err != nil {
		t.Fatal(err)
	}

	sum := sha256.Sum256([]byte(p.Verifier))
	want := base64.RawURLEncoding.EncodeToString(sum[:])
	if p.Challenge != want {
		t.Errorf("challenge = %q, want %q", p.Challenge, want)
	}

	// Padding would be rejected by the server.
	if got := p.Challenge; got != "" && got[len(got)-1] == '=' {
		t.Error("challenge must be base64url without padding")
	}
}

func TestNewPKCEIsUniquePerCall(t *testing.T) {
	seen := make(map[string]bool, 100)
	for i := 0; i < 100; i++ {
		p, err := NewPKCE()
		if err != nil {
			t.Fatal(err)
		}
		if seen[p.Verifier] {
			t.Fatalf("verifier repeated after %d calls", i)
		}
		seen[p.Verifier] = true
	}
}

func TestChallengeMethodIsS256(t *testing.T) {
	if challengeMethod != "S256" {
		t.Errorf("challenge method = %q; plain offers no protection", challengeMethod)
	}
}
