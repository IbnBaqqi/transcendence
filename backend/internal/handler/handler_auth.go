package handler

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/IbnBaqqi/transcendence/internal/dtos"
)

const (
	refreshTokenCookie = "refresh_token"
	refreshTokenTTL    = 7 * 24 * time.Hour
)

// Signup creates a new user account.
func (h *Handler) Signup(w http.ResponseWriter, r *http.Request) {
	var input dtos.CreateUserRequest
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		respondWithError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	result, err := h.Auth.Signup(r.Context(), input)
	if err != nil {
		respondWithError(w, statusFromServiceError(err), err.Error())
		return
	}

	setRefreshTokenCookie(w, result.RefreshToken, refreshTokenTTL)

	respondWithJSON(w, http.StatusCreated, dtos.AuthResponse{
		AccessToken: result.AccessToken,
		User:        result.User,
	})
}

// Login authenticates an existing user and returns tokens.
func (h *Handler) Login(w http.ResponseWriter, r *http.Request) {
	var req dtos.LoginRequest

	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()

	if err := decoder.Decode(&req); err != nil {
		respondWithError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	result, err := h.Auth.Login(r.Context(), req)
	if err != nil {
		respondWithError(w, statusFromServiceError(err), err.Error())
		return
	}

	setRefreshTokenCookie(w, result.RefreshToken, refreshTokenTTL)

	respondWithJSON(w, http.StatusOK, dtos.AuthResponse{
		AccessToken: result.AccessToken,
		User:        result.User,
	})
}

// Logout clears the refresh token cookie.
func (h *Handler) Logout(w http.ResponseWriter, r *http.Request) {
	setRefreshTokenCookie(w, "", -1)
	w.WriteHeader(http.StatusNoContent)
}

func setRefreshTokenCookie(w http.ResponseWriter, value string, ttl time.Duration) {
	maxAge := int(ttl.Seconds())
	if ttl < 0 {
		maxAge = -1
	}
	http.SetCookie(w, &http.Cookie{
		Name:     refreshTokenCookie,
		Value:    value,
		Path:     "/",
		MaxAge:   maxAge,
		HttpOnly: true,
		Secure:   true,
		SameSite: http.SameSiteLaxMode,
	})
}
