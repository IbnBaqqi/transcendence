package handler

import (
	"fmt"
	"net/http"
	"time"
)

// ExportMyData answers with every row the API already tells this account about
// itself, as one JSON document.
//
// Content-Disposition makes a browser save it rather than render it: the
// frontend fetches this and hands it to the user as a file, and a person who
// hits the URL directly should get the same thing rather than a wall of JSON.
func (h *Handler) ExportMyData(w http.ResponseWriter, r *http.Request) {
	userID, err := getUserID(r)
	if err != nil {
		respondWithError(w, http.StatusUnauthorized, "Authentication required")
		return
	}

	export, err := h.User.ExportData(r.Context(), userID)
	if err != nil {
		respondWithServiceError(w, r, err)
		return
	}

	name := fmt.Sprintf("metsatori-data-%s.json", time.Now().UTC().Format("2006-01-02"))
	w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=%q", name))

	respondWithJSON(w, http.StatusOK, export)
}
