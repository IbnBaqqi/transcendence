package handler

import (
	"errors"
	"log/slog"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/google/uuid"

	"github.com/IbnBaqqi/transcendence/internal/auth"
	"github.com/IbnBaqqi/transcendence/internal/database"
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

// parseUUIDParam reads a named URL segment as a UUID. Every id in the API is
// one, so this replaced the int32 and uuid variants that used to sit alongside
// each other.
func parseUUIDParam(r *http.Request, name string) (uuid.UUID, error) {
	return uuid.Parse(chi.URLParam(r, name))
}

// parseIDParam reads the {id} segment, which is the common case.
func parseIDParam(r *http.Request) (uuid.UUID, error) {
	return parseUUIDParam(r, "id")
}

// parseOptionalUUID reads a UUID from a query string, treating an empty value
// as absent. Cursors use it: no "after" means the first page.
func parseOptionalUUID(raw string) (uuid.UUID, error) {
	if raw == "" {
		return uuid.Nil, nil
	}
	return uuid.Parse(raw)
}

func statusFromServiceError(err error) int {
	var validationErr *service.ValidationError
	var notFoundErr *service.NotFoundError
	var forbiddenErr *service.ForbiddenError
	var conflictErr *service.ConflictError

	var authValidationErr *auth.ValidationError
	var authConflictErr *auth.ConflictError
	var authAccountExistsErr *auth.AccountExistsError
	var authRetryErr *auth.RetryError
	var authErr *auth.AuthError
	var authForbiddenErr *auth.ForbiddenError
	var authSuspendedErr *auth.SuspendedError

	switch {
	case errors.As(err, &validationErr), errors.As(err, &authValidationErr):
		return http.StatusBadRequest
	case errors.As(err, &notFoundErr):
		return http.StatusNotFound
	case errors.As(err, &forbiddenErr), errors.As(err, &authSuspendedErr),
		errors.As(err, &authForbiddenErr):
		return http.StatusForbidden
	case errors.As(err, &conflictErr), errors.As(err, &authConflictErr),
		errors.As(err, &authAccountExistsErr), errors.As(err, &authRetryErr):
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
			"method", r.Method,
			"path", r.URL.Path,
			"error", err)

		respondWithError(w, status, "Something went wrong")
		return
	}

	respondWithError(w, status, err.Error())
}

// viewerID is the authenticated user, or uuid.Nil on a public route where
// nobody is signed in. Used where a response is shaped by who is reading it -
// unlike getUserID, an anonymous viewer is not an error here.
func viewerID(r *http.Request) uuid.UUID {
	user, ok := auth.UserFromContext(r.Context())
	if !ok {
		return uuid.Nil
	}
	return user.ID
}

// hidePresenceIfBlocked blanks a user's online status when the viewer must not
// see it. The list endpoints do this in SQL; these are the paths that fetch one
// user at a time.
//
// Anonymous viewers get no presence at all. GET /users/{id} is public, so
// leaving them alone would let a blocked user read the blocker's status simply
// by logging out - which would make the whole rule a speed bump.
//
// Fails closed on a lookup error: a wrong "hidden" is cosmetic, a wrong
// "online" means someone keeps watching a person who blocked them.
func (h *Handler) hidePresenceIfBlocked(r *http.Request, viewer uuid.UUID, other *database.User) {
	if viewer == other.ID {
		return
	}

	if viewer == uuid.Nil {
		other.ShowOnlineStatus = false
		return
	}

	blocked, err := h.Block.ExistsBetween(r.Context(), viewer, other.ID)
	if err != nil {
		slog.Warn("presence block check failed, hiding presence",
			"error", err, "request_id", middleware.GetReqID(r.Context()))
		other.ShowOnlineStatus = false
		return
	}

	if blocked {
		other.ShowOnlineStatus = false
	}
}

// viewerName is the authenticated caller's username, for a response that
// echoes back something they authored. Empty when nobody is signed in.
func viewerName(r *http.Request) string {
	user, ok := auth.UserFromContext(r.Context())
	if !ok {
		return ""
	}
	return user.Name
}
