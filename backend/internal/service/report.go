package service

import (
	"context"
	"database/sql"
	"errors"
	"strings"

	"unicode"
	"unicode/utf8"

	"github.com/google/uuid"

	"github.com/IbnBaqqi/transcendence/internal/database"
)

// sanitizeReportDetail drops control characters a moderator's terminal or admin
// UI would act on rather than display - ANSI escapes can recolour and reposition
// output, and bidi overrides can visually reverse the text that follows. This is
// attacker-controlled text read by the person deciding the report, so it is
// stripped at write time rather than trusted to whoever renders it.
func sanitizeReportDetail(detail string) string {
	cleaned := strings.Map(func(r rune) rune {
		if r == '\n' || r == '\t' {
			return r
		}
		if unicode.IsControl(r) || unicode.Is(unicode.Bidi_Control, r) {
			return -1
		}
		return r
	}, detail)

	return strings.TrimSpace(cleaned)
}

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

	if !utf8.ValidString(detail) || strings.ContainsRune(detail, 0) {
		return &ValidationError{Message: "Report detail must be valid UTF-8 without null bytes"}
	}

	detail = sanitizeReportDetail(detail)

	if utf8.RuneCountInString(detail) > maxReportDetail {
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
