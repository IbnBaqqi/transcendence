package service

import (
	"context"
	"database/sql"
	"errors"
	"testing"

	"github.com/google/uuid"

	"github.com/IbnBaqqi/transcendence/internal/database"
	"github.com/IbnBaqqi/transcendence/internal/storage"
	"github.com/IbnBaqqi/transcendence/internal/testdb"
)

type reorderFixture struct {
	svc     *ListingImageService
	db      *database.DB
	seller  uuid.UUID
	other   uuid.UUID
	listing uuid.UUID
	images  []uuid.UUID
}

func newReorderFixture(t *testing.T) reorderFixture {
	t.Helper()
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

	seller := user("seller")
	other := user("other")

	listing, err := db.CreateListing(ctx, database.CreateListingParams{
		ID:       database.NewID(),
		SellerID: seller, Title: "Chanterelles", Category: "mushrooms",
		Price: "18.10", Quantity: 5, Unit: "kg",
	})
	if err != nil {
		t.Fatalf("creating the listing: %v", err)
	}

	var images []uuid.UUID
	for _, name := range []string{"a.png", "b.png", "c.png"} {
		img, err := db.CreateListingImage(ctx, database.CreateListingImageParams{
			ID: database.NewID(), ListingID: listing.ID, Filename: name,
		})
		if err != nil {
			t.Fatalf("creating %s: %v", name, err)
		}
		images = append(images, img.ID)
	}

	files, err := storage.NewLocal(t.TempDir())
	if err != nil {
		t.Fatalf("temporary upload dir: %v", err)
	}
	t.Cleanup(func() { _ = files.Close() })

	return reorderFixture{
		svc:     NewListingImageService(db, files, 5),
		db:      db,
		seller:  seller,
		other:   other,
		listing: listing.ID,
		images:  images,
	}
}

func (f reorderFixture) order(t *testing.T) []uuid.UUID {
	t.Helper()
	rows, err := f.db.ListListingImages(context.Background(), f.listing)
	if err != nil {
		t.Fatalf("reading the gallery: %v", err)
	}
	out := make([]uuid.UUID, 0, len(rows))
	for _, r := range rows {
		out = append(out, r.ID)
	}
	return out
}

func sameOrder(a, b []uuid.UUID) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

// The whole point: position is only ever written at insert, so without this a
// gallery's order is frozen as its upload order.
func TestReorderingRewritesTheGallery(t *testing.T) {
	f := newReorderFixture(t)
	want := []uuid.UUID{f.images[2], f.images[0], f.images[1]}

	if err := f.svc.ReorderImages(context.Background(), f.seller, f.listing, want); err != nil {
		t.Fatalf("reordering: %v", err)
	}

	if got := f.order(t); !sameOrder(got, want) {
		t.Errorf("gallery = %v, want %v", got, want)
	}
}

// A swap is momentarily two rows at the same position. It is only legal
// because the unique constraint is DEFERRABLE and the work happens in one
// transaction - under autocommit this fails on the first statement.
func TestSwappingTwoAdjacentImagesIsAllowed(t *testing.T) {
	f := newReorderFixture(t)
	want := []uuid.UUID{f.images[1], f.images[0], f.images[2]}

	if err := f.svc.ReorderImages(context.Background(), f.seller, f.listing, want); err != nil {
		t.Fatalf("swapping the first two: %v", err)
	}

	if got := f.order(t); !sameOrder(got, want) {
		t.Errorf("gallery = %v, want %v", got, want)
	}
}

// A gallery half in the old order and half in the new is worse than one that
// did not change, so an inexact list is refused rather than partly applied.
func TestAnInexactListIsRefusedAndChangesNothing(t *testing.T) {
	f := newReorderFixture(t)
	before := f.order(t)

	cases := map[string][]uuid.UUID{
		"missing an image":     {f.images[1], f.images[0]},
		"an extra id":          {f.images[2], f.images[1], f.images[0], database.NewID()},
		"the same id twice":    {f.images[0], f.images[0], f.images[1]},
		"an id from elsewhere": {f.images[0], f.images[1], database.NewID()},
		"an empty list":        {},
	}

	for name, ids := range cases {
		t.Run(name, func(t *testing.T) {
			err := f.svc.ReorderImages(context.Background(), f.seller, f.listing, ids)

			var invalid *ValidationError
			if !errors.As(err, &invalid) {
				t.Fatalf("err = %v, want ValidationError", err)
			}
			if got := f.order(t); !sameOrder(got, before) {
				t.Errorf("the gallery changed: %v, want %v", got, before)
			}
		})
	}
}

func TestOnlyTheSellerCanReorder(t *testing.T) {
	f := newReorderFixture(t)
	before := f.order(t)

	err := f.svc.ReorderImages(context.Background(), f.other, f.listing,
		[]uuid.UUID{f.images[2], f.images[1], f.images[0]})

	var forbidden *ForbiddenError
	if !errors.As(err, &forbidden) {
		t.Fatalf("err = %v, want ForbiddenError", err)
	}
	if got := f.order(t); !sameOrder(got, before) {
		t.Error("a stranger reordered somebody else's gallery")
	}
}

// Reordering to the order it already has is a no-op, not an error - which is
// what makes a full-list PUT idempotent.
func TestReorderingToTheSameOrderIsFine(t *testing.T) {
	f := newReorderFixture(t)
	before := f.order(t)

	if err := f.svc.ReorderImages(context.Background(), f.seller, f.listing, before); err != nil {
		t.Fatalf("reordering to the same order: %v", err)
	}

	if got := f.order(t); !sameOrder(got, before) {
		t.Errorf("gallery = %v, want %v", got, before)
	}
}
