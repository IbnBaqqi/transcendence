package dtos

import "github.com/IbnBaqqi/transcendence/internal/database"

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

// --- Search DTOs ---

// ListingSearchQuery holds raw, unparsed values pulled from URL query
// params. Parsing/validation happens in service layer.
type ListingSearchQuery struct {
	Keyword  string
	Category string
	MinPrice string
	MaxPrice string
	Location string
	Page     string
	Limit    string
}

type PaginatedListings struct {
	Items      []database.Listing `json:"items"`
	Total      int64              `json:"total"`
	Page       int                `json:"page"`
	Limit      int                `json:"limit"`
	TotalPages int                `json:"total_pages"`
}
