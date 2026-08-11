package dtos

import "github.com/IbnBaqqi/transcendence/internal/database"

// UploadURLPrefix is the public path the router serves uploaded files from.
const UploadURLPrefix = "/uploads/"

// ListingImageResponse is the public JSON for one poto.
type ListingImageResponse struct {
	ID       int32  `json:"id"`
	URL      string `json:"url"`
	Position int32  `json:"position"`
}

// ToListingImageResponse maps a single image row into the response dto.
func ToListingImageResponse(img database.ListingImage) ListingImageResponse {
	return ListingImageResponse{
		ID:       img.ID,
		URL:      UploadURLPrefix + img.Filename,
		Position: img.Position,
	}
}

// ToListingImageResponses maps multiple image rows to a response slice.
func ToListingImageResponses(rows []database.ListingImage) []ListingImageResponse {
	out := make([]ListingImageResponse, 0, len(rows))
	for _, r := range rows {
		out = append(out, ToListingImageResponse(r))
	}
	return out
}
