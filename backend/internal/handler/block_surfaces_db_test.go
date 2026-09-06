package handler

import (
	"context"
	"database/sql"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"

	"github.com/IbnBaqqi/transcendence/internal/auth"
	"github.com/IbnBaqqi/transcendence/internal/database"
	"github.com/IbnBaqqi/transcendence/internal/service"
	"github.com/IbnBaqqi/transcendence/internal/storage"
	"github.com/IbnBaqqi/transcendence/internal/testdb"
)

// Every public read that exposes a person or their listing, in one table.
//
// One test per endpoint is what let GET /listings/{id}/images ship readable
// while GET /listings/{id} answered 404 - the two guards are the same three
// terms and only one of them was updated. Enumerating the surfaces means the
// next sibling endpoint fails here rather than in review.
//
// The pair mattering is not only about the one endpoint: a removed listing
// 404s on both, so an endpoint that stays open turns two answers into a signal
// that separates "blocked" from "taken down".
func TestEveryPublicReadIsClosedToABlockedViewer(t *testing.T) {
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
		if err := db.EnsureProfile(ctx, user.ID); err != nil {
			t.Fatalf("profile for %s: %v", name, err)
		}
		return user.ID
	}
	seller, viewer := mk("seller"), mk("viewer")

	listing, err := db.CreateListing(ctx, database.CreateListingParams{
		ID:       database.NewID(),
		SellerID: seller, Title: "Chanterelles", Category: "mushrooms",
		Price: "18.00", Quantity: 4, Unit: "kg",
	})
	if err != nil {
		t.Fatalf("creating a listing: %v", err)
	}

	files, err := storage.NewLocal(t.TempDir())
	if err != nil {
		t.Fatalf("temporary upload dir: %v", err)
	}
	t.Cleanup(func() { _ = files.Close() })

	blocks := service.NewBlockService(db.Queries)
	h := New(Deps{
		DB:           db,
		Block:        blocks,
		Listing:      service.NewListingService(db, files),
		ListingImage: service.NewListingImageService(db, files, 6),
		Profile:      service.NewProfileService(db, files),
		Review:       service.NewReviewService(db),
	})

	surfaces := []struct {
		name    string
		path    string
		param   string
		id      uuid.UUID
		handler http.HandlerFunc
	}{
		{"the listing", "/listings/", "id", listing.ID, h.GetListing},
		{"its images", "/listings/{id}/images", "id", listing.ID, h.GetListingImages},
		{"the seller's profile", "/users/", "id", seller, h.GetPublicProfile},
		{"the seller's reviews", "/users/{id}/reviews", "id", seller, h.GetSellerReviews},
	}

	call := func(t *testing.T, s int, as uuid.UUID) int {
		t.Helper()
		sf := surfaces[s]

		req := httptest.NewRequest(http.MethodGet, sf.path, nil)
		routeCtx := chi.NewRouteContext()
		routeCtx.URLParams.Add(sf.param, sf.id.String())
		reqCtx := context.WithValue(req.Context(), chi.RouteCtxKey, routeCtx)
		reqCtx = auth.WithUser(reqCtx, auth.User{ID: as, Role: auth.RoleUser})

		rec := httptest.NewRecorder()
		sf.handler(rec, req.WithContext(reqCtx))
		return rec.Code
	}

	// The control. Without it a handler that 404s for an unrelated reason - a
	// missing service, a bad route param - would look like the block working.
	for i, sf := range surfaces {
		if code := call(t, i, viewer); code != http.StatusOK {
			t.Fatalf("before any block, %s answered %d, want 200", sf.name, code)
		}
	}

	if err := blocks.Block(ctx, seller, viewer); err != nil {
		t.Fatalf("blocking: %v", err)
	}

	for i, sf := range surfaces {
		t.Run(sf.name, func(t *testing.T) {
			if code := call(t, i, viewer); code != http.StatusNotFound {
				t.Errorf("%s answered %d to a blocked viewer, want 404", sf.name, code)
			}
		})
	}

	t.Run("unblocking reopens all of them", func(t *testing.T) {
		if err := blocks.Unblock(ctx, seller, viewer); err != nil {
			t.Fatalf("unblocking: %v", err)
		}
		for i, sf := range surfaces {
			if code := call(t, i, viewer); code != http.StatusOK {
				t.Errorf("after unblocking, %s answered %d, want 200", sf.name, code)
			}
		}
	})
}
