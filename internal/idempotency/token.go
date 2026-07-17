package idempotency

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"errors"
)

const tokenByteLength = 32

var ErrInvalidToken = errors.New("invalid idempotency token")

// Generate creates a cryptographically secure, URL-safe token suitable for a
// hidden HTML form field. Only the token hash should be persisted.
func Generate() (string, error) {
	tokenBytes := make([]byte, tokenByteLength)

	if _, err := rand.Read(tokenBytes); err != nil {
		return "", err
	}

	return base64.RawURLEncoding.EncodeToString(
		tokenBytes,
	), nil
}

// Hash validates the token's canonical encoding and returns its SHA-256 hash.
func Hash(rawToken string) (string, error) {
	decodedToken, err :=
		base64.RawURLEncoding.DecodeString(rawToken)
	if err != nil ||
		len(decodedToken) != tokenByteLength {
		return "", ErrInvalidToken
	}

	if base64.RawURLEncoding.EncodeToString(
		decodedToken,
	) != rawToken {
		return "", ErrInvalidToken
	}

	tokenHash := sha256.Sum256(decodedToken)

	return hex.EncodeToString(tokenHash[:]), nil
}
