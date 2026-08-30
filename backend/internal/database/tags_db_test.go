package database_test

import (
	"context"
	"testing"

	"github.com/google/uuid"

	"github.com/IbnBaqqi/transcendence/internal/database"
	"github.com/IbnBaqqi/transcendence/internal/testdb"
)

func tagged(t *testing.T, db *database.DB, name, title string, tags ...string) {
	t.Helper()

	id := uuid.New()
	if _, err := db.Exec(
		`INSERT INTO users (id, email, username, password) VALUES ($1, $2, $3, 'x')`,
		id, name+"@example.test", name,
	); err != nil {
		t.Fatalf("creating %s: %v", name, err)
	}

	listingID := database.NewID()
	if _, err := db.Exec(
		`INSERT INTO listings (id, seller_id, title, description, category, price, quantity, unit)
		 VALUES ($1, $2, $3, 'fresh', 'mushrooms', 10.00, 5, 'kg')`,
		listingID, id, title,
	); err != nil {
		t.Fatalf("creating %s's listing: %v", name, err)
	}

	for _, tag := range tags {
		if _, err := db.Exec(
			`WITH t AS (INSERT INTO tags (name) VALUES ($1)
			            ON CONFLICT (name) DO UPDATE SET name = EXCLUDED.name RETURNING id)
			 INSERT INTO listing_tags (listing_id, tag_id) SELECT $2, id FROM t`,
			tag, listingID,
		); err != nil {
			t.Fatalf("tagging %s with %s: %v", title, tag, err)
		}
	}
}

func TestFilteringByTag(t *testing.T) {
	db := testdb.New(t)
	ctx := context.Background()

	tagged(t, db, "aino", "Chanterelles", "roadside", "sunny")
	tagged(t, db, "veikko", "Morels", "roadside")
	tagged(t, db, "sisko", "Bilberries")

	tests := []struct {
		name string
		tag  string
		want int
	}{
		{"a shared tag finds both", "roadside", 2},
		{"a tag only one listing has", "sunny", 1},
		{"an untagged listing is never matched", "nonsense", 0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			params := database.SearchListingsParams{Tag: tt.tag, Limit: 50}

			rows, err := db.Queries.SearchListingsDynamic(ctx, params)
			if err != nil {
				t.Fatalf("searching: %v", err)
			}
			if len(rows) != tt.want {
				t.Errorf("rows = %d, want %d", len(rows), tt.want)
			}

			count, err := db.Queries.CountSearchListingsDynamic(ctx, params)
			if err != nil {
				t.Fatalf("counting: %v", err)
			}
			if int(count) != tt.want {
				t.Errorf("count = %d, want %d - the total disagrees with the rows", count, tt.want)
			}
		})
	}
}
