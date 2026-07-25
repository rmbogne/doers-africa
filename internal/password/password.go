package password

import (
	"fmt"
	"unicode/utf8"

	"golang.org/x/crypto/bcrypt"

)

const (
	MinimumLength = 12
	MaximumBytes  = 72
)

var (
	ErrTooShort = fmt.Errorf(
		"password must contain at least %d characters",
		MinimumLength,
	)

	ErrTooLong = fmt.Errorf(
		"password must not exceed %d bytes",
		MaximumBytes,
	)
)

func Validate(plainText string) error {
	if utf8.RuneCountInString(plainText) <
		MinimumLength {
		return ErrTooShort
	}

	if len([]byte(plainText)) > MaximumBytes {
		return ErrTooLong
	}

	return nil
}

func Hash(plainText string) (string, error) {
	if err := Validate(plainText); err != nil {
		return "", err
	}

	hashedPassword, err :=
		bcrypt.GenerateFromPassword(
			[]byte(plainText),
			bcrypt.DefaultCost,
		)
	if err != nil {
		return "", fmt.Errorf(
			"generate password hash: %w",
			err,
		)
	}

	return string(hashedPassword), nil
}

func Matches(
	passwordHash string,
	plainTextPassword string,
) bool {
	err := bcrypt.CompareHashAndPassword(
		[]byte(passwordHash),
		[]byte(plainTextPassword),
	)

	return err == nil
}
