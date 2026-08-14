package handler

import (
	"net/http"

	"github.com/IbnBaqqi/transcendence/internal/dtos"
)

func (h *Handler) SaveListing(w http.ResponseWriter, r *http.Request) {
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

	if err := h.Saved.SaveListing(r.Context(), userID, listingID); err != nil {
		respondWithServiceError(w, r, err)
		return
	}

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
		respondWithServiceError(w, r, err)
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
		respondWithServiceError(w, r, err)
		return
	}

	ids := make([]int32, 0, len(listings))
	for _, l := range listings {
		ids = append(ids, l.ID)
	}

	byListing, err := h.ListingImage.ImagesByListing(r.Context(), ids)
	if err != nil {
		respondWithServiceError(w, r, err)
		return
	}

	respondWithJSON(w, http.StatusOK, dtos.ToListingResponsesWithImages(listings, byListing))
}
