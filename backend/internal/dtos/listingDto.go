package dtos

import (
	"time"

	"github.com/google/uuid"

	"github.com/IbnBaqqi/transcendence/internal/database"
)

// --- Request DTOs ---

type CreateListingInput struct {
	Title       string  `json:"title"`
	Description string  `json:"description"`
	Category    string  `json:"category"`
	Price       float64 `json:"price"`
	Quantity    int32   `json:"quantity"`
	Unit        string  `json:"unit"`
}

type UpdateListingInput struct {
	Title       string  `json:"title"`
	Description string  `json:"description"`
	Category    string  `json:"category"`
	Price       float64 `json:"price"`
	Quantity    int32   `json:"quantity"`
	Unit        string  `json:"unit"`
}

type ListingSearchQuery struct {
	Keyword  string
	Category string
	MinPrice string
	MaxPrice string
	Location string
	Sort     string
	Page     string
	Limit    string
}

type PaginatedListings struct {
	Items      []ListingResponse `json:"items"`
	Total      int64             `json:"total"`
	Page       int               `json:"page"`
	Limit      int               `json:"limit"`
	TotalPages int               `json:"total_pages"`
}

type ListingResponse struct {
	ID          int32                  `json:"id"`
	SellerID    uuid.UUID              `json:"seller_id"`
	Title       string                 `json:"title"`
	Description string                 `json:"description"`
	Category    string                 `json:"category"`
	Price       float64                `json:"price"`
	Quantity    int32                  `json:"quantity"`
	Unit        string                 `json:"unit"`
	CreatedAt   time.Time              `json:"created_at"`
	UpdatedAt   time.Time              `json:"updated_at"`
	Images      []ListingImageResponse `json:"images"`
}

func ToListingResponse(l database.Listing) ListingResponse {
	price := numericToFloat(l.Price)

	return ListingResponse{
		ID:          l.ID,
		SellerID:    l.SellerID,
		Title:       l.Title,
		Description: l.Description.String,
		Category:    l.Category,
		Price:       price,
		Quantity:    l.Quantity,
		Unit:        l.Unit,
		CreatedAt:   l.CreatedAt.Time,
		UpdatedAt:   l.UpdatedAt.Time,
		Images:      []ListingImageResponse{},
	}
}

func ToListingResponses(rows []database.Listing) []ListingResponse {
	out := make([]ListingResponse, 0, len(rows))
	for _, r := range rows {
		out = append(out, ToListingResponse(r))
	}
	return out
}

func ToListingResponseWithImages(l database.Listing, imgs []database.ListingImage) ListingResponse {
	res := ToListingResponse(l)
	res.Images = ToListingImageResponses(imgs)
	return res
}

func ToListingResponsesWithImages(
	rows []database.Listing,
	byListing map[int32][]database.ListingImage,
) []ListingResponse {
	out := make([]ListingResponse, 0, len(rows))
	for _, r := range rows {
		out = append(out, ToListingResponseWithImages(r, byListing[r.ID]))
	}
	return out
}
