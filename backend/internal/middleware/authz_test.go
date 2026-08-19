package middleware

import (
	"context"
	"database/sql"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/google/uuid"

	"github.com/IbnBaqqi/transcendence/internal/auth"
)

type fakeRoleStore struct {
	role string
	err  error
}

func (f fakeRoleStore) GetUserRole(context.Context, uuid.UUID) (string, error) {
	return f.role, f.err
}

func TestRequireRole(t *testing.T) {
	tests := []struct {
		name          string
		authenticated bool
		store         fakeRoleStore
		wantStatus    int
		wantNext      bool
	}{
		{"an admin passes through", true, fakeRoleStore{role: RoleAdmin}, http.StatusOK, true},
		{"an ordinary user is refused", true, fakeRoleStore{role: RoleUser}, http.StatusForbidden, false},
		{"a deleted account is refused, not a 500", true, fakeRoleStore{err: sql.ErrNoRows}, http.StatusForbidden, false},
		{"a database failure is a 500", true, fakeRoleStore{err: errors.New("connection refused")}, http.StatusInternalServerError, false},
		{"nobody authenticated", false, fakeRoleStore{role: RoleAdmin}, http.StatusUnauthorized, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			called := false
			next := http.HandlerFunc(func(http.ResponseWriter, *http.Request) { called = true })

			req := httptest.NewRequest(http.MethodGet, "/admin", nil)
			if tt.authenticated {
				req = req.WithContext(auth.WithUser(req.Context(),
					auth.User{ID: uuid.New(), Role: RoleUser}))
			}
			rec := httptest.NewRecorder()

			RequireRole(tt.store, RoleAdmin)(next).ServeHTTP(rec, req)

			if rec.Code != tt.wantStatus {
				t.Errorf("status = %d, want %d", rec.Code, tt.wantStatus)
			}
			if called != tt.wantNext {
				t.Errorf("handler ran = %v, want %v", called, tt.wantNext)
			}
		})
	}
}
