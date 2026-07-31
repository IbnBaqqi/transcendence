package middleware

import (
	"log/slog"
	"net/http"
	"strings"

	"github.com/IbnBaqqi/transcendence/internal/auth"
	"github.com/google/uuid"
)

// sanitizeLog strips newlines and carriage returns from a string to
// prevent log injection (G706).
func sanitizeLog(s string) string {
	s = strings.ReplaceAll(s, "\n", "")
	s = strings.ReplaceAll(s, "\r", "")
	return s
}

func Authenticate(authService *auth.JwtService) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			tokenStr, err := auth.GetBearerToken(r.Header)
			if err != nil {
				slog.Debug("no auth token", "path", sanitizeLog(r.URL.Path), "error", err.Error()) // #nosec G706 -- path sanitized by sanitizeLog
				next.ServeHTTP(w, r)
				return
			}

			claims, err := authService.VerifyAccessToken(tokenStr)
			if err != nil {
				slog.Warn("invalid auth token", "path", sanitizeLog(r.URL.Path), "error", err.Error()) // #nosec G706 -- path sanitized by sanitizeLog
				next.ServeHTTP(w, r)
				return
			}

			// Parse user ID from claims
			id, err := uuid.Parse(claims.Subject)
			if err != nil {
				slog.Error("invalid user ID in token", "subject", sanitizeLog(claims.Subject), "error", err.Error()) // #nosec G706 -- subject sanitized by sanitizeLog
				next.ServeHTTP(w, r)
				return
			}

			user := auth.User{
				ID:   id,
				Role: claims.Role,
				Name: claims.Name,
			}

			slog.Debug("authenticated request", "user_id", user.ID, "role", user.Role, "path", sanitizeLog(r.URL.Path)) // #nosec G706 -- path sanitized by sanitizeLog
			ctx := auth.WithUser(r.Context(), user)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

// RequiredAuth blocks the request unless Authenticate already attached a user.
//
// Why two middlewares and not one: Authenticate is deliberately "optional auth"
// - it attaches a user when there's a valid token and quietly continues when
// there isn't. That's what public pages want /browse listings logged out, but
// personalise if logged in). The catch is that Authenticate ALONE never rejects
// anyone, so every non-public route has to be wrapped in this as well.
func RequiredAuth(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// ok is false when no valid token was presented, so no user was stored.
		if _, ok := auth.UserFromContext(r.Context()); !ok {
			// Written out by hand rather than reusing the handler package's
			// respondWithError: middleware sits underneath handler and importing
			// it upward would tangle the two. The JSON shape matches errorRespones.
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusUnauthorized)
			_, _ = w.Write([]byte(`{"error":"authentication required"}`))
			return
		}
		next.ServeHTTP(w, r)
	})
}
