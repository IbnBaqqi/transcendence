package service

import (
	"context"
	"database/sql"
	"testing"

	"github.com/google/uuid"

	"github.com/IbnBaqqi/transcendence/internal/database"
	"github.com/IbnBaqqi/transcendence/internal/storage"
	"github.com/IbnBaqqi/transcendence/internal/testdb"
)

// The rating reaches a listing card through SellersFor, which batches one
// page's sellers into a single query. These pin what that query computes.
func TestSellersForCarriesTheRating(t *testing.T) {
	ctx := context.Background()
	db := testdb.New(t)

	user := func(name string) uuid.UUID {
		t.Helper()
		u, err := db.CreateUser(ctx, database.CreateUserParams{
			ID:       database.NewID(),
			Username: name, Email: name + "@example.test",
			Password: sql.NullString{String: "irrelevant", Valid: true},
		})
		if err != nil {
			t.Fatalf("creating %s: %v", name, err)
		}
		return u.ID
	}

	rated := user("rated")
	unrated := user("unrated")

	listingFor := func(seller uuid.UUID, title string) database.Listing {
		t.Helper()
		l, err := db.CreateListing(ctx, database.CreateListingParams{
			ID: database.NewID(), SellerID: seller, Title: title,
			Category: "mushrooms", Price: "18.10", Quantity: 5, Unit: "kg",
		})
		if err != nil {
			t.Fatalf("creating %s: %v", title, err)
		}
		return l
	}

	ratedListing := listingFor(rated, "Chanterelles")
	unratedListing := listingFor(unrated, "Chicken of the woods")

	// 5 and 4 average to 4.5 - a value no COALESCE-to-zero could produce by
	// accident, so a wrong join shows up as a wrong number rather than a zero.
	for _, score := range []int32{5, 4} {
		buyer := user("buyer" + string(rune('a'+score)))
		order, err := db.CreateOrder(ctx, database.CreateOrderParams{
			ID: database.NewID(), ListingID: uuid.NullUUID{UUID: ratedListing.ID, Valid: true},
			BuyerID: buyer, SellerID: rated, Quantity: 1, UnitPrice: "18.10",
		})
		if err != nil {
			t.Fatalf("creating an order: %v", err)
		}
		if _, err := db.CreateReview(ctx, database.CreateReviewParams{
			ID: database.NewID(), OrderID: order.ID, SellerID: rated,
			ReviewerID: uuid.NullUUID{UUID: buyer, Valid: true}, Rating: score,
		}); err != nil {
			t.Fatalf("creating a review: %v", err)
		}
	}

	files, err := storage.NewLocal(t.TempDir())
	if err != nil {
		t.Fatalf("temporary upload dir: %v", err)
	}
	t.Cleanup(func() { _ = files.Close() })

	svc := NewListingService(db, files)
	bySeller, err := svc.SellersFor(ctx, []database.Listing{ratedListing, unratedListing})
	if err != nil {
		t.Fatalf("decorating the sellers: %v", err)
	}

	if got := bySeller[rated].Rating; got.Average != 4.5 || got.Count != 2 {
		t.Errorf("rated seller = %+v, want {4.5 2}", got)
	}

	// Zero average and zero count: the average alone cannot say which of the
	// two this is, which is why the count travels with it.
	if got := bySeller[unrated].Rating; got.Average != 0 || got.Count != 0 {
		t.Errorf("unrated seller = %+v, want {0 0}", got)
	}
}
