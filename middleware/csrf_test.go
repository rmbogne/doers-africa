package middleware

import (
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"
)

func configureTestCSRF(t *testing.T) {
	t.Helper()

	err := ConfigureCSRF(
		CSRFConfig{
			Secret: strings.Repeat(
				"t",
				48,
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
}

func TestCSRFAcceptsValidPOST(t *testing.T) {
	configureTestCSRF(t)

	handler := CSRF(
		http.HandlerFunc(
			func(
				w http.ResponseWriter,
				r *http.Request,
			) {
				w.WriteHeader(
					http.StatusNoContent,
				)
			},
		),
	)

	getRecorder := httptest.NewRecorder()
	getRequest := httptest.NewRequest(
		http.MethodGet,
		"/form",
		nil,
	)

	handler.ServeHTTP(
		getRecorder,
		getRequest,
	)

	cookies := getRecorder.Result().Cookies()
	if len(cookies) == 0 {
		t.Fatal("expected CSRF cookie")
	}

	token := cookies[0].Value

	form := url.Values{}
	form.Set(CSRFFieldName, token)

	postRequest := httptest.NewRequest(
		http.MethodPost,
		"/submit",
		strings.NewReader(form.Encode()),
	)
	postRequest.Header.Set(
		"Content-Type",
		"application/x-www-form-urlencoded",
	)
	postRequest.AddCookie(cookies[0])

	postRecorder := httptest.NewRecorder()

	handler.ServeHTTP(
		postRecorder,
		postRequest,
	)

	if postRecorder.Code !=
		http.StatusNoContent {
		t.Fatalf(
			"expected %d, got %d",
			http.StatusNoContent,
			postRecorder.Code,
		)
	}
}

func TestCSRFRejectsMissingToken(t *testing.T) {
	configureTestCSRF(t)

	handler := CSRF(
		http.HandlerFunc(
			func(
				w http.ResponseWriter,
				r *http.Request,
			) {
				w.WriteHeader(
					http.StatusNoContent,
				)
			},
		),
	)

	getRecorder := httptest.NewRecorder()

	handler.ServeHTTP(
		getRecorder,
		httptest.NewRequest(
			http.MethodGet,
			"/form",
			nil,
		),
	)

	cookie := getRecorder.Result().Cookies()[0]

	postRequest := httptest.NewRequest(
		http.MethodPost,
		"/submit",
		nil,
	)
	postRequest.AddCookie(cookie)

	postRecorder := httptest.NewRecorder()

	handler.ServeHTTP(
		postRecorder,
		postRequest,
	)

	if postRecorder.Code !=
		http.StatusForbidden {
		t.Fatalf(
			"expected %d, got %d",
			http.StatusForbidden,
			postRecorder.Code,
		)
	}
}

func TestCSRFCookieIsNotSecureOnLocalhost(
	t *testing.T,
) {
	configureTestCSRF(t)

	request := httptest.NewRequest(
		http.MethodGet,
		"http://localhost:8080/login",
		nil,
	)

	if csrfCookieSecure(request) {
		t.Fatal(
			"expected non-Secure cookie on localhost HTTP",
		)
	}
}

func TestCSRFCookieUsesConfiguredProductionSetting(
	t *testing.T,
) {
	err := ConfigureCSRF(
		CSRFConfig{
			Secret: strings.Repeat(
				"p",
				48,
			),
			CookieSecure:         true,
			MaxAge:               12 * time.Hour,
			MultipartMaxBytes:    12 << 20,
			MultipartMemoryBytes: 2 << 20,
		},
	)
	if err != nil {
		t.Fatal(err)
	}

	request := httptest.NewRequest(
		http.MethodGet,
		"http://app.example.com/login",
		nil,
	)

	if !csrfCookieSecure(request) {
		t.Fatal(
			"expected Secure cookie for configured production host",
		)
	}
}
