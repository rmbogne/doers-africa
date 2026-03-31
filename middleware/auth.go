package middleware

import (
	"context"
	"net/http"
	"strconv"
	"strings"
)

// SessionInfo holds the session data
type SessionInfo struct {
	ID   int
	Role string
}

type contextKey string

const SessionKey = contextKey("session")

// Auth middleware extracts session from cookie and sets it in context
func Auth(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		cookie, err := r.Cookie("session")
		if err == nil && cookie.Value != "" {
			// Expected format: "role:id"
			parts := strings.Split(cookie.Value, ":")
			if len(parts) == 2 {
				role := parts[0]
				id, _ := strconv.Atoi(parts[1])

				info := SessionInfo{ID: id, Role: role}
				ctx := context.WithValue(r.Context(), SessionKey, info)
				next.ServeHTTP(w, r.WithContext(ctx))
				return
			}
		}
		next.ServeHTTP(w, r)
	})
}

// RequireRole wraps a handler to ensure only specific roles can access
func RequireRole(role string, next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		val := r.Context().Value(SessionKey)
		if val == nil {
			http.Redirect(w, r, "/login?role="+role, http.StatusSeeOther)
			return
		}
		info := val.(SessionInfo)
		if info.Role != role {
			http.Error(w, "Forbidden: Invalid Role", http.StatusForbidden)
			return
		}
		next(w, r)
	}
}
