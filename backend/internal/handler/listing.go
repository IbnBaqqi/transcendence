package handler

import (
	"encoding/json"
	"log/slog"
	"net/http"

	"github.com/IbnBaqqi/transcendence/internal/dtos"
)

func (h *Handler) CreateListing(w http.ResponseWriter, r *http.Request) {
	userID, err := getUserID(r)
	if err != nil {
		respondWithError(w, http.StatusUnauthorized, "authentication required")
		return
	}

	var input dtos.CreateListingInput
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		respondWithError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	listing, err := h.Listing.CreateListing(r.Context(), userID, input)
	if err != nil {
		respondWithError(w, statusFromServiceError(err), err.Error())
		return
	}

	respondWithJSON(w, http.StatusCreated, dtos.ToListingResponse(listing))
}

func (h *Handler) GetListings(w http.ResponseWriter, r *http.Request) {
	listings, err := h.Listing.ListListings(r.Context())
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "could not fetch listings")
		return
	}
	respondWithJSON(w, http.StatusOK, dtos.ToListingResponses(listings))
}

func (h *Handler) GetListing(w http.ResponseWriter, r *http.Request) {
	id, err := parseIDParam(r)
	if err != nil {
		respondWithError(w, http.StatusBadRequest, "invalid listing id")
		return
	}

	listing, err := h.Listing.GetListing(r.Context(), id)
	if err != nil {
		respondWithError(w, statusFromServiceError(err), err.Error())
		return
	}

	respondWithJSON(w, http.StatusOK, dtos.ToListingResponse(listing))
}

func (h *Handler) UpdateListing(w http.ResponseWriter, r *http.Request) {
	userID, err := getUserID(r)
	if err != nil {
		respondWithError(w, http.StatusUnauthorized, "authentication required")
		return
	}

	id, err := parseIDParam(r)
	if err != nil {
		respondWithError(w, http.StatusBadRequest, "invalid listing id")
		return
	}

	var input dtos.UpdateListingInput
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		respondWithError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	updated, err := h.Listing.UpdateListing(r.Context(), userID, id, input)
	if err != nil {
		respondWithError(w, statusFromServiceError(err), err.Error())
		return
	}

	respondWithJSON(w, http.StatusOK, dtos.ToListingResponse(updated))
}

func (h *Handler) DeleteListing(w http.ResponseWriter, r *http.Request) {
	userID, err := getUserID(r)
	if err != nil {
		respondWithError(w, http.StatusUnauthorized, "authentication required")
		return
	}

	id, err := parseIDParam(r)
	if err != nil {
		respondWithError(w, http.StatusBadRequest, "invalid listing id")
		return
	}

	if err := h.Listing.DeleteListing(r.Context(), userID, id); err != nil {
		respondWithError(w, statusFromServiceError(err), err.Error())
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
		Sort:     q.Get("sort"),
		Page:     q.Get("page"),
		Limit:    q.Get("limit"),
	}

	result, err := h.Listing.SearchListings(r.Context(), query)
	if err != nil {
		status := statusFromServiceError(err)
		if status >= http.StatusInternalServerError {
			slog.Error("listing search failed", "error", err)
			respondWithError(w, status, "could not search listings")
			return
		}
		respondWithError(w, status, err.Error())
		return
	}

	respondWithJSON(w, http.StatusOK, result)
}
