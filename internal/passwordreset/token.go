package passwordreset

import (
	"crypto/rand"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"fmt"
	"time"
	"strings"
)

type PasswordResetToken struct {
	ID        int64
	UserID    int64
	Role      string
	TokenHash string
	ExpiresAt time.Time
	UsedAt    *time.Time
	CreatedAt time.Time
}

const (
	// tokenSizeBytes controls the amount of cryptographically secure random data used to generate a password-reset token.
	//
	// 32 bytes provides 256 bits of entropy, which is sufficient for a one-time password-reset token.
	tokenSizeBytes = 32
	// tokenHashLength is the number of hexadecimal characters produced by a SHA-256 hash.
	//
	// SHA-256 produces 32 bytes, and hexadecimal encoding represents each byte using two characters:
	tokenHashLength = sha256.Size * 2
)

var (
	ErrEmptyToken = errors.New("password reset token must not be empty")
	ErrInvalidToken = errors.New("password reset token is invalid")
	ErrInvalidTokenHash = errors.New("password reset token hash is invalid")
)
// Generate cryptographically secure random bytes.
// Return URL-safe encoded token.
func NewToken() (string, error) {
	randomBytes := make(
		[]byte,
		tokenSizeBytes,
	)

	if _, err := rand.Read(randomBytes); err != nil {
		return "", fmt.Errorf(
			"generate password reset token: %w",
			err,
		)
	}

	rawToken := base64.RawURLEncoding.EncodeToString(
		randomBytes,
	)

	return rawToken, nil
}

// HashToken creates a SHA-256 hexadecimal hash of a raw password-reset token.
//
// Only this hash should be stored in the database. When a user submits a raw
// token, hash it again and look up the resulting value.
func HashToken(token string) (string, error) {
	token = strings.TrimSpace(token)

	if token == "" {
		return "", ErrEmptyToken
	}

	sum := sha256.Sum256(
		[]byte(token),
	)

	return hex.EncodeToString(
		sum[:],
	), nil
}

// ValidateToken verifies that a raw token has the expected format.
//
// It confirms that:
//
// - The token is not empty
// - The token is valid unpadded URL-safe Base64
// - The token decodes to exactly tokenSizeBytes bytes
//
// This function does not determine whether the token exists in the database,
// has expired, or has already been used. Those checks belong in the store
// layer.
func ValidateTokenHash(tokenHash string) error {
	tokenHash = strings.TrimSpace(tokenHash)

	if len(tokenHash) != tokenHashLength {
		return ErrInvalidTokenHash
	}

	decodedHash, err :=
		hex.DecodeString(tokenHash)
	if err != nil {
		return ErrInvalidTokenHash
	}

	if len(decodedHash) != sha256.Size {
		return ErrInvalidTokenHash
	}

	return nil
}

// ValidateToken verifies that a raw password-reset token has the expected
// format and length.
//
// It checks that the token:
//   - is not empty
//   - uses unpadded URL-safe Base64 encoding
//   - decodes to exactly tokenSizeBytes bytes
//
// This function does not verify whether the token exists in the database,
// has expired, or has already been used.
func ValidateToken(token string) error {
	token = strings.TrimSpace(token)

	if token == "" {
		return ErrEmptyToken
	}

	decodedToken, err :=
		base64.RawURLEncoding.DecodeString(token)
	if err != nil {
		return ErrInvalidToken
	}

	if len(decodedToken) != tokenSizeBytes {
		return ErrInvalidToken
	}

	return nil
}

// Matches reports whether a raw token corresponds to a stored SHA-256 token
// hash.
//
// For the normal reset flow, the preferred approach is:
//
//  1. Hash the submitted raw token
//  2. Query the database using the hash
//
// Matches is useful when direct comparison is necessary, such as in unit
// tests or when a token record has already been loaded.
//
// Constant-time comparison is used to reduce timing side channels.
func Matches(
	rawToken string,
	storedTokenHash string,
) bool {
	if err := ValidateToken(rawToken); err != nil {
		return false
	}

	if err := ValidateTokenHash(
		storedTokenHash,
	); err != nil {
		return false
	}

	calculatedHash, err := HashToken(rawToken)
	if err != nil {
		return false
	}

	return subtle.ConstantTimeCompare(
		[]byte(calculatedHash),
		[]byte(
			strings.ToLower(
				storedTokenHash,
			),
		),
	) == 1
}

