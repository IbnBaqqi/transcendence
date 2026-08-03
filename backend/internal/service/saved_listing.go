package service

import (
	"context"

	"github.com/IbnBaqqi/transcendence/internal/database"
	"github.com/google/uuid"
)

// SavedListingService holds the wishlist rules. Same shape as ListingService:
// a db handle and nothing else - no http.Request, no ResponseWriter. Keeping
// HTTP out of here is what lets the rules be tested without a server.
type SavedListingService struct {
	db *database.Queries
}

func NewSavedListingService(db *database.Queries) *SavedListingService {
	return &SavedListingService{db: db}
}

// SaveListing bookmarks a listing for a user.
//
// We look the listing up first so a bad id produces a clear 404. Without that
// check the insert would trip the foreign key instead, and a raw FK violation
// srfaces as a vague 500.
func (s *SavedListingService) SaveListing(ctx context.Context, userID uuid.UUID, listingID int32) error {
	if _, err := s.db.GetListing(ctx, listingID); err != nil {
		return &NotFoundError{Message: "listing not found"}
	}

	// A repeat save is deliberately NOT an error: the query's ON CONFLICT DO
	// NOTHING makes this idempotent, so double-clicking the heart is harmless.
	return s.db.SaveListing(ctx, database.SaveListingParams{
		UserID: userID,
		ListingID: listingID,
	})
}

// UnsaveListing removes a bookmark, reporting 404 when there wasn't ine.
func (s *SavedListingService) UnsaveListing(ctx context.Context, userID uuid.UUID, listingID int32) error {
	// rows is the affected-row count that :execrow gives us. 0 means the
	// DELETE matched nothing - the user never saved this listing.
	rows, err := s.db.UnsaveListing(ctx, database.UnsaveListingParams{
		UserID: userID,
		ListingID: listingID,
	})
	if err != nil {
		return err
	}
	if rows == 0 {
		return &NotFoundError{Message: "listing was not saved"}
	}
	return nil
}

// ListSaved restuns the user's wishlist, most recently saved first.
//
// It return []database.Listing (raw rows) rather than DTOs on purpose -
// mapping to the public JSON shape is the handler's job, same split as
// ListingService.
func (s *SavedListingService) ListSaved(ctx context.Context, userID uuid.UUID) ([]database.Listing, error) {
	return s.db.ListSaveListings(ctx, userID)
}