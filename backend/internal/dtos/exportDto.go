package dtos

import (
	"time"

	"github.com/IbnBaqqi/transcendence/internal/database"
)

// DataExport is everything the API already tells this account about itself,
// gathered into one document. The field types are the response shapes the
// endpoints publish, so a reader who knows the API knows this file - and a
// change to a response cannot leave the export describing something else.
type DataExport struct {
	ExportedAt time.Time          `json:"exported_at"`
	Account    OwnProfileResponse `json:"account"`
	Providers  []string           `json:"oauth_providers"`

	Listings      []ListingResponse      `json:"listings"`
	Orders        []OrderResponse        `json:"orders"`
	SavedListings []ListingResponse      `json:"saved_listings"`
	Conversations []ConversationListItem `json:"conversations"`
	Messages      []MessageResponse      `json:"messages"`

	ReviewsWritten  []ReviewResponse `json:"reviews_written"`
	ReviewsReceived []ReviewResponse `json:"reviews_received"`

	Following []ChatUserResponse `json:"following"`
	Followers []ChatUserResponse `json:"followers"`

	// Who this account has blocked. Who has blocked THEM is deliberately
	// absent: no endpoint exposes it, and a block is symmetric in its effects
	// precisely so neither side learns who acted.
	Blocks []BlockedUserResponse `json:"blocks"`

	Notifications []NotificationResponse `json:"notifications"`
	APIKeys       []APIKeyResponse       `json:"api_keys"`
}

// ToExportReviews maps plain review rows, which the export reads unpaginated.
// The reviewer's name is not joined in: a review row names the author by id,
// and the export is about this account rather than a directory of others.
func ToExportReviews(rows []database.Review) []ReviewResponse {
	out := make([]ReviewResponse, 0, len(rows))
	for _, r := range rows {
		out = append(out, ToReviewResponse(r, ""))
	}
	return out
}

func ToExportOrders(rows []database.Order) []OrderResponse {
	out := make([]OrderResponse, 0, len(rows))
	for _, o := range rows {
		out = append(out, NewOrderResponse(o))
	}
	return out
}
