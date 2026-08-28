package handler

import (
	"encoding/json"
	"net/http"

	"github.com/IbnBaqqi/transcendence/internal/dtos"
)

func (h *Handler) CreateReview(w http.ResponseWriter, r *http.Request) {
	userID, err := getUserID(r)
	if err != nil {
		respondWithError(w, http.StatusUnauthorized, "Authentication required")
		return
	}

	orderID, err := parseIDParam(r)
	if err != nil {
		respondWithError(w, http.StatusBadRequest, "Invalid order id")
		return
	}

	req, ok := decodeReview(w, r)
	if !ok {
		return
	}

	review, err := h.Review.Create(r.Context(), userID, orderID, req.Rating, req.Comment)
	if err != nil {
		respondWithServiceError(w, r, err)
		return
	}

	respondWithJSON(w, http.StatusCreated, dtos.ToReviewResponse(review, ""))
}

func (h *Handler) UpdateReview(w http.ResponseWriter, r *http.Request) {
	userID, err := getUserID(r)
	if err != nil {
		respondWithError(w, http.StatusUnauthorized, "Authentication required")
		return
	}

	reviewID, err := parseIDParam(r)
	if err != nil {
		respondWithError(w, http.StatusBadRequest, "Invalid review id")
		return
	}

	req, ok := decodeReview(w, r)
	if !ok {
		return
	}

	review, err := h.Review.Update(r.Context(), userID, reviewID, req.Rating, req.Comment)
	if err != nil {
		respondWithServiceError(w, r, err)
		return
	}

	respondWithJSON(w, http.StatusOK, dtos.ToReviewResponse(review, ""))
}

func (h *Handler) GetSellerReviews(w http.ResponseWriter, r *http.Request) {
	sellerID, err := parseIDParam(r)
	if err != nil {
		respondWithError(w, http.StatusBadRequest, "Invalid user id")
		return
	}

	rows, err := h.Review.ListForSeller(r.Context(), sellerID)
	if err != nil {
		respondWithServiceError(w, r, err)
		return
	}

	respondWithJSON(w, http.StatusOK, dtos.ToReviewResponses(rows))
}

func decodeReview(w http.ResponseWriter, r *http.Request) (dtos.ReviewRequest, bool) {
	var req dtos.ReviewRequest

	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()

	if err := decoder.Decode(&req); err != nil {
		respondWithError(w, http.StatusBadRequest, "Invalid request body")
		return dtos.ReviewRequest{}, false
	}

	return req, true
}
