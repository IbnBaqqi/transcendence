package handler

import (
	"errors"
	"log/slog"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/google/uuid"

	"github.com/IbnBaqqi/transcendence/internal/auth"
	"github.com/IbnBaqqi/transcendence/internal/service"
)

// getUserID returns the id of the authenticated caller.
func getUserID(r *http.Request) (uuid.UUID, error) {
	user, ok := auth.UserFromContext(r.Context())
	if !ok {
		return uuid.Nil, errors.New("no authenticated user in context")
	}
	return user.ID, nil
}

// parseInt32Param reads a named URL segment and converts it to int32.
func parseInt32Param(r *http.Request, name string) (int32, error) {
	idStr := chi.URLParam(r, name)
	id, err := strconv.ParseInt(idStr, 10, 32)
	if err != nil {
		return 0, err
	}
	return int32(id), nil
}

// parseIDParam reads the {id} segment out of the URL and converts it to int32.
func parseIDParam(r *http.Request) (int32, error) {
	return parseInt32Param(r, "id")
}

func statusFromServiceError(err error) int {
	var validationErr *service.ValidationError
	var notFoundErr *service.NotFoundError
	var forbiddenErr *service.ForbiddenError
	var conflictErr *service.ConflictError

	var authValidationErr *auth.ValidationError
	var authConflictErr *auth.ConflictError
	var authErr *auth.AuthError

	switch {
	case errors.As(err, &validationErr), errors.As(err, &authValidationErr):
		return http.StatusBadRequest
	case errors.As(err, &notFoundErr):
		return http.StatusNotFound
	case errors.As(err, &forbiddenErr):
		return http.StatusForbidden
	case errors.As(err, &conflictErr), errors.As(err, &authConflictErr):
		return http.StatusConflict
	case errors.As(err, &authErr):
		return http.StatusUnauthorized
	default:
		return http.StatusInternalServerError
	}
}

// respondWithServiceError turns a service error into a response, choosing the
// audience by status.
func respondWithServiceError(w http.ResponseWriter, r *http.Request, err error) {
	status := statusFromServiceError(err)

	if status >= http.StatusInternalServerError {
		slog.Error("unhandled error",
			"request_id", middleware.GetReqID(r.Context()),
			"path", r.URL.Path,
			"error", err)

		respondWithError(w, status, "something went wrong")
		return
	}

	respondWithError(w, status, err.Error())
}
