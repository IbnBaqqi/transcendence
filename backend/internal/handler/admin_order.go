package handler

import (
	"encoding/json"
	"net/http"

	"github.com/IbnBaqqi/transcendence/internal/dtos"
)

func (h *Handler) ListOrdersForAdmin(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()

	page, err := h.AdminOrder.List(r.Context(), dtos.AdminOrderQuery{
		Status:      q.Get("status"),
		CreatedFrom: q.Get("created_from"),
		CreatedTo:   q.Get("created_to"),
		Stuck:       q.Get("stuck"),
		Page:        q.Get("page"),
		Limit:       q.Get("limit"),
	})
	if err != nil {
		respondWithServiceError(w, r, err)
		return
	}

	respondWithJSON(w, http.StatusOK, page)
}

func (h *Handler) ResolveOrder(w http.ResponseWriter, r *http.Request) {
	adminID, err := getUserID(r)
	if err != nil {
		respondWithError(w, http.StatusUnauthorized, "Authentication required")
		return
	}

	orderID, err := parseIDParam(r)
	if err != nil {
		respondWithError(w, http.StatusBadRequest, "Invalid order id")
		return
	}

	var req dtos.ResolveOrderRequest

	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()

	if err := decoder.Decode(&req); err != nil {
		respondWithError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	order, err := h.AdminOrder.Resolve(r.Context(), adminID, orderID, req.Outcome, req.Reason)
	if err != nil {
		respondWithServiceError(w, r, err)
		return
	}

	respondWithJSON(w, http.StatusOK, dtos.ToAdminOrderResponse(order, false))
}
