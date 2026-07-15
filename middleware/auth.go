package middleware

import (
	"context"
	"database/sql"
	"errors"
	"log"
	"net/http"

	sessionutil "github.com/mbogne/african-doers/internal/session"
	"github.com/mbogne/african-doers/store"
)

const sessionCookieName = "session"

type SessionInfo struct {
	ID   int
	Role string
}

type contextKey string

const SessionKey = contextKey("session")

func Auth(next http.Handler) http.Handler {
	return http.HandlerFunc(
		func(w http.ResponseWriter, r *http.Request) {
			cookie, err := r.Cookie(sessionCookieName)
			if err != nil || cookie.Value == "" {
				next.ServeHTTP(w, r)
				return
			}

			tokenHash := sessionutil.Hash(cookie.Value)

			authSession, err := store.DB.GetSession(
				r.Context(),
				tokenHash,
			)
			if err != nil {
				if !errors.Is(err, sql.ErrNoRows) {
					log.Printf(
						"Session lookup error: %v",
						err,
					)
				}

				next.ServeHTTP(w, r)
				return
			}

			sessionInfo := SessionInfo{
				ID:   authSession.UserID,
				Role: authSession.Role,
			}

			ctx := context.WithValue(
				r.Context(),
				SessionKey,
				sessionInfo,
			)

			next.ServeHTTP(
				w,
				r.WithContext(ctx),
			)
		},
	)
}

func RequireRole(
	requiredRole string,
	next http.Handler,
) http.Handler {
	return http.HandlerFunc(
		func(w http.ResponseWriter, r *http.Request) {
			sessionInfo, ok := r.Context().
				Value(SessionKey).(SessionInfo)

			if !ok {
				http.Error(
					w,
					"Authentication required",
					http.StatusUnauthorized,
				)
				return
			}

			if sessionInfo.Role != requiredRole {
				http.Error(
					w,
					"Forbidden",
					http.StatusForbidden,
				)
				return
			}

			next.ServeHTTP(w, r)
		},
	)
}

func GetRoleAndID(r *http.Request) (string, int) {
	sessionInfo, ok := r.Context().
		Value(SessionKey).(SessionInfo)

	if !ok {
		return "", 0
	}

	return sessionInfo.Role, sessionInfo.ID
}
