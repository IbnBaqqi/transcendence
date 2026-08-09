package dtos

import (
	"database/sql"
	"time"

	"github.com/IbnBaqqi/transcendence/internal/database"
	"github.com/google/uuid"
)

const onlineWindow = 2 * time.Minute

const (
	RoleBuyer  = "buyer"
	RoleSeller = "seller"
)

type StartConversationInput struct {
	ListingID int32  `json:"listing_id"`
	Body      string `json:"body"`
}

type SendMessageInput struct {
	Body string `json:"body"`
}

type PresenceResponse struct {
	IsOnline   bool       `json:"is_online"`
	LastSeenAt *time.Time `json:"last_seen_at,omitempty"`
}

type ChatUserResponse struct {
	ID       uuid.UUID        `json:"id"`
	Username string           `json:"username"`
	Presence PresenceResponse `json:"presence"`
}

type MessageResponse struct {
	ID             int32      `json:"id"`
	ConversationID int32      `json:"conversation_id"`
	SenderID       uuid.UUID  `json:"sender_id"`
	Body           string     `json:"body"`
	ReadAt         *time.Time `json:"read_at,omitempty"`
	CreatedAt      time.Time  `json:"created_at"`
}

type MessagePreview struct {
	Body      string    `json:"body"`
	CreatedAt time.Time `json:"created_at"`
}

type ConversationResponse struct {
	ID           int32            `json:"id"`
	ListingID    int32            `json:"listing_id"`
	ListingTitle string           `json:"listing_title"`
	Status       string           `json:"status"`
	Role         string           `json:"role"`
	OtherUser    ChatUserResponse `json:"other_user"`
	CreatedAt    time.Time        `json:"created_at"`
	UpdatedAt    time.Time        `json:"updated_at"`
}

type ConversationListItem struct {
	ID           int32            `json:"id"`
	ListingID    int32            `json:"listing_id"`
	ListingTitle string           `json:"listing_title"`
	Status       string           `json:"status"`
	Role         string           `json:"role"`
	OtherUser    ChatUserResponse `json:"other_user"`
	LastMessage  *MessagePreview  `json:"last_message"`
	UnreadCount  int64            `json:"unread_count"`
	UpdatedAt    time.Time        `json:"updated_at"`
}

type UnreadCountResponse struct {
	UnreadCount int64 `json:"unread_count"`
}

func toPresence(lastSeen sql.NullTime, showOnlineStatus bool) PresenceResponse {
	if !showOnlineStatus || !lastSeen.Valid {
		return PresenceResponse{}
	}

	seenAt := lastSeen.Time
	return PresenceResponse{
		IsOnline:   time.Since(seenAt) < onlineWindow,
		LastSeenAt: &seenAt,
	}
}

func roleFor(buyerID, viewerID uuid.UUID) string {
	if buyerID == viewerID {
		return RoleBuyer
	}
	return RoleSeller
}

func nullTimePtr(t sql.NullTime) *time.Time {
	if !t.Valid {
		return nil
	}
	at := t.Time
	return &at
}

func ToMessageResponse(m database.Message) MessageResponse {
	return MessageResponse{
		ID:             m.ID,
		ConversationID: m.ConversationID,
		SenderID:       m.SenderID,
		Body:           m.Body,
		ReadAt:         nullTimePtr(m.ReadAt),
		CreatedAt:      m.CreatedAt.Time,
	}
}

func ToMessageResponses(rows []database.Message) []MessageResponse {
	out := make([]MessageResponse, 0, len(rows))
	for _, r := range rows {
		out = append(out, ToMessageResponse(r))
	}
	return out
}

func ToConversationResponse(
	c database.Conversation,
	listingTitle string,
	other database.User,
	viewerID uuid.UUID,
) ConversationResponse {
	return ConversationResponse{
		ID:           c.ID,
		ListingID:    c.ListingID,
		ListingTitle: listingTitle,
		Status:       c.Status,
		Role:         roleFor(c.BuyerID, viewerID),
		OtherUser: ChatUserResponse{
			ID:       other.ID,
			Username: other.Username,
			Presence: toPresence(other.LastSeenAt, other.ShowOnlineStatus),
		},
		CreatedAt: c.CreatedAt.Time,
		UpdatedAt: c.UpdatedAt.Time,
	}
}

func ToConversationListItem(row database.ListConversationsForUserRow, viewerID uuid.UUID) ConversationListItem {
	item := ConversationListItem{
		ID:           row.ID,
		ListingID:    row.ListingID,
		ListingTitle: row.ListingTitle,
		Status:       row.Status,
		Role:         roleFor(row.BuyerID, viewerID),
		OtherUser: ChatUserResponse{
			ID:       row.OtherUserID,
			Username: row.OtherUsername,
			Presence: toPresence(row.OtherLastSeenAt, row.OtherShowOnlineStatus),
		},
		UnreadCount: row.UnreadCount,
		UpdatedAt:   row.UpdatedAt.Time,
	}

	if row.LastMessageAt.Valid {
		item.LastMessage = &MessagePreview{
			Body:      row.LastMessageBody,
			CreatedAt: row.LastMessageAt.Time,
		}
	}

	return item
}

func ToConversationListItems(rows []database.ListConversationsForUserRow, viewerID uuid.UUID) []ConversationListItem {
	out := make([]ConversationListItem, 0, len(rows))
	for _, r := range rows {
		out = append(out, ToConversationListItem(r, viewerID))
	}
	return out
}
