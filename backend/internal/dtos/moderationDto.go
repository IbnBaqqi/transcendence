package dtos

import (
	"time"

	"github.com/google/uuid"

	"github.com/IbnBaqqi/transcendence/internal/database"
)

type ModerateListingRequest struct {
	Action string `json:"action"`
	Note   string `json:"note"`
}

type ModerateListingResponse struct {
	Listing         ListingResponse `json:"listing"`
	ReportsResolved int64           `json:"reports_resolved"`
}

type ReportedListingResponse struct {
	ListingID       uuid.UUID  `json:"listing_id"`
	Title           string     `json:"title"`
	SellerID        uuid.UUID  `json:"seller_id"`
	RemovedAt       *time.Time `json:"removed_at"`
	ReportCount     int64      `json:"report_count"`
	FirstReportedAt time.Time  `json:"first_reported_at"`
}

func ToReportedListingResponses(rows []database.ListReportedListingsRow) []ReportedListingResponse {
	out := make([]ReportedListingResponse, 0, len(rows))
	for _, r := range rows {
		out = append(out, ReportedListingResponse{
			ListingID:       r.ListingID,
			Title:           r.Title,
			SellerID:        r.SellerID,
			RemovedAt:       nullTimePtr(r.RemovedAt),
			ReportCount:     r.ReportCount,
			FirstReportedAt: r.FirstReportedAt,
		})
	}
	return out
}

type ReportResponse struct {
	ID         uuid.UUID  `json:"id"`
	ReporterID *uuid.UUID `json:"reporter_id"`
	Reason     string     `json:"reason"`
	Detail     string     `json:"detail"`
	Status     string     `json:"status"`
	CreatedAt  time.Time  `json:"created_at"`
}

func ToReportResponses(rows []database.ListReportsForListingRow) []ReportResponse {
	out := make([]ReportResponse, 0, len(rows))
	for _, r := range rows {
		out = append(out, ReportResponse{
			ID:         r.ID,
			ReporterID: nullUUIDPtr(r.ReporterID),
			Reason:     r.Reason,
			Detail:     r.Detail.String,
			Status:     r.Status,
			CreatedAt:  r.CreatedAt,
		})
	}
	return out
}

type ModerationActionResponse struct {
	ID          uuid.UUID  `json:"id"`
	ListingID   uuid.UUID  `json:"listing_id"`
	ModeratorID *uuid.UUID `json:"moderator_id"`
	Action      string     `json:"action"`
	Note        string     `json:"note"`
	CreatedAt   time.Time  `json:"created_at"`
}

func ToModerationActionResponses(rows []database.ModerationAction) []ModerationActionResponse {
	out := make([]ModerationActionResponse, 0, len(rows))
	for _, r := range rows {
		out = append(out, ModerationActionResponse{
			ID:          r.ID,
			ListingID:   r.ListingID,
			ModeratorID: nullUUIDPtr(r.ModeratorID),
			Action:      r.Action,
			Note:        r.Note.String,
			CreatedAt:   r.CreatedAt,
		})
	}
	return out
}
