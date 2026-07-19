package middleware

import (
	"context"
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/base64"
	"errors"
	"fmt"
	"log"
	"net"
	"net/http"
	"strings"
	"sync"
	"time"
)

const (
	csrfCookieName = "aos_csrf"
	CSRFFieldName  = "csrf_token"
	CSRFHeaderName = "X-CSRF-Token"

	csrfTokenBytes = 32
)

type CSRFConfig struct {
	Secret               string
	CookieSecure         bool
	MaxAge               time.Duration
	MultipartMaxBytes    int64
	MultipartMemoryBytes int64
}

type csrfContextKey struct{}

var errCSRFMulitpartTooLarge = errors.New(
	"multipart form exceeds request limit",
)

var csrfRuntimeConfig struct {
	sync.RWMutex
	configured           bool
	secret               []byte
	cookieSecure         bool
	maxAge               time.Duration
	multipartMaxBytes    int64
	multipartMemoryBytes int64
}

// ConfigureCSRF must be called once during application startup using validated
// environment configuration.
func ConfigureCSRF(
	configuration CSRFConfig,
) error {
	if len(strings.TrimSpace(
		configuration.Secret,
	)) < 32 {
		return fmt.Errorf(
			"CSRF secret must contain at least 32 characters",
		)
	}

	if configuration.MaxAge <= 0 {
		return fmt.Errorf(
			"CSRF max age must be positive",
		)
	}

	if configuration.MultipartMaxBytes <= 0 {
		return fmt.Errorf(
			"CSRF multipart maximum must be positive",
		)
	}

	if configuration.MultipartMemoryBytes <= 0 ||
		configuration.MultipartMemoryBytes >
			configuration.MultipartMaxBytes {
		return fmt.Errorf(
			"invalid CSRF multipart memory limit",
		)
	}

	csrfRuntimeConfig.Lock()
	defer csrfRuntimeConfig.Unlock()

	csrfRuntimeConfig.secret = []byte(
		configuration.Secret,
	)
	csrfRuntimeConfig.cookieSecure =
		configuration.CookieSecure
	csrfRuntimeConfig.maxAge =
		configuration.MaxAge
	csrfRuntimeConfig.multipartMaxBytes =
		configuration.MultipartMaxBytes
	csrfRuntimeConfig.multipartMemoryBytes =
		configuration.MultipartMemoryBytes
	csrfRuntimeConfig.configured = true

	return nil
}

func CSRF(next http.Handler) http.Handler {
	return http.HandlerFunc(
		func(
			w http.ResponseWriter,
			r *http.Request,
		) {
			if !csrfConfigured() {
				http.Error(
					w,
					"Application security is not configured",
					http.StatusInternalServerError,
				)
				return
			}

			token, validCookie :=
				readCSRFCookie(r)

			if !validCookie {
				var err error

				token, err = newCSRFToken()
				if err != nil {
					log.Printf(
						"CSRF token generation error: %v",
						err,
					)
					http.Error(
						w,
						"Unable to initialize request security",
						http.StatusInternalServerError,
					)
					return
				}

				setCSRFCookie(w, r, token)
			}

			requestContext := context.WithValue(
				r.Context(),
				csrfContextKey{},
				token,
			)
			r = r.WithContext(requestContext)

			if isStateChangingMethod(r.Method) {
				submittedToken, err := submittedCSRFToken(
					w,
					r,
				)
				if err != nil {
					var maximumBytesError *http.MaxBytesError

					if errors.As(
						err,
						&maximumBytesError,
					) {
						http.Error(
							w,
							"Request body is too large",
							http.StatusRequestEntityTooLarge,
						)
						return
					}

					if errors.Is(
						err,
						errCSRFMulitpartTooLarge,
					) {
						redirectMultipartFormError(
							w,
							r,
							"too_large",
						)
						return
					}

					http.Error(
						w,
						"Invalid form submission",
						http.StatusBadRequest,
					)
					return
				}

				if !validSubmittedCSRFToken(
					token,
					submittedToken,
				) {
					http.Error(
						w,
						"Invalid or missing CSRF token",
						http.StatusForbidden,
					)
					return
				}
			}

			next.ServeHTTP(w, r)
		},
	)
}

func CSRFToken(r *http.Request) string {
	token, _ := r.Context().
		Value(csrfContextKey{}).(string)

	return token
}

func submittedCSRFToken(
	w http.ResponseWriter,
	r *http.Request,
) (string, error) {
	headerToken := strings.TrimSpace(
		r.Header.Get(CSRFHeaderName),
	)
	if headerToken != "" {
		return headerToken, nil
	}

	contentType := strings.ToLower(
		r.Header.Get("Content-Type"),
	)

	if strings.HasPrefix(
		contentType,
		"multipart/form-data",
	) {
		maximumBytes,
			memoryBytes :=
			csrfMultipartLimits()

		if r.ContentLength > maximumBytes {
			return "",
				errCSRFMulitpartTooLarge
		}

		r.Body = http.MaxBytesReader(
			w,
			r.Body,
			maximumBytes,
		)

		if err := r.ParseMultipartForm(
			memoryBytes,
		); err != nil {
			var maximumBytesError *http.MaxBytesError

			if errors.As(
				err,
				&maximumBytesError,
			) {
				return "",
					errCSRFMulitpartTooLarge
			}

			return "", err
		}
	} else if err := r.ParseForm(); err != nil {
		return "", err
	}

	return strings.TrimSpace(
		r.Form.Get(CSRFFieldName),
	), nil
}

func redirectMultipartFormError(
	w http.ResponseWriter,
	r *http.Request,
	errorCode string,
) {
	queryValues := r.URL.Query()
	queryValues.Set(
		"upload_error",
		errorCode,
	)

	redirectURL := r.URL.Path

	if encodedQuery := queryValues.Encode(); encodedQuery != "" {
		redirectURL += "?" + encodedQuery
	}

	http.Redirect(
		w,
		r,
		redirectURL,
		http.StatusSeeOther,
	)
}

func readCSRFCookie(
	r *http.Request,
) (string, bool) {
	cookie, err := r.Cookie(csrfCookieName)
	if err != nil {
		return "", false
	}

	token := strings.TrimSpace(cookie.Value)

	return token, verifySignedCSRFToken(token)
}

func newCSRFToken() (string, error) {
	randomBytes := make([]byte, csrfTokenBytes)

	if _, err := rand.Read(randomBytes); err != nil {
		return "", err
	}

	randomToken :=
		base64.RawURLEncoding.EncodeToString(
			randomBytes,
		)

	signature := signCSRFValue(randomToken)

	return randomToken + "." + signature, nil
}

func verifySignedCSRFToken(token string) bool {
	parts := strings.Split(token, ".")
	if len(parts) != 2 {
		return false
	}

	randomToken := parts[0]
	providedSignature := parts[1]

	decodedToken, err :=
		base64.RawURLEncoding.DecodeString(
			randomToken,
		)
	if err != nil ||
		len(decodedToken) != csrfTokenBytes {
		return false
	}

	expectedSignature :=
		signCSRFValue(randomToken)

	return subtle.ConstantTimeCompare(
		[]byte(providedSignature),
		[]byte(expectedSignature),
	) == 1
}

func validSubmittedCSRFToken(
	cookieToken string,
	submittedToken string,
) bool {
	if !verifySignedCSRFToken(cookieToken) ||
		!verifySignedCSRFToken(submittedToken) {
		return false
	}

	return subtle.ConstantTimeCompare(
		[]byte(cookieToken),
		[]byte(submittedToken),
	) == 1
}

func signCSRFValue(value string) string {
	mac := hmac.New(
		sha256.New,
		csrfSecret(),
	)

	_, _ = mac.Write([]byte(value))

	return base64.RawURLEncoding.EncodeToString(
		mac.Sum(nil),
	)
}

func csrfSecret() []byte {
	csrfRuntimeConfig.RLock()
	defer csrfRuntimeConfig.RUnlock()

	return csrfRuntimeConfig.secret
}

func setCSRFCookie(
	w http.ResponseWriter,
	r *http.Request,
	token string,
) {
	maximumAge := csrfMaxAge()

	http.SetCookie(
		w,
		&http.Cookie{
			Name:     csrfCookieName,
			Value:    token,
			Path:     "/",
			MaxAge:   int(maximumAge.Seconds()),
			Expires:  time.Now().Add(maximumAge),
			HttpOnly: true,
			Secure:   csrfCookieSecure(r),
			SameSite: http.SameSiteStrictMode,
		},
	)
}

func csrfCookieSecure(r *http.Request) bool {
	if r.TLS != nil {
		return true
	}

	if isLoopbackHost(r.Host) {
		return false
	}

	csrfRuntimeConfig.RLock()
	defer csrfRuntimeConfig.RUnlock()

	return csrfRuntimeConfig.cookieSecure
}

func csrfConfigured() bool {
	csrfRuntimeConfig.RLock()
	defer csrfRuntimeConfig.RUnlock()

	return csrfRuntimeConfig.configured
}

func csrfMaxAge() time.Duration {
	csrfRuntimeConfig.RLock()
	defer csrfRuntimeConfig.RUnlock()

	return csrfRuntimeConfig.maxAge
}

func csrfMultipartLimits() (
	maximumBytes int64,
	memoryBytes int64,
) {
	csrfRuntimeConfig.RLock()
	defer csrfRuntimeConfig.RUnlock()

	return csrfRuntimeConfig.multipartMaxBytes,
		csrfRuntimeConfig.multipartMemoryBytes
}

func isLoopbackHost(rawHost string) bool {
	host := strings.TrimSpace(rawHost)

	if parsedHost, _, err := net.SplitHostPort(
		host,
	); err == nil {
		host = parsedHost
	}

	host = strings.Trim(
		strings.ToLower(host),
		"[]",
	)

	if host == "localhost" {
		return true
	}

	ipAddress := net.ParseIP(host)

	return ipAddress != nil &&
		ipAddress.IsLoopback()
}

func isStateChangingMethod(method string) bool {
	switch method {
	case http.MethodPost,
		http.MethodPut,
		http.MethodPatch,
		http.MethodDelete:
		return true

	default:
		return false
	}
}
