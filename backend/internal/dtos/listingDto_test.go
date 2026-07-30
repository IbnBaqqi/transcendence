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

// The point of the DTO is the JSON that comes out, so assert on the marshalled
// bytes rather than field-by-field. A field-by-field test would still pass if
// someone removed the json tags.
func TestToListingResponseJSON(t *testing.T) {
	row := database.Listing{
		ID:       1,
		SellerID: uuid.MustParse("3f1a7c2e-8b4d-4e91-9a5f-2c6d8e0b1a34"),
		Title:    "Golden Chanterelles",
		// zero value = Valid:false = the column was NULL in Postgres.
		// This is the case that produced {"String":"","Valid":false} before.
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

	// Compare the whole document rather than substrings. `"price":18` is a
	// prefix of `"price":18.5`, so a Contains check would pass for the wrong
	// value - and this is the only test guarding the NUMERIC-string -> float
	// conversion. An exact match also catches renamed, dropped and extra keys.
	//
	// Note "price":18 and not 18.00: JSON numbers carry no trailing zeros.
	// "description":"" is the NULL column - an empty string, not a wrapper.
	want := `{"id":1,"seller_id":"3f1a7c2e-8b4d-4e91-9a5f-2c6d8e0b1a34",` +
		`"title":"Golden Chanterelles","description":"","category":"mushrooms",` +
		`"price":18,"quantity":4,"unit":"kg",` +
		`"created_at":"1970-01-01T00:00:00Z","updated_at":"1970-01-01T00:00:00Z"}`

	if got != want {
		t.Errorf("JSON shape changed\n got: %s\nwant: %s", got, want)
	}

	// what must never leak: Go field names and sql null wrappers
	for _, bad := range []string{`"Title"`, `"Valid"`, `"String"`} {
		if strings.Contains(got, bad) {
			t.Errorf("raw sqlc shape leaked (%s) in JSON: %s", bad, got)
		}
	}
}

// The happy path for the null-wrapper unwrap, plus the slice mapper.
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

// A nil slice marshals to `null`, an empty one to `[]`. Clients shouldn't
// need a null check, so this guards the make(..., 0, len) in the mapper.
func TestToListingResponsesEmptyIsArray(t *testing.T) {
	b, err := json.Marshal(ToListingResponses(nil))
	if err != nil {
		t.Fatalf("marshal failed: %v", err)
	}
	if got := string(b); got != "[]" {
		t.Errorf("expected [] for an empty result, got %s", got)
	}
}
