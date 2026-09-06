package service

import (
	"context"
	"database/sql"
	"errors"
	"sync"
	"testing"

	"github.com/google/uuid"

	"github.com/IbnBaqqi/transcendence/internal/database"
	"github.com/IbnBaqqi/transcendence/internal/dtos"
	"github.com/IbnBaqqi/transcendence/internal/notify"
	"github.com/IbnBaqqi/transcendence/internal/testdb"
)

// Two buyers, one unit, at the same instant. Exactly one may win.
//
// CreateOrder takes GetListingForUpdate (SELECT ... FOR UPDATE) and then
// decrements with `WHERE quantity >= $2`, so there are two independent guards.
// This fires them both at a listing with a single unit left.
func TestTwoBuyersCannotTakeTheSameLastUnit(t *testing.T) {
	db := testdb.New(t)
	ctx := context.Background()

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
	seller, alice, bob := mk("seller"), mk("alice"), mk("bob")

	listing, err := db.CreateListing(ctx, database.CreateListingParams{
		ID:       database.NewID(),
		SellerID: seller, Title: "The last punnet", Category: "berries",
		Price: "9.00", Quantity: 1, Unit: "kg",
	})
	if err != nil {
		t.Fatalf("creating the listing: %v", err)
	}

	orders := NewOrderService(db, notify.Disabled{})

	// Both goroutines wait on the same gate, so they enter CreateOrder together
	// rather than one simply finishing first.
	var gate sync.WaitGroup
	gate.Add(1)
	results := make(chan error, 2)

	var runners sync.WaitGroup
	for _, buyer := range []uuid.UUID{alice, bob} {
		runners.Add(1)
		go func(buyer uuid.UUID) {
			defer runners.Done()
			gate.Wait()
			_, err := orders.CreateOrder(ctx, buyer, dtos.CreateOrderInput{
				ListingID: listing.ID, Quantity: 1,
			})
			results <- err
		}(buyer)
	}
	gate.Done()
	runners.Wait()
	close(results)

	var won int
	var refusals []error
	for err := range results {
		if err == nil {
			won++
			continue
		}
		refusals = append(refusals, err)
	}

	if won != 1 {
		t.Errorf("%d buyers got the last unit, want exactly 1", won)
	}
	for _, err := range refusals {
		var conflict *ConflictError
		if !errors.As(err, &conflict) {
			t.Errorf("the loser got %#v, want *ConflictError - a race should read as "+
				"'out of stock', not as a crash", err)
		}
	}

	after, err := db.GetListing(ctx, listing.ID)
	if err != nil {
		t.Fatalf("re-reading the listing: %v", err)
	}
	if after.Quantity != 0 {
		t.Errorf("quantity = %d, want 0 - the stock did not balance", after.Quantity)
	}

	var placed int
	if err := db.QueryRow(
		`SELECT count(*) FROM orders WHERE listing_id = $1`, listing.ID,
	).Scan(&placed); err != nil {
		t.Fatal(err)
	}
	if placed != 1 {
		t.Errorf("%d orders exist for one unit of stock", placed)
	}
}
