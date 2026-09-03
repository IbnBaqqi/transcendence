package dtos

import (
	"database/sql"
	"time"

	"github.com/IbnBaqqi/transcendence/internal/database"
	"github.com/IbnBaqqi/transcendence/internal/presence"
	"github.com/google/uuid"
)

const (
	RoleBuyer  = "buyer"
	RoleSeller = "seller"
)

type StartConversationInput struct {
	ListingID uuid.UUID `json:"listing_id"`
	Body      string    `json:"body"`
}

type SendMessageInput struct {
	Body string `json:"body"`
}

type PresenceResponse struct {
	IsOnline   bool       `json:"is_online"`
	LastSeenAt *time.Time `json:"last_seen_at,omitempty"`
}

type ChatUserResponse struct {
	ID        uuid.UUID        `json:"id"`
	Username  string           `json:"username"`
	AvatarURL *string          `json:"avatar_url"`
	Presence  PresenceResponse `json:"presence"`
}

type MessageResponse struct {
	ID             uuid.UUID  `json:"id"`
	ConversationID uuid.UUID  `json:"conversation_id"`
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
	ID           uuid.UUID        `json:"id"`
	ListingID    *uuid.UUID       `json:"listing_id"`
	ListingTitle string           `json:"listing_title"`
	Status       string           `json:"status"`
	Role         string           `json:"role"`
	OtherUser    ChatUserResponse `json:"other_user"`
	CreatedAt    time.Time        `json:"created_at"`
	UpdatedAt    time.Time        `json:"updated_at"`
}

type ConversationListItem struct {
	ID           uuid.UUID        `json:"id"`
	ListingID    *uuid.UUID       `json:"listing_id"`
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

const DeletedUserName = "Deleted user"

// A deleted account keeps its row so the other party's thread still resolves,
// but its username is a machine placeholder chosen to satisfy a unique index,
// not something to show a person.
func displayName(username string, deletedAt sql.NullTime) string {
	if deletedAt.Valid {
		return DeletedUserName
	}
	return username
}

// Deleting the deletedAt check leaves a deleted account's photo beside the
// name that was anonymised to hide exactly that person.
func displayAvatar(filename sql.NullString, deletedAt sql.NullTime) *string {
	if deletedAt.Valid {
		return nil
	}
	return avatarURL(filename)
}

func toPresence(lastSeen sql.NullTime, showOnlineStatus bool) PresenceResponse {
	if !showOnlineStatus || !lastSeen.Valid {
		return PresenceResponse{}
	}

	online := presence.IsOnline(lastSeen.Time, time.Now())
	seenAt := lastSeen.Time.Truncate(presence.Interval)

	return PresenceResponse{
		IsOnline:   online,
		LastSeenAt: &seenAt,
	}
}

func roleFor(buyerID, viewerID uuid.UUID) string {
	if buyerID == viewerID {
		return RoleBuyer
	}
	return RoleSeller
}

// nullUUIDPtr converts sqlc's uuid.NullUUID into a *uuid.UUID, so a listing
// that has been deleted marshals as JSON null rather than a zero uuid.
func nullUUIDPtr(v uuid.NullUUID) *uuid.UUID {
	if !v.Valid {
		return nil
	}
	id := v.UUID
	return &id
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
	other database.GetChatUserRow,
	viewerID uuid.UUID,
) ConversationResponse {
	return ConversationResponse{
		ID:           c.ID,
		ListingID:    nullUUIDPtr(c.ListingID),
		ListingTitle: c.ListingTitle,
		Status:       c.Status,
		Role:         roleFor(c.BuyerID, viewerID),
		OtherUser: ChatUserResponse{
			ID:        other.ID,
			Username:  displayName(other.Username, other.DeletedAt),
			AvatarURL: displayAvatar(other.AvatarFilename, other.DeletedAt),
			Presence:  toPresence(other.LastSeenAt, other.ShowOnlineStatus),
		},
		CreatedAt: c.CreatedAt.Time,
		UpdatedAt: c.UpdatedAt.Time,
	}
}

func ToConversationListItem(row database.ListConversationsForUserRow, viewerID uuid.UUID) ConversationListItem {
	item := ConversationListItem{
		ID:           row.ID,
		ListingID:    nullUUIDPtr(row.ListingID),
		ListingTitle: row.ListingTitle,
		Status:       row.Status,
		Role:         roleFor(row.BuyerID, viewerID),
		OtherUser: ChatUserResponse{
			ID:        row.OtherUserID,
			Username:  displayName(row.OtherUsername, row.OtherDeletedAt),
			AvatarURL: displayAvatar(row.OtherAvatarFilename, row.OtherDeletedAt),
			Presence:  toPresence(row.OtherLastSeenAt, row.OtherShowOnlineStatus),
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
