package middleware

import (
	"net/http"
	"strings"
)

const (
	DefaultStandardRequestBodyMaxBytes int64 = 64 << 10
	DefaultUploadRequestBodyMaxBytes   int64 = 3 << 20
)

type RequestSizeConfig struct {
	StandardMaxBytes int64
	UploadMaxBytes   int64
}

// RequestSizeLimits must wrap CSRF so the body is bounded before CSRF parses it.
func RequestSizeLimits(configuration RequestSizeConfig, next http.Handler) http.Handler {
	standardMaximum := configuration.StandardMaxBytes
	if standardMaximum <= 0 {
		standardMaximum = DefaultStandardRequestBodyMaxBytes
	}
	uploadMaximum := configuration.UploadMaxBytes
	if uploadMaximum <= 0 {
		uploadMaximum = DefaultUploadRequestBodyMaxBytes
	}

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !isStateChangingMethod(r.Method) || r.Body == nil {
			next.ServeHTTP(w, r)
			return
		}

		maximumBytes := standardMaximum
		uploadRequest := isUploadRequest(r)
		if uploadRequest {
			maximumBytes = uploadMaximum
		}

		if r.ContentLength > maximumBytes {
			rejectOversizedRequest(w, r, uploadRequest)
			return
		}

		r.Body = http.MaxBytesReader(w, r.Body, maximumBytes)
		next.ServeHTTP(w, r)
	})
}

func isUploadRequest(r *http.Request) bool {
	if !strings.HasPrefix(strings.ToLower(r.Header.Get("Content-Type")), "multipart/form-data") {
		return false
	}
	path := strings.TrimSuffix(r.URL.Path, "/")

	switch path {
	case "/register", "/doer/event/new", "/doer/event/edit", "/doer/service/new", "/doer/service/edit":
		return true
	}

	return strings.HasPrefix(path, "/doer/event/edit/") ||
		strings.HasPrefix(path, "/doer/service/edit/")
}

func rejectOversizedRequest(w http.ResponseWriter, r *http.Request, uploadRequest bool) {
	if uploadRequest {
		queryValues := r.URL.Query()
		queryValues.Set("upload_error", "too_large")
		redirectURL := r.URL.Path
		if encodedQuery := queryValues.Encode(); encodedQuery != "" {
			redirectURL += "?" + encodedQuery
		}
		http.Redirect(w, r, redirectURL, http.StatusSeeOther)
		return
	}
	http.Error(w, "Request body is too large", http.StatusRequestEntityTooLarge)
}
