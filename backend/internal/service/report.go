package service

import (
	"context"
	"database/sql"
	"errors"
	"strings"

	"github.com/google/uuid"

	"github.com/IbnBaqqi/transcendence/internal/database"
)

const maxReportDetail = 500

var reportReasons = map[string]bool{
	"spam":       true,
	"prohibited": true,
	"misleading": true,
	"offensive":  true,
	"other":      true,
}

const reportedConstraint = "listing_reports_listing_reporter_uq"

type ReportService struct {
	db *database.Queries
}

func NewReportService(db *database.Queries) *ReportService {
	return &ReportService{db: db}
}

func (s *ReportService) Report(
	ctx context.Context,
	reporterID uuid.UUID,
	listingID uuid.UUID,
	reason string,
	detail string,
) error {
	if !reportReasons[reason] {
		return &ValidationError{Message: "Unknown report reason"}
	}

	detail = strings.TrimSpace(detail)
	if len(detail) > maxReportDetail {
		return &ValidationError{Message: "Report detail is too long"}
	}

	listing, err := s.db.GetListing(ctx, listingID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return &NotFoundError{Message: "Listing not found"}
		}
		return err
	}
	if listing.SellerID == reporterID {
		return &ValidationError{Message: "You cannot report your own listing"}
	}

	err = s.db.CreateReport(ctx, database.CreateReportParams{
		ID:         database.NewID(),
		ListingID:  listingID,
		ReporterID: uuid.NullUUID{UUID: reporterID, Valid: true},
		Reason:     reason,
		Detail:     sql.NullString{String: detail, Valid: detail != ""},
	})
	if isUniqueViolation(err, reportedConstraint) {
		return &ConflictError{Message: "You have already reported this listing"}
	}
	return err
}
