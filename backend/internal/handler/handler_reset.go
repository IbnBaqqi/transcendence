package handler

import (
	"encoding/json"
	"net/http"

	"github.com/IbnBaqqi/transcendence/internal/dtos"
)

// ForgotPassword always answers 202, whether or not the address exists.
// Anything else lets a caller test an address list for membership.
func (h *Handler) ForgotPassword(w http.ResponseWriter, r *http.Request) {
	var req dtos.ForgotPasswordRequest

	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()

	if err := decoder.Decode(&req); err != nil {
		respondWithError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if err := h.Auth.RequestReset(r.Context(), req.Email); err != nil {
		respondWithServiceError(w, r, err)
		return
	}

	w.WriteHeader(http.StatusAccepted)
}

func (h *Handler) ResetPassword(w http.ResponseWriter, r *http.Request) {
	var req dtos.ResetPasswordRequest

	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()

	if err := decoder.Decode(&req); err != nil {
		respondWithError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if err := h.Auth.ResetPassword(r.Context(), req.Token, req.Password); err != nil {
		respondWithServiceError(w, r, err)
		return
	}

	// No session is issued: following the link proves control of the mailbox,
	// not of the account, and every session was just revoked.
	w.WriteHeader(http.StatusNoContent)
}
