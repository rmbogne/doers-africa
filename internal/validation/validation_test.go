package validation

import (
	"strings"
	"testing"
	"time"
)

func TestSingleLineRejectsNewline(t *testing.T) {
	if _, err := SingleLine("Title", "Valid\nInjected", 3, 150); err == nil {
		t.Fatal("expected newline to be rejected")
	}
}

func TestMultilineAllowsNewline(t *testing.T) {
	value, err := Multiline("Description", "First line\nSecond line", 1, 100)
	if err != nil {
		t.Fatalf("Multiline returned error: %v", err)
	}
	if !strings.Contains(value, "\n") {
		t.Fatal("expected newline to be preserved")
	}
}

func TestEmailNormalizesCase(t *testing.T) {
	value, err := Email(" USER@Example.COM ")
	if err != nil {
		t.Fatalf("Email returned error: %v", err)
	}
	if value != "user@example.com" {
		t.Fatalf("unexpected normalized email: %s", value)
	}
}

func TestDateTodayOrFuture(t *testing.T) {
	now := time.Date(2026, time.July, 19, 12, 0, 0, 0, time.UTC)
	if _, err := DateTodayOrFuture("Date", "2026-07-18", 5, now); err == nil {
		t.Fatal("expected past date to fail")
	}
	if _, err := DateTodayOrFuture("Date", "2026-07-20", 5, now); err != nil {
		t.Fatalf("expected future date to pass: %v", err)
	}
}

func TestIntegerRange(t *testing.T) {
	if _, err := Integer("Price", "1000001", 0, 1000000); err == nil {
		t.Fatal("expected out-of-range integer to fail")
	}
}

func TestOptionalURL(t *testing.T) {
	if _, err := OptionalURL("Website", "javascript:alert(1)"); err == nil {
		t.Fatal("expected unsafe URL scheme to fail")
	}
}
