package middleware_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"

	"github.com/IbnBaqqi/transcendence/internal/auth"
	"github.com/IbnBaqqi/transcendence/internal/database"
	mw "github.com/IbnBaqqi/transcendence/internal/middleware"
	"github.com/IbnBaqqi/transcendence/internal/testdb"
)

func adminRouter(t *testing.T, db *database.DB, startingRole string) (func() int, uuid.UUID) {
	t.Helper()

	id := uuid.New()
	if _, err := db.Exec(
		`INSERT INTO users (id, email, username, password, role)
		 VALUES ($1, $2, $3, 'x', $4)`,
		id, id.String()+"@example.test", "user_"+id.String()[:8], startingRole,
	); err != nil {
		t.Fatalf("creating the user: %v", err)
	}

	user, err := db.Queries.GetUser(context.Background(), id)
	if err != nil {
		t.Fatalf("reading the user back: %v", err)
	}

	jwt := auth.NewJwtService("test-secret", time.Hour)
	token, err := jwt.IssueAccessToken(user)
	if err != nil {
		t.Fatalf("issuing a token: %v", err)
	}

	r := chi.NewRouter()
	r.Use(mw.Authenticate(jwt))
	r.Group(func(r chi.Router) {
		r.Use(mw.RequiredAuth)
		r.Use(mw.RequireRole(db.Queries, auth.RoleAdmin))
		r.Get("/admin", func(w http.ResponseWriter, _ *http.Request) {
			w.WriteHeader(http.StatusNoContent)
		})
	})

	call := func() int {
		req := httptest.NewRequest(http.MethodGet, "/admin", nil)
		req.Header.Set("Authorization", "Bearer "+token)
		rec := httptest.NewRecorder()
		r.ServeHTTP(rec, req)
		return rec.Code
	}

	return call, id
}

func TestDemotionTakesEffectOnTheNextRequest(t *testing.T) {
	db := testdb.New(t)

	call, id := adminRouter(t, db, auth.RoleAdmin)

	if code := call(); code != http.StatusNoContent {
		t.Fatalf("as an admin: status = %d, want %d", code, http.StatusNoContent)
	}

	if _, err := db.Exec(`UPDATE users SET role = $2 WHERE id = $1`, id, auth.RoleUser); err != nil {
		t.Fatal(err)
	}

	if code := call(); code != http.StatusForbidden {
		t.Errorf("after demotion: status = %d, want %d", code, http.StatusForbidden)
	}
}

func TestPromotionNeedsNoNewToken(t *testing.T) {
	db := testdb.New(t)

	call, id := adminRouter(t, db, auth.RoleUser)

	if code := call(); code != http.StatusForbidden {
		t.Fatalf("as an ordinary user: status = %d, want %d", code, http.StatusForbidden)
	}

	if _, err := db.Exec(`UPDATE users SET role = $2 WHERE id = $1`, id, auth.RoleAdmin); err != nil {
		t.Fatal(err)
	}

	if code := call(); code != http.StatusNoContent {
		t.Errorf("after promotion: status = %d, want %d", code, http.StatusNoContent)
	}
}
