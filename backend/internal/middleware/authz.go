package middleware

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"

	"github.com/google/uuid"

	"github.com/IbnBaqqi/transcendence/internal/auth"
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
				slog.Info("role check failed", "user_id", user.ID, "have", current, "want", role)
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

	if err := json.NewEncoder(w).Encode(map[string]string{"error": msg}); err != nil {
		slog.Error("could not write the error body", "error", err)
	}
}
