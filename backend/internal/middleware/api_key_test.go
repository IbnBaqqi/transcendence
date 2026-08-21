package middleware

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/IbnBaqqi/transcendence/internal/auth"
	"github.com/IbnBaqqi/transcendence/internal/database"
)

type fakeKeyStore struct {
	owner auth.User
	good  string
	keyID uuid.UUID
}

func (f fakeKeyStore) Authenticate(_ context.Context, raw string) (uuid.UUID, auth.User, error) {
	if raw == f.good {
		return f.keyID, f.owner, nil
	}
	return uuid.Nil, auth.User{}, errors.New("not usable")
}

func TestAuthenticateAcceptsEitherCredential(t *testing.T) {
	owner := auth.User{ID: uuid.New(), Name: "aino", Role: "USER"}
	jwt := auth.NewJwtService("test-secret", time.Hour)
	token, err := jwt.IssueAccessToken(database.User{ID: uuid.New(), Username: "browser", Role: "ADMIN"})
	if err != nil {
		t.Fatal(err)
	}
	keys := fakeKeyStore{owner: owner, good: "fk_live_good", keyID: uuid.New()}

	tests := []struct {
		name       string
		set        func(*http.Request)
		wantStatus int
		wantUser   string
		wantKeyID  bool
	}{
		{"no credential", func(*http.Request) {}, http.StatusOK, "", false},
		{"a good key", func(r *http.Request) { r.Header.Set("X-API-Key", "fk_live_good") }, http.StatusOK, "aino", true},
		{"a bad key", func(r *http.Request) { r.Header.Set("X-API-Key", "fk_live_bad") }, http.StatusUnauthorized, "", false},
		{"a bearer token", func(r *http.Request) { r.Header.Set("Authorization", "Bearer "+token) }, http.StatusOK, "browser", false},
		{"both, the token wins", func(r *http.Request) {
			r.Header.Set("Authorization", "Bearer "+token)
			r.Header.Set("X-API-Key", "fk_live_good")
		}, http.StatusOK, "browser", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var gotUser auth.User
			var gotKeyID bool
			next := http.HandlerFunc(func(_ http.ResponseWriter, r *http.Request) {
				gotUser, _ = auth.UserFromContext(r.Context())
				_, gotKeyID = apiKeyID(r.Context())
			})

			req := httptest.NewRequest(http.MethodGet, "/listings", nil)
			tt.set(req)
			rec := httptest.NewRecorder()

			Authenticate(jwt, keys)(next).ServeHTTP(rec, req)

			if rec.Code != tt.wantStatus {
				t.Errorf("status = %d, want %d", rec.Code, tt.wantStatus)
			}
			if gotUser.Name != tt.wantUser {
				t.Errorf("user = %q, want %q", gotUser.Name, tt.wantUser)
			}
			// A bearer request must leave no key id behind, or the limiter
			// would throttle browser sessions.
			if gotKeyID != tt.wantKeyID {
				t.Errorf("key id in context = %v, want %v", gotKeyID, tt.wantKeyID)
			}
		})
	}
}

func TestSessionOnlyBlocksAPIKeys(t *testing.T) {
	tests := []struct {
		name       string
		viaKey     bool
		wantStatus int
		wantNext   bool
	}{
		{"authenticated by key", true, http.StatusForbidden, false},
		{"authenticated by token", false, http.StatusOK, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			called := false
			next := http.HandlerFunc(func(http.ResponseWriter, *http.Request) { called = true })

			req := httptest.NewRequest(http.MethodPost, "/me/api-keys", nil)
			if tt.viaKey {
				req = req.WithContext(context.WithValue(req.Context(), apiKeyIDKey{}, uuid.New()))
			}
			rec := httptest.NewRecorder()

			SessionOnly(next).ServeHTTP(rec, req)

			if rec.Code != tt.wantStatus {
				t.Errorf("status = %d, want %d", rec.Code, tt.wantStatus)
			}
			if called != tt.wantNext {
				t.Errorf("handler ran = %v, want %v", called, tt.wantNext)
			}
		})
	}
}
