package middleware

import (
	"net/http"
)

// MaxBody bounds every request body under the group it wraps.
//
// The limit MUST stay at or above the upload cap. http.MaxBytesReader wraps
// r.Body, so a later, larger wrap in a handler reads THROUGH this one and the
// smaller limit wins - nesting composes as a minimum, not an override. Set n
// below MAX_UPLOAD_BYTES and every avatar and listing photo silently fails at
// n instead, with the handler tests none the wiser because they never run
// this.
func MaxBody(n int64) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Declared length first, so an oversized body is refused before a
			// byte of it is read and with the status that says why. The reader
			// below still covers a chunked body, where the length is unknown
			// until it arrives - there the handler's decoder reports the
			// failure instead, as a 400.
			if r.ContentLength > n {
				writeError(w, http.StatusRequestEntityTooLarge, "Request body is too large")
				return
			}

			r.Body = http.MaxBytesReader(w, r.Body, n)
			next.ServeHTTP(w, r)
		})
	}
}
