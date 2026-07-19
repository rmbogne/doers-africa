package middleware

import "net/http"

const DefaultMaximumRequestBodyBytes int64 = 12 << 20

// RequestBodyLimit must wrap CSRF middleware so multipart requests are bounded
// before CSRF form parsing occurs.
func RequestBodyLimit(
	maximumBytes int64,
	next http.Handler,
) http.Handler {
	if maximumBytes <= 0 {
		maximumBytes =
			DefaultMaximumRequestBodyBytes
	}

	return http.HandlerFunc(
		func(
			w http.ResponseWriter,
			r *http.Request,
		) {
			if r.Body != nil {
				r.Body = http.MaxBytesReader(
					w,
					r.Body,
					maximumBytes,
				)
			}

			next.ServeHTTP(w, r)
		},
	)
}
