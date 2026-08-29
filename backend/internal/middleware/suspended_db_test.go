package middleware_test

import (
	"context"
	"database/sql"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/google/uuid"

	"github.com/IbnBaqqi/transcendence/internal/auth"
	"github.com/IbnBaqqi/transcendence/internal/database"
	mw "github.com/IbnBaqqi/transcendence/internal/middleware"
	"github.com/IbnBaqqi/transcendence/internal/testdb"
)

// The one place the two inactive states are meant to differ: a deleted account
// looks like it never existed, a suspended one says so and says why, because it
// is meant to be appealed.
func TestSuspendedAndDeletedAnswerDifferently(t *testing.T) {
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
	active, suspended, deleted := mk("active"), mk("suspended"), mk("deleted")
	gone := mk("gone")

	if _, err := db.SuspendUser(ctx, database.SuspendUserParams{
		ID: suspended, Reason: sql.NullString{String: "spamming listings", Valid: true},
	}); err != nil {
		t.Fatalf("suspending: %v", err)
	}
	if _, err := db.AnonymiseUser(ctx, deleted); err != nil {
		t.Fatalf("deleting: %v", err)
	}

	// Suspended first, then deleted. A deletion does not clear suspended_at,
	// so this row is in both states at once and the precedence has to hold.
	if _, err := db.SuspendUser(ctx, database.SuspendUserParams{
		ID: gone, Reason: sql.NullString{String: "spamming listings", Valid: true},
	}); err != nil {
		t.Fatalf("suspending before deleting: %v", err)
	}
	if _, err := db.AnonymiseUser(ctx, gone); err != nil {
		t.Fatalf("deleting a suspended account: %v", err)
	}

	guarded := mw.RequireActiveUser(db.Queries)(http.HandlerFunc(
		func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(http.StatusNoContent) }))

	call := func(t *testing.T, id uuid.UUID) *httptest.ResponseRecorder {
		t.Helper()
		req := httptest.NewRequest(http.MethodGet, "/anything", nil)
		req = req.WithContext(auth.WithUser(req.Context(), auth.User{ID: id, Role: auth.RoleUser}))
		rec := httptest.NewRecorder()
		guarded.ServeHTTP(rec, req)
		return rec
	}

	t.Run("an active account passes", func(t *testing.T) {
		if code := call(t, active).Code; code != http.StatusNoContent {
			t.Errorf("status = %d, want 204", code)
		}
	})

	t.Run("a deleted account gets a silent 401", func(t *testing.T) {
		rec := call(t, deleted)
		if rec.Code != http.StatusUnauthorized {
			t.Errorf("status = %d, want 401", rec.Code)
		}
		if strings.Contains(strings.ToLower(rec.Body.String()), "suspend") {
			t.Errorf("a deleted account was described as suspended:\n%s", rec.Body.String())
		}
	})

	t.Run("a suspended account gets a 403 naming the reason", func(t *testing.T) {
		rec := call(t, suspended)
		if rec.Code != http.StatusForbidden {
			t.Errorf("status = %d, want 403 - a 401 sends them to the password reset instead", rec.Code)
		}
		if !strings.Contains(rec.Body.String(), "spamming listings") {
			t.Errorf("the reason is missing, so there is nothing to appeal:\n%s", rec.Body.String())
		}
	})

	t.Run("a deleted account that was suspended first is still silent", func(t *testing.T) {
		rec := call(t, gone)
		if rec.Code != http.StatusUnauthorized {
			t.Errorf("status = %d, want 401 - deletion outranks the suspension it kept", rec.Code)
		}
		if strings.Contains(rec.Body.String(), "spamming listings") {
			t.Errorf("a deleted account published the reason it was suspended for:\n%s", rec.Body.String())
		}
	})
}
