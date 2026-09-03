package dtos

import (
	"time"

	"github.com/google/uuid"

	"github.com/IbnBaqqi/transcendence/internal/database"
)

// Kind and the ids, not a sentence: the wording is the client's, so it can be
// read in the reader's language rather than the one the server was written in.
type NotificationResponse struct {
	ID             uuid.UUID  `json:"id"`
	Kind           string     `json:"kind"`
	ListingTitle   string     `json:"listing_title"`
	OrderID        *uuid.UUID `json:"order_id"`
	ConversationID *uuid.UUID `json:"conversation_id"`
	ReadAt         *time.Time `json:"read_at"`
	CreatedAt      time.Time  `json:"created_at"`
}

func ToNotificationResponses(rows []database.Notification) []NotificationResponse {
	out := make([]NotificationResponse, 0, len(rows))
	for _, r := range rows {
		out = append(out, NotificationResponse{
			ID:             r.ID,
			Kind:           r.Kind,
			ListingTitle:   r.ListingTitle,
			OrderID:        nullUUIDPtr(r.OrderID),
			ConversationID: nullUUIDPtr(r.ConversationID),
			ReadAt:         nullTimePtr(r.ReadAt),
			CreatedAt:      r.CreatedAt.Time,
		})
	}
	return out
}
