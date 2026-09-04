package handler

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-chi/chi/v5"

	"github.com/IbnBaqqi/transcendence/internal/database"
	"github.com/IbnBaqqi/transcendence/internal/service"
	"github.com/IbnBaqqi/transcendence/internal/storage"
	"github.com/IbnBaqqi/transcendence/internal/testdb"
)

// ListImages is a bare delegate, so on its own it would answer 200 [] for a
// listing that never existed. It does not, because the handler establishes the
// listing first - and this pins that, so the guard cannot be dropped as
// redundant on the grounds that the service will catch it.
func TestTheImagesOfAListingThatDoesNotExistAre404(t *testing.T) {
	db := testdb.New(t)

	files, err := storage.NewLocal(t.TempDir())
	if err != nil {
		t.Fatalf("temporary upload dir: %v", err)
	}
	t.Cleanup(func() { _ = files.Close() })

	h := New(Deps{
		DB:           db,
		Listing:      service.NewListingService(db, files),
		ListingImage: service.NewListingImageService(db, files, 5),
	})

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	routeCtx := chi.NewRouteContext()
	routeCtx.URLParams.Add("id", database.NewID().String())
	rec := httptest.NewRecorder()

	h.GetListingImages(rec, req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, routeCtx)))

	if rec.Code != http.StatusNotFound {
		t.Fatalf("got %d with body %s, want 404 - an empty gallery and no such listing are different answers",
			rec.Code, rec.Body.String())
	}
}
