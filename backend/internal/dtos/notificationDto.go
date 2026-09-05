package dtos

import (
	"time"

	"github.com/google/uuid"

	"github.com/IbnBaqqi/transcendence/internal/database"
)

// Kind and the ids, not a sentence: the wording is the client's, so it can be
// read in the reader's language rather than the one the server was written in.
type NotificationResponse struct {
	ID   uuid.UUID `json:"id"`
	Kind string    `json:"kind"`
	// Null for a kind with no listing - a follow has none. Always present, so
	// a client never has to test for the key's absence, only for null.
	ListingTitle *string `json:"listing_title"`
	// Exactly one of these four is set, which is what lets a client route the
	// row without inferring the destination from which id happens to be
	// present. The database enforces it.
	OrderID        *uuid.UUID `json:"order_id"`
	ConversationID *uuid.UUID `json:"conversation_id"`
	ActorID        *uuid.UUID `json:"actor_id"`
	ListingID      *uuid.UUID `json:"listing_id"`
	ReadAt         *time.Time `json:"read_at"`
	CreatedAt      time.Time  `json:"created_at"`
}

func ToNotificationResponses(rows []database.Notification) []NotificationResponse {
	out := make([]NotificationResponse, 0, len(rows))
	for _, r := range rows {
		out = append(out, NotificationResponse{
			ID:             r.ID,
			Kind:           r.Kind,
			ListingTitle:   nullStringPtr(r.ListingTitle),
			OrderID:        nullUUIDPtr(r.OrderID),
			ConversationID: nullUUIDPtr(r.ConversationID),
			ActorID:        nullUUIDPtr(r.ActorID),
			ListingID:      nullUUIDPtr(r.ListingID),
			ReadAt:         nullTimePtr(r.ReadAt),
			CreatedAt:      r.CreatedAt,
		})
	}
	return out
}
