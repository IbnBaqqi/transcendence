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
	"github.com/IbnBaqqi/transcendence/internal/dtos"
	"github.com/IbnBaqqi/transcendence/internal/service"
	"github.com/IbnBaqqi/transcendence/internal/storage"
	"github.com/IbnBaqqi/transcendence/internal/testdb"
)

// profileRouter builds the two routes the way router.go does, so the test
// exercises the real wiring rather than calling a handler directly.
func profileRouter(t *testing.T) (http.Handler, uuid.UUID) {
	t.Helper()

	db := testdb.New(t)
	ctx := context.Background()

	user, err := db.CreateUser(ctx, database.CreateUserParams{
		ID:       database.NewID(),
		Username: "aino",
		Email:    "aino@example.test",
		Password: sql.NullString{String: "irrelevant", Valid: true},
	})
	if err != nil {
		t.Fatalf("creating a user: %v", err)
	}
	if err := db.EnsureProfile(ctx, user.ID); err != nil {
		t.Fatalf("creating the profile: %v", err)
	}

	files, err := storage.NewLocal(t.TempDir())
	if err != nil {
		t.Fatalf("creating a temporary upload dir: %v", err)
	}
	t.Cleanup(func() { _ = files.Close() })

	profiles := service.NewProfileService(db, files)

	if _, err := profiles.Update(ctx, user.ID, dtos.UpdateProfileInput{
		Firstname:   dtos.SetString("Aino"),
		PhoneNumber: dtos.SetString("+358401234567"),
		DateOfBirth: dtos.SetString("2001-05-14"),
		Location:    dtos.SetString("Espoo"),
	}); err != nil {
		t.Fatalf("filling the profile: %v", err)
	}

	h := New(Deps{DB: db, Profile: profiles, CookieSecure: true})

	r := chi.NewRouter()
	r.Get("/users/{id}", h.GetPublicProfile)
	r.Group(func(r chi.Router) {
		r.Use(func(next http.Handler) http.Handler {
			return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
				ctx := auth.WithUser(req.Context(), auth.User{ID: user.ID})
				next.ServeHTTP(w, req.WithContext(ctx))
			})
		})
		r.Get("/me/profile", h.GetOwnProfile)
	})

	return r, user.ID
}

func get(t *testing.T, r http.Handler, path string) (int, string) {
	t.Helper()

	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, path, nil))

	return rec.Code, rec.Body.String()
}

// The route has to be wired to the PUBLIC mapper. The DTO test proves the
// mapper is clean; only this proves the endpoint uses it.
func TestGetPublicProfileHidesPrivateFields(t *testing.T) {
	r, userID := profileRouter(t)

	code, body := get(t, r, "/users/"+userID.String())

	if code != http.StatusOK {
		t.Fatalf("status = %d, want 200: %s", code, body)
	}
	for _, private := range []string{"email", "phone_number", "date_of_birth", "aino@example.test", "+358401234567"} {
		if strings.Contains(body, private) {
			t.Errorf("public profile contains %q:\n%s", private, body)
		}
	}
	if !strings.Contains(body, `"username":"aino"`) {
		t.Errorf("public profile is missing the username:\n%s", body)
	}
}

func TestGetOwnProfileShowsPrivateFields(t *testing.T) {
	r, _ := profileRouter(t)

	code, body := get(t, r, "/me/profile")

	if code != http.StatusOK {
		t.Fatalf("status = %d, want 200: %s", code, body)
	}
	for _, expected := range []string{`"email":"aino@example.test"`, `"phone_number":"+358401234567"`, `"date_of_birth":"2001-05-14"`} {
		if !strings.Contains(body, expected) {
			t.Errorf("own profile is missing %s:\n%s", expected, body)
		}
	}
}

func TestGetPublicProfileBadInput(t *testing.T) {
	r, _ := profileRouter(t)

	if code, _ := get(t, r, "/users/not-a-uuid"); code != http.StatusBadRequest {
		t.Errorf("malformed uuid = %d, want 400", code)
	}
	if code, _ := get(t, r, "/users/"+uuid.New().String()); code != http.StatusNotFound {
		t.Errorf("unknown user = %d, want 404", code)
	}
}
