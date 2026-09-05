package handler

import (
	"context"
	"database/sql"
	"encoding/json"
	"net/http"
	"testing"

	"github.com/google/uuid"

	"github.com/IbnBaqqi/transcendence/internal/database"
	"github.com/IbnBaqqi/transcendence/internal/dtos"
	"github.com/IbnBaqqi/transcendence/internal/service"
	"github.com/IbnBaqqi/transcendence/internal/storage"
	"github.com/IbnBaqqi/transcendence/internal/testdb"
)

// A listing carries only seller_id, so the name and picture on a card are
// decorated onto the response. Nothing else asserts they survive the trip.
func TestListingResponsesCarryTheirSeller(t *testing.T) {
	db := testdb.New(t)
	ctx := context.Background()

	user, err := db.CreateUser(ctx, database.CreateUserParams{
		ID:       database.NewID(),
		Username: "mushroom_matti", Email: "matti@example.test",
		Password: sql.NullString{String: "irrelevant", Valid: true},
	})
	if err != nil {
		t.Fatalf("creating the seller: %v", err)
	}
	if err := db.EnsureProfile(ctx, user.ID); err != nil {
		t.Fatalf("creating the profile: %v", err)
	}
	if _, err := db.SetAvatar(ctx, database.SetAvatarParams{
		ID:             user.ID,
		AvatarFilename: sql.NullString{String: "matti.png", Valid: true},
	}); err != nil {
		t.Fatalf("setting the avatar: %v", err)
	}

	listing, err := db.CreateListing(ctx, database.CreateListingParams{
		ID:       database.NewID(),
		SellerID: user.ID, Title: "Chanterelles", Category: "mushrooms",
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

	h := New(Deps{
		DB:           db,
		Listing:      service.NewListingService(db, files),
		ListingImage: service.NewListingImageService(db, files, 5),
	})

	assertSeller := func(t *testing.T, seller *dtos.ListingSeller) {
		t.Helper()
		if seller == nil {
			t.Fatal("seller is null - the response was never decorated")
		}
		if seller.ID != user.ID || seller.Username != "mushroom_matti" {
			t.Errorf("seller = %+v, want mushroom_matti", seller)
		}
		if seller.AvatarURL == nil || *seller.AvatarURL != dtos.UploadURLPrefix+"matti.png" {
			t.Errorf("avatar_url = %v, want the stored path", seller.AvatarURL)
		}
	}

	t.Run("the detail response", func(t *testing.T) {
		rec := fetchListingAs(t, h, listing.ID, nil)
		if rec.Code != http.StatusOK {
			t.Fatalf("status = %d, want 200", rec.Code)
		}

		var got dtos.ListingResponse
		if err := json.Unmarshal(rec.Body.Bytes(), &got); err != nil {
			t.Fatalf("decoding: %v", err)
		}
		assertSeller(t, got.Seller)
	})

	// The search path decorates a whole page at once through a different
	// helper, so passing the detail case says nothing about this one.
	t.Run("the search results", func(t *testing.T) {
		result, err := h.Listing.SearchListings(ctx, uuid.Nil, dtos.ListingSearchQuery{
			SellerID: user.ID.String(),
		})
		if err != nil {
			t.Fatalf("searching: %v", err)
		}
		if len(result.Items) == 0 {
			t.Fatal("no items came back, so nothing was asserted")
		}
		assertSeller(t, result.Items[0].Seller)
	})
}

// Confirms the LEFT JOIN behind a follow row: the person is listed whether or
// not they ever uploaded a picture.
func TestFollowListsSurviveAMissingAvatar(t *testing.T) {
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
	follower, followee := mk("follower"), mk("followee")

	if _, err := db.FollowUser(ctx, database.FollowUserParams{
		FollowerID: follower, FolloweeID: followee,
	}); err != nil {
		t.Fatalf("following: %v", err)
	}

	rows, err := db.ListFollowing(ctx, database.ListFollowingParams{
		ViewerID: follower, SubjectID: follower,
	})
	if err != nil {
		t.Fatalf("listing: %v", err)
	}
	if len(rows) != 1 {
		t.Fatalf("got %d rows, want 1 - an inner join would drop a user with no profile row", len(rows))
	}
	if rows[0].AvatarFilename.Valid {
		t.Errorf("avatar_filename = %q, want null", rows[0].AvatarFilename.String)
	}
}
