package app

import (
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"reflect"
	"testing"

	"github.com/go-chi/chi/v5"

	"github.com/IbnBaqqi/transcendence/internal/database"
	mw "github.com/IbnBaqqi/transcendence/internal/middleware"
	"github.com/IbnBaqqi/transcendence/internal/storage"
)

func TestOrderEventsRouteRequiresAuth(t *testing.T) {
	files, err := storage.NewLocal(t.TempDir())
	if err != nil {
		t.Fatalf("temporary upload dir: %v", err)
	}
	t.Cleanup(func() { _ = files.Close() })

	handler := NewRouter(
		slog.New(slog.NewTextHandler(io.Discard, nil)),
		&api{Files: files, DB: &database.DB{}},
	)

	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/api/v1/orders/1/events", nil))
	if rec.Code != http.StatusUnauthorized {
		t.Errorf("status = %d, want 401", rec.Code)
	}

	routes, ok := handler.(chi.Routes)
	if !ok {
		t.Fatalf("NewRouter no longer returns a chi router")
	}

	const want = "GET /api/v1/orders/{id}/events"
	requiredAuth := reflect.ValueOf(mw.RequiredAuth).Pointer()

	found := false
	if err := chi.Walk(routes, func(method, route string, _ http.Handler, mws ...func(http.Handler) http.Handler) error {
		if method+" "+route != want {
			return nil
		}
		found = true
		for _, m := range mws {
			if reflect.ValueOf(m).Pointer() == requiredAuth {
				return nil
			}
		}
		t.Errorf("%s is routed outside the requiredAuth group", want)
		return nil
	}); err != nil {
		t.Fatalf("walking the router: %v", err)
	}
	if !found {
		t.Errorf("%s is not registered", want)
	}
}
