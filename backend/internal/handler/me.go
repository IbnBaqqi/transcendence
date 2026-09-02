package handler

import (
	"encoding/json"
	"errors"
	"net/http"

	"github.com/IbnBaqqi/transcendence/internal/auth"
	"github.com/IbnBaqqi/transcendence/internal/dtos"
)

var errInvalidNumber = errors.New("invalid number")

func (h *Handler) GetSettings(w http.ResponseWriter, r *http.Request) {
	userID, err := getUserID(r)
	if err != nil {
		respondWithError(w, http.StatusUnauthorized, "Authentication required")
		return
	}

	user, err := h.User.Get(r.Context(), userID)
	if err != nil {
		respondWithServiceError(w, r, err)
		return
	}

	respondWithJSON(w, http.StatusOK, dtos.ToUserSettingsResponse(user))
}

func (h *Handler) UpdateSettings(w http.ResponseWriter, r *http.Request) {
	userID, err := getUserID(r)
	if err != nil {
		respondWithError(w, http.StatusUnauthorized, "Authentication required")
		return
	}

	var input dtos.UpdateSettingsInput
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		respondWithError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	if input.ShowOnlineStatus == nil {
		respondWithError(w, http.StatusBadRequest, "No settings to update")
		return
	}

	user, err := h.User.SetShowOnlineStatus(r.Context(), userID, *input.ShowOnlineStatus)
	if err != nil {
		respondWithServiceError(w, r, err)
		return
	}

	respondWithJSON(w, http.StatusOK, dtos.ToUserSettingsResponse(user))
}

func (h *Handler) GetUnreadCount(w http.ResponseWriter, r *http.Request) {
	userID, err := getUserID(r)
	if err != nil {
		respondWithError(w, http.StatusUnauthorized, "Authentication required")
		return
	}

	count, err := h.Conversation.CountUnread(r.Context(), userID)
	if err != nil {
		respondWithServiceError(w, r, err)
		return
	}

	respondWithJSON(w, http.StatusOK, dtos.UnreadCountResponse{UnreadCount: count})
}

func (h *Handler) DeleteAccount(w http.ResponseWriter, r *http.Request) {
	userID, err := getUserID(r)
	if err != nil {
		respondWithError(w, http.StatusUnauthorized, "Authentication required")
		return
	}

	var req dtos.DeleteAccountRequest

	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()

	if err := decoder.Decode(&req); err != nil {
		respondWithError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	if err := h.User.DeleteAccount(r.Context(), userID, req.Username); err != nil {
		respondWithServiceError(w, r, err)
		return
	}

	h.setRefreshTokenCookie(w, "", -1)
	w.WriteHeader(http.StatusNoContent)
}

func (h *Handler) ChangePassword(w http.ResponseWriter, r *http.Request) {
	userID, err := getUserID(r)
	if err != nil {
		respondWithError(w, http.StatusUnauthorized, "Authentication required")
		return
	}

	var req dtos.ChangePasswordRequest

	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()

	if err := decoder.Decode(&req); err != nil {
		respondWithError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	refreshToken, err := h.Auth.ChangePassword(r.Context(), userID, req.CurrentPassword, req.NewPassword)
	if err != nil {
		respondWithServiceError(w, r, err)
		return
	}

	// The caller's own session was revoked with the rest, so the cookie has to
	// carry the replacement or this request signs its own sender out.
	h.setRefreshTokenCookie(w, refreshToken, auth.RefreshTokenTTL)
	w.WriteHeader(http.StatusNoContent)
}
