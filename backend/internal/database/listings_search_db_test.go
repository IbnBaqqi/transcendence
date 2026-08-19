package database_test

import (
	"context"
	"testing"

	"github.com/google/uuid"

	"github.com/IbnBaqqi/transcendence/internal/database"
	"github.com/IbnBaqqi/transcendence/internal/testdb"
)

func seller(t *testing.T, db *database.DB, name, title, location string) string {
	t.Helper()

	id := uuid.New()
	if _, err := db.Exec(
		`INSERT INTO users (id, email, username, password) VALUES ($1, $2, $3, 'x')`,
		id, name+"@example.test", name,
	); err != nil {
		t.Fatalf("creating %s: %v", name, err)
	}

	if _, err := db.Exec(
		`INSERT INTO listings (seller_id, title, description, category, price, quantity, unit)
		 VALUES ($1, $2, 'fresh', 'mushrooms', 10.00, 5, 'kg')`,
		id, title,
	); err != nil {
		t.Fatalf("creating %s's listing: %v", name, err)
	}

	if location != "" {
		if _, err := db.Exec(
			`INSERT INTO addresses (user_id, location) VALUES ($1, $2)`, id, location,
		); err != nil {
			t.Fatalf("creating %s's address: %v", name, err)
		}
	}

	return title
}

func TestLocationFilterMatchesOnlySellersWithAnAddress(t *testing.T) {
	db := testdb.New(t)
	ctx := context.Background()

	seller(t, db, "aino", "Chanterelles", "Helsinki")
	seller(t, db, "veikko", "Bilberries", "")

	tests := []struct {
		name     string
		location string
		want     []string
	}{
		{"no filter returns both", "", []string{"Bilberries", "Chanterelles"}},
		{"matches the addressed seller", "helsinki", []string{"Chanterelles"}},
		{"partial, case-insensitive", "HELS", []string{"Chanterelles"}},
		{"a location nobody has", "turku", nil},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			params := database.SearchListingsParams{Location: tt.location, Limit: 50}

			rows, err := db.Queries.SearchListingsDynamic(ctx, params)
			if err != nil {
				t.Fatalf("searching: %v", err)
			}

			got := make([]string, 0, len(rows))
			for _, row := range rows {
				got = append(got, row.Title)
			}
			if len(got) != len(tt.want) {
				t.Fatalf("titles = %v, want %v", got, tt.want)
			}
			for i, title := range tt.want {
				if got[i] != title {
					t.Errorf("titles = %v, want %v", got, tt.want)
					break
				}
			}

			count, err := db.Queries.CountSearchListingsDynamic(ctx, params)
			if err != nil {
				t.Fatalf("counting: %v", err)
			}
			if int(count) != len(tt.want) {
				t.Errorf("count = %d, want %d", count, len(tt.want))
			}
		})
	}
}
