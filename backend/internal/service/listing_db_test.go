package service

import (
	"context"
	"database/sql"
	"errors"
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
			page, err := listings.SearchListings(ctx, dtos.ListingSearchQuery{Category: filter})
			if err != nil {
				t.Fatalf("searching: %v", err)
			}
			if page.Total != 1 {
				t.Errorf("?category=%q found %d, want 1", filter, page.Total)
			}
		})
	}
}
