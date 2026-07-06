package handler

import (
	"database/sql"
	"encoding/json"
	"errors"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"

	"github.com/IbnBaqqi/transcendence/internal/database"
	"github.com/IbnBaqqi/transcendence/internal/dtos"
)

// ListingHandler holds dependancies needed by listing endpoints.
type ListingHandler struct {
	db *database.Queries
}

func NewListingHandler(db *database.Queries) *ListingHandler {
	return &ListingHandler{db: db}
}

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
	id, err := strconv.Atoi(idStr)
	if err != nil {
		return 0, errors.New("invalid X-User-ID header")
	}
	return int32(id), nil
}

// --- Validation ---

func validateListingInput(title, category, unit string, price float64, quantity int32) error {
	if title == "" || len(title) > 100 {
		return errors.New("title is required and must be under 100 characters")
	}
	if category == "" {
		return errors.New("category is required")
	}
	if unit == "" {
		return errors.New("unit is required")
	}
	if price <= 0 {
		return errors.New("price must be greater than 0")
	}
	if quantity <= 0 {
		return errors.New("quantity must be greater then 0")
	}
	return nil
}

// --- Handlers ---

func (h *ListingHandler) Create(w http.ResponseWriter, r *http.Request) {
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

	if err := validateListingInput(input.Title, input.Category, input.Unit, input.Price, input.Quantity); err != nil {
		Error(w, http.StatusBadRequest, err.Error())
		return
	}

	listing, err := h.db.CreateListing(r.Context(), database.CreateListingParams{
		SellerID:		userID,
		Title:			input.Title,
		Description:	sql.NullString{String: input.Description, Valid: input.Description != ""},
		Category:		input.Category,
		Price:			strconv.FormatFloat(input.Price, 'f', 2, 64),
		Quantity:		input.Quantity,
		Unit:			input.Unit,
	})
	if err != nil {
		Error(w, http.StatusInternalServerError, "could not create listing")
		return
	}

	JSON(w, http.StatusCreated, listing)
}

func (h *ListingHandler) List(w http.ResponseWriter, r *http.Request) {
	listings, err := h.db.ListListings(r.Context())
	if err != nil {
		Error(w, http.StatusInternalServerError, "could not fetch listings")
		return
	}
	JSON(w, http.StatusOK, listings)
}

func (h *ListingHandler) Get(w http.ResponseWriter, r *http.Request) {
	id, err := parseIDParam(r)
	if err != nil {
		Error(w, http.StatusBadRequest, "invalid listing id")
		return
	}
	listing, err :=h.db.GetListing(r.Context(), id)
	if err != nil {
		Error(w, http.StatusNotFound, "listing not found")
		return
	}

	JSON(w, http.StatusOK, listing)
}

func (h *ListingHandler) Update(w http.ResponseWriter, r *http.Request) {
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
 
	existing, err := h.db.GetListing(r.Context(), id)
	if err != nil {
		Error(w, http.StatusNotFound, "listing not found")
		return
	}
	if existing.SellerID != userID {
		Error(w, http.StatusForbidden, "you do not own this listing")
		return
	}
 
	var input dtos.UpdateListingInput
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		Error(w, http.StatusBadRequest, "invalid request body")
		return
	}
 
	if err := validateListingInput(input.Title, input.Category, input.Unit, input.Price, input.Quantity); err != nil {
		Error(w, http.StatusBadRequest, err.Error())
		return
	}
 
	updated, err := h.db.UpdateListing(r.Context(), database.UpdateListingParams{
		ID:          id,
		Title:       input.Title,
		Description: sql.NullString{String: input.Description, Valid: input.Description != ""},
		Category:    input.Category,
		Price:       strconv.FormatFloat(input.Price, 'f', 2, 64),
		Quantity:    input.Quantity,
		Unit:        input.Unit,
	})
	if err != nil {
		Error(w, http.StatusInternalServerError, "could not update listing")
		return
	}
 
	JSON(w, http.StatusOK, updated)
}
 
func (h *ListingHandler) Delete(w http.ResponseWriter, r *http.Request) {
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
 
	existing, err := h.db.GetListing(r.Context(), id)
	if err != nil {
		Error(w, http.StatusNotFound, "listing not found")
		return
	}
	if existing.SellerID != userID {
		Error(w, http.StatusForbidden, "you do not own this listing")
		return
	}
 
	if err := h.db.DeleteListing(r.Context(), id); err != nil {
		Error(w, http.StatusInternalServerError, "could not delete listing")
		return
	}
 
	w.WriteHeader(http.StatusNoContent)
}
 
// --- Helpers ---
 
func parseIDParam(r *http.Request) (int32, error) {
	idStr := chi.URLParam(r, "id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		return 0, err
	}
	return int32(id), nil
}