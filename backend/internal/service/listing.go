package service

import (
	"context"
	"database/sql"
	"errors"
	"log/slog"
	"math"
	"strconv"

	"github.com/IbnBaqqi/transcendence/internal/database"
	"github.com/IbnBaqqi/transcendence/internal/dtos"
	"github.com/google/uuid"
)

// ListingService contains the business logic for listings: validation,
// ownership rules, and orchestration of database calls. It knows
// nothing about HTTP.
//
// It takes *database.DB rather than *database.Queries because UpdateListing
// needs a transaction (BeginTx), which only DB can start. DB embeds *Queries,
// so the simple non-transactional calls like s.db.GetListing(...) still work
// exactly as before.
type ListingService struct {
	db *database.DB
}

func NewListingService(db *database.DB) *ListingService {
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

// UpdateListing edits a listing the caller owns.
//
// The whole read-check-write runs inside ONE transaction holding SELECT ... FOR
// UPDATE on the row, for the same reason CreateOrder does: `quantity` is live
// stock that orders decrement and cancellations restore. Reading it unlocked
// and writing it back afterwards is a classic lost update - an order that
// commits in between gets silently overwritten, and the sold-out check below
// can pass on a value that is already stale.
func (s *ListingService) UpdateListing(ctx context.Context, userID uuid.UUID, listingID int32, input dtos.UpdateListingInput) (database.Listing, error) {
	// Validate first: bad input should never take a row lock.
	if err := validateListingInput(input.Title, input.Category, input.Unit, input.Price, input.Quantity); err != nil {
		return database.Listing{}, err
	}

	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return database.Listing{}, err
	}
	defer func() {
		// A committed transaction returns ErrTxDone here, which is the happy
		// path - only log a rollback that actually went wrong.
		if err := tx.Rollback(); err != nil && !errors.Is(err, sql.ErrTxDone) {
			slog.Error("listing update transaction rollback failed", "error", err)
		}
	}()

	qtx := s.db.Queries.WithTx(tx.Tx)

	// FOR UPDATE holds this row until we commit, so a concurrent CreateOrder or
	// CancelOrder queues behind us rather than interleaving with our write.
	existing, err := qtx.GetListingForUpdate(ctx, listingID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return database.Listing{}, &NotFoundError{Message: "listing not found"}
		}
		return database.Listing{}, err
	}

	if existing.SellerID != userID {
		return database.Listing{}, &ForbiddenError{Message: "you do not own this listing"}
	}

	// Checked against the LOCKED row. Unlocked, this could read 1 while an order
	// in flight was taking it to 0, letting a sold-out listing be edited anyway.
	if existing.Quantity == 0 {
		return database.Listing{}, &ConflictError{Message: "listing is sold out and can no longer be edited; create new listing"}
	}

	updated, err := qtx.UpdateListing(ctx, database.UpdateListingParams{
		ID:          listingID,
		Title:       input.Title,
		Description: sql.NullString{String: input.Description, Valid: input.Description != ""},
		Category:    input.Category,
		Price:       strconv.FormatFloat(input.Price, 'f', 2, 64),
		Quantity:    input.Quantity,
		Unit:        input.Unit,
	})
	if err != nil {
		return database.Listing{}, err
	}

	if err := tx.Commit(); err != nil {
		return database.Listing{}, err
	}

	return updated, nil
}

// DeleteListing removes a listing the caller owns.
//
// Locked and transactional for the same reason as UpdateListing, plus one extra
// rule: a listing that any order references cannot be deleted at all. Orders are
// historical records - a buyer's completed order must not vanish because the
// seller tidied up their listings. The foreign key is ON DELETE RESTRICT
// (migration 008) as a backstop, so even a code path that skips this check
// cannot destroy order history.
//
// Holding FOR UPDATE across the count and the delete is what stops a new order
// slipping in between them: CreateOrder takes the same lock, so it queues behind
// us instead of racing.
func (s *ListingService) DeleteListing(ctx context.Context, userID uuid.UUID, listingID int32) error {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer func() {
		if err := tx.Rollback(); err != nil && !errors.Is(err, sql.ErrTxDone) {
			slog.Error("listing delete transaction rollback failed", "error", err)
		}
	}()

	qtx := s.db.Queries.WithTx(tx.Tx)

	existing, err := qtx.GetListingForUpdate(ctx, listingID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return &NotFoundError{Message: "listing not found"}
		}
		return err
	}
	if existing.SellerID != userID {
		return &ForbiddenError{Message: "you do not own this listing"}
	}

	orderCount, err := qtx.CountOrdersForListing(ctx, listingID)
	if err != nil {
		return err
	}
	if orderCount > 0 {
		return &ConflictError{Message: "this listing has orders and cannot be deleted; its order history has to be kept"}
	}

	if err := qtx.DeleteListing(ctx, listingID); err != nil {
		return err
	}

	return tx.Commit()
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
		Items:      dtos.ToListingResponses(items),
		Total:      total,
		Page:       page,
		Limit:      limit,
		TotalPages: totalPages,
	}, nil
}
