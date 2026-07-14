package password

import (
	"errors"
	"strings"
	"testing"
)

func TestHashAndMatches(t *testing.T) {
	plainText := "Correct-Horse-2026!"

	hash, err := Hash(plainText)
	if err != nil {
		t.Fatalf("Hash returned an error: %v", err)
	}

	if hash == plainText {
		t.Fatal("stored password must not equal plaintext password")
	}

	if !Matches(hash, plainText) {
		t.Fatal("expected password to match its hash")
	}

	if Matches(hash, "Wrong-Password-2026!") {
		t.Fatal("wrong password must not match")
	}
}

func TestHashUsesDifferentSalt(t *testing.T) {
	plainText := "Correct-Horse-2026!"

	firstHash, err := Hash(plainText)
	if err != nil {
		t.Fatalf("first Hash returned an error: %v", err)
	}

	secondHash, err := Hash(plainText)
	if err != nil {
		t.Fatalf("second Hash returned an error: %v", err)
	}

	if firstHash == secondHash {
		t.Fatal("two hashes of the same password should differ")
	}
}

func TestHashRejectsShortPassword(t *testing.T) {
	_, err := Hash("short")

	if !errors.Is(err, ErrTooShort) {
		t.Fatalf("expected ErrTooShort, received %v", err)
	}
}

func TestHashRejectsPasswordLongerThan72Bytes(t *testing.T) {
	_, err := Hash(strings.Repeat("a", 73))

	if !errors.Is(err, ErrTooLong) {
		t.Fatalf("expected ErrTooLong, received %v", err)
	}
}
