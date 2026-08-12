package handler

import (
	"encoding/json"
	"errors"
	"net/http"

	"github.com/IbnBaqqi/transcendence/internal/dtos"
)

var errInvalidNumber = errors.New("invalid number")

func (h *Handler) GetSettings(w http.ResponseWriter, r *http.Request) {
	userID, err := getUserID(r)
	if err != nil {
		respondWithError(w, http.StatusUnauthorized, "authentication required")
		return
	}

	user, err := h.User.Get(r.Context(), userID)
	if err != nil {
		respondWithError(w, statusFromServiceError(err), err.Error())
		return
	}

	respondWithJSON(w, http.StatusOK, dtos.ToUserSettingsResponse(user))
}

func (h *Handler) UpdateSettings(w http.ResponseWriter, r *http.Request) {
	userID, err := getUserID(r)
	if err != nil {
		respondWithError(w, http.StatusUnauthorized, "authentication required")
		return
	}

	var input dtos.UpdateSettingsInput
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		respondWithError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if input.ShowOnlineStatus == nil {
		respondWithError(w, http.StatusBadRequest, "no settings to update")
		return
	}

	user, err := h.User.SetShowOnlineStatus(r.Context(), userID, *input.ShowOnlineStatus)
	if err != nil {
		respondWithError(w, statusFromServiceError(err), err.Error())
		return
	}

	respondWithJSON(w, http.StatusOK, dtos.ToUserSettingsResponse(user))
}

func (h *Handler) GetUnreadCount(w http.ResponseWriter, r *http.Request) {
	userID, err := getUserID(r)
	if err != nil {
		respondWithError(w, http.StatusUnauthorized, "authentication required")
		return
	}

	count, err := h.Conversation.CountUnread(r.Context(), userID)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "could not count unread messages")
		return
	}

	respondWithJSON(w, http.StatusOK, dtos.UnreadCountResponse{UnreadCount: count})
}
