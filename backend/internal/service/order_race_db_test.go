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

	// Both goroutines park on the same gate so they enter CreateOrder together
	// rather than one simply finishing first.
	//
	// ready is what makes that true. Opening the gate straight after spawning
	// would not: nothing says either goroutine has reached Wait yet, so the
	// gate could be a no-op and the two calls could run one after the other -
	// which still passes, because the plain check at order.go:86 refuses the
	// second. The test would look green while never racing anything.
	var ready, gate sync.WaitGroup
	ready.Add(2)
	gate.Add(1)

	type attempt struct {
		buyer uuid.UUID
		err   error
	}
	results := make(chan attempt, 2)

	var runners sync.WaitGroup
	for _, buyer := range []uuid.UUID{alice, bob} {
		runners.Add(1)
		go func(buyer uuid.UUID) {
			defer runners.Done()
			ready.Done()
			gate.Wait()
			_, err := orders.CreateOrder(ctx, buyer, dtos.CreateOrderInput{
				ListingID: listing.ID, Quantity: 1,
			})
			results <- attempt{buyer: buyer, err: err}
		}(buyer)
	}

	ready.Wait() // both are at the gate
	gate.Done()
	runners.Wait()
	close(results)

	var winner uuid.UUID
	var won int
	var refusals []error
	for got := range results {
		if got.err == nil {
			won++
			winner = got.buyer
			continue
		}
		refusals = append(refusals, got.err)
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

	// Raw SQL rather than a sql/queries entry: this asks a question only the
	// test has, and a generated query existing solely for one assertion is the
	// worse trade.
	//
	// The buyer ids, not a count: how many rows there are and who owns them are
	// the same question here, and one row proves a single sale happened rather
	// than that it belongs to the caller who got nil back.
	rows, err := db.Query(`SELECT buyer_id FROM orders WHERE listing_id = $1`, listing.ID)
	if err != nil {
		t.Fatal(err)
	}
	defer rows.Close()

	var owners []uuid.UUID
	for rows.Next() {
		var owner uuid.UUID
		if err := rows.Scan(&owner); err != nil {
			t.Fatal(err)
		}
		owners = append(owners, owner)
	}
	if err := rows.Err(); err != nil {
		t.Fatal(err)
	}

	if len(owners) != 1 {
		t.Fatalf("%d orders exist for one unit of stock", len(owners))
	}
	if won == 1 && owners[0] != winner {
		t.Errorf("the order belongs to %s but %s is the buyer whose call succeeded",
			owners[0], winner)
	}
}
