package password

import (
	"fmt"
	"unicode/utf8"

	"golang.org/x/crypto/bcrypt"
)

const (
	minimumPasswordLength = 12
	maximumPasswordBytes  = 72
)

var (
	ErrTooShort = fmt.Errorf(
		"password must contain at least %d characters",
		minimumPasswordLength,
	)
	ErrTooLong = fmt.Errorf(
		"password must not exceed %d bytes",
		maximumPasswordBytes,
	)
)

// Hash validates and hashes a plaintext password with bcrypt.
func Hash(plainText string) (string, error) {
	if utf8.RuneCountInString(plainText) < minimumPasswordLength {
		return "", ErrTooShort
	}

	// bcrypt rejects passwords longer than 72 bytes.
	if len([]byte(plainText)) > maximumPasswordBytes {
		return "", ErrTooLong
	}

	hashedPassword, err := bcrypt.GenerateFromPassword(
		[]byte(plainText),
		bcrypt.DefaultCost,
	)
	if err != nil {
		return "", fmt.Errorf("generate password hash: %w", err)
	}

	return string(hashedPassword), nil
}

// Matches reports whether a plaintext password matches a bcrypt hash.
func Matches(passwordHash, plainTextPassword string) bool {
	if passwordHash == "" || plainTextPassword == "" {
		return false
	}

	return bcrypt.CompareHashAndPassword(
		[]byte(passwordHash),
		[]byte(plainTextPassword),
	) == nil
}
