package handler

import (
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"

	"github.com/IbnBaqqi/transcendence/internal/dtos"
)

func (h *Handler) FollowUser(w http.ResponseWriter, r *http.Request) {
	followerID, err := getUserID(r)
	if err != nil {
		respondWithError(w, http.StatusUnauthorized, "authentication required")
		return
	}

	followeeID, err := targetUserID(r)
	if err != nil {
		respondWithError(w, http.StatusBadRequest, "invalid user id")
		return
	}

	if err := h.Follow.Follow(r.Context(), followerID, followeeID); err != nil {
		respondWithServiceError(w, r, err)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func (h *Handler) UnfollowUser(w http.ResponseWriter, r *http.Request) {
	followerID, err := getUserID(r)
	if err != nil {
		respondWithError(w, http.StatusUnauthorized, "authentication required")
		return
	}

	followeeID, err := targetUserID(r)
	if err != nil {
		respondWithError(w, http.StatusBadRequest, "invalid user id")
		return
	}

	if err := h.Follow.Unfollow(r.Context(), followerID, followeeID); err != nil {
		respondWithServiceError(w, r, err)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func (h *Handler) GetFollowing(w http.ResponseWriter, r *http.Request) {
	userID, err := getUserID(r)
	if err != nil {
		respondWithError(w, http.StatusUnauthorized, "authentication required")
		return
	}

	rows, err := h.Follow.ListFollowing(r.Context(), userID)
	if err != nil {
		respondWithServiceError(w, r, err)
		return
	}

	respondWithJSON(w, http.StatusOK, dtos.ToFollowingResponses(rows))
}

func (h *Handler) GetUserFollowing(w http.ResponseWriter, r *http.Request) {
	userID, err := targetUserID(r)
	if err != nil {
		respondWithError(w, http.StatusBadRequest, "invalid user id")
		return
	}

	rows, err := h.Follow.ListFollowing(r.Context(), userID)
	if err != nil {
		respondWithServiceError(w, r, err)
		return
	}

	respondWithJSON(w, http.StatusOK, dtos.ToFollowingResponses(rows))
}

func (h *Handler) GetFollowers(w http.ResponseWriter, r *http.Request) {
	userID, err := targetUserID(r)
	if err != nil {
		respondWithError(w, http.StatusBadRequest, "invalid user id")
		return
	}

	rows, err := h.Follow.ListFollowers(r.Context(), userID)
	if err != nil {
		respondWithServiceError(w, r, err)
		return
	}

	respondWithJSON(w, http.StatusOK, dtos.ToFollowerResponses(rows))
}

func targetUserID(r *http.Request) (uuid.UUID, error) {
	return uuid.Parse(chi.URLParam(r, "id"))
}
