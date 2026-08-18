package handler

import (
	"encoding/json"
	"log/slog"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5/middleware"

	"github.com/IbnBaqqi/transcendence/internal/auth"
	"github.com/IbnBaqqi/transcendence/internal/dtos"
)

const refreshTokenCookie = "refresh_token"

func (h *Handler) Signup(w http.ResponseWriter, r *http.Request) {
	var input dtos.CreateUserRequest
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		respondWithError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	result, err := h.Auth.Signup(r.Context(), input)
	if err != nil {
		respondWithServiceError(w, r, err)
		return
	}

	setRefreshTokenCookie(w, result.RefreshToken, auth.RefreshTokenTTL)

	respondWithJSON(w, http.StatusCreated, dtos.AuthResponse{
		AccessToken: result.AccessToken,
		User:        result.User,
	})
}

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
		respondWithServiceError(w, r, err)
		return
	}

	setRefreshTokenCookie(w, result.RefreshToken, auth.RefreshTokenTTL)

	respondWithJSON(w, http.StatusOK, dtos.AuthResponse{
		AccessToken: result.AccessToken,
		User:        result.User,
	})
}

func (h *Handler) Logout(w http.ResponseWriter, r *http.Request) {
	if cookie, err := r.Cookie(refreshTokenCookie); err == nil {
		if err := h.Auth.EndSession(r.Context(), cookie.Value); err != nil {
			slog.Error("revoking the session failed",
				"request_id", middleware.GetReqID(r.Context()), "error", err)
		}
	}

	setRefreshTokenCookie(w, "", -1)
	w.WriteHeader(http.StatusNoContent)
}

func (h *Handler) Refresh(w http.ResponseWriter, r *http.Request) {
	cookie, err := r.Cookie(refreshTokenCookie)
	if err != nil {
		respondWithError(w, http.StatusUnauthorized, "invalid refresh token")
		return
	}

	result, err := h.Auth.RedeemSession(r.Context(), cookie.Value)
	if err != nil {
		respondWithServiceError(w, r, err)
		return
	}

	setRefreshTokenCookie(w, result.RefreshToken, auth.RefreshTokenTTL)

	respondWithJSON(w, http.StatusOK, dtos.AuthResponse{
		AccessToken: result.AccessToken,
		User:        result.User,
	})
}

func (h *Handler) Me(w http.ResponseWriter, r *http.Request) {
	userID, err := getUserID(r)
	if err != nil {
		respondWithError(w, http.StatusUnauthorized, "authentication required")
		return
	}

	user, err := h.User.Get(r.Context(), userID)
	if err != nil {
		respondWithServiceError(w, r, err)
		return
	}

	respondWithJSON(w, http.StatusOK, dtos.UserInfo{
		ID:       user.ID.String(),
		Username: user.Username,
		Email:    user.Email,
		Role:     user.Role,
	})
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
