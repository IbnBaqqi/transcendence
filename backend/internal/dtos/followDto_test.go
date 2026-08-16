package dtos

import (
	"database/sql"
	"encoding/json"
	"reflect"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/IbnBaqqi/transcendence/internal/database"
)

func TestFollowListsAreAlwaysArrays(t *testing.T) {
	following, err := json.Marshal(ToFollowingResponses(nil))
	if err != nil {
		t.Fatalf("marshalling: %v", err)
	}
	if string(following) != "[]" {
		t.Errorf("empty following = %s, want []", following)
	}

	followers, err := json.Marshal(ToFollowerResponses(nil))
	if err != nil {
		t.Fatalf("marshalling: %v", err)
	}
	if string(followers) != "[]" {
		t.Errorf("empty followers = %s, want []", followers)
	}
}

func TestFollowListHidesPresenceWhenAsked(t *testing.T) {
	seen := sql.NullTime{Time: time.Now(), Valid: true}

	rows := []database.ListFollowingRow{
		{ID: uuid.New(), Username: "visible", LastSeenAt: seen, ShowOnlineStatus: true},
		{ID: uuid.New(), Username: "hidden", LastSeenAt: seen, ShowOnlineStatus: false},
		{ID: uuid.New(), Username: "neverseen", ShowOnlineStatus: true},
	}

	got := ToFollowingResponses(rows)
	if len(got) != 3 {
		t.Fatalf("len = %d, want 3", len(got))
	}

	if !got[0].Presence.IsOnline || got[0].Presence.LastSeenAt == nil {
		t.Error("a visible, recently seen user should report online with a timestamp")
	}
	if got[1].Presence.IsOnline || got[1].Presence.LastSeenAt != nil {
		t.Error("a hidden user leaked presence")
	}
	if got[2].Presence.LastSeenAt != nil {
		t.Error("a user who has never been seen should have no timestamp")
	}

	hidden, _ := json.Marshal(got[1].Presence)
	never, _ := json.Marshal(got[2].Presence)
	if string(hidden) != string(never) {
		t.Errorf("hidden %s and never-seen %s are distinguishable", hidden, never)
	}
}

func TestBothFollowMappersAgree(t *testing.T) {
	first, second := uuid.New(), uuid.New()
	seen := sql.NullTime{Time: time.Date(2026, 8, 16, 12, 0, 0, 0, time.UTC), Valid: true}

	following := ToFollowingResponses([]database.ListFollowingRow{
		{ID: first, Username: "aino", LastSeenAt: seen, ShowOnlineStatus: true},
		{ID: second, Username: "hidden", LastSeenAt: seen, ShowOnlineStatus: false},
	})
	followers := ToFollowerResponses([]database.ListFollowersRow{
		{ID: first, Username: "aino", LastSeenAt: seen, ShowOnlineStatus: true},
		{ID: second, Username: "hidden", LastSeenAt: seen, ShowOnlineStatus: false},
	})

	if !reflect.DeepEqual(following, followers) {
		t.Errorf("the two mappers disagree:\n following = %+v\n followers = %+v", following, followers)
	}
}
