package service

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"io"
	"log/slog"

	"github.com/IbnBaqqi/transcendence/internal/database"
	"github.com/google/uuid"
)

// fileStore is the slice of storage.Local this service actually uses.
type fileStore interface {
	Save(r io.Reader, ext string) (string, error)
	Delete(name string) error
}

// ListingImageService holds the rules for listing photos.
type ListingImageService struct {
	db            *database.DB
	files         fileStore
	maxPerListing int
}

func NewListingImageService(db *database.DB, files fileStore, maxPerListing int) *ListingImageService {
	return &ListingImageService{
		db:            db,
		files:         files,
		maxPerListing: maxPerListing,
	}
}

func (s *ListingImageService) ownedListing(ctx context.Context, userID uuid.UUID, listingID int32) error {
	listing, err := s.db.GetListing(ctx, listingID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return &NotFoundError{Message: "listing not found"}
		}
		return err
	}
	if listing.SellerID != userID {
		return &ForbiddenError{Message: "you do not own this listing"}
	}
	return nil
}

// AddImage stores an uploaded file and records it against the listing.
func (s *ListingImageService) AddImage(
	ctx context.Context,
	userID uuid.UUID,
	listingID int32,
	r io.Reader,
	ext string,
) (database.ListingImage, error) {
	if err := s.ownedListing(ctx, userID, listingID); err != nil {
		return database.ListingImage{}, err
	}

	count, err := s.db.CountListingImages(ctx, listingID)
	if err != nil {
		return database.ListingImage{}, err
	}
	if count >= int64(s.maxPerListing) {
		return database.ListingImage{}, &ConflictError{
			Message: fmt.Sprintf("a listing can have at most %d images", s.maxPerListing),
		}
	}

	filename, err := s.files.Save(r, ext)
	if err != nil {
		return database.ListingImage{}, err
	}

	img, err := s.db.CreateListingImage(ctx, database.CreateListingImageParams{
		ListingID: listingID,
		Filename:  filename,
	})
	if err != nil {
		if delErr := s.files.Delete(filename); delErr != nil {
			slog.Error("orphaned upload: file written but row insert failed",
				"filename", filename, "error", delErr)
		}
		return database.ListingImage{}, err
	}

	return img, nil
}

// ListImages returns one listing's photos in display order.
func (s *ListingImageService) ListImages(ctx context.Context, listingID int32) ([]database.ListingImage, error) {
	return s.db.ListListingImages(ctx, listingID)
}

// ImagesByListing groups photos for MANY listings using a single query.
func (s *ListingImageService) ImagesByListing(ctx context.Context, listingIDs []int32) (map[int32][]database.ListingImage, error) {
	out := make(map[int32][]database.ListingImage, len(listingIDs))
	if len(listingIDs) == 0 {
		return out, nil
	}

	rows, err := s.db.ListImagesForListings(ctx, listingIDs)
	if err != nil {
		return nil, err
	}

	for _, row := range rows {
		out[row.ListingID] = append(out[row.ListingID], row)
	}
	return out, nil
}

// DeleteImage remove one photo from a listing the caller owns.
func (s *ListingImageService) DeleteImage(ctx context.Context, userID uuid.UUID, listingID, imageID int32) error {
	if err := s.ownedListing(ctx, userID, listingID); err != nil {
		return err
	}

	filename, err := s.db.DeleteListingImage(ctx, database.DeleteListingImageParams{
		ID:        imageID,
		ListingID: listingID,
	})
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return &NotFoundError{Message: "image not found"}
		}
		return err
	}

	if err := s.files.Delete(filename); err != nil {
		slog.Error("failed to delete image file", "filename", filename, "error", err)
	}

	return nil
}
