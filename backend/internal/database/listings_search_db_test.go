package database_test

import (
	"context"
	"sort"
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
		`INSERT INTO listings (id, seller_id, title, description, category, price, quantity, unit)
		 VALUES ($1, $2, $3, 'fresh', 'mushrooms', 10.00, 5, 'kg')`,
		database.NewID(), id, title,
	); err != nil {
		t.Fatalf("creating %s's listing: %v", name, err)
	}

	if location != "" {
		if _, err := db.Exec(
			`INSERT INTO addresses (id, user_id, location) VALUES ($1, $2, $3)`, database.NewID(), id, location,
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

func categorised(t *testing.T, db *database.DB, name, title, category string) {
	t.Helper()

	id := uuid.New()
	if _, err := db.Exec(
		`INSERT INTO users (id, email, username, password) VALUES ($1, $2, $3, 'x')`,
		id, name+"@example.test", name,
	); err != nil {
		t.Fatalf("creating %s: %v", name, err)
	}

	if _, err := db.Exec(
		`INSERT INTO listings (id, seller_id, title, description, category, price, quantity, unit)
		 VALUES ($1, $2, $3, 'fresh', $4, 10.00, 5, 'kg')`,
		database.NewID(), id, title, category,
	); err != nil {
		t.Fatalf("creating %s's listing: %v", name, err)
	}
}

func TestFilteringByAParentIncludesItsChildren(t *testing.T) {
	db := testdb.New(t)
	ctx := context.Background()

	if _, err := db.Exec(
		`INSERT INTO categories (slug, name, parent_slug) VALUES ('chanterelles', 'Chanterelles', 'mushrooms')`,
	); err != nil {
		t.Fatalf("adding a child category: %v", err)
	}

	categorised(t, db, "aino", "Plain Mushrooms", "mushrooms")
	categorised(t, db, "veikko", "Golden Chanterelles", "chanterelles")
	categorised(t, db, "sisko", "Bilberries", "berries")

	tests := []struct {
		name     string
		category string
		want     []string
	}{
		{"a parent reaches its children", "mushrooms", []string{"Golden Chanterelles", "Plain Mushrooms"}},
		{"a child does not climb back up", "chanterelles", []string{"Golden Chanterelles"}},
		{"an unrelated category is unaffected", "berries", []string{"Bilberries"}},
		{"an unknown slug is empty, not an error", "nonsense", nil},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rows, err := db.Queries.SearchListingsDynamic(ctx, database.SearchListingsParams{
				Category: tt.category, Limit: 50,
			})
			if err != nil {
				t.Fatalf("searching: %v", err)
			}

			got := make([]string, 0, len(rows))
			for _, row := range rows {
				got = append(got, row.Title)
			}
			sort.Strings(got)

			if len(got) != len(tt.want) {
				t.Fatalf("titles = %v, want %v", got, tt.want)
			}
			for i, title := range tt.want {
				if got[i] != title {
					t.Fatalf("titles = %v, want %v", got, tt.want)
				}
			}
		})
	}
}

func makeSeller(t *testing.T, db *database.DB, name string) uuid.UUID {
	t.Helper()

	id := uuid.New()
	if _, err := db.Exec(
		`INSERT INTO users (id, email, username, password) VALUES ($1, $2, $3, 'x')`,
		id, name+"@example.test", name,
	); err != nil {
		t.Fatalf("creating %s: %v", name, err)
	}
	return id
}

func addListing(t *testing.T, db *database.DB, sellerID uuid.UUID, title, category string) {
	t.Helper()

	if _, err := db.Exec(
		`INSERT INTO listings (id, seller_id, title, description, category, price, quantity, unit)
		 VALUES ($1, $2, $3, 'fresh', $4, 10.00, 5, 'kg')`,
		database.NewID(), sellerID, title, category,
	); err != nil {
		t.Fatalf("creating %s: %v", title, err)
	}
}

func searchTitles(t *testing.T, db *database.DB, params database.SearchListingsParams) []string {
	t.Helper()

	rows, err := db.Queries.SearchListingsDynamic(context.Background(), params)
	if err != nil {
		t.Fatalf("searching: %v", err)
	}

	titles := make([]string, 0, len(rows))
	for _, row := range rows {
		titles = append(titles, row.Title)
	}
	sort.Strings(titles)
	return titles
}

func TestTheSellerFilterReturnsOnlyThatSellersListings(t *testing.T) {
	db := testdb.New(t)

	aino := makeSeller(t, db, "aino")
	veikko := makeSeller(t, db, "veikko")
	addListing(t, db, aino, "Chanterelles", "mushrooms")
	addListing(t, db, veikko, "Bilberries", "berries")

	got := searchTitles(t, db, database.SearchListingsParams{SellerID: aino.String(), Limit: 50})
	if len(got) != 1 || got[0] != "Chanterelles" {
		t.Errorf("titles = %v, want [Chanterelles]", got)
	}

	unknown := searchTitles(t, db, database.SearchListingsParams{SellerID: uuid.New().String(), Limit: 50})
	if len(unknown) != 0 {
		t.Errorf("titles = %v for a seller nobody is, want none", unknown)
	}
}

func TestTheSellerFilterComposesWithCategory(t *testing.T) {
	db := testdb.New(t)

	aino := makeSeller(t, db, "aino")
	veikko := makeSeller(t, db, "veikko")
	addListing(t, db, aino, "Chanterelles", "mushrooms")
	addListing(t, db, aino, "Aino's Bilberries", "berries")
	addListing(t, db, veikko, "Veikko's Morels", "mushrooms")

	// Each filter alone matches two listings, so only the conjunction can
	// return one - which is what a clause landing in the wrong branch of the
	// builder would get wrong.
	if got := searchTitles(t, db, database.SearchListingsParams{SellerID: aino.String(), Limit: 50}); len(got) != 2 {
		t.Fatalf("the seller filter alone = %v, want two listings", got)
	}
	if got := searchTitles(t, db, database.SearchListingsParams{Category: "mushrooms", Limit: 50}); len(got) != 2 {
		t.Fatalf("the category filter alone = %v, want two listings", got)
	}

	got := searchTitles(t, db, database.SearchListingsParams{
		SellerID: aino.String(), Category: "mushrooms", Limit: 50,
	})
	if len(got) != 1 || got[0] != "Chanterelles" {
		t.Errorf("titles = %v, want [Chanterelles] - the two filters did not compose", got)
	}
}
