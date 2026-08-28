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
	"github.com/IbnBaqqi/transcendence/internal/service"
	"github.com/IbnBaqqi/transcendence/internal/storage"
	"github.com/IbnBaqqi/transcendence/internal/testdb"
)

// The public-profile path is the one that needs a handler test: the SQL
// predicate covers the lists, but this surface is guarded in Go, and it is the
// only presence surface reachable without a token.
func TestThePublicProfileHidesPresenceFromBlockedViewersAndStrangers(t *testing.T) {
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
	alice, bob, carol := mk("alice"), mk("bob"), mk("carol")

	for _, id := range []uuid.UUID{alice, bob, carol} {
		if _, err := db.ExecContext(ctx,
			`UPDATE users SET show_online_status = true, last_seen_at = now() WHERE id = $1`, id,
		); err != nil {
			t.Fatalf("enabling presence: %v", err)
		}
	}

	files, err := storage.NewLocal(t.TempDir())
	if err != nil {
		t.Fatalf("creating a temporary upload dir: %v", err)
	}
	t.Cleanup(func() { _ = files.Close() })

	blocks := service.NewBlockService(db.Queries)
	if err := blocks.Block(ctx, alice, bob); err != nil {
		t.Fatalf("blocking: %v", err)
	}

	h := New(db, nil, nil, nil, nil, nil, nil, service.NewProfileService(db, files),
		nil, blocks, nil, nil, nil, nil, 0, true, nil, "")

	profileOf := func(t *testing.T, subject uuid.UUID, viewer *uuid.UUID) string {
		t.Helper()

		req := httptest.NewRequest(http.MethodGet, "/users/"+subject.String(), nil)

		routeCtx := chi.NewRouteContext()
		routeCtx.URLParams.Add("id", subject.String())
		reqCtx := context.WithValue(req.Context(), chi.RouteCtxKey, routeCtx)
		if viewer != nil {
			reqCtx = auth.WithUser(reqCtx, auth.User{ID: *viewer, Role: auth.RoleUser})
		}

		rec := httptest.NewRecorder()
		h.GetPublicProfile(rec, req.WithContext(reqCtx))
		if rec.Code != http.StatusOK {
			t.Fatalf("status = %d: %s", rec.Code, rec.Body.String())
		}
		return rec.Body.String()
	}

	online := func(body string) bool {
		return strings.Contains(body, `"is_online":true`)
	}

	t.Run("a stranger sees presence", func(t *testing.T) {
		if body := profileOf(t, alice, &carol); !online(body) {
			t.Errorf("carol should see alice online:\n%s", body)
		}
	})

	t.Run("the blocked user does not", func(t *testing.T) {
		if body := profileOf(t, alice, &bob); online(body) {
			t.Errorf("bob was blocked by alice and still sees her online:\n%s", body)
		}
	})

	t.Run("the blocker does not either", func(t *testing.T) {
		if body := profileOf(t, bob, &alice); online(body) {
			t.Errorf("alice blocked bob and still sees him online:\n%s", body)
		}
	})

	// Without this the whole rule is a speed bump: log out and read it anyway.
	// The field is omitted rather than sent as offline, so a client can tell
	// "you are not signed in" apart from "this person is offline" - otherwise
	// every public profile renders a permanent, false "Offline".
	t.Run("an anonymous caller gets no presence field at all", func(t *testing.T) {
		body := profileOf(t, alice, nil)

		if strings.Contains(body, "presence") {
			t.Errorf("presence was served without a token:\n%s", body)
		}
		if !strings.Contains(body, `"username"`) {
			t.Errorf("the rest of the profile should still be public:\n%s", body)
		}
	})
}
