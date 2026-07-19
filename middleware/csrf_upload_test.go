package middleware

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestCSRFOversizedMultipartRedirectsToForm(
	t *testing.T,
) {
	configureTestCSRF(t)

	maximumBytes, _ := csrfMultipartLimits()

	handler := CSRF(
		http.HandlerFunc(
			func(
				w http.ResponseWriter,
				r *http.Request,
			) {
				t.Fatal(
					"protected handler should not execute",
				)
			},
		),
	)

	request := httptest.NewRequest(
		http.MethodPost,
		"/doer/service/new",
		strings.NewReader("body"),
	)

	request.Header.Set(
		"Content-Type",
		"multipart/form-data; boundary=test",
	)

	request.ContentLength =
		maximumBytes + 1

	recorder := httptest.NewRecorder()

	handler.ServeHTTP(
		recorder,
		request,
	)

	if recorder.Code != http.StatusSeeOther {
		t.Fatalf(
			"expected %d, got %d",
			http.StatusSeeOther,
			recorder.Code,
		)
	}

	location := recorder.Header().Get(
		"Location",
	)

	expected :=
		"/doer/service/new?upload_error=too_large"

	if location != expected {
		t.Fatalf(
			"expected %q, got %q",
			expected,
			location,
		)
	}
}

func TestMultipartErrorRedirectPreservesObjectID(
	t *testing.T,
) {
	request := httptest.NewRequest(
		http.MethodPost,
		"/doer/event/edit?id=507f1f77bcf86cd799439011",
		nil,
	)
	recorder := httptest.NewRecorder()

	redirectMultipartFormError(
		recorder,
		request,
		"too_large",
	)

	location := recorder.Header().Get("Location")

	expected :=
		"/doer/event/edit?" +
			"id=507f1f77bcf86cd799439011&" +
			"upload_error=too_large"

	if location != expected {
		t.Fatalf(
			"expected %q, got %q",
			expected,
			location,
		)
	}
}
