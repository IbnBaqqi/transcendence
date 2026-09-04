package handler

import (
	"context"
	"database/sql"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"

	"github.com/IbnBaqqi/transcendence/internal/auth"
	"github.com/IbnBaqqi/transcendence/internal/database"
	"github.com/IbnBaqqi/transcendence/internal/dtos"
	"github.com/IbnBaqqi/transcendence/internal/service"
	"github.com/IbnBaqqi/transcendence/internal/storage"
	"github.com/IbnBaqqi/transcendence/internal/testdb"
)

func reorderHandler(t *testing.T) (*Handler, uuid.UUID, uuid.UUID, []uuid.UUID) {
	t.Helper()
	ctx := context.Background()
	db := testdb.New(t)

	seller, err := db.CreateUser(ctx, database.CreateUserParams{
		ID:       database.NewID(),
		Username: "seller", Email: "seller@example.test",
		Password: sql.NullString{String: "irrelevant", Valid: true},
	})
	if err != nil {
		t.Fatalf("creating the seller: %v", err)
	}

	listing, err := db.CreateListing(ctx, database.CreateListingParams{
		ID:       database.NewID(),
		SellerID: seller.ID, Title: "Chanterelles", Category: "mushrooms",
		Price: "18.10", Quantity: 5, Unit: "kg",
	})
	if err != nil {
		t.Fatalf("creating the listing: %v", err)
	}

	var images []uuid.UUID
	for _, name := range []string{"a.png", "b.png"} {
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

	h := New(Deps{DB: db, ListingImage: service.NewListingImageService(db, files, 5)})
	return h, seller.ID, listing.ID, images
}

func reorderAs(t *testing.T, h *Handler, caller uuid.UUID, idParam, body string) *httptest.ResponseRecorder {
	t.Helper()

	req := httptest.NewRequest(http.MethodPut, "/", strings.NewReader(body))
	routeCtx := chi.NewRouteContext()
	routeCtx.URLParams.Add("id", idParam)
	ctx := context.WithValue(req.Context(), chi.RouteCtxKey, routeCtx)
	ctx = auth.WithUser(ctx, auth.User{ID: caller, Role: auth.RoleUser})

	rec := httptest.NewRecorder()
	h.ReorderListingImages(rec, req.WithContext(ctx))
	return rec
}

// The response is the gallery in its new order, so the caller does not have to
// refetch to find out what it now looks like.
func TestReorderingAnswersWithTheNewOrder(t *testing.T) {
	h, seller, listing, images := reorderHandler(t)

	body := `{"image_ids":["` + images[1].String() + `","` + images[0].String() + `"]}`
	rec := reorderAs(t, h, seller, listing.String(), body)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200: %s", rec.Code, rec.Body.String())
	}

	var got []dtos.ListingImageResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &got); err != nil {
		t.Fatalf("decoding: %v", err)
	}
	if len(got) != 2 || got[0].ID != images[1] || got[1].ID != images[0] {
		t.Errorf("gallery = %v, want the reversed order", got)
	}
}

func TestReorderingRejectsAnInexactList(t *testing.T) {
	h, seller, listing, images := reorderHandler(t)

	body := `{"image_ids":["` + images[0].String() + `"]}`
	rec := reorderAs(t, h, seller, listing.String(), body)

	if rec.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want 400", rec.Code)
	}
}

func TestReorderingRejectsAMalformedBody(t *testing.T) {
	h, seller, listing, _ := reorderHandler(t)

	for name, body := range map[string]string{
		"not an id":     `{"image_ids":["not-a-uuid"]}`,
		"unknown field": `{"image_ids":[],"cover":"a"}`,
		"not json":      `{`,
	} {
		t.Run(name, func(t *testing.T) {
			rec := reorderAs(t, h, seller, listing.String(), body)
			if rec.Code != http.StatusBadRequest {
				t.Errorf("status = %d, want 400", rec.Code)
			}
		})
	}
}
