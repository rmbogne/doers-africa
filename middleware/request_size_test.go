package middleware

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestRequestSizeLimitsRejectsStandardForm(t *testing.T) {
	handler := RequestSizeLimits(RequestSizeConfig{StandardMaxBytes: 16, UploadMaxBytes: 100}, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Fatal("handler should not execute")
	}))
	request := httptest.NewRequest(http.MethodPost, "/login", strings.NewReader(strings.Repeat("x", 17)))
	recorder := httptest.NewRecorder()
	handler.ServeHTTP(recorder, request)
	if recorder.Code != http.StatusRequestEntityTooLarge {
		t.Fatalf("expected %d, got %d", http.StatusRequestEntityTooLarge, recorder.Code)
	}
}

func TestRequestSizeLimitsRedirectsUploadForm(t *testing.T) {
	handler := RequestSizeLimits(RequestSizeConfig{StandardMaxBytes: 16, UploadMaxBytes: 32}, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Fatal("handler should not execute")
	}))
	request := httptest.NewRequest(http.MethodPost, "/doer/service/edit?id=507f1f77bcf86cd799439011", strings.NewReader(strings.Repeat("x", 33)))
	request.Header.Set("Content-Type", "multipart/form-data; boundary=test")
	recorder := httptest.NewRecorder()
	handler.ServeHTTP(recorder, request)
	if recorder.Code != http.StatusSeeOther {
		t.Fatalf("expected %d, got %d", http.StatusSeeOther, recorder.Code)
	}
	expected := "/doer/service/edit?id=507f1f77bcf86cd799439011&upload_error=too_large"
	if recorder.Header().Get("Location") != expected {
		t.Fatalf("unexpected redirect: %s", recorder.Header().Get("Location"))
	}
}
