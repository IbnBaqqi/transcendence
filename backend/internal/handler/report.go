package handler

import (
	"encoding/json"
	"net/http"

	"github.com/IbnBaqqi/transcendence/internal/dtos"
)

func (h *Handler) ReportListing(w http.ResponseWriter, r *http.Request) {
	reporterID, err := getUserID(r)
	if err != nil {
		respondWithError(w, http.StatusUnauthorized, "Authentication required")
		return
	}

	listingID, err := parseIDParam(r)
	if err != nil {
		respondWithError(w, http.StatusBadRequest, "Invalid listing id")
		return
	}

	var req dtos.ReportListingRequest

	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()

	if err := decoder.Decode(&req); err != nil {
		respondWithError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	if err := h.Report.Report(r.Context(), reporterID, listingID, req.Reason, req.Detail); err != nil {
		respondWithServiceError(w, r, err)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}
