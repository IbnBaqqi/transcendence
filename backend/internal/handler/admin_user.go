package handler

import (
	"encoding/json"
	"net/http"

	"github.com/google/uuid"

	"github.com/IbnBaqqi/transcendence/internal/dtos"
)

func (h *Handler) ListUsers(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()

	page, err := h.AdminUser.List(r.Context(), dtos.AdminUserQuery{
		Role:   q.Get("role"),
		Status: q.Get("status"),
		Page:   q.Get("page"),
		Limit:  q.Get("limit"),
	})
	if err != nil {
		respondWithServiceError(w, r, err)
		return
	}

	respondWithJSON(w, http.StatusOK, page)
}

func (h *Handler) GetUserHistory(w http.ResponseWriter, r *http.Request) {
	subjectID, err := parseIDParam(r)
	if err != nil {
		respondWithError(w, http.StatusBadRequest, "Invalid user id")
		return
	}

	actions, err := h.AdminUser.History(r.Context(), subjectID)
	if err != nil {
		respondWithServiceError(w, r, err)
		return
	}

	respondWithJSON(w, http.StatusOK, dtos.ToUserActionResponses(actions))
}

func (h *Handler) SuspendUser(w http.ResponseWriter, r *http.Request) {
	adminID, subjectID, ok := adminAndSubject(w, r)
	if !ok {
		return
	}

	var req dtos.SuspendRequest
	if !decodeStrict(w, r, &req) {
		return
	}

	user, err := h.AdminUser.Suspend(r.Context(), adminID, subjectID, req.Reason)
	if err != nil {
		respondWithServiceError(w, r, err)
		return
	}

	respondWithJSON(w, http.StatusOK, dtos.ToAdminUserResponse(user))
}

func (h *Handler) SetUserRole(w http.ResponseWriter, r *http.Request) {
	adminID, subjectID, ok := adminAndSubject(w, r)
	if !ok {
		return
	}

	var req dtos.SetRoleRequest
	if !decodeStrict(w, r, &req) {
		return
	}

	user, err := h.AdminUser.SetRole(r.Context(), adminID, subjectID, req.Role, req.Note)
	if err != nil {
		respondWithServiceError(w, r, err)
		return
	}

	respondWithJSON(w, http.StatusOK, dtos.ToAdminUserResponse(user))
}

func (h *Handler) ReinstateUser(w http.ResponseWriter, r *http.Request) {
	adminID, subjectID, ok := adminAndSubject(w, r)
	if !ok {
		return
	}

	var req dtos.ReinstateRequest
	if !decodeStrict(w, r, &req) {
		return
	}

	user, err := h.AdminUser.Reinstate(r.Context(), adminID, subjectID, req.Note)
	if err != nil {
		respondWithServiceError(w, r, err)
		return
	}

	respondWithJSON(w, http.StatusOK, dtos.ToAdminUserResponse(user))
}

func (h *Handler) DeleteUserAsAdmin(w http.ResponseWriter, r *http.Request) {
	adminID, subjectID, ok := adminAndSubject(w, r)
	if !ok {
		return
	}

	var req dtos.AdminDeleteRequest
	if !decodeStrict(w, r, &req) {
		return
	}

	if err := h.AdminUser.Delete(r.Context(), adminID, subjectID, req.Username, req.Reason); err != nil {
		respondWithServiceError(w, r, err)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func adminAndSubject(w http.ResponseWriter, r *http.Request) (adminID, subjectID uuid.UUID, ok bool) {
	adminID, err := getUserID(r)
	if err != nil {
		respondWithError(w, http.StatusUnauthorized, "Authentication required")
		return uuid.Nil, uuid.Nil, false
	}

	subjectID, err = parseIDParam(r)
	if err != nil {
		respondWithError(w, http.StatusBadRequest, "Invalid user id")
		return uuid.Nil, uuid.Nil, false
	}

	return adminID, subjectID, true
}

func decodeStrict(w http.ResponseWriter, r *http.Request, into any) bool {
	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()

	if err := decoder.Decode(into); err != nil {
		respondWithError(w, http.StatusBadRequest, "Invalid request body")
		return false
	}

	return true
}
