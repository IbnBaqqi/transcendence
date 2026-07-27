package service

import (
	"context"
	"database/sql"
	"math"
	"strconv"

	"github.com/IbnBaqqi/transcendence/internal/database"
	"github.com/IbnBaqqi/transcendence/internal/dtos"
	"github.com/google/uuid"
)

// ListingService contains the business logic for listings: validation,
// ownership rules, and orchestration of database calls. It knows
// nothing about HTTP.
type ListingService struct {
	db *database.Queries
}

func NewListingService(db *database.Queries) *ListingService {
	return &ListingService{db: db}
}

func validateListingInput(title, category, unit string, price float64, quantity int32) error {
	if title == "" || len(title) > 100 {
		return &ValidationError{Message: "title is required and must be under 100 characters"}
	}
	if category == "" {
		return &ValidationError{Message: "category is required"}
	}
	if unit == "" {
		return &ValidationError{Message: "unit is required"}
	}
	if price <= 0 {
		return &ValidationError{Message: "price must be greater than 0"}
	}
	if quantity <= 0 {
		return &ValidationError{Message: "Quantity must be greater than 0"}
	}
	return nil
}

func (s *ListingService) CreateListing(ctx context.Context, sellerID uuid.UUID, input dtos.CreateListingInput) (database.Listing, error) {
	if err := validateListingInput(input.Title, input.Category, input.Unit, input.Price, input.Quantity); err != nil {
		return database.Listing{}, err
	}

	return s.db.CreateListing(ctx, database.CreateListingParams{
		SellerID:    sellerID,
		Title:       input.Title,
		Description: sql.NullString{String: input.Description, Valid: input.Description != ""},
		Category:    input.Category,
		Price:       strconv.FormatFloat(input.Price, 'f', 2, 64),
		Quantity:    input.Quantity,
		Unit:        input.Unit,
	})
}

func (s *ListingService) GetListing(ctx context.Context, id int32) (database.Listing, error) {
	listing, err := s.db.GetListing(ctx, id)
	if err != nil {
		return database.Listing{}, &NotFoundError{Message: "listing not found"}
	}
	return listing, nil
}

func (s *ListingService) ListListings(ctx context.Context) ([]database.Listing, error) {
	return s.db.ListListings(ctx)
}

func (s *ListingService) UpdateListing(ctx context.Context, userID uuid.UUID, listingID int32, input dtos.UpdateListingInput) (database.Listing, error) {
	existing, err := s.db.GetListing(ctx, listingID)
	if err != nil {
		return database.Listing{}, &NotFoundError{Message: "listing not found"}
	}
	if existing.SellerID != userID {
		return database.Listing{}, &ForbiddenError{Message: "you do not own this listing"}
	}

	if err := validateListingInput(input.Title, input.Category, input.Unit, input.Price, input.Quantity); err != nil {
		return database.Listing{}, err
	}

	return s.db.UpdateListing(ctx, database.UpdateListingParams{
		ID:          listingID,
		Title:       input.Title,
		Description: sql.NullString{String: input.Description, Valid: input.Description != ""},
		Category:    input.Category,
		Price:       strconv.FormatFloat(input.Price, 'f', 2, 64),
		Quantity:    input.Quantity,
		Unit:        input.Unit,
	})
}

func (s *ListingService) DeleteListing(ctx context.Context, userID uuid.UUID, listingID int32) error {
	existing, err := s.db.GetListing(ctx, listingID)
	if err != nil {
		return &NotFoundError{Message: "listing not found"}
	}
	if existing.SellerID != userID {
		return &ForbiddenError{Message: "you do not own this listing"}
	}

	return s.db.DeleteListing(ctx, listingID)
}

const (
	defaultPage  = 1
	defaultLimit = 20
	maxLimit     = 50
)

func (s *ListingService) SearchListings(ctx context.Context, q dtos.ListingSearchQuery) (dtos.PaginatedListings, error) {
	page := defaultPage
	if q.Page != "" {
		p, err := strconv.Atoi(q.Page)
		if err != nil || p < 1 {
			return dtos.PaginatedListings{}, &ValidationError{Message: "page must be a positive integer"}
		}
		page = p
	}

	limit := defaultLimit
	if q.Limit != "" {
		l, err := strconv.Atoi(q.Limit)
		if err != nil || l < 1 {
			return dtos.PaginatedListings{}, &ValidationError{Message: "limit must be a positive integer"}
		}
		limit = l
	}
	if limit > maxLimit {
		limit = maxLimit
	}

	var minPrice, maxPrice sql.NullString
	var minVal, maxVal float64

	if q.MinPrice != "" {
		v, err := strconv.ParseFloat(q.MinPrice, 64)
		if err != nil || v < 0 {
			return dtos.PaginatedListings{}, &ValidationError{Message: "min_price must be a non-negative number"}
		}
		minVal = v
		minPrice = sql.NullString{String: strconv.FormatFloat(v, 'f', 2, 64), Valid: true}
	}
	if q.MaxPrice != "" {
		v, err := strconv.ParseFloat(q.MaxPrice, 64)
		if err != nil || v < 0 {
			return dtos.PaginatedListings{}, &ValidationError{Message: "max_price must be a non-negative number"}
		}
		maxVal = v
		maxPrice = sql.NullString{String: strconv.FormatFloat(v, 'f', 2, 64), Valid: true}
	}
	if minPrice.Valid && maxPrice.Valid && minVal > maxVal {
		return dtos.PaginatedListings{}, &ValidationError{Message: "min_price must not exceed max_price"}
	}

	offset := (page - 1) * limit
	if offset < 0 || offset > math.MaxInt32 {
		return dtos.PaginatedListings{}, &ValidationError{Message: "page is too large"}
	}

	params := database.SearchListingsParams{
		Keyword:  q.Keyword,
		Category: q.Category,
		Location: q.Location,
		Offset:   int32(offset),
		Limit:    int32(limit),
	}
	if minPrice.Valid {
		params.MinPrice = minPrice.String
	}
	if maxPrice.Valid {
		params.MaxPrice = maxPrice.String
	}

	items, err := s.db.SearchListingsDynamic(ctx, params)
	if err != nil {
		return dtos.PaginatedListings{}, err
	}
	total, err := s.db.CountSearchListingsDynamic(ctx, params)
	if err != nil {
		return dtos.PaginatedListings{}, err
	}

	totalPages := int((total + int64(limit) - 1) / int64(limit))

	return dtos.PaginatedListings{
		Items:      items,
		Total:      total,
		Page:       page,
		Limit:      limit,
		TotalPages: totalPages,
	}, nil
}
