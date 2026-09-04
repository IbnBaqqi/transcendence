package service

import (
	"context"
	"database/sql"
	"errors"
	"log/slog"
	"strings"
	"unicode/utf8"

	"github.com/google/uuid"

	"github.com/IbnBaqqi/transcendence/internal/database"
)

const maxModerationNote = 500

var moderationActions = map[string]string{
	"remove":  "removed",
	"restore": "restored",
	"dismiss": "dismissed",
}

// quarantineStore is the slice of the file store moderation needs: removal
// must stop the images being served, and restore must bring them back.
type quarantineStore interface {
	Quarantine(name string) error
	Release(name string) error
}

type ModerationService struct {
	db    *database.DB
	files quarantineStore
}

func NewModerationService(db *database.DB, files quarantineStore) *ModerationService {
	return &ModerationService{db: db, files: files}
}

func (s *ModerationService) Queue(ctx context.Context) ([]database.ListReportedListingsRow, error) {
	return s.db.ListReportedListings(ctx)
}

func (s *ModerationService) ReportsFor(ctx context.Context, listingID uuid.UUID) ([]database.ListReportsForListingRow, error) {
	if err := s.requireListing(ctx, listingID); err != nil {
		return nil, err
	}

	return s.db.ListReportsForListing(ctx, listingID)
}

func (s *ModerationService) History(ctx context.Context, listingID uuid.UUID) ([]database.ModerationAction, error) {
	if err := s.requireListing(ctx, listingID); err != nil {
		return nil, err
	}

	return s.db.ListModerationActions(ctx, listingID)
}

func (s *ModerationService) requireListing(ctx context.Context, listingID uuid.UUID) error {
	if _, err := s.db.GetListing(ctx, listingID); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return &NotFoundError{Message: "Listing not found"}
		}
		return err
	}
	return nil
}

func (s *ModerationService) Moderate(
	ctx context.Context,
	moderatorID uuid.UUID,
	listingID uuid.UUID,
	action string,
	note string,
) (database.Listing, int64, error) {
	stored, ok := moderationActions[action]
	if !ok {
		return database.Listing{}, 0, &ValidationError{Message: "Action must be remove, restore or dismiss"}
	}

	if !utf8.ValidString(note) || strings.ContainsRune(note, 0) {
		return database.Listing{}, 0, &ValidationError{Message: "Note must be valid UTF-8 without null bytes"}
	}
	note = sanitizeFreeText(note)
	if utf8.RuneCountInString(note) > maxModerationNote {
		return database.Listing{}, 0, &ValidationError{Message: "Note is too long"}
	}
	if action == "remove" && note == "" {
		return database.Listing{}, 0, &ValidationError{Message: "Removing a listing needs a reason"}
	}

	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return database.Listing{}, 0, err
	}
	defer func() {
		if err := tx.Rollback(); err != nil && !errors.Is(err, sql.ErrTxDone) {
			slog.Error("moderation transaction rollback failed", "error", err)
		}
	}()

	qtx := s.db.Queries.WithTx(tx.Tx)

	listing, err := qtx.GetListingForUpdate(ctx, listingID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return database.Listing{}, 0, &NotFoundError{Message: "Listing not found"}
		}
		return database.Listing{}, 0, err
	}

	switch action {
	case "remove":
		if listing.RemovedAt.Valid {
			return database.Listing{}, 0, &ConflictError{Message: "Listing is already removed"}
		}
		listing, err = qtx.SetListingRemoved(ctx, listingID)
	case "restore":
		if !listing.RemovedAt.Valid {
			return database.Listing{}, 0, &ConflictError{Message: "Listing is not removed"}
		}
		listing, err = qtx.RestoreListing(ctx, listingID)
	case "dismiss":
		// Dismissing would clear the queue while the listing stayed invisible,
		// and nothing lists removed listings - it would be reachable only by
		// uuid. Restore it first, or uphold what is already done.
		if listing.RemovedAt.Valid {
			return database.Listing{}, 0, &ConflictError{Message: "Listing is removed; restore it before dismissing its reports"}
		}
	}
	if err != nil {
		return database.Listing{}, 0, err
	}

	resolution := "dismissed"
	if action == "remove" {
		resolution = "upheld"
	}

	resolved, err := qtx.ResolveOpenReports(ctx, database.ResolveOpenReportsParams{
		ListingID: listingID,
		Status:    resolution,
	})
	if err != nil {
		return database.Listing{}, 0, err
	}

	if _, err := qtx.CreateModerationAction(ctx, database.CreateModerationActionParams{
		ID:          database.NewID(),
		ListingID:   listingID,
		ModeratorID: uuid.NullUUID{UUID: moderatorID, Valid: true},
		Action:      stored,
		Note:        sql.NullString{String: note, Valid: note != ""},
	}); err != nil {
		return database.Listing{}, 0, err
	}

	if err := tx.Commit(); err != nil {
		return database.Listing{}, 0, err
	}

	// After the commit, like every other file move in this codebase: the
	// database is the record, the filesystem follows it. A failure here leaves
	// the listing correctly removed but its images still served, which is why
	// it is logged loudly rather than swallowed.
	s.applyToImages(ctx, action, listingID)

	return listing, resolved, nil
}

// applyToImages stops serving a removed listing's photos, or starts again when
// it is restored. Filenames stay in listing_images either way - they are how a
// restore finds the files.
func (s *ModerationService) applyToImages(ctx context.Context, action string, listingID uuid.UUID) {
	if s.files == nil || action == "dismiss" {
		return
	}

	images, err := s.db.ListListingImages(ctx, listingID)
	if err != nil {
		slog.Error("could not list images to move", "listing_id", listingID, "error", err)
		return
	}

	for _, img := range images {
		var moveErr error
		if action == "remove" {
			moveErr = s.files.Quarantine(img.Filename)
		} else {
			moveErr = s.files.Release(img.Filename)
		}

		if moveErr != nil {
			slog.Error("could not move a moderated listing's image",
				"listing_id", listingID, "filename", img.Filename,
				"action", action, "error", moveErr)
		}
	}
}
