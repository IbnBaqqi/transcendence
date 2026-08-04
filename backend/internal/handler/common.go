package handler

import (
	"errors"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"

	"github.com/IbnBaqqi/transcendence/internal/auth"
	"github.com/IbnBaqqi/transcendence/internal/service"
)

// Helpers shared by every file in the handler package.
//
// They started life in listing.go, but order.go and handler_auth.go use them
// too - a helper three features depend on shouldn't be owned by one feature's
// file. Same idea as response.go, which already holds respondWithJSON and
// respondWithError. Everything here is lowercase (unexported), so it's visible
// inside package handler and nowhere else.

// getUserID returns the id of the authenticated caller.
//
// The user is placed in the context by the Authenticate middleware *after* it
// has verified the JWT signature, so this value is trustworthy.
//
// It returns an error rather than assuming success: a route that forgot
// mw.RequireAuth would otherwise silently run as uuid.Nil.
func getUserID(r *http.Request) (uuid.UUID, error) {
	user, ok := auth.UserFromContext(r.Context())
	if !ok {
		return uuid.Nil, errors.New("no authenticated user in context")
	}
	return user.ID, nil
}

// parseIDParam reads the {id} segment out of the URL and converts it to int32.
//
// chi.URLParam pulls the named piece out of the route pattern; ParseInt's
// bitSize of 32 makes sure the value actually fits in an int32 before we cast.
func parseIDParam(r *http.Request) (int32, error) {
	idStr := chi.URLParam(r, "id")
	id, err := strconv.ParseInt(idStr, 10, 32)
	if err != nil {
		return 0, err
	}
	return int32(id), nil
}

// statusFromServiceError maps service-layer error types to HTTP status codes,
// so handlers don't need to know about business rules - only about what kind
// of error occurred.
//
// errors.As walks the error chain looking for one that matches the target's
// type, which is why wrapped errors still map correctly.
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
