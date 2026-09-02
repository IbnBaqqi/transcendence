package dtos

import (
	"database/sql"
	"encoding/json"
	"strings"
	"testing"
	"time"

	"github.com/IbnBaqqi/transcendence/internal/database"
	"github.com/IbnBaqqi/transcendence/internal/presence"
	"github.com/google/uuid"
)

func userWithPresence(lastSeen sql.NullTime, show bool) database.GetChatUserRow {
	return database.GetChatUserRow{
		ID:               uuid.New(),
		Username:         "seller",
		LastSeenAt:       lastSeen,
		ShowOnlineStatus: show,
	}
}

func conversationFor(buyer, seller uuid.UUID) database.Conversation {
	now := sql.NullTime{Time: time.Unix(0, 0).UTC(), Valid: true}
	return database.Conversation{
		ID:           uuid.New(),
		ListingID:    uuid.NullUUID{UUID: uuid.New(), Valid: true},
		ListingTitle: "Chanterelles",
		BuyerID:      buyer,
		SellerID:     seller,
		Status:       "accepted",
		CreatedAt:    now,
		UpdatedAt:    now,
	}
}

func TestPresenceHiddenLeavesNoTimestampInJSON(t *testing.T) {
	seenNow := sql.NullTime{Time: time.Now(), Valid: true}
	buyer, seller := uuid.New(), uuid.New()

	res := ToConversationResponse(
		conversationFor(buyer, seller),
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
			lastSeen:   sql.NullTime{Time: time.Now().Add(-presence.Window - time.Second), Valid: true},
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

func TestConversationSurvivesADeletedListing(t *testing.T) {
	buyer, seller := uuid.New(), uuid.New()

	conv := conversationFor(buyer, seller)
	conv.ListingID = uuid.NullUUID{}

	res := ToConversationResponse(conv, userWithPresence(sql.NullTime{}, true), buyer)

	if res.ListingID != nil {
		t.Errorf("listing_id = %v, want nil", *res.ListingID)
	}
	if res.ListingTitle != "Chanterelles" {
		t.Errorf("listing_title = %q, want the snapshot to survive", res.ListingTitle)
	}

	b, err := json.Marshal(res)
	if err != nil {
		t.Fatalf("marshal failed: %v", err)
	}
	if !strings.Contains(string(b), `"listing_id":null`) {
		t.Errorf("want listing_id null, got %s", b)
	}
}

func TestConversationRoleFollowsViewer(t *testing.T) {
	buyer, seller := uuid.New(), uuid.New()
	conv := conversationFor(buyer, seller)
	other := userWithPresence(sql.NullTime{}, true)

	if got := ToConversationResponse(conv, other, buyer).Role; got != RoleBuyer {
		t.Errorf("role = %q, want %q", got, RoleBuyer)
	}
	if got := ToConversationResponse(conv, other, seller).Role; got != RoleSeller {
		t.Errorf("role = %q, want %q", got, RoleSeller)
	}
}

func TestListItemWithoutMessages(t *testing.T) {
	buyer := uuid.New()
	row := database.ListConversationsForUserRow{
		ID:              uuid.New(),
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
		ID:              uuid.New(),
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
	unread := ToMessageResponse(database.Message{ID: uuid.New(), Body: "hei"})
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
		ID:     uuid.New(),
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

// last_seen_at is a public field on a profile, so it is truncated to the
// minute: TouchLastSeen only writes once per presence.Interval, and the extra
// precision would be a microsecond-accurate activity log for anyone holding a
// user id. Freshness is still judged on the untruncated value.
func TestPresenceTimestampIsTruncatedToTheMinute(t *testing.T) {
	buyer, seller := uuid.New(), uuid.New()

	// A time with obvious sub-minute detail to lose.
	seen := time.Date(2026, 8, 16, 12, 34, 56, 789012345, time.UTC)

	res := ToConversationResponse(
		conversationFor(buyer, seller),
		userWithPresence(sql.NullTime{Time: seen, Valid: true}, true),
		buyer,
	)

	got := res.OtherUser.Presence.LastSeenAt
	if got == nil {
		t.Fatal("last_seen_at is missing")
	}
	if want := seen.Truncate(time.Minute); !got.Equal(want) {
		t.Errorf("last_seen_at = %s, want %s", got.Format(time.RFC3339Nano), want.Format(time.RFC3339Nano))
	}
	if got.Second() != 0 || got.Nanosecond() != 0 {
		t.Errorf("last_seen_at = %s, want no seconds or nanoseconds", got.Format(time.RFC3339Nano))
	}
}

// Truncating the exposed value must not make a fresh user look offline.
func TestPresenceFreshnessUsesTheUntruncatedTime(t *testing.T) {
	buyer, seller := uuid.New(), uuid.New()

	// Inside the window, but truncating would push it outside.
	seen := time.Now().Add(-presence.Window + 5*time.Second)

	res := ToConversationResponse(
		conversationFor(buyer, seller),
		userWithPresence(sql.NullTime{Time: seen, Valid: true}, true),
		buyer,
	)

	if !res.OtherUser.Presence.IsOnline {
		t.Error("is_online = false, want true - freshness was judged on the truncated value")
	}
}

func TestChatUserAvatarURLIsAPathOrNull(t *testing.T) {
	buyer, seller := uuid.New(), uuid.New()
	other := userWithPresence(sql.NullTime{}, true)
	other.AvatarFilename = sql.NullString{String: "abc.png", Valid: true}

	got := ToConversationResponse(conversationFor(buyer, seller), other, buyer).OtherUser.AvatarURL
	if got == nil || *got != UploadURLPrefix+"abc.png" {
		t.Errorf("avatar_url = %v, want %q", got, UploadURLPrefix+"abc.png")
	}

	other.AvatarFilename = sql.NullString{}
	if got := ToConversationResponse(conversationFor(buyer, seller), other, buyer).OtherUser.AvatarURL; got != nil {
		t.Errorf("avatar_url with none set = %q, want nil", *got)
	}
}

// The username is anonymised for a deleted account; leaving the photo would
// identify exactly the person that anonymisation exists to hide.
func TestDeletedAccountKeepsNeitherNameNorAvatar(t *testing.T) {
	buyer, seller := uuid.New(), uuid.New()
	other := userWithPresence(sql.NullTime{}, true)
	other.AvatarFilename = sql.NullString{String: "abc.png", Valid: true}
	other.DeletedAt = sql.NullTime{Time: time.Now(), Valid: true}

	chat := ToConversationResponse(conversationFor(buyer, seller), other, buyer).OtherUser
	if chat.Username != DeletedUserName {
		t.Errorf("username = %q, want %q", chat.Username, DeletedUserName)
	}
	if chat.AvatarURL != nil {
		t.Errorf("avatar_url = %q, want nil for a deleted account", *chat.AvatarURL)
	}
}

// avatar_url is in the schema's required list, so it has to be present and
// null rather than omitted - a client reading `"avatar_url" in user` must not
// see the key disappear when nobody has set a picture.
func TestChatUserAlwaysCarriesTheAvatarKey(t *testing.T) {
	buyer, seller := uuid.New(), uuid.New()
	res := ToConversationResponse(conversationFor(buyer, seller), userWithPresence(sql.NullTime{}, true), buyer)

	encoded, err := json.Marshal(res.OtherUser)
	if err != nil {
		t.Fatalf("marshalling: %v", err)
	}
	if !strings.Contains(string(encoded), `"avatar_url":null`) {
		t.Errorf("other_user = %s, want an explicit null avatar_url", encoded)
	}
}

// The inbox list is a separate construction site from the detail response, so
// each needs its own check: fixing one and forgetting the other is invisible
// until the list renders every row with a blank face.
func TestListItemCarriesTheAvatarAndDropsItWhenDeleted(t *testing.T) {
	buyer := uuid.New()
	row := database.ListConversationsForUserRow{
		ID:                  uuid.New(),
		BuyerID:             buyer,
		SellerID:            uuid.New(),
		Status:              "accepted",
		ListingTitle:        "Chanterelles",
		OtherUserID:         uuid.New(),
		OtherUsername:       "seller",
		OtherAvatarFilename: sql.NullString{String: "c.png", Valid: true},
	}

	got := ToConversationListItem(row, buyer).OtherUser.AvatarURL
	if got == nil || *got != UploadURLPrefix+"c.png" {
		t.Errorf("avatar_url = %v, want %q", got, UploadURLPrefix+"c.png")
	}

	row.OtherDeletedAt = sql.NullTime{Time: time.Now(), Valid: true}
	if got := ToConversationListItem(row, buyer).OtherUser.AvatarURL; got != nil {
		t.Errorf("avatar_url for a deleted account = %q, want nil", *got)
	}
}
