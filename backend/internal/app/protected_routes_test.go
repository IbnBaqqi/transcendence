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

func TestProtectedRoutesRejectAnonymousCallers(t *testing.T) {
	files, err := storage.NewLocal(t.TempDir())
	if err != nil {
		t.Fatalf("temporary upload dir: %v", err)
	}
	t.Cleanup(func() { _ = files.Close() })

	router := NewRouter(
		slog.New(slog.NewTextHandler(io.Discard, nil)),
		&api{Files: files, DB: &database.DB{}},
	)

	const id = "11111111-1111-1111-1111-111111111111"

	tests := []struct {
		method string
		path   string
	}{
		{http.MethodGet, "/api/v1/me/following"},
		{http.MethodPost, "/api/v1/users/" + id + "/follow"},
		{http.MethodDelete, "/api/v1/users/" + id + "/follow"},
		{http.MethodGet, "/api/v1/users/" + id + "/followers"},
	}

	for _, tt := range tests {
		t.Run(tt.method+" "+tt.path, func(t *testing.T) {
			rec := httptest.NewRecorder()
			router.ServeHTTP(rec, httptest.NewRequest(tt.method, tt.path, nil))

			if rec.Code != http.StatusUnauthorized {
				t.Errorf("status = %d, want 401 - is this route inside the RequiredAuth group?", rec.Code)
			}
		})
	}
}

func TestProtectedRoutesAreInsideTheAuthGroup(t *testing.T) {
	files, err := storage.NewLocal(t.TempDir())
	if err != nil {
		t.Fatalf("temporary upload dir: %v", err)
	}
	t.Cleanup(func() { _ = files.Close() })

	handler := NewRouter(
		slog.New(slog.NewTextHandler(io.Discard, nil)),
		&api{Files: files, DB: &database.DB{}},
	)
	routes, ok := handler.(chi.Routes)
	if !ok {
		t.Fatal("NewRouter no longer returns a chi router")
	}

	want := map[string]bool{
		"GET /api/v1/me/following":         true,
		"POST /api/v1/users/{id}/follow":   true,
		"DELETE /api/v1/users/{id}/follow": true,
		"GET /api/v1/users/{id}/followers": true,
	}
	requiredAuth := reflect.ValueOf(mw.RequiredAuth).Pointer()

	seen := map[string]bool{}
	err = chi.Walk(routes, func(method, route string, _ http.Handler, mws ...func(http.Handler) http.Handler) error {
		key := method + " " + route
		if !want[key] {
			return nil
		}
		seen[key] = true

		for _, m := range mws {
			if reflect.ValueOf(m).Pointer() == requiredAuth {
				return nil
			}
		}
		t.Errorf("%s is routed outside the RequiredAuth group", key)
		return nil
	})
	if err != nil {
		t.Fatalf("walking the router: %v", err)
	}

	for key := range want {
		if !seen[key] {
			t.Errorf("%s is not registered at all", key)
		}
	}
}
