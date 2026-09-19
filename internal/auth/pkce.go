package auth

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"fmt"
)

// verifierBytes is the amount of entropy behind a code verifier. RFC 7636
// requires the encoded verifier to be 43 to 128 characters; 32 raw bytes
// encode to exactly 43, the length the RFC recommends.
const verifierBytes = 32

// PKCE holds one authorization attempt's proof-of-possession pair.
//
// The Challenge travels to Yandex in the authorization URL, while the Verifier
// stays in memory and is presented when the code is exchanged. Only the holder
// of the Verifier can redeem a code issued for the Challenge, which is what
// lets a public client such as this one authenticate without a client secret.
type PKCE struct {
	// Verifier is the secret half. It must not be logged or written to disk.
	Verifier string
	// Challenge is SHA-256 of the Verifier, base64url encoded without padding.
	Challenge string
}

// NewPKCE generates a fresh verifier and its challenge.
func NewPKCE() (*PKCE, error) {
	raw := make([]byte, verifierBytes)
	if _, err := rand.Read(raw); err != nil {
		return nil, fmt.Errorf("generating PKCE verifier: %w", err)
	}

	verifier := base64.RawURLEncoding.EncodeToString(raw)
	sum := sha256.Sum256([]byte(verifier))

	return &PKCE{
		Verifier:  verifier,
		Challenge: base64.RawURLEncoding.EncodeToString(sum[:]),
	}, nil
}

// challengeMethod is the only transformation worth using. Yandex also accepts
// "plain", which offers no protection at all.
const challengeMethod = "S256"
