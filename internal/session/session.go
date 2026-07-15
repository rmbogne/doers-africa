package session

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"fmt"
)

const tokenSizeBytes = 32

// NewToken creates a cryptographically secure session token.
//
// rawToken is sent to the browser.
// tokenHash is stored in PostgreSQL.
func NewToken() (
	rawToken string,
	tokenHash string,
	err error,
) {
	randomBytes := make([]byte, tokenSizeBytes)

	if _, err := rand.Read(randomBytes); err != nil {
		return "", "", fmt.Errorf(
			"generate session token: %w",
			err,
		)
	}

	rawToken = base64.RawURLEncoding.EncodeToString(
		randomBytes,
	)

	tokenHash = Hash(rawToken)

	return rawToken, tokenHash, nil
}

// Hash creates the value stored in PostgreSQL.
func Hash(rawToken string) string {
	sum := sha256.Sum256([]byte(rawToken))

	return hex.EncodeToString(sum[:])
}
