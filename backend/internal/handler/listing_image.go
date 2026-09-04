package handler

import (
	"encoding/json"
	"net/http"

	"github.com/google/uuid"

	"github.com/IbnBaqqi/transcendence/internal/dtos"
)

func (h *Handler) UploadListingImage(w http.ResponseWriter, r *http.Request) {
	userID, err := getUserID(r)
	if err != nil {
		respondWithError(w, http.StatusUnauthorized, "Authentication required")
		return
	}

	listingID, err := parseIDParam(r)
	if err != nil {
		respondWithError(w, http.StatusBadRequest, "Invalid listing id")
		return
	}

	upload, ok := h.readImageUpload(w, r, "image")
	if !ok {
		return
	}
	defer upload.Close()

	img, err := h.ListingImage.AddImage(r.Context(), userID, listingID, upload.Body, upload.Ext)
	if err != nil {
		if isTooLarge(err) {
			respondWithError(w, http.StatusRequestEntityTooLarge, "Image is too large")
			return
		}
		respondWithServiceError(w, r, err)
		return
	}

	respondWithJSON(w, http.StatusCreated, dtos.ToListingImageResponse(img))
}

func (h *Handler) GetListingImages(w http.ResponseWriter, r *http.Request) {
	listingID, err := parseIDParam(r)
	if err != nil {
		respondWithError(w, http.StatusBadRequest, "Invalid listing id")
		return
	}

	listing, err := h.Listing.GetListing(r.Context(), listingID)
	if err != nil {
		respondWithServiceError(w, r, err)
		return
	}

	hidden := listing.RemovedAt.Valid || h.sellerIsHidden(r, listing.SellerID)
	if hidden && !h.maySeeRemovedListing(r, listing.SellerID) {
		respondWithError(w, http.StatusNotFound, "Listing not found")
		return
	}

	imgs, err := h.ListingImage.ListImages(r.Context(), listingID)
	if err != nil {
		respondWithServiceError(w, r, err)
		return
	}
	respondWithJSON(w, http.StatusOK, dtos.ToListingImageResponses(imgs))
}

func (h *Handler) DeleteListingImage(w http.ResponseWriter, r *http.Request) {
	userID, err := getUserID(r)
	if err != nil {
		respondWithError(w, http.StatusUnauthorized, "Authentication required")
		return
	}

	listingID, err := parseIDParam(r)
	if err != nil {
		respondWithError(w, http.StatusBadRequest, "Invalid listing id")
		return
	}

	imageID, err := parseUUIDParam(r, "imageID")
	if err != nil {
		respondWithError(w, http.StatusBadRequest, "Invalid image id")
		return
	}

	if err := h.ListingImage.DeleteImage(r.Context(), userID, listingID, imageID); err != nil {
		respondWithServiceError(w, r, err)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func (h *Handler) ReorderListingImages(w http.ResponseWriter, r *http.Request) {
	userID, err := getUserID(r)
	if err != nil {
		respondWithError(w, http.StatusUnauthorized, "Authentication required")
		return
	}

	listingID, err := parseIDParam(r)
	if err != nil {
		respondWithError(w, http.StatusBadRequest, "Invalid listing id")
		return
	}

	var req dtos.ReorderImagesRequest
	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&req); err != nil {
		respondWithError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	ids := make([]uuid.UUID, 0, len(req.ImageIDs))
	for _, raw := range req.ImageIDs {
		id, parseErr := uuid.Parse(raw)
		if parseErr != nil {
			respondWithError(w, http.StatusBadRequest, "Invalid image id")
			return
		}
		ids = append(ids, id)
	}

	if err := h.ListingImage.ReorderImages(r.Context(), userID, listingID, ids); err != nil {
		respondWithServiceError(w, r, err)
		return
	}

	imgs, err := h.ListingImage.ListImages(r.Context(), listingID)
	if err != nil {
		respondWithServiceError(w, r, err)
		return
	}

	respondWithJSON(w, http.StatusOK, dtos.ToListingImageResponses(imgs))
}
