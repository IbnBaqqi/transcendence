package service

import (
	"context"
	"database/sql"
	"testing"

	"github.com/google/uuid"

	"github.com/IbnBaqqi/transcendence/internal/database"
	"github.com/IbnBaqqi/transcendence/internal/dtos"
	"github.com/IbnBaqqi/transcendence/internal/storage"
	"github.com/IbnBaqqi/transcendence/internal/testdb"
)

type soldOutFixture struct {
	listings *ListingService
	seller   uuid.UUID
	stranger uuid.UUID
}

// Quantity cannot be created at zero - a listing reaches it by selling out - so
// the fixture drops it directly rather than running the whole order handshake.
func newSoldOutFixture(t *testing.T) soldOutFixture {
	t.Helper()

	db := testdb.New(t)
	ctx := context.Background()

	files, err := storage.NewLocal(t.TempDir())
	if err != nil {
		t.Fatalf("temporary upload dir: %v", err)
	}
	t.Cleanup(func() { _ = files.Close() })

	mk := func(name string) uuid.UUID {
		user, err := db.CreateUser(ctx, database.CreateUserParams{
			ID:       database.NewID(),
			Username: name, Email: name + "@example.test",
			Password: sql.NullString{String: "irrelevant", Valid: true},
		})
		if err != nil {
			t.Fatalf("creating %s: %v", name, err)
		}
		return user.ID
	}
	seller, stranger := mk("seller"), mk("stranger")

	listings := NewListingService(db, files)
	for _, title := range []string{"In stock", "Sold out"} {
		created, err := listings.CreateListing(ctx, seller, dtos.CreateListingInput{
			Title: title, Category: "mushrooms", Price: 18.10, Quantity: 4, Unit: "kg",
		})
		if err != nil {
			t.Fatalf("creating %q: %v", title, err)
		}
		if title == "Sold out" {
			if _, err := db.ExecContext(ctx,
				`UPDATE listings SET quantity = 0 WHERE id = $1`, created.ID); err != nil {
				t.Fatalf("selling out %q: %v", title, err)
			}
		}
	}

	return soldOutFixture{listings: listings, seller: seller, stranger: stranger}
}

func (f soldOutFixture) search(t *testing.T, viewer uuid.UUID, q dtos.ListingSearchQuery) dtos.PaginatedListings {
	t.Helper()

	page, err := f.listings.SearchListings(context.Background(), viewer, q)
	if err != nil {
		t.Fatalf("searching: %v", err)
	}
	return page
}

func titles(page dtos.PaginatedListings) []string {
	out := make([]string, 0, len(page.Items))
	for _, item := range page.Items {
		out = append(out, item.Title)
	}
	return out
}

func has(page dtos.PaginatedListings, title string) bool {
	for _, got := range titles(page) {
		if got == title {
			return true
		}
	}
	return false
}

// The row a seller most needs: it is the one to restock or delist, and hiding
// it is how it silently disappears from their own inventory.
func TestASellerSeesTheirOwnSoldOutListing(t *testing.T) {
	f := newSoldOutFixture(t)

	page := f.search(t, f.seller, dtos.ListingSearchQuery{
		SellerID: f.seller.String(), IncludeSoldOut: "true",
	})

	if !has(page, "Sold out") {
		t.Errorf("the seller's own inventory = %v, want the sold-out listing in it", titles(page))
	}
	if !has(page, "In stock") {
		t.Errorf("the seller's own inventory = %v, want the in-stock listing too", titles(page))
	}
}

// Honoured for the seller, ignored for everyone else - including a signed-in
// stranger who has simply read the parameter name off the documentation.
func TestOnlyTheSellerThemselvesCanAskForSoldOut(t *testing.T) {
	anonymous := func(soldOutFixture) uuid.UUID { return uuid.Nil }

	for _, tt := range []struct {
		name     string
		viewer   func(f soldOutFixture) uuid.UUID
		sellerID func(f soldOutFixture) string
	}{
		{
			"a signed-in stranger",
			func(f soldOutFixture) uuid.UUID { return f.stranger },
			func(f soldOutFixture) string { return f.seller.String() },
		},
		{"nobody signed in", anonymous, func(f soldOutFixture) string { return f.seller.String() }},
		// Both sides are uuid.Nil here, and Nil == Nil. Without the guard on
		// the seller id this one case opens sold-out listings to everybody.
		{"no seller_id at all", anonymous, func(soldOutFixture) string { return "" }},
	} {
		t.Run(tt.name, func(t *testing.T) {
			f := newSoldOutFixture(t)

			page := f.search(t, tt.viewer(f), dtos.ListingSearchQuery{
				SellerID: tt.sellerID(f), IncludeSoldOut: "true",
			})

			if has(page, "Sold out") {
				t.Errorf("public search = %v, want the sold-out listing hidden", titles(page))
			}
			if !has(page, "In stock") {
				t.Errorf("public search = %v, want the in-stock listing present", titles(page))
			}
		})
	}
}

// total is counted by a second query. If the flag reaches the page and not the
// count, pagination promises rows it will not deliver.
func TestTheTotalCountsTheSameSetThePageComesFrom(t *testing.T) {
	f := newSoldOutFixture(t)

	page := f.search(t, f.seller, dtos.ListingSearchQuery{
		SellerID: f.seller.String(), IncludeSoldOut: "true",
	})

	if page.Total != int64(len(page.Items)) {
		t.Errorf("total = %d, items = %d - the count query and the page query disagree",
			page.Total, len(page.Items))
	}
}
