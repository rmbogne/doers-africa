package middleware

import (
	"net/http/httptest"
	"os"
	"testing"
)

func TestCSRFCookieIsNotSecureOnHTTPDevelopmentHosts(
	t *testing.T,
) {
	originalValue := os.Getenv(
		"CSRF_COOKIE_SECURE",
	)
	t.Cleanup(
		func() {
			_ = os.Setenv(
				"CSRF_COOKIE_SECURE",
				originalValue,
			)
		},
	)

	if err := os.Setenv(
		"CSRF_COOKIE_SECURE",
		"true",
	); err != nil {
		t.Fatal(err)
	}

	testCases := []string{
		"http://localhost:8080/login",
		"http://127.0.0.1:8080/login",
		"http://[::1]:8080/login",
	}

	for _, target := range testCases {
		request := httptest.NewRequest(
			"GET",
			target,
			nil,
		)

		if csrfCookieSecure(request) {
			t.Fatalf(
				"expected non-Secure cookie for %s",
				target,
			)
		}
	}
}

func TestCSRFCookieIsSecureBehindHTTPSProxy(
	t *testing.T,
) {
	request := httptest.NewRequest(
		"GET",
		"http://app.example.com/login",
		nil,
	)
	request.Header.Set(
		"X-Forwarded-Proto",
		"https",
	)

	if !csrfCookieSecure(request) {
		t.Fatal(
			"expected Secure cookie behind HTTPS proxy",
		)
	}
}
