package handler

import (
	"net/http"
)

func (h *Handler) SaveListing(w http.ResponseWriter, r *http.Request) {
	userID, err := getUserID(r)
	if err != nil {
		respondWithError(w, http.StatusUnauthorized, "Authentication required")
		return
	}

	listingID, err := parseIDParam(r)
	if err != nil {
		respondWithError(w, http.StatusBadRequest, "Invalid listing id")
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
		respondWithError(w, http.StatusUnauthorized, "Authentication required")
		return
	}

	listingID, err := parseIDParam(r)
	if err != nil {
		respondWithError(w, http.StatusBadRequest, "Invalid listing id")
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
		respondWithError(w, http.StatusUnauthorized, "Authentication required")
		return
	}

	listings, err := h.Saved.ListSaved(r.Context(), userID)
	if err != nil {
		respondWithServiceError(w, r, err)
		return
	}

	items, err := h.Listing.Responses(r.Context(), listings)
	if err != nil {
		respondWithServiceError(w, r, err)
		return
	}

	respondWithJSON(w, http.StatusOK, items)
}
