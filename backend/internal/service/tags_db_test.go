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

func tagFixture(t *testing.T) (*ListingService, *database.DB, uuid.UUID) {
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

	return NewListingService(db, files), db, seller.ID
}

func createTagged(
	t *testing.T,
	s *ListingService,
	seller uuid.UUID,
	title string,
	tags []string,
) database.Listing {
	t.Helper()

	listing, err := s.CreateListing(context.Background(), seller, dtos.CreateListingInput{
		Title: title, Category: "mushrooms", Price: 18.00, Quantity: 4, Unit: "kg",
		Tags: tags,
	})
	if err != nil {
		t.Fatalf("creating %s: %v", title, err)
	}
	return listing
}

func TestTagsSurviveARoundTrip(t *testing.T) {
	listings, db, seller := tagFixture(t)
	ctx := context.Background()

	created := createTagged(t, listings, seller, "Chanterelles",
		[]string{"Chanterelle", "chanterelle", "  roadside  ", "", "SUNNY"})

	got, err := db.ListTagsForListing(ctx, created.ID)
	if err != nil {
		t.Fatalf("reading the tags back: %v", err)
	}

	want := []string{"chanterelle", "roadside", "sunny"}
	if len(got) != len(want) {
		t.Fatalf("tags = %q, want %q", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("tags = %q, want %q", got, want)
		}
	}
}

func TestUpdatingWithoutTagsClearsThem(t *testing.T) {
	listings, db, seller := tagFixture(t)
	ctx := context.Background()

	created := createTagged(t, listings, seller, "Chanterelles", []string{"roadside"})

	if _, err := listings.UpdateListing(ctx, seller, created.ID, dtos.UpdateListingInput{
		Title: "Chanterelles", Category: "mushrooms", Price: 18.00, Quantity: 4, Unit: "kg",
	}); err != nil {
		t.Fatalf("updating: %v", err)
	}

	got, err := db.ListTagsForListing(ctx, created.ID)
	if err != nil {
		t.Fatalf("reading the tags back: %v", err)
	}
	if len(got) != 0 {
		t.Errorf("tags = %q, want none - PUT replaces the whole set", got)
	}
}

func TestATagOutlivesTheListingThatUsedIt(t *testing.T) {
	listings, db, seller := tagFixture(t)
	ctx := context.Background()

	first := createTagged(t, listings, seller, "Chanterelles", []string{"roadside"})
	second := createTagged(t, listings, seller, "Morels", []string{"roadside"})

	if err := listings.DeleteListing(ctx, seller, first.ID); err != nil {
		t.Fatalf("deleting the first listing: %v", err)
	}

	got, err := db.ListTagsForListing(ctx, second.ID)
	if err != nil {
		t.Fatalf("reading the survivor's tags: %v", err)
	}
	if len(got) != 1 || got[0] != "roadside" {
		t.Errorf("the survivor's tags = %q, want [roadside] - deleting one listing took the tag with it", got)
	}
}
