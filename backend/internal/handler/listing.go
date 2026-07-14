package handler

import (
	"encoding/json"
	"errors"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"

	"github.com/IbnBaqqi/transcendence/internal/dtos"
	"github.com/IbnBaqqi/transcendence/internal/service"
)

// --- Auth Placeholder ---
//
// TODO(auth): Replace this once real middleware exists.
// This currently trusts a client-supplied header, which is NOT secure -
// it's only here so CRUD logic can be built and tested locally.
// Once middleware is in place, this should read the user ID from
// r.Context() instead (set by that middleware after verifying a
// session/JWT), and this function can likely be deleted entirely.
func getUserID(r *http.Request) (int32, error) {
	idStr := r.Header.Get("X-USER-ID")
	if idStr == "" {
		return 0, errors.New("missing X-User-ID header")
	}
	id, err := strconv.ParseInt(idStr, 10, 32)
	if err != nil {
		return 0, errors.New("invalid X-User-ID header")
	}
	return int32(id), nil
}

// statusFromServiceError maps service-layer error types to HTTP status
// codes, so the handler doesn't need to know about business rules,
// only about what kind of error occure.
func statusFromServiceError(err error) int {
	var validationErr *service.ValidationError
	var notFoundErr *service.NotFoundError
	var forbiddenErr *service.ForbiddenError

	switch {
	case errors.As(err, &validationErr):
		return http.StatusBadRequest
	case errors.As(err, &notFoundErr):
		return http.StatusNotFound
	case errors.As(err, &forbiddenErr):
		return http.StatusForbidden
	default:
		return http.StatusInternalServerError
	}
}

func (h *Handler) CreateListing(w http.ResponseWriter, r *http.Request) {
	userID, err := getUserID(r)
	if err != nil {
		Error(w, http.StatusUnauthorized, "authentication required")
		return
	}

	var input dtos.CreateListingInput
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		Error(w, http.StatusBadRequest, "invalid request body")
		return
	}

	listing, err := h.Listing.CreateListing(r.Context(), userID, input)
	if err != nil {
		Error(w, statusFromServiceError(err), err.Error())
		return
	}

	JSON(w, http.StatusCreated, listing)
}

func (h *Handler) GetListings(w http.ResponseWriter, r *http.Request) {
	listings, err := h.Listing.ListListings(r.Context())
	if err != nil {
		Error(w, http.StatusInternalServerError, "could not fetch listings")
		return
	}
	JSON(w, http.StatusOK, listings)
}

func (h *Handler) GetListing(w http.ResponseWriter, r *http.Request) {
	id, err := parseIDParam(r)
	if err != nil {
		Error(w, http.StatusBadRequest, "invalid listing id")
		return
	}

	listing, err := h.Listing.GetListing(r.Context(), id)
	if err != nil {
		Error(w, statusFromServiceError(err), err.Error())
		return
	}

	JSON(w, http.StatusOK, listing)
}

func (h *Handler) UpdateListing(w http.ResponseWriter, r *http.Request) {
	userID, err := getUserID(r)
	if err != nil {
		Error(w, http.StatusUnauthorized, "authentication required")
		return
	}

	id, err := parseIDParam(r)
	if err != nil {
		Error(w, http.StatusBadRequest, "invalid listing id")
		return
	}

	var input dtos.UpdateListingInput
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		Error(w, http.StatusBadRequest, "invalid request body")
		return
	}

	updated, err := h.Listing.UpdateListing(r.Context(), userID, id, input)
	if err != nil {
		Error(w, statusFromServiceError(err), err.Error())
		return
	}

	JSON(w, http.StatusOK, updated)
}

func (h *Handler) DeleteListing(w http.ResponseWriter, r *http.Request) {
	userID, err := getUserID(r)
	if err != nil {
		Error(w, http.StatusUnauthorized, "authentication required")
		return
	}

	id, err := parseIDParam(r)
	if err != nil {
		Error(w, http.StatusBadRequest, "invalid listing id")
		return
	}

	if err := h.Listing.DeleteListing(r.Context(), userID, id); err != nil {
		Error(w, statusFromServiceError(err), err.Error())
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func (h *Handler) SearchListings(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	query := dtos.ListingSearchQuery{
		Keyword:  q.Get("keyword"),
		Category: q.Get("category"),
		MinPrice: q.Get("min_price"),
		MaxPrice: q.Get("max_price"),
		Location: q.Get("location"),
		Page:     q.Get("page"),
		Limit:    q.Get("limit"),
	}

	result, err := h.Listing.SearchListings(r.Context(), query)
	if err != nil {
		Error(w, statusFromServiceError(err), err.Error())
		return
	}

	JSON(w, http.StatusOK, result)
}

// --- Helpers ---

func parseIDParam(r *http.Request) (int32, error) {
	idStr := chi.URLParam(r, "id")
	id, err := strconv.ParseInt(idStr, 10, 32)
	if err != nil {
		return 0, err
	}
	return int32(id), nil
}
