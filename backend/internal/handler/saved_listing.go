package handler

import (
	"net/http"

	"github.com/IbnBaqqi/transcendence/internal/dtos"
)

// Note what's NOT here: no getUserID, no parseIDParam. Both already exist in
// listing.go, and everything in package handler shares them. Redeclaring
// either would be a duplicate-symbol compile error.

func (h *Handler) SaveListing(w http.ResponseWriter, r *http.Request) {
	userID, err := getUserID(r)
	if err != nil {
		respondWithError(w, http.StatusUnauthorized, "authentication required")
		return
	}

	// {id} here is the LISTING's id, from /listings/{id}/save.
	listingID, err := parseIDParam(r)
	if err != nil {
		respondWithError(w, http.StatusBadRequest, "invalid listing id")
		return
	}

	if err := h.Saved.SaveListing(r.Context(), userID, listingID); err != nil {
		respondWithError(w, statusFromServiceError(err), err.Error())
		return
	}

	// 204: it worked, and there's no body worth sending back.
	w.WriteHeader(http.StatusNoContent)
}

func (h *Handler) UnsaveListing(w http.ResponseWriter, r *http.Request) {
	userID, err := getUserID(r)
	if err != nil {
		respondWithError(w, http.StatusUnauthorized, "authentication required")
		return
	}

	listingID, err := parseIDParam(r)
	if err != nil {
		respondWithError(w, http.StatusBadRequest, "invalid listing id")
		return
	}

	if err := h.Saved.UnsaveListing(r.Context(), userID, listingID); err != nil {
		respondWithError(w, statusFromServiceError(err), err.Error())
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func (h *Handler) GetSavedListings(w http.ResponseWriter, r *http.Request) {
	userID, err := getUserID(r)
	if err != nil {
		respondWithError(w, http.StatusUnauthorized, "authentication required")
		return
	}

	listings, err := h.Saved.ListSaved(r.Context(), userID)
	if err != nil {
		// A DB failure here isn't the caller's fault, so don't leak err.Error()
		// into the response - log-worthy, not client-worthy.
		respondWithError(w, http.StatusInternalServerError, "could not fetch saved listings")
		return
	}

	// Same mapper GET /listings uses (added in #81), so a wishlist entry and a
	// feed entry are byte identical shapes - the frontend reuses one type.
	respondWithJSON(w, http.StatusOK, dtos.ToListingResponses(listings))
}
