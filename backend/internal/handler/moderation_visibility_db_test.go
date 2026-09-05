package handler

import (
	"context"
	"database/sql"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"

	"github.com/IbnBaqqi/transcendence/internal/auth"
	"github.com/IbnBaqqi/transcendence/internal/database"
	"github.com/IbnBaqqi/transcendence/internal/notify"
	"github.com/IbnBaqqi/transcendence/internal/service"
	"github.com/IbnBaqqi/transcendence/internal/storage"
	"github.com/IbnBaqqi/transcendence/internal/testdb"
)

func fetchListingAs(t *testing.T, h *Handler, listingID uuid.UUID, viewer *auth.User) *httptest.ResponseRecorder {
	t.Helper()

	req := httptest.NewRequest(http.MethodGet, "/listings/"+listingID.String(), nil)

	routeCtx := chi.NewRouteContext()
	routeCtx.URLParams.Add("id", listingID.String())
	ctx := context.WithValue(req.Context(), chi.RouteCtxKey, routeCtx)
	if viewer != nil {
		ctx = auth.WithUser(ctx, *viewer)
	}

	rec := httptest.NewRecorder()
	h.GetListing(rec, req.WithContext(ctx))
	return rec
}

func TestARemovedListingIsHiddenFromEveryoneButItsSellerAndAdmins(t *testing.T) {
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
	seller, stranger, admin := mk("seller"), mk("stranger"), mk("admin")

	listing, err := db.CreateListing(ctx, database.CreateListingParams{
		ID:       database.NewID(),
		SellerID: seller, Title: "Chanterelles", Category: "mushrooms",
		Price: "18.10", Quantity: 5, Unit: "kg",
	})
	if err != nil {
		t.Fatalf("creating a listing: %v", err)
	}

	// The visibility check re-reads the role from the database, so a token
	// claiming ADMIN is not enough - the row has to say so too.
	if _, err := db.ExecContext(ctx, `UPDATE users SET role = 'ADMIN' WHERE id = $1`, admin); err != nil {
		t.Fatalf("promoting the admin: %v", err)
	}

	files, err := storage.NewLocal(t.TempDir())
	if err != nil {
		t.Fatalf("creating a temporary upload dir: %v", err)
	}
	t.Cleanup(func() { _ = files.Close() })

	mod := service.NewModerationService(db, files)
	if _, _, err := mod.Moderate(ctx, admin, listing.ID, "remove", "prohibited species"); err != nil {
		t.Fatalf("removing: %v", err)
	}

	h := New(Deps{
		DB:           db,
		Listing:      service.NewListingService(db, files),
		ListingImage: service.NewListingImageService(db, files, 5),
	})

	t.Run("anonymous gets 404", func(t *testing.T) {
		if code := fetchListingAs(t, h, listing.ID, nil).Code; code != http.StatusNotFound {
			t.Errorf("status = %d, want 404", code)
		}
	})

	t.Run("a stranger gets 404", func(t *testing.T) {
		viewer := auth.User{ID: stranger, Role: auth.RoleUser}
		if code := fetchListingAs(t, h, listing.ID, &viewer).Code; code != http.StatusNotFound {
			t.Errorf("status = %d, want 404 - a removed listing must look like one that never existed", code)
		}
	})

	t.Run("the seller sees it, marked removed", func(t *testing.T) {
		viewer := auth.User{ID: seller, Role: auth.RoleUser}
		rec := fetchListingAs(t, h, listing.ID, &viewer)
		if rec.Code != http.StatusOK {
			t.Fatalf("status = %d, want 200 - the seller must be able to see what happened", rec.Code)
		}
		if !strings.Contains(rec.Body.String(), `"removed_at"`) {
			t.Errorf("body has no removed_at, so the seller cannot tell:\n%s", rec.Body.String())
		}
	})

	t.Run("an admin sees it", func(t *testing.T) {
		viewer := auth.User{ID: admin, Role: auth.RoleAdmin}
		if code := fetchListingAs(t, h, listing.ID, &viewer).Code; code != http.StatusOK {
			t.Errorf("status = %d, want 200", code)
		}
	})

	t.Run("a token claiming ADMIN over a demoted account gets 404", func(t *testing.T) {
		viewer := auth.User{ID: stranger, Role: auth.RoleAdmin}
		if code := fetchListingAs(t, h, listing.ID, &viewer).Code; code != http.StatusNotFound {
			t.Errorf("status = %d, want 404 - the role must come from the database, like RequireRole", code)
		}
	})

	fetchImagesAs := func(t *testing.T, viewer *auth.User) int {
		t.Helper()

		req := httptest.NewRequest(http.MethodGet, "/listings/"+listing.ID.String()+"/images", nil)

		routeCtx := chi.NewRouteContext()
		routeCtx.URLParams.Add("id", listing.ID.String())
		ctx := context.WithValue(req.Context(), chi.RouteCtxKey, routeCtx)
		if viewer != nil {
			ctx = auth.WithUser(ctx, *viewer)
		}

		rec := httptest.NewRecorder()
		h.GetListingImages(rec, req.WithContext(ctx))
		return rec.Code
	}

	t.Run("its images are 404 for an outsider", func(t *testing.T) {
		if code := fetchImagesAs(t, nil); code != http.StatusNotFound {
			t.Errorf("status = %d, want 404 - a 200 here confirms the listing exists and was moderated", code)
		}
	})

	t.Run("its images stay visible to the seller", func(t *testing.T) {
		viewer := auth.User{ID: seller, Role: auth.RoleUser}
		if code := fetchImagesAs(t, &viewer); code != http.StatusOK {
			t.Errorf("status = %d, want 200", code)
		}
	})
}

// The seller-deleted check lives here rather than in the GetListing query, so
// that an admin can still open a reported listing whose seller has left.
func TestADepartedSellersListingIsHiddenFromEveryoneButAdmins(t *testing.T) {
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
			t.Fatalf("creating %s's profile: %v", name, err)
		}
		return user.ID
	}
	seller, admin, stranger := mk("seller"), mk("admin"), mk("stranger")

	if _, err := db.ExecContext(ctx, `UPDATE users SET role = 'ADMIN' WHERE id = $1`, admin); err != nil {
		t.Fatalf("promoting the admin: %v", err)
	}

	listing, err := db.CreateListing(ctx, database.CreateListingParams{
		ID:       database.NewID(),
		SellerID: seller, Title: "Chanterelles", Category: "mushrooms",
		Price: "18.10", Quantity: 5, Unit: "kg",
	})
	if err != nil {
		t.Fatalf("creating a listing: %v", err)
	}

	files, err := storage.NewLocal(t.TempDir())
	if err != nil {
		t.Fatalf("creating a temporary upload dir: %v", err)
	}
	t.Cleanup(func() { _ = files.Close() })

	if err := service.NewUserService(db, files, notify.Disabled{}).DeleteAccount(ctx, seller, "seller"); err != nil {
		t.Fatalf("deleting the seller: %v", err)
	}

	h := New(Deps{
		DB:           db,
		Listing:      service.NewListingService(db, files),
		ListingImage: service.NewListingImageService(db, files, 5),
	})

	t.Run("a stranger gets 404", func(t *testing.T) {
		viewer := auth.User{ID: stranger, Role: auth.RoleUser}
		if code := fetchListingAs(t, h, listing.ID, &viewer).Code; code != http.StatusNotFound {
			t.Errorf("status = %d, want 404", code)
		}
	})

	t.Run("an admin can still open it to judge a report", func(t *testing.T) {
		viewer := auth.User{ID: admin, Role: auth.RoleAdmin}
		if code := fetchListingAs(t, h, listing.ID, &viewer).Code; code != http.StatusOK {
			t.Errorf("status = %d, want 200 - moderation cannot judge what it cannot open", code)
		}
	})
}
