package handler

import (
	"net/http"

	"github.com/IbnBaqqi/transcendence/internal/dtos"
)

func (h *Handler) UploadAvatar(w http.ResponseWriter, r *http.Request) {
	userID, err := getUserID(r)
	if err != nil {
		respondWithError(w, http.StatusUnauthorized, "Authentication required")
		return
	}

	upload, ok := h.readImageUpload(w, r, "avatar")
	if !ok {
		return
	}
	defer upload.Close()

	filename, err := h.Profile.SetAvatar(r.Context(), userID, upload.Body, upload.Ext)
	if err != nil {
		if isTooLarge(err) {
			respondWithError(w, http.StatusRequestEntityTooLarge, "Image is too large")
			return
		}
		respondWithServiceError(w, r, err)
		return
	}

	respondWithJSON(w, http.StatusOK, dtos.AvatarResponse{
		AvatarURL: dtos.UploadURLPrefix + filename,
	})
}

func (h *Handler) DeleteAvatar(w http.ResponseWriter, r *http.Request) {
	userID, err := getUserID(r)
	if err != nil {
		respondWithError(w, http.StatusUnauthorized, "Authentication required")
		return
	}

	if err := h.Profile.RemoveAvatar(r.Context(), userID); err != nil {
		respondWithServiceError(w, r, err)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}
