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

func TestToListingResponseJSON(t *testing.T) {
	row := database.Listing{
		ID:          1,
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

	want := `{"id":1,"seller_id":"3f1a7c2e-8b4d-4e91-9a5f-2c6d8e0b1a34",` +
		`"title":"Golden Chanterelles","description":"","category":"mushrooms",` +
		`"price":18,"quantity":4,"unit":"kg",` +
		`"created_at":"1970-01-01T00:00:00Z","updated_at":"1970-01-01T00:00:00Z"}`

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
