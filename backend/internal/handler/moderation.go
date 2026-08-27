package handler

import (
	"encoding/json"
	"net/http"

	"github.com/IbnBaqqi/transcendence/internal/dtos"
)

func (h *Handler) GetReportQueue(w http.ResponseWriter, r *http.Request) {
	rows, err := h.Moderation.Queue(r.Context())
	if err != nil {
		respondWithServiceError(w, r, err)
		return
	}

	respondWithJSON(w, http.StatusOK, dtos.ToReportedListingResponses(rows))
}

func (h *Handler) GetListingReports(w http.ResponseWriter, r *http.Request) {
	listingID, err := parseIDParam(r)
	if err != nil {
		respondWithError(w, http.StatusBadRequest, "Invalid listing id")
		return
	}

	rows, err := h.Moderation.ReportsFor(r.Context(), listingID)
	if err != nil {
		respondWithServiceError(w, r, err)
		return
	}

	respondWithJSON(w, http.StatusOK, dtos.ToReportResponses(rows))
}

func (h *Handler) GetModerationHistory(w http.ResponseWriter, r *http.Request) {
	listingID, err := parseIDParam(r)
	if err != nil {
		respondWithError(w, http.StatusBadRequest, "Invalid listing id")
		return
	}

	rows, err := h.Moderation.History(r.Context(), listingID)
	if err != nil {
		respondWithServiceError(w, r, err)
		return
	}

	respondWithJSON(w, http.StatusOK, dtos.ToModerationActionResponses(rows))
}

func (h *Handler) ModerateListing(w http.ResponseWriter, r *http.Request) {
	moderatorID, err := getUserID(r)
	if err != nil {
		respondWithError(w, http.StatusUnauthorized, "Authentication required")
		return
	}

	listingID, err := parseIDParam(r)
	if err != nil {
		respondWithError(w, http.StatusBadRequest, "Invalid listing id")
		return
	}

	var req dtos.ModerateListingRequest

	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()

	if err := decoder.Decode(&req); err != nil {
		respondWithError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	listing, resolved, err := h.Moderation.Moderate(r.Context(), moderatorID, listingID, req.Action, req.Note)
	if err != nil {
		respondWithServiceError(w, r, err)
		return
	}

	imgs, err := h.ListingImage.ListImages(r.Context(), listingID)
	if err != nil {
		respondWithServiceError(w, r, err)
		return
	}

	respondWithJSON(w, http.StatusOK, dtos.ModerateListingResponse{
		Listing:         dtos.ToListingResponseWithImages(listing, imgs),
		ReportsResolved: resolved,
	})
}
