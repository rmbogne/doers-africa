package middleware

import (
	"log"
	"net/http"
	"time"
)

// Logger is a basic logging middleware
func Logger(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()

		log.Printf("[AUDIT] Started %s %s", r.Method, r.URL.Path)

		// Create a custom response writer to capture the status code
		rw := &responseWriter{ResponseWriter: w, status: http.StatusOK}
		
		next.ServeHTTP(rw, r)

		duration := time.Since(start)
		log.Printf("[MONITOR] Completed %s %s with %d in %v", r.Method, r.URL.Path, rw.status, duration)
	})
}

// responseWriter wraps http.ResponseWriter to capture the status code
type responseWriter struct {
	http.ResponseWriter
	status int
}

func (rw *responseWriter) WriteHeader(code int) {
	rw.status = code
	rw.ResponseWriter.WriteHeader(code)
}
