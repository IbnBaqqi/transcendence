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

// The admin twin of the route above. It carries no membership check of its own -
// an admin is never a party to the order they are judging - so the role guard on
// the group is the only thing in front of every order's history. Moving this
// line out of that block would compile and pass every handler test.
//
// Compared against a known admin route rather than looked up by pointer:
// RequireRole is a closure factory, and an instance built here does not match
// the one the router installed - it does not match for /admin/reports either,
// so the pointer trick used above for RequiredAuth simply does not apply. Two
// routes in the same group do share a chain, and that is what this asserts.
func TestAdminOrderEventsRouteIsGuardedLikeTheOtherAdminRoutes(t *testing.T) {
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
	handler.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/api/v1/admin/orders/1/events", nil))
	if rec.Code != http.StatusUnauthorized {
		t.Errorf("status = %d, want 401", rec.Code)
	}

	routes, ok := handler.(chi.Routes)
	if !ok {
		t.Fatalf("NewRouter no longer returns a chi router")
	}

	chains := map[string][]uintptr{}
	if err := chi.Walk(routes, func(method, route string, _ http.Handler, mws ...func(http.Handler) http.Handler) error {
		key := method + " " + route
		for _, m := range mws {
			chains[key] = append(chains[key], reflect.ValueOf(m).Pointer())
		}
		return nil
	}); err != nil {
		t.Fatalf("walking the router: %v", err)
	}

	const (
		mine  = "GET /api/v1/admin/orders/{id}/events"
		known = "GET /api/v1/admin/reports"
		party = "GET /api/v1/orders/{id}/events"
	)

	if chains[mine] == nil {
		t.Fatalf("%s is not registered", mine)
	}
	if !reflect.DeepEqual(chains[mine], chains[known]) {
		t.Errorf("%s does not carry the same middleware as %s - is it outside the RequireRole(ADMIN) group?", mine, known)
	}
	// The control: the parties' route is authenticated but not role-guarded, so
	// a chain equal to that one would mean the admin guard is missing.
	if reflect.DeepEqual(chains[mine], chains[party]) {
		t.Errorf("%s carries the same middleware as the parties-only route, so it is not admin-guarded", mine)
	}
}
