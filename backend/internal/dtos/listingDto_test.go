package dtos

import (
	"database/sql"
	"encoding/json"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/IbnBaqqi/transcendence/internal/database"
)

var testListingDtoID = uuid.MustParse("33333333-3333-4333-8333-333333333333")

func TestToListingResponseJSON(t *testing.T) {
	row := database.Listing{
		ID:          testListingDtoID,
		SellerID:    uuid.MustParse("3f1a7c2e-8b4d-4e91-9a5f-2c6d8e0b1a34"),
		Title:       "Golden Chanterelles",
		Description: sql.NullString{},
		Category:    "mushrooms",
		Price:       "18.00",
		Quantity:    4,
		Unit:        "kg",
		CreatedAt:   sql.NullTime{Time: time.Unix(0, 0).UTC(), Valid: true},
		UpdatedAt:   sql.NullTime{Time: time.Unix(0, 0).UTC(), Valid: true},
	}

	b, err := json.Marshal(ToListingResponse(row))
	if err != nil {
		t.Fatalf("marshal failed: %v", err)
	}
	got := string(b)

	want := `{"id":"` + testListingDtoID.String() + `",` +
		`"seller_id":"3f1a7c2e-8b4d-4e91-9a5f-2c6d8e0b1a34",` +
		`"title":"Golden Chanterelles","description":"","category":"mushrooms",` +
		`"price":18,"quantity":4,"unit":"kg",` +
		`"created_at":"1970-01-01T00:00:00Z","updated_at":"1970-01-01T00:00:00Z",` +
		`"images":[],"tags":[],"seller":null}`

	if got != want {
		t.Errorf("JSON shape changed\n got: %s\nwant: %s", got, want)
	}

	for _, bad := range []string{`"Title"`, `"Valid"`, `"String"`} {
		if strings.Contains(got, bad) {
			t.Errorf("raw sqlc shape leaked (%s) in JSON: %s", bad, got)
		}
	}
}

func TestToListingResponsesMapsRows(t *testing.T) {
	rows := []database.Listing{
		{
			Title:       "Wild Blueberries",
			Description: sql.NullString{String: "Hand-picked", Valid: true},
			Price:       "7.50",
		},
	}

	out := ToListingResponses(rows)

	if len(out) != 1 {
		t.Fatalf("expected 1 listing, got %d", len(out))
	}
	if out[0].Description != "Hand-picked" {
		t.Errorf("expected description to unwrap, got %q", out[0].Description)
	}
	if out[0].Price != 7.5 {
		t.Errorf("expected price 7.5, got %v", out[0].Price)
	}
}

func TestToListingResponsesEmptyIsArray(t *testing.T) {
	b, err := json.Marshal(ToListingResponses(nil))
	if err != nil {
		t.Fatalf("marshal failed: %v", err)
	}
	if got := string(b); got != "[]" {
		t.Errorf("expected [] for an empty result, got %s", got)
	}
}

func TestWithSellerEachMatchesOnSellerID(t *testing.T) {
	sellerA, sellerB := uuid.New(), uuid.New()
	items := []ListingResponse{
		{ID: uuid.New(), SellerID: sellerA},
		{ID: uuid.New(), SellerID: sellerB},
		{ID: uuid.New(), SellerID: sellerA},
	}
	bySeller := map[uuid.UUID]ListingSeller{
		sellerA: {ID: sellerA, Username: "matti"},
	}

	got := WithSellerEach(items, bySeller)

	if got[0].Seller == nil || got[0].Seller.Username != "matti" {
		t.Errorf("first item seller = %+v, want matti", got[0].Seller)
	}
	// Two listings from the same seller both get one, from a single map entry.
	if got[2].Seller == nil || got[2].Seller.Username != "matti" {
		t.Errorf("third item seller = %+v, want matti", got[2].Seller)
	}
	// A seller the lookup did not return stays null rather than borrowing
	// whichever entry happened to be last.
	if got[1].Seller != nil {
		t.Errorf("second item seller = %+v, want nil", got[1].Seller)
	}
}

func TestToListingSellerBuildsTheAvatarPath(t *testing.T) {
	id := uuid.New()

	with := ToListingSeller(id, "matti", sql.NullString{String: "a.png", Valid: true})
	if with.AvatarURL == nil || *with.AvatarURL != UploadURLPrefix+"a.png" {
		t.Errorf("avatar_url = %v, want %q", with.AvatarURL, UploadURLPrefix+"a.png")
	}

	without := ToListingSeller(id, "matti", sql.NullString{})
	if without.AvatarURL != nil {
		t.Errorf("avatar_url with none set = %q, want nil", *without.AvatarURL)
	}
}
