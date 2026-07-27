package middleware

import (
	"log/slog"
	"net/http"
	"strconv"

	"github.com/IbnBaqqi/transcendence/internal/auth"
)

func Authenticate(authService *auth.JwtService) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			tokenStr, err := auth.GetBearerToken(r.Header)
			if err != nil {
				slog.Debug("no auth token", "path", r.URL.Path, "error", err.Error())
				next.ServeHTTP(w, r)
				return
			}

			claims, err := authService.VerifyAccessToken(tokenStr)
			if err != nil {
				slog.Warn("invalid auth token", "path", r.URL.Path, "error", err.Error())
				next.ServeHTTP(w, r)
				return
			}

			// Parse user ID from claims
			id, err := strconv.ParseInt(claims.Subject, 10, 64)
			if err != nil {
				slog.Error("invalid user ID in token", "subject", claims.Subject, "error", err.Error())
				next.ServeHTTP(w, r)
				return
			}

			user := auth.User{
				ID:   id,
				Role: claims.Role,
				Name: claims.Name,
			}

			slog.Debug("authenticated request", "user_id", user.ID, "role", user.Role, "path", r.URL.Path)
			ctx := auth.WithUser(r.Context(), user)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}
