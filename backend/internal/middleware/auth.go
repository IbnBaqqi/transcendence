package middleware

import (
	"log/slog"
	"net/http"
	"strconv"
	"strings"

	"github.com/IbnBaqqi/transcendence/internal/auth"
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
			id, err := strconv.ParseInt(claims.Subject, 10, 64)
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
