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

func filledProfile() (database.User, database.Profile, sql.NullString) {
	id := uuid.MustParse("11111111-1111-1111-1111-111111111111")

	user := database.User{
		ID:               id,
		Username:         "forager",
		Email:            "forager@example.test",
		LastSeenAt:       sql.NullTime{Time: time.Now(), Valid: true},
		ShowOnlineStatus: true,
	}
	profile := database.Profile{
		ID:          id,
		Firstname:   sql.NullString{String: "Aino", Valid: true},
		Lastname:    sql.NullString{String: "Virtanen", Valid: true},
		Bio:         sql.NullString{String: "picks chanterelles", Valid: true},
		PhoneNumber: sql.NullString{String: "+358401234567", Valid: true},
		DateOfBirth: sql.NullTime{Time: time.Date(2001, 5, 14, 0, 0, 0, 0, time.UTC), Valid: true},
	}
	return user, profile, sql.NullString{String: "Espoo", Valid: true}
}

// The whole reason there are two response types. Asserting on the marshalled
// JSON rather than the struct, because JSON is what actually reaches a client.
func TestPublicProfileHasNoPrivateFields(t *testing.T) {
	user, profile, location := filledProfile()

	body, err := json.Marshal(ToPublicProfileResponse(user, profile, location, database.SellerRatingRow{Average: 4.25, Total: 4}, true))
	if err != nil {
		t.Fatalf("marshalling: %v", err)
	}

	for _, private := range []string{"email", "phone_number", "date_of_birth", "forager@example.test", "+358401234567"} {
		if strings.Contains(string(body), private) {
			t.Errorf("public profile contains %q:\n%s", private, body)
		}
	}

	for _, expected := range []string{"username", "firstname", "bio", "location", "presence"} {
		if !strings.Contains(string(body), expected) {
			t.Errorf("public profile is missing %q:\n%s", expected, body)
		}
	}
}

// Presence is omitted entirely for an anonymous caller, not sent as offline:
// a client cannot tell a blanket false apart from a real one, so it would be
// a claim about every user on the site that happens to be untrue.
func TestAnAnonymousViewerGetsNoPresenceField(t *testing.T) {
	user, profile, location := filledProfile()

	body, err := json.Marshal(ToPublicProfileResponse(user, profile, location, database.SellerRatingRow{}, false))
	if err != nil {
		t.Fatalf("marshalling: %v", err)
	}

	if strings.Contains(string(body), "presence") {
		t.Errorf("presence was sent to an anonymous viewer:\n%s", body)
	}
	if strings.Contains(string(body), "is_online") {
		t.Errorf("an online claim reached an anonymous viewer:\n%s", body)
	}

	// The rest of the profile is still public.
	for _, expected := range []string{"username", "firstname", "bio", "location"} {
		if !strings.Contains(string(body), expected) {
			t.Errorf("anonymous profile is missing %q:\n%s", expected, body)
		}
	}
}

func TestOwnProfileHasThePrivateFields(t *testing.T) {
	user, profile, location := filledProfile()

	body, err := json.Marshal(ToOwnProfileResponse(user, profile, location))
	if err != nil {
		t.Fatalf("marshalling: %v", err)
	}

	for _, expected := range []string{`"email":"forager@example.test"`, `"phone_number":"+358401234567"`} {
		if !strings.Contains(string(body), expected) {
			t.Errorf("own profile is missing %s:\n%s", expected, body)
		}
	}
}

// A DATE must not grow a midnight and a timezone on the way out.
func TestProfileDateOfBirthIsADate(t *testing.T) {
	user, profile, location := filledProfile()

	body, err := json.Marshal(ToOwnProfileResponse(user, profile, location))
	if err != nil {
		t.Fatalf("marshalling: %v", err)
	}

	if !strings.Contains(string(body), `"date_of_birth":"2001-05-14"`) {
		t.Errorf("date_of_birth is not a plain date:\n%s", body)
	}
}

// Unset fields are null rather than missing, so a form can render an empty box
// instead of guessing whether the field exists.
func TestEmptyProfileSendsNulls(t *testing.T) {
	id := uuid.MustParse("11111111-1111-1111-1111-111111111111")
	user := database.User{ID: id, Username: "new", Email: "new@example.test"}

	body, err := json.Marshal(ToOwnProfileResponse(user, database.Profile{ID: id}, sql.NullString{}))
	if err != nil {
		t.Fatalf("marshalling: %v", err)
	}

	for _, key := range []string{"firstname", "lastname", "bio", "phone_number", "date_of_birth", "location"} {
		if !strings.Contains(string(body), `"`+key+`":null`) {
			t.Errorf("%q is not null:\n%s", key, body)
		}
	}
}

// The average is rounded once, at the boundary: 4.333333333333333 in JSON is
// noise, and a star display wants one decimal.
func TestTheRatingIsRoundedAndAlwaysPresent(t *testing.T) {
	user, profile, location := filledProfile()

	body, err := json.Marshal(ToPublicProfileResponse(
		user, profile, location,
		database.SellerRatingRow{Average: 4.333333333333333, Total: 3},
		true,
	))
	if err != nil {
		t.Fatalf("marshalling: %v", err)
	}

	if !strings.Contains(string(body), `"average":4.3`) {
		t.Errorf("average was not rounded to one decimal:\n%s", body)
	}

	// A seller with nothing gets zeros, not an absent object - count is what
	// separates "no reviews yet" from "rated zero".
	empty, err := json.Marshal(ToPublicProfileResponse(
		user, profile, location, database.SellerRatingRow{}, true,
	))
	if err != nil {
		t.Fatalf("marshalling: %v", err)
	}
	if !strings.Contains(string(empty), `"rating":{"average":0,"count":0}`) {
		t.Errorf("an unrated seller should still carry a rating object:\n%s", empty)
	}
}
