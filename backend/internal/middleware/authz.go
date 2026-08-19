package middleware

import (
	"context"
	"database/sql"
	"errors"
	"log/slog"
	"net/http"

	"github.com/IbnBaqqi/transcendence/internal/auth"
	"github.com/google/uuid"
)

const (
	RoleUser  = "USER"
	RoleAdmin = "ADMIN"
)

type roleStore interface {
	GetUserRole(ctx context.Context, id uuid.UUID) (string, error)
}

func RequireRole(store roleStore, role string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			user, ok := auth.UserFromContext(r.Context())
			if !ok {
				writeAuthzError(w, http.StatusUnauthorized, "authentication required")
				return
			}

			current, err := store.GetUserRole(r.Context(), user.ID)
			if err != nil {
				if errors.Is(err, sql.ErrNoRows) {
					writeAuthzError(w, http.StatusForbidden, "forbidden")
					return
				}
				slog.Error("could not read role", "user_id", user.ID, "error", err)
				writeAuthzError(w, http.StatusInternalServerError, "internal server error")
				return
			}

			if current != role {
				slog.Warn("role check failed", "user_id", user.ID, "have", current, "want", role)
				writeAuthzError(w, http.StatusForbidden, "forbidden")
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}

func writeAuthzError(w http.ResponseWriter, status int, msg string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_, _ = w.Write([]byte(`{"error":"` + msg + `"}`))
}
