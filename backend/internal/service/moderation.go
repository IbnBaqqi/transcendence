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

type ModerationService struct {
	db *database.DB
}

func NewModerationService(db *database.DB) *ModerationService {
	return &ModerationService{db: db}
}

func (s *ModerationService) Queue(ctx context.Context) ([]database.ListReportedListingsRow, error) {
	return s.db.ListReportedListings(ctx)
}

func (s *ModerationService) ReportsFor(ctx context.Context, listingID uuid.UUID) ([]database.ListReportsForListingRow, error) {
	return s.db.ListReportsForListing(ctx, listingID)
}

func (s *ModerationService) History(ctx context.Context, listingID uuid.UUID) ([]database.ModerationAction, error) {
	return s.db.ListModerationActions(ctx, listingID)
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
	note = sanitizeReportDetail(note)
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

	return listing, resolved, nil
}
