package app

import (
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/IbnBaqqi/transcendence/internal/database"
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
