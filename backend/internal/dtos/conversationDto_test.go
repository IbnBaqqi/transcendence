package dtos

import (
	"database/sql"
	"encoding/json"
	"strings"
	"testing"
	"time"

	"github.com/IbnBaqqi/transcendence/internal/database"
	"github.com/google/uuid"
)

func userWithPresence(lastSeen sql.NullTime, show bool) database.User {
	return database.User{
		ID:               uuid.New(),
		Username:         "seller",
		LastSeenAt:       lastSeen,
		ShowOnlineStatus: show,
	}
}

func conversationFor(buyer, seller uuid.UUID) database.Conversation {
	now := sql.NullTime{Time: time.Unix(0, 0).UTC(), Valid: true}
	return database.Conversation{
		ID:        1,
		ListingID: 7,
		BuyerID:   buyer,
		SellerID:  seller,
		Status:    "accepted",
		CreatedAt: now,
		UpdatedAt: now,
	}
}

func TestPresenceHiddenLeavesNoTimestampInJSON(t *testing.T) {
	seenNow := sql.NullTime{Time: time.Now(), Valid: true}
	buyer, seller := uuid.New(), uuid.New()

	res := ToConversationResponse(
		conversationFor(buyer, seller),
		"Chanterelles",
		userWithPresence(seenNow, false),
		buyer,
	)

	if res.OtherUser.Presence.IsOnline {
		t.Error("is_online = true for a hidden user")
	}
	if res.OtherUser.Presence.LastSeenAt != nil {
		t.Error("last_seen_at was set for a hidden user")
	}

	b, err := json.Marshal(res)
	if err != nil {
		t.Fatalf("marshal failed: %v", err)
	}
	if strings.Contains(string(b), "last_seen_at") {
		t.Errorf("hidden presence still shipped a last_seen_at key: %s", b)
	}
}

func TestPresenceVisible(t *testing.T) {
	buyer, seller := uuid.New(), uuid.New()

	tests := []struct {
		name       string
		lastSeen   sql.NullTime
		wantOnline bool
		wantStamp  bool
	}{
		{
			name:       "seen just now",
			lastSeen:   sql.NullTime{Time: time.Now(), Valid: true},
			wantOnline: true,
			wantStamp:  true,
		},
		{
			name:       "seen an hour ago",
			lastSeen:   sql.NullTime{Time: time.Now().Add(-time.Hour), Valid: true},
			wantOnline: false,
			wantStamp:  true,
		},
		{
			name:       "just outside the window",
			lastSeen:   sql.NullTime{Time: time.Now().Add(-onlineWindow - time.Second), Valid: true},
			wantOnline: false,
			wantStamp:  true,
		},
		{
			name:       "never seen",
			lastSeen:   sql.NullTime{},
			wantOnline: false,
			wantStamp:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			res := ToConversationResponse(
				conversationFor(buyer, seller),
				"Chanterelles",
				userWithPresence(tt.lastSeen, true),
				buyer,
			)

			if got := res.OtherUser.Presence.IsOnline; got != tt.wantOnline {
				t.Errorf("is_online = %v, want %v", got, tt.wantOnline)
			}
			if got := res.OtherUser.Presence.LastSeenAt != nil; got != tt.wantStamp {
				t.Errorf("last_seen_at present = %v, want %v", got, tt.wantStamp)
			}
		})
	}
}

func TestConversationRoleFollowsViewer(t *testing.T) {
	buyer, seller := uuid.New(), uuid.New()
	conv := conversationFor(buyer, seller)
	other := userWithPresence(sql.NullTime{}, true)

	if got := ToConversationResponse(conv, "t", other, buyer).Role; got != RoleBuyer {
		t.Errorf("role = %q, want %q", got, RoleBuyer)
	}
	if got := ToConversationResponse(conv, "t", other, seller).Role; got != RoleSeller {
		t.Errorf("role = %q, want %q", got, RoleSeller)
	}
}

func TestListItemWithoutMessages(t *testing.T) {
	buyer := uuid.New()
	row := database.ListConversationsForUserRow{
		ID:              1,
		BuyerID:         buyer,
		SellerID:        uuid.New(),
		Status:          "pending",
		ListingTitle:    "Chanterelles",
		OtherUserID:     uuid.New(),
		OtherUsername:   "seller",
		LastMessageBody: "",
		LastMessageAt:   sql.NullTime{},
	}

	item := ToConversationListItem(row, buyer)
	if item.LastMessage != nil {
		t.Fatalf("last_message = %+v, want nil", item.LastMessage)
	}

	b, err := json.Marshal(item)
	if err != nil {
		t.Fatalf("marshal failed: %v", err)
	}
	if !strings.Contains(string(b), `"last_message":null`) {
		t.Errorf("want last_message null, got %s", b)
	}
}

func TestListItemWithMessages(t *testing.T) {
	buyer := uuid.New()
	at := time.Unix(1000, 0).UTC()
	row := database.ListConversationsForUserRow{
		ID:              1,
		BuyerID:         buyer,
		SellerID:        uuid.New(),
		Status:          "accepted",
		LastMessageBody: "Yes, 4kg left",
		LastMessageAt:   sql.NullTime{Time: at, Valid: true},
		UnreadCount:     2,
	}

	item := ToConversationListItem(row, buyer)
	if item.LastMessage == nil {
		t.Fatal("last_message = nil, want a preview")
	}
	if item.LastMessage.Body != "Yes, 4kg left" {
		t.Errorf("body = %q", item.LastMessage.Body)
	}
	if item.UnreadCount != 2 {
		t.Errorf("unread_count = %d, want 2", item.UnreadCount)
	}
}

func TestMessageResponseReadState(t *testing.T) {
	unread := ToMessageResponse(database.Message{ID: 1, Body: "hei"})
	if unread.ReadAt != nil {
		t.Error("read_at set on an unread message")
	}

	b, err := json.Marshal(unread)
	if err != nil {
		t.Fatalf("marshal failed: %v", err)
	}
	if strings.Contains(string(b), "read_at") {
		t.Errorf("unread message shipped a read_at key: %s", b)
	}

	read := ToMessageResponse(database.Message{
		ID:     2,
		Body:   "hei",
		ReadAt: sql.NullTime{Time: time.Unix(0, 0).UTC(), Valid: true},
	})
	if read.ReadAt == nil {
		t.Error("read_at nil on a read message")
	}
}

func TestToMessageResponsesIsAlwaysAnArray(t *testing.T) {
	b, err := json.Marshal(ToMessageResponses(nil))
	if err != nil {
		t.Fatalf("marshal failed: %v", err)
	}
	if string(b) != "[]" {
		t.Errorf("empty messages = %s, want []", b)
	}
}
