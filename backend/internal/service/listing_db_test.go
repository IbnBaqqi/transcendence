package service

import (
	"context"
	"database/sql"
	"errors"
	"strconv"
	"testing"

	"github.com/google/uuid"

	"github.com/IbnBaqqi/transcendence/internal/database"
	"github.com/IbnBaqqi/transcendence/internal/dtos"
	"github.com/IbnBaqqi/transcendence/internal/storage"
	"github.com/IbnBaqqi/transcendence/internal/testdb"
)

func listingFixture(t *testing.T) (*ListingService, uuid.UUID) {
	t.Helper()

	db := testdb.New(t)

	files, err := storage.NewLocal(t.TempDir())
	if err != nil {
		t.Fatalf("creating a temporary upload dir: %v", err)
	}
	t.Cleanup(func() { _ = files.Close() })

	seller, err := db.CreateUser(context.Background(), database.CreateUserParams{
		ID:       database.NewID(),
		Username: "seller",
		Email:    "seller@example.test",
		Password: sql.NullString{String: "irrelevant", Valid: true},
	})
	if err != nil {
		t.Fatalf("creating the seller: %v", err)
	}

	return NewListingService(db, files), seller.ID
}

func TestAnUnknownCategoryIsRejectedWithAMessage(t *testing.T) {
	listings, seller := listingFixture(t)
	ctx := context.Background()

	_, err := listings.CreateListing(ctx, seller, dtos.CreateListingInput{
		Title: "Golden Chanterelles", Category: "nonsense",
		Price: 18.00, Quantity: 4, Unit: "kg",
	})

	var invalid *ValidationError
	if !errors.As(err, &invalid) {
		t.Fatalf("creating with an unknown category: err = %#v, want *ValidationError - an unmapped 23503 is a 500", err)
	}
	if invalid.Message != "Category is not recognised" {
		t.Errorf("message = %q", invalid.Message)
	}
}

func TestUpdatingToAnUnknownCategoryIsRejectedToo(t *testing.T) {
	listings, seller := listingFixture(t)
	ctx := context.Background()

	created, err := listings.CreateListing(ctx, seller, dtos.CreateListingInput{
		Title: "Golden Chanterelles", Category: "mushrooms",
		Price: 18.00, Quantity: 4, Unit: "kg",
	})
	if err != nil {
		t.Fatalf("creating: %v", err)
	}

	_, err = listings.UpdateListing(ctx, seller, created.ID, dtos.UpdateListingInput{
		Title: "Golden Chanterelles", Category: "nonsense",
		Price: 18.00, Quantity: 4, Unit: "kg",
	})

	var invalid *ValidationError
	if !errors.As(err, &invalid) {
		t.Fatalf("updating to an unknown category: err = %#v, want *ValidationError", err)
	}
	if invalid.Message != "Category is not recognised" {
		t.Errorf("message = %q", invalid.Message)
	}
}

func TestACategoryIsNormalisedTheWayTheMigrationNormalised(t *testing.T) {
	listings, seller := listingFixture(t)
	ctx := context.Background()

	tests := []string{"Mushrooms", "  mushrooms  ", "MUSHROOMS", " Mushrooms "}

	for _, given := range tests {
		t.Run(given, func(t *testing.T) {
			created, err := listings.CreateListing(ctx, seller, dtos.CreateListingInput{
				Title: "Golden Chanterelles", Category: given,
				Price: 18.00, Quantity: 4, Unit: "kg",
			})
			if err != nil {
				t.Fatalf("creating with %q: %v", given, err)
			}
			if created.Category != "mushrooms" {
				t.Errorf("stored %q, want %q", created.Category, "mushrooms")
			}
		})
	}
}

func TestSearchFindsAListingWhateverCaseTheFilterUses(t *testing.T) {
	listings, seller := listingFixture(t)
	ctx := context.Background()

	if _, err := listings.CreateListing(ctx, seller, dtos.CreateListingInput{
		Title: "Golden Chanterelles", Category: "Mushrooms",
		Price: 18.00, Quantity: 4, Unit: "kg",
	}); err != nil {
		t.Fatalf("creating: %v", err)
	}

	for _, filter := range []string{"mushrooms", "Mushrooms", "  MUSHROOMS "} {
		t.Run(filter, func(t *testing.T) {
			page, err := listings.SearchListings(ctx, uuid.Nil, dtos.ListingSearchQuery{Category: filter})
			if err != nil {
				t.Fatalf("searching: %v", err)
			}
			if page.Total != 1 {
				t.Errorf("?category=%q found %d, want 1", filter, page.Total)
			}
		})
	}
}

func twoSellerFixture(t *testing.T) (*ListingService, uuid.UUID, uuid.UUID) {
	t.Helper()

	db := testdb.New(t)

	files, err := storage.NewLocal(t.TempDir())
	if err != nil {
		t.Fatalf("creating a temporary upload dir: %v", err)
	}
	t.Cleanup(func() { _ = files.Close() })

	ids := make([]uuid.UUID, 0, 2)
	for _, name := range []string{"aino", "veikko"} {
		user, err := db.CreateUser(context.Background(), database.CreateUserParams{
			ID:       database.NewID(),
			Username: name,
			Email:    name + "@example.test",
			Password: sql.NullString{String: "irrelevant", Valid: true},
		})
		if err != nil {
			t.Fatalf("creating %s: %v", name, err)
		}
		ids = append(ids, user.ID)
	}

	return NewListingService(db, files), ids[0], ids[1]
}

func TestTheSellerFilterCountsWhatItReturns(t *testing.T) {
	listings, aino, veikko := twoSellerFixture(t)
	ctx := context.Background()

	for i, owner := range []uuid.UUID{aino, aino, aino, veikko, veikko} {
		if _, err := listings.CreateListing(ctx, owner, dtos.CreateListingInput{
			Title: "Listing " + strconv.Itoa(i), Category: "mushrooms",
			Price: 10.00, Quantity: 5, Unit: "kg",
		}); err != nil {
			t.Fatalf("creating listing %d: %v", i, err)
		}
	}

	page, err := listings.SearchListings(ctx, uuid.Nil, dtos.ListingSearchQuery{
		SellerID: aino.String(), Limit: "2",
	})
	if err != nil {
		t.Fatalf("searching: %v", err)
	}

	// total and total_pages describe the filtered set, not the whole table.
	// Move the seller clause below the builder's countOnly split and the rows
	// stay right while these report on all five listings.
	if len(page.Items) != 2 {
		t.Errorf("items = %d, want 2 - the page size", len(page.Items))
	}
	if page.Total != 3 {
		t.Errorf("total = %d, want 3 - aino's listings, not all five", page.Total)
	}
	if page.TotalPages != 2 {
		t.Errorf("total_pages = %d, want 2", page.TotalPages)
	}
}

func TestASellerIdThatIsNotAUUIDIsRejected(t *testing.T) {
	listings, _ := listingFixture(t)

	_, err := listings.SearchListings(context.Background(), uuid.Nil, dtos.ListingSearchQuery{
		SellerID: "not-a-uuid",
	})

	var invalid *ValidationError
	if !errors.As(err, &invalid) {
		t.Fatalf("error = %v, want a validation error - the ::uuid cast alone would make this a 500", err)
	}
}

// The busiest read in the app: GET /listings/{id} and the images route behind
// it both come through here. A 404 test cannot see the collapse - only a real
// listing whose lookup genuinely fails can.
func TestADatabaseFailureIsNotReportedAsAMissingListingToABuyer(t *testing.T) {
	listings, seller := listingFixture(t)

	listing, err := listings.CreateListing(context.Background(), seller, dtos.CreateListingInput{
		Title: "Chanterelles", Category: "mushrooms",
		Price: 18.10, Quantity: 4, Unit: "kg",
	})
	if err != nil {
		t.Fatalf("creating the listing: %v", err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	_, err = listings.GetListing(ctx, listing.ID)

	var notFound *NotFoundError
	if errors.As(err, &notFound) {
		t.Fatalf("err = %v, want the underlying failure rather than a 404", err)
	}
	if !errors.Is(err, context.Canceled) {
		t.Errorf("err = %v, want the context cancellation to survive", err)
	}
}

func TestAListingThatDoesNotExistIsStillNotFound(t *testing.T) {
	listings, _ := listingFixture(t)

	_, err := listings.GetListing(context.Background(), database.NewID())

	var notFound *NotFoundError
	if !errors.As(err, &notFound) {
		t.Fatalf("err = %v, want NotFoundError", err)
	}
}

// The premise of 024: an order does not need its listing. Every field a reader
// sees is snapshotted on the order row at purchase, so SET NULL lets the
// listing go while the order stays readable. Exercised through the database
// rather than the service because it is the constraint being tested, not a
// branch - the service is not allowed to delete a listing with orders yet.
func TestAnOrderOutlivesTheListingItWasPlacedOn(t *testing.T) {
	ctx := context.Background()
	db := testdb.New(t)

	seller, err := db.CreateUser(ctx, database.CreateUserParams{
		ID: database.NewID(), Username: "seller", Email: "seller@example.test",
		Password: sql.NullString{String: "irrelevant", Valid: true},
	})
	if err != nil {
		t.Fatalf("creating the seller: %v", err)
	}
	buyer, err := db.CreateUser(ctx, database.CreateUserParams{
		ID: database.NewID(), Username: "buyer", Email: "buyer@example.test",
		Password: sql.NullString{String: "irrelevant", Valid: true},
	})
	if err != nil {
		t.Fatalf("creating the buyer: %v", err)
	}

	listing, err := db.CreateListing(ctx, database.CreateListingParams{
		ID: database.NewID(), SellerID: seller.ID, Title: "Chanterelles",
		Category: "mushrooms", Price: "18.00", Quantity: 3, Unit: "kg",
	})
	if err != nil {
		t.Fatalf("creating the listing: %v", err)
	}

	order, err := db.CreateOrder(ctx, database.CreateOrderParams{
		ID: database.NewID(), ListingID: uuid.NullUUID{UUID: listing.ID, Valid: true},
		BuyerID: buyer.ID, SellerID: seller.ID, Quantity: 2, UnitPrice: "18.00",
		ListingTitle: listing.Title,
	})
	if err != nil {
		t.Fatalf("creating the order: %v", err)
	}

	if err := db.DeleteListing(ctx, listing.ID); err != nil {
		t.Fatalf("deleting the listing: %v", err)
	}

	survived, err := db.GetOrder(ctx, order.ID)
	if err != nil {
		t.Fatalf("reading the order back: %v", err)
	}
	if survived.ListingID.Valid {
		t.Errorf("listing_id = %v, want null once the listing is deleted", survived.ListingID.UUID)
	}
	// The snapshot is the whole reason this is safe.
	if survived.ListingTitle != "Chanterelles" {
		t.Errorf("listing title = %q, want %q", survived.ListingTitle, "Chanterelles")
	}
	if survived.Quantity != 2 {
		t.Errorf("quantity = %d, want 2", survived.Quantity)
	}
}

// Seller, buyer and a listing with one order in the status the caller wants.
func orderedListingFixture(t *testing.T, status string) (*ListingService, *database.DB, uuid.UUID, uuid.UUID) {
	t.Helper()
	ctx := context.Background()
	db := testdb.New(t)

	files, err := storage.NewLocal(t.TempDir())
	if err != nil {
		t.Fatalf("creating a temporary upload dir: %v", err)
	}
	t.Cleanup(func() { _ = files.Close() })

	seller, err := db.CreateUser(ctx, database.CreateUserParams{
		ID: database.NewID(), Username: "seller", Email: "seller@example.test",
		Password: sql.NullString{String: "irrelevant", Valid: true},
	})
	if err != nil {
		t.Fatalf("creating the seller: %v", err)
	}
	buyer, err := db.CreateUser(ctx, database.CreateUserParams{
		ID: database.NewID(), Username: "buyer", Email: "buyer@example.test",
		Password: sql.NullString{String: "irrelevant", Valid: true},
	})
	if err != nil {
		t.Fatalf("creating the buyer: %v", err)
	}

	listing, err := db.CreateListing(ctx, database.CreateListingParams{
		ID: database.NewID(), SellerID: seller.ID, Title: "Chanterelles",
		Category: "mushrooms", Price: "18.00", Quantity: 3, Unit: "kg",
	})
	if err != nil {
		t.Fatalf("creating the listing: %v", err)
	}

	order, err := db.CreateOrder(ctx, database.CreateOrderParams{
		ID: database.NewID(), ListingID: uuid.NullUUID{UUID: listing.ID, Valid: true},
		BuyerID: buyer.ID, SellerID: seller.ID, Quantity: 1, UnitPrice: "18.00",
		ListingTitle: listing.Title,
	})
	if err != nil {
		t.Fatalf("creating the order: %v", err)
	}
	if status != "pending" {
		if _, err := db.UpdateOrderStatus(ctx, database.UpdateOrderStatusParams{
			ID: order.ID, Status: status,
		}); err != nil {
			t.Fatalf("setting the order to %s: %v", status, err)
		}
	}

	return NewListingService(db, files), db, seller.ID, listing.ID
}

// The rule in one table: an order still in flight blocks the delete, a finished
// one does not. Before 024 every one of these blocked, and a seller who had
// ever sold anything could never remove the listing.
func TestOnlyASaleInFlightBlocksDeletingAListing(t *testing.T) {
	for _, tc := range []struct {
		status  string
		deletes bool
	}{
		{status: "pending", deletes: false},
		{status: "confirmed", deletes: false},
		{status: "completed", deletes: true},
		{status: "cancelled", deletes: true},
		{status: "refunded", deletes: true},
	} {
		t.Run(tc.status, func(t *testing.T) {
			ctx := context.Background()
			listings, db, seller, listingID := orderedListingFixture(t, tc.status)

			err := listings.DeleteListing(ctx, seller, listingID)

			if !tc.deletes {
				var conflict *ConflictError
				if !errors.As(err, &conflict) {
					t.Fatalf("err = %#v, want *ConflictError - %s is still in flight", err, tc.status)
				}
				if _, err := db.GetListing(ctx, listingID); err != nil {
					t.Errorf("listing should still be there after a refused delete: %v", err)
				}
				return
			}

			if err != nil {
				t.Fatalf("deleting with only a %s order: %v", tc.status, err)
			}
			if _, err := db.GetListing(ctx, listingID); !errors.Is(err, sql.ErrNoRows) {
				t.Errorf("GetListing err = %v, want sql.ErrNoRows - the row should be gone", err)
			}
		})
	}
}

// The other half of the same change: the order the seller sold is still there
// afterwards, reading off its own snapshot. Deleting a listing must not delete
// somebody else's receipt.
func TestACompletedOrderSurvivesTheSellerDeletingTheListing(t *testing.T) {
	ctx := context.Background()
	listings, db, seller, listingID := orderedListingFixture(t, "completed")

	if err := listings.DeleteListing(ctx, seller, listingID); err != nil {
		t.Fatalf("deleting the listing: %v", err)
	}

	orders, err := db.ListOrdersForUser(ctx, seller)
	if err != nil {
		t.Fatalf("listing the seller's orders: %v", err)
	}
	if len(orders) != 1 {
		t.Fatalf("orders = %d, want 1 - the sale must outlive the listing", len(orders))
	}
	if orders[0].ListingID.Valid {
		t.Errorf("listing_id = %v, want null", orders[0].ListingID.UUID)
	}
	if orders[0].ListingTitle != "Chanterelles" {
		t.Errorf("listing title = %q, want %q", orders[0].ListingTitle, "Chanterelles")
	}
}
