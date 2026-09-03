package handler

import (
	"net/http"

	"github.com/IbnBaqqi/transcendence/internal/dtos"
)

func (h *Handler) GetNotifications(w http.ResponseWriter, r *http.Request) {
	userID, err := getUserID(r)
	if err != nil {
		respondWithError(w, http.StatusUnauthorized, "Authentication required")
		return
	}

	rows, err := h.Notification.List(r.Context(), userID)
	if err != nil {
		respondWithServiceError(w, r, err)
		return
	}

	respondWithJSON(w, http.StatusOK, dtos.ToNotificationResponses(rows))
}

func (h *Handler) MarkNotificationsRead(w http.ResponseWriter, r *http.Request) {
	userID, err := getUserID(r)
	if err != nil {
		respondWithError(w, http.StatusUnauthorized, "Authentication required")
		return
	}

	if _, err := h.Notification.MarkAllRead(r.Context(), userID); err != nil {
		respondWithServiceError(w, r, err)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}
