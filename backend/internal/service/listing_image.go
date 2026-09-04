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

func (s *ListingImageService) ownedListing(ctx context.Context, userID uuid.UUID, listingID uuid.UUID) error {
	listing, err := s.db.GetListing(ctx, listingID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return &NotFoundError{Message: "Listing not found"}
		}
		return err
	}
	if listing.SellerID != userID {
		return &ForbiddenError{Message: "You do not own this listing"}
	}
	return nil
}

// AddImage stores an uploaded file and records it against the listing.
func (s *ListingImageService) AddImage(
	ctx context.Context,
	userID uuid.UUID,
	listingID uuid.UUID,
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
			Message: fmt.Sprintf("A listing can have at most %d images", s.maxPerListing),
		}
	}

	filename, err := s.files.Save(r, ext)
	if err != nil {
		return database.ListingImage{}, err
	}

	img, err := s.createImageRow(ctx, listingID, filename)
	if err != nil {
		if delErr := s.files.Delete(filename); delErr != nil {
			slog.Error("orphaned upload: file written but row insert failed",
				"filename", filename, "error", delErr)
		}
		return database.ListingImage{}, err
	}

	return img, nil
}

// ReorderImages rewrites a listing's gallery order to the order of the ids it
// is given. Positions become 0..n-1, derived from the array rather than sent
// alongside it.
//
// The list must be exactly the listing's current images: a missing, extra or
// duplicated id is a 400 rather than a partial reorder, because a gallery half
// in the old order and half in the new is worse than an unchanged one.
//
// One transaction, and it needs to be: the unique constraint on
// (listing_id, position) is DEFERRABLE INITIALLY DEFERRED, so a swap is
// momentarily a duplicate and is only legal if the check waits for the commit.
// Under autocommit it fires per statement and the swap fails.
func (s *ListingImageService) ReorderImages(
	ctx context.Context,
	userID uuid.UUID,
	listingID uuid.UUID,
	imageIDs []uuid.UUID,
) error {
	if err := s.ownedListing(ctx, userID, listingID); err != nil {
		return err
	}

	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer func() {
		if err := tx.Rollback(); err != nil && !errors.Is(err, sql.ErrTxDone) {
			slog.Error("reorder images transaction rollback failed", "error", err)
		}
	}()

	qtx := s.db.Queries.WithTx(tx.Tx)

	if _, err := qtx.GetListingForUpdate(ctx, listingID); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return &NotFoundError{Message: "Listing not found"}
		}
		return err
	}

	total, err := qtx.CountListingImages(ctx, listingID)
	if err != nil {
		return err
	}
	if total != int64(len(imageIDs)) {
		return &ValidationError{Message: "The image list must name every image on the listing, once each"}
	}

	updated, err := qtx.SetListingImagePositions(ctx, database.SetListingImagePositionsParams{
		ImageIds:  imageIDs,
		ListingID: listingID,
	})
	if err != nil {
		return err
	}
	// The two checks catch different things and both are needed.
	//
	// The count above rejects a list of the wrong length - missing an image,
	// carrying an extra id, or empty. It cannot see a list of the right length
	// that names the wrong images: [A, A, B] against three images is length
	// three against count three, and passes.
	//
	// This one rejects exactly those. Every updated row belongs to this
	// listing, so if as many rows changed as ids were sent, the ids were n
	// distinct images of this listing - a repeat updates its row once and a
	// foreign id updates none, so either leaves fewer rows changed than sent.
	if updated != int64(len(imageIDs)) {
		return &ValidationError{Message: "The image list must name every image on the listing, once each"}
	}

	return tx.Commit()
}

// createImageRow locks the listing so concurrent uploads can't read the same
// position or slip past the per-listing cap.
func (s *ListingImageService) createImageRow(
	ctx context.Context,
	listingID uuid.UUID,
	filename string,
) (database.ListingImage, error) {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return database.ListingImage{}, err
	}
	defer func() {
		if err := tx.Rollback(); err != nil && !errors.Is(err, sql.ErrTxDone) {
			slog.Error("add image transaction rollback failed", "error", err)
		}
	}()

	qtx := s.db.Queries.WithTx(tx.Tx)

	if _, err := qtx.GetListingForUpdate(ctx, listingID); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return database.ListingImage{}, &NotFoundError{Message: "Listing not found"}
		}
		return database.ListingImage{}, err
	}

	count, err := qtx.CountListingImages(ctx, listingID)
	if err != nil {
		return database.ListingImage{}, err
	}
	if count >= int64(s.maxPerListing) {
		return database.ListingImage{}, &ConflictError{
			Message: fmt.Sprintf("A listing can have at most %d images", s.maxPerListing),
		}
	}

	img, err := qtx.CreateListingImage(ctx, database.CreateListingImageParams{
		ID:        database.NewID(),
		ListingID: listingID,
		Filename:  filename,
	})
	if err != nil {
		return database.ListingImage{}, err
	}

	if err := tx.Commit(); err != nil {
		return database.ListingImage{}, err
	}

	return img, nil
}

// ListImages returns one listing's photos in display order.
func (s *ListingImageService) ListImages(ctx context.Context, listingID uuid.UUID) ([]database.ListingImage, error) {
	return s.db.ListListingImages(ctx, listingID)
}

// ImagesByListing groups photos for MANY listings using a single query.
func (s *ListingImageService) ImagesByListing(ctx context.Context, listingIDs []uuid.UUID) (map[uuid.UUID][]database.ListingImage, error) {
	return imagesByListing(ctx, s.db.Queries, listingIDs)
}

// imagesByListing groups one batch query's rows by listing, so any service can
// attach photos to a page of listings without an N+1.
func imagesByListing(ctx context.Context, db *database.Queries, listingIDs []uuid.UUID) (map[uuid.UUID][]database.ListingImage, error) {
	out := make(map[uuid.UUID][]database.ListingImage, len(listingIDs))
	if len(listingIDs) == 0 {
		return out, nil
	}

	rows, err := db.ListImagesForListings(ctx, listingIDs)
	if err != nil {
		return nil, err
	}

	for _, row := range rows {
		out[row.ListingID] = append(out[row.ListingID], row)
	}
	return out, nil
}

// DeleteImage remove one photo from a listing the caller owns.
func (s *ListingImageService) DeleteImage(ctx context.Context, userID uuid.UUID, listingID, imageID uuid.UUID) error {
	if err := s.ownedListing(ctx, userID, listingID); err != nil {
		return err
	}

	filename, err := s.db.DeleteListingImage(ctx, database.DeleteListingImageParams{
		ID:        imageID,
		ListingID: listingID,
	})
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return &NotFoundError{Message: "Image not found"}
		}
		return err
	}

	if err := s.files.Delete(filename); err != nil {
		slog.Error("failed to delete image file", "filename", filename, "error", err)
	}

	return nil
}
