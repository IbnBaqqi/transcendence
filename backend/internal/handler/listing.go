package handler

import (
	"encoding/json"
	"log/slog"
	"net/http"

	"github.com/go-chi/chi/v5/middleware"

	"github.com/google/uuid"

	"github.com/IbnBaqqi/transcendence/internal/auth"
	"github.com/IbnBaqqi/transcendence/internal/dtos"
)

// maySeeRemovedListing decides who may still see a hidden listing. A removed
// one must look like it never existed, so every route that can reach one asks
// this rather than re-deriving the rule.
//
// The admin case re-reads the role from the database rather than trusting the
// token's claim, matching mw.RequireRole - otherwise a demoted admin keeps
// seeing removed listings until their access token expires.
func (h *Handler) maySeeRemovedListing(r *http.Request, sellerID uuid.UUID) bool {
	viewer, ok := auth.UserFromContext(r.Context())
	if !ok {
		return false
	}
	if viewer.ID == sellerID {
		return true
	}

	role, err := h.db.GetUserRole(r.Context(), viewer.ID)
	if err != nil {
		return false
	}
	return role == auth.RoleAdmin
}

// sellerIsHidden reports whether the listing's seller is deleted or suspended.
// Kept out of the shared GetListing query on purpose: every service reads
// through it, and filtering there would 404 a reported listing for the admin
// judging the report.
//
// Fails closed, and says so in the log: without the line an outage would look
// like listings quietly vanishing.
func (h *Handler) sellerIsHidden(r *http.Request, sellerID uuid.UUID) bool {
	visible, err := h.db.UserIsVisible(r.Context(), sellerID)
	if err != nil {
		slog.Error("could not check whether the seller is visible, hiding the listing",
			"seller_id", sellerID, "request_id", middleware.GetReqID(r.Context()), "error", err)
		return true
	}
	return !visible
}

func (h *Handler) CreateListing(w http.ResponseWriter, r *http.Request) {
	userID, err := getUserID(r)
	if err != nil {
		respondWithError(w, http.StatusUnauthorized, "Authentication required")
		return
	}

	var input dtos.CreateListingInput
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		respondWithError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	listing, err := h.Listing.CreateListing(r.Context(), userID, input)
	if err != nil {
		respondWithServiceError(w, r, err)
		return
	}

	tags, err := h.Listing.TagsForListing(r.Context(), listing.ID)
	if err != nil {
		respondWithServiceError(w, r, err)
		return
	}

	respondWithJSON(w, http.StatusCreated, dtos.WithTags(dtos.ToListingResponse(listing), tags))
}

func (h *Handler) GetListings(w http.ResponseWriter, r *http.Request) {
	listings, err := h.Listing.ListListings(r.Context())
	if err != nil {
		respondWithServiceError(w, r, err)
		return
	}

	ids := make([]uuid.UUID, 0, len(listings))
	for _, l := range listings {
		ids = append(ids, l.ID)
	}

	byListing, err := h.ListingImage.ImagesByListing(r.Context(), ids)
	if err != nil {
		respondWithServiceError(w, r, err)
		return
	}

	tagsByListing, err := h.Listing.TagsByListing(r.Context(), ids)
	if err != nil {
		respondWithServiceError(w, r, err)
		return
	}

	respondWithJSON(w, http.StatusOK,
		dtos.WithTagsEach(dtos.ToListingResponsesWithImages(listings, byListing), tagsByListing))
}

func (h *Handler) GetListing(w http.ResponseWriter, r *http.Request) {
	id, err := parseIDParam(r)
	if err != nil {
		respondWithError(w, http.StatusBadRequest, "Invalid listing id")
		return
	}

	listing, err := h.Listing.GetListing(r.Context(), id)
	if err != nil {
		respondWithServiceError(w, r, err)
		return
	}

	hidden := listing.RemovedAt.Valid || h.sellerIsHidden(r, listing.SellerID)
	if hidden && !h.maySeeRemovedListing(r, listing.SellerID) {
		respondWithError(w, http.StatusNotFound, "Listing not found")
		return
	}

	imgs, err := h.ListingImage.ListImages(r.Context(), id)
	if err != nil {
		respondWithServiceError(w, r, err)
		return
	}

	tags, err := h.Listing.TagsForListing(r.Context(), id)
	if err != nil {
		respondWithServiceError(w, r, err)
		return
	}

	respondWithJSON(w, http.StatusOK, dtos.WithTags(dtos.ToListingResponseWithImages(listing, imgs), tags))
}

func (h *Handler) UpdateListing(w http.ResponseWriter, r *http.Request) {
	userID, err := getUserID(r)
	if err != nil {
		respondWithError(w, http.StatusUnauthorized, "Authentication required")
		return
	}

	id, err := parseIDParam(r)
	if err != nil {
		respondWithError(w, http.StatusBadRequest, "Invalid listing id")
		return
	}

	var input dtos.UpdateListingInput
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		respondWithError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	updated, err := h.Listing.UpdateListing(r.Context(), userID, id, input)
	if err != nil {
		respondWithServiceError(w, r, err)
		return
	}

	imgs, err := h.ListingImage.ListImages(r.Context(), id)
	if err != nil {
		respondWithServiceError(w, r, err)
		return
	}

	tags, err := h.Listing.TagsForListing(r.Context(), id)
	if err != nil {
		respondWithServiceError(w, r, err)
		return
	}

	respondWithJSON(w, http.StatusOK, dtos.WithTags(dtos.ToListingResponseWithImages(updated, imgs), tags))
}

func (h *Handler) DeleteListing(w http.ResponseWriter, r *http.Request) {
	userID, err := getUserID(r)
	if err != nil {
		respondWithError(w, http.StatusUnauthorized, "Authentication required")
		return
	}

	id, err := parseIDParam(r)
	if err != nil {
		respondWithError(w, http.StatusBadRequest, "Invalid listing id")
		return
	}

	if err := h.Listing.DeleteListing(r.Context(), userID, id); err != nil {
		respondWithServiceError(w, r, err)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func (h *Handler) SearchListings(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	query := dtos.ListingSearchQuery{
		Keyword:  q.Get("keyword"),
		Category: q.Get("category"),
		Tag:      q.Get("tag"),
		MinPrice: q.Get("min_price"),
		MaxPrice: q.Get("max_price"),
		Location: q.Get("location"),
		SellerID: q.Get("seller_id"),
		Sort:     q.Get("sort"),
		Page:     q.Get("page"),
		Limit:    q.Get("limit"),
	}

	result, err := h.Listing.SearchListings(r.Context(), query)
	if err != nil {
		respondWithServiceError(w, r, err)
		return
	}

	respondWithJSON(w, http.StatusOK, result)
}

func (h *Handler) GetCategories(w http.ResponseWriter, r *http.Request) {
	rows, err := h.Listing.ListCategories(r.Context())
	if err != nil {
		respondWithServiceError(w, r, err)
		return
	}

	respondWithJSON(w, http.StatusOK, dtos.ToCategoryResponses(rows))
}
