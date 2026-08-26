package handler

import (
	"net/http"

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
