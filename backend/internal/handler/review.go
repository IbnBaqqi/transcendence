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

	review, err := h.Review.Create(r.Context(), userID, orderID, req.Rating, commentOf(req.Comment))
	if err != nil {
		respondWithServiceError(w, r, err)
		return
	}

	respondWithJSON(w, http.StatusCreated, dtos.ToReviewResponse(review, viewerName(r)))
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

	respondWithJSON(w, http.StatusOK, dtos.ToReviewResponse(review, viewerName(r)))
}

func (h *Handler) GetSellerReviews(w http.ResponseWriter, r *http.Request) {
	sellerID, err := parseIDParam(r)
	if err != nil {
		respondWithError(w, http.StatusBadRequest, "Invalid user id")
		return
	}

	// These are what renders on the seller's profile, which a block already
	// answers with 404 - leaving them readable by user id would hand a blocked
	// viewer the reputation and the review text off the page they cannot open.
	if h.blockedBetween(r, viewerID(r), sellerID) {
		respondWithError(w, http.StatusNotFound, "User not found")
		return
	}

	q := r.URL.Query()

	page, err := h.Review.ListForSeller(r.Context(), sellerID, dtos.ReviewQuery{
		Page:  q.Get("page"),
		Limit: q.Get("limit"),
	})
	if err != nil {
		respondWithServiceError(w, r, err)
		return
	}

	respondWithJSON(w, http.StatusOK, page)
}

func (h *Handler) GetOrderReview(w http.ResponseWriter, r *http.Request) {
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

	review, reviewer, err := h.Review.GetForOrder(r.Context(), userID, orderID)
	if err != nil {
		respondWithServiceError(w, r, err)
		return
	}

	respondWithJSON(w, http.StatusOK, dtos.ToReviewResponse(review, reviewer))
}

// commentOf flattens the optional to a plain string for create, where absent
// and empty mean the same thing - there is no prior text to preserve.
func commentOf(c dtos.OptionalString) string {
	if c.Set && c.Value != nil {
		return *c.Value
	}
	return ""
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
