package service

import (
	"context"
	"database/sql"
	"strconv"

	"github.com/IbnBaqqi/transcendence/internal/database"
	"github.com/IbnBaqqi/transcendence/internal/dtos"
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

func (s *ListingService) CreateListing(ctx context.Context, sellerID int32, input dtos.CreateListingInput) (database.Listing, error) {
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

func (s *ListingService) UpdateListing(ctx context.Context, userID, listingID int32, input dtos.UpdateListingInput) (database.Listing, error) {
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

func (s *ListingService) DeleteListing(ctx context.Context, userID, listingID int32) error {
	existing, err := s.db.GetListing(ctx, listingID)
	if err != nil {
		return &NotFoundError{Message: "listing not found"}
	}
	if existing.SellerID != userID {
		return &ForbiddenError{Message: "you do not own this listing"}
	}

	return s.db.DeleteListing(ctx, listingID)
}
