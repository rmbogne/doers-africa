package middleware

import (
	"context"
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/base64"
	"log"
	"net"
	"net/http"
	"os"
	"strings"
	"sync"
	"time"
)

const (
	csrfCookieName = "aos_csrf"
	CSRFFieldName  = "csrf_token"
	CSRFHeaderName = "X-CSRF-Token"

	csrfTokenBytes        = 32
	csrfMaxAge            = 12 * time.Hour
	csrfMultipartMaxBytes = 12 << 20
	csrfMultipartMemory   = 10 << 20
)

type csrfContextKey struct{}

var (
	csrfSecretOnce sync.Once
	csrfSecretKey  []byte
)

// CSRF protects POST, PUT, PATCH, and DELETE requests using a signed
// double-submit token. It also exposes a token through CSRFToken(r) so the
// server-rendered templates can place it in forms.
func CSRF(next http.Handler) http.Handler {
	return http.HandlerFunc(
		func(
			w http.ResponseWriter,
			r *http.Request,
		) {
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

// CSRFToken returns the request token made available by CSRF middleware.
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
		r.Body = http.MaxBytesReader(
			w,
			r.Body,
			csrfMultipartMaxBytes,
		)

		if err := r.ParseMultipartForm(
			csrfMultipartMemory,
		); err != nil {
			return "", err
		}
	} else if err := r.ParseForm(); err != nil {
		return "", err
	}

	return strings.TrimSpace(
		r.Form.Get(CSRFFieldName),
	), nil
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
	csrfSecretOnce.Do(
		func() {
			configuredSecret := strings.TrimSpace(
				os.Getenv("CSRF_SECRET"),
			)

			if len(configuredSecret) >= 32 {
				csrfSecretKey =
					[]byte(configuredSecret)
				return
			}

			// Local-development fallback. Production must configure
			// CSRF_SECRET so tokens remain valid across instances/restarts.
			csrfSecretKey = make([]byte, 32)

			if _, err := rand.Read(
				csrfSecretKey,
			); err != nil {
				panic(
					"unable to initialize CSRF secret: " +
						err.Error(),
				)
			}

			log.Print(
				"Warning: CSRF_SECRET is not configured; " +
					"using an ephemeral development secret",
			)
		},
	)

	return csrfSecretKey
}

func setCSRFCookie(
	w http.ResponseWriter,
	r *http.Request,
	token string,
) {
	http.SetCookie(
		w,
		&http.Cookie{
			Name:     csrfCookieName,
			Value:    token,
			Path:     "/",
			MaxAge:   int(csrfMaxAge.Seconds()),
			Expires:  time.Now().Add(csrfMaxAge),
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

	// Safari does not consistently accept/send Secure cookies over plain
	// http://localhost. Always keep the cookie non-Secure for an HTTP
	// loopback development address, even if an inherited shell variable is
	// accidentally set to true.
	if isLoopbackHost(r.Host) {
		return false
	}

	forwardedProto := strings.TrimSpace(
		strings.Split(
			r.Header.Get("X-Forwarded-Proto"),
			",",
		)[0],
	)

	if strings.EqualFold(
		forwardedProto,
		"https",
	) {
		return true
	}

	configuredValue := strings.ToLower(
		strings.TrimSpace(
			os.Getenv("CSRF_COOKIE_SECURE"),
		),
	)

	return configuredValue == "true" ||
		configuredValue == "1" ||
		configuredValue == "yes"
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
