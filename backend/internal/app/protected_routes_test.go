package app

import (
	"context"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"reflect"
	"testing"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"

	"github.com/IbnBaqqi/transcendence/internal/auth"
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

// The inverse of the table above, and it exists because of how this endpoint
// fails. /auth/providers feeds the signed-out login screen, so a visitor with
// no session is its whole audience, and the client deliberately renders nothing
// when it cannot read the list. A 401 here therefore draws zero sign-in buttons
// with no error and nothing in the console - which is exactly what "no provider
// is configured" looks like, the legitimate state that endpoint exists to
// report. Moving it inside an auth group would look like the feature working.
//
// TestSpecMatchesRouter compares paths and methods, not middleware, and the
// handler tests call the handler directly, so nothing else can catch it.
func TestPublicRoutesAnswerAnonymousCallers(t *testing.T) {
	files, err := storage.NewLocal(t.TempDir())
	if err != nil {
		t.Fatalf("temporary upload dir: %v", err)
	}
	t.Cleanup(func() { _ = files.Close() })

	router := NewRouter(
		slog.New(slog.NewTextHandler(io.Discard, nil)),
		&api{Files: files, DB: &database.DB{}},
	)

	tests := []struct {
		method string
		path   string
		want   int
	}{
		// No credentials are configured in this router, so the honest answer to
		// an anonymous caller is an empty list rather than a refusal.
		{http.MethodGet, "/api/v1/auth/providers", http.StatusOK},
	}

	for _, tt := range tests {
		t.Run(tt.method+" "+tt.path, func(t *testing.T) {
			rec := httptest.NewRecorder()
			router.ServeHTTP(rec, httptest.NewRequest(tt.method, tt.path, nil))

			if rec.Code == http.StatusUnauthorized {
				t.Fatalf("status = 401 - has this route moved inside an auth group? it is read before a session exists")
			}
			if rec.Code != tt.want {
				t.Errorf("status = %d, want %d", rec.Code, tt.want)
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

// Identifying the limiter by behaviour rather than by pointer, because
// RateLimitByUser returns a fresh closure per call and there is nothing stable
// to compare against. A middleware on this route that allows the first
// exportsPerHour requests from one account and then refuses is the limiter.
func exportIsRateLimitedPerUser(mws []func(http.Handler) http.Handler) bool {
	ctx := auth.WithUser(context.Background(), auth.User{ID: uuid.New()})

	for _, m := range mws {
		if limits(m, ctx) {
			return true
		}
	}
	return false
}

func limits(m func(http.Handler) http.Handler, ctx context.Context) (found bool) {
	// Most of the inherited chain needs wiring this test does not build.
	defer func() {
		if recover() != nil {
			found = false
		}
	}()

	h := m(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	}))

	for i := 0; i < exportsPerHour; i++ {
		rec := httptest.NewRecorder()
		h.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/api/v1/me/export", nil).WithContext(ctx))
		if rec.Code != http.StatusNoContent {
			return false
		}
	}

	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/api/v1/me/export", nil).WithContext(ctx))
	return rec.Code == http.StatusTooManyRequests
}

// The export runs fifteen unpaginated queries over an entire account history
// and queues an email. Nothing else in the chain bounds it: the group limiter
// keys on the API key, and this route is session-only.
func TestTheExportRouteIsRateLimited(t *testing.T) {
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

	seen := false
	err = chi.Walk(routes, func(method, route string, _ http.Handler, mws ...func(http.Handler) http.Handler) error {
		if method != http.MethodGet || route != "/api/v1/me/export" {
			return nil
		}
		seen = true

		if !exportIsRateLimitedPerUser(mws) {
			t.Error("GET /api/v1/me/export carries no per-user rate limit")
		}
		return nil
	})
	if err != nil {
		t.Fatalf("walking the router: %v", err)
	}
	if !seen {
		t.Fatal("GET /api/v1/me/export is not registered at all")
	}
}
