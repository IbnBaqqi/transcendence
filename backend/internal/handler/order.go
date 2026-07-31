package handler

import (
	"encoding/json"
	"net/http"

	"github.com/google/uuid"

	"github.com/IbnBaqqi/transcendence/internal/dtos"
)

// Note what this file does NOT import: "database". Handlers talk to the service
// and to dtos, never to the DB directly. The three helpers used below -
// getUserId, parseIDParam, statusFromServiceError - live in listing.go; they're
// in the same `handler` package, so they're available without an import.

// CreateOrder handles POST /api/v1/orders.
//
// The buyer's identity comes from the request context/header, NOT the body -
// otherwise anyone could place orders as someone else.
func (h *Handler) CreateOrder(w http.ResponseWriter, r *http.Request) {
	userID, err := getUserID(r)
	if err != nil {
		respondWithError(w, http.StatusUnauthorized, "authentication required")
		return
	}

	// Decode reads the request body and fills in the struct. Fields the client
	// didn't send are left at their zero value (0 for int32) - which is why the
	// service still validates listing_id and quantity rather than trusting this.
	var input dtos.CreateOrderInput
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		respondWithError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	order, err := h.Order.CreateOrder(r.Context(), userID, input)
	if err != nil {
		// One line covers every failure the service can produce: 400 for bad
		// input, 404 for a missing listing, 409 for insufficient stock.
		respondWithError(w, statusFromServiceError(err), err.Error())
		return
	}

	// 201 Created, not 200 - a new resource exists now.
	respondWithJSON(w, http.StatusCreated, dtos.NewOrderResponse(order))
}

// GetOrder handles GET /api/v1/orders/{id}.
func (h *Handler) GetOrder(w http.ResponseWriter, r *http.Request) {
	userID, err := getUserID(r)
	if err != nil {
		respondWithError(w, http.StatusUnauthorized, "authentication required")
		return
	}

	// Reads the {id} segment from the URL and converts it to int32.
	id, err := parseIDParam(r)
	if err != nil {
		respondWithError(w, http.StatusBadRequest, "invalid order id")
		return
	}

	// userID is passed in so the service can enforce that only the buyer and
	// the seller may read this order. That check is a business rule, so it
	// belongs in the service, not here.
	order, err := h.Order.GetOrder(r.Context(), userID, id)
	if err != nil {
		respondWithError(w, statusFromServiceError(err), err.Error())
		return
	}

	respondWithJSON(w, http.StatusOK, dtos.NewOrderResponse(order))
}

// GetOrders handles GET /api/v1/orders - every where the caller is
// either the buyer or the seller.
func (h *Handler) GetOrders(w http.ResponseWriter, r *http.Request) {
	userID, err := getUserID(r)
	if err != nil {
		respondWithError(w, http.StatusUnauthorized, "authentication required")
		return
	}

	orders, err := h.Order.ListOrders(r.Context(), userID)
	if err != nil {
		// This path only fails if the DB itself is unhappy, so there's no typed
		// service error to map - a generic 500 with a safe message is right.
		respondWithError(w, http.StatusInternalServerError, "could not fetch orders")
		return
	}

	// NewOrderResponses (plural) guarantees `[]` intead of `null` for a user
	// with no orders yet, so the frontend can always call .map() on it.
	respondWithJSON(w, http.StatusOK, dtos.NewOrderResponses(orders))
}

// --- Status transition endpoints ---
//
// All four are the same six steps: identify the caller, parse {id}, call the
// matching service method, map the error, return the updated order. The service
// owns every rule about who may do what from which state; these handlers
// stay dumb on purpose.

// parseOrderRequest pulls out the two things every status endpoint needs and
// writes the error response itself if either is missin.
//
// It returns a bool rather than an error because the caller has nothing left to
// decide: `ok == false` means "a response was already sent, just return".
// Tha keeps each handler below to a single early-exit line.
func parseOrderRequest(w http.ResponseWriter, r *http.Request) (uuid.UUID, int32, bool) {
	userID, err := getUserID(r)
	if err != nil {
		respondWithError(w, http.StatusUnauthorized, "authentication required")
		return uuid.Nil, 0, false
	}

	id, err := parseIDParam(r)
	if err != nil {
		respondWithError(w, http.StatusBadRequest, "invalid order id")
		return uuid.Nil, 0, false
	}

	return userID, id, true
}

// ConfirmOrder handles POST /api/v1/orders/{id}/confirm - seller accepts.
// pending -> confirm
func (h *Handler) ConfirmOrder(w http.ResponseWriter, r *http.Request) {
	userID, id, ok := parseOrderRequest(w, r)
	if !ok {
		return
	}

	order, err := h.Order.ConfirmOrder(r.Context(), userID, id)
	if err != nil {
		respondWithError(w, statusFromServiceError(err), err.Error())
		return
	}

	respondWithJSON(w, http.StatusOK, dtos.NewOrderResponse(order))
}

// PayOrder handles POST /api/v1/orders/{id}/pay - buyer pays (simulated).
// confirm -> paid
func (h *Handler) PayOrder(w http.ResponseWriter, r *http.Request) {
	userID, id, ok := parseOrderRequest(w, r)
	if !ok {
		return
	}

	order, err := h.Order.PayOrder(r.Context(), userID, id)
	if err != nil {
		respondWithError(w, statusFromServiceError(err), err.Error())
		return
	}

	respondWithJSON(w, http.StatusOK, dtos.NewOrderResponse(order))
}

// CompleteOrder handles POST /api/v1/orders/{id}/complete - seller marks it
// handed over. paid -> completed (terminal)
func (h *Handler) CompleteOrder(w http.ResponseWriter, r *http.Request) {
	userID, id, ok := parseOrderRequest(w, r)
	if !ok {
		return
	}

	order, err := h.Order.CompleteOrder(r.Context(), userID, id)
	if err != nil {
		respondWithError(w, statusFromServiceError(err), err.Error())
		return
	}

	respondWithJSON(w, http.StatusOK, dtos.NewOrderResponse(order))
}

// CancelOrder handles POST /api/v1/orders/{id}/cancel - either side backs out.
// pending|confirmed -> cancelled (terminal, and restores the listings's stock)
func (h *Handler) CancelOrder(w http.ResponseWriter, r *http.Request) {
	userID, id, ok := parseOrderRequest(w, r)
	if !ok {
		return
	}

	order, err := h.Order.CancelOrder(r.Context(), userID, id)
	if err != nil {
		respondWithError(w, statusFromServiceError(err), err.Error())
		return
	}

	respondWithJSON(w, http.StatusOK, dtos.NewOrderResponse(order))
}
