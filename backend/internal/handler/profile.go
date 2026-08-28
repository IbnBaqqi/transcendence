package handler

import (
	"encoding/json"
	"net/http"

	"github.com/google/uuid"

	"github.com/IbnBaqqi/transcendence/internal/dtos"
)

func (h *Handler) GetOwnProfile(w http.ResponseWriter, r *http.Request) {
	userID, err := getUserID(r)
	if err != nil {
		respondWithError(w, http.StatusUnauthorized, "Authentication required")
		return
	}

	detail, err := h.Profile.Get(r.Context(), userID)
	if err != nil {
		respondWithServiceError(w, r, err)
		return
	}

	respondWithJSON(w, http.StatusOK, dtos.ToOwnProfileResponse(detail.User, detail.Profile, detail.Location))
}

func (h *Handler) UpdateOwnProfile(w http.ResponseWriter, r *http.Request) {
	userID, err := getUserID(r)
	if err != nil {
		respondWithError(w, http.StatusUnauthorized, "Authentication required")
		return
	}

	var input dtos.UpdateProfileInput

	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()

	if err := decoder.Decode(&input); err != nil {
		respondWithError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	detail, err := h.Profile.Update(r.Context(), userID, input)
	if err != nil {
		respondWithServiceError(w, r, err)
		return
	}

	respondWithJSON(w, http.StatusOK, dtos.ToOwnProfileResponse(detail.User, detail.Profile, detail.Location))
}

// The same service read as GetOwnProfile - the difference is entirely in
// which mapper runs, so the private fields have no way to reach this response.
func (h *Handler) GetPublicProfile(w http.ResponseWriter, r *http.Request) {
	userID, err := parseUUIDParam(r, "id")
	if err != nil {
		respondWithError(w, http.StatusBadRequest, "Invalid user id")
		return
	}

	detail, err := h.Profile.Get(r.Context(), userID)
	if err != nil {
		respondWithServiceError(w, r, err)
		return
	}

	// Presence is for signed-in callers only. Anonymous ones get the profile
	// without the field at all, rather than a blanket "offline" that would be
	// false for every user on the site - and that a client cannot tell apart
	// from a real one.
	viewer := viewerID(r)
	if viewer != uuid.Nil {
		h.hidePresenceIfBlocked(r, viewer, &detail.User)
	}

	respondWithJSON(w, http.StatusOK,
		dtos.ToPublicProfileResponse(detail.User, detail.Profile, detail.Location, viewer != uuid.Nil))
}
