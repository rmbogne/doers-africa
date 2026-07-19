package middleware

import (
	"strings"
	"testing"
	"time"
)

func TestConfigureCSRF(
	t *testing.T,
) {
	err := ConfigureCSRF(
		CSRFConfig{
			Secret: strings.Repeat(
				"x",
				32,
			),
			CookieSecure:         false,
			MaxAge:               12 * time.Hour,
			MultipartMaxBytes:    12 << 20,
			MultipartMemoryBytes: 2 << 20,
		},
	)

	if err != nil {
		t.Fatalf(
			"ConfigureCSRF returned an error: %v",
			err,
		)
	}

	if !csrfConfigured() {
		t.Fatal(
			"expected CSRF to be configured",
		)
	}
}

func TestConfigureCSRFRejectsShortSecret(
	t *testing.T,
) {
	err := ConfigureCSRF(
		CSRFConfig{
			Secret:               "short",
			MaxAge:               time.Hour,
			MultipartMaxBytes:    12 << 20,
			MultipartMemoryBytes: 2 << 20,
		},
	)

	if err == nil {
		t.Fatal(
			"expected short CSRF secret to fail",
		)
	}
}
