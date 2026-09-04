package app

import (
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/IbnBaqqi/transcendence/internal/config"
	"github.com/IbnBaqqi/transcendence/internal/database"
	"github.com/IbnBaqqi/transcendence/internal/storage"
)

const testUploadCap = 5 << 20

func routerWithUploadCap(t *testing.T) http.Handler {
	t.Helper()

	files, err := storage.NewLocal(t.TempDir())
	if err != nil {
		t.Fatalf("temporary upload dir: %v", err)
	}
	t.Cleanup(func() { _ = files.Close() })

	return NewRouter(
		slog.New(slog.NewTextHandler(io.Discard, nil)),
		&api{
			Files:  files,
			DB:     &database.DB{},
			Upload: config.UploadConfig{MaxBytes: testUploadCap},
		},
	)
}

func post(t *testing.T, h http.Handler, path string, size int) *httptest.ResponseRecorder {
	t.Helper()

	req := httptest.NewRequest(http.MethodPost, path, strings.NewReader(strings.Repeat("x", size)))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	return rec
}

// The whole point of the group cap: every JSON route is bounded without any of
// them saying so. A handler test cannot see this, because it never runs the
// middleware.
func TestAnOversizeBodyIsRefusedAtTheGroup(t *testing.T) {
	h := routerWithUploadCap(t)

	rec := post(t, h, "/api/v1/listings", testUploadCap+uploadHeadroom+1)

	if rec.Code != http.StatusRequestEntityTooLarge {
		t.Errorf("status = %d, want 413 - is MaxBody still on the /api/v1 group?", rec.Code)
	}
}

// The regression this exists to prevent. MaxBytesReader nests as a MINIMUM, so
// a group cap at or below MAX_UPLOAD_BYTES silently becomes the upload limit -
// and every upload test in internal/handler calls the handler directly, so
// none of them would notice. Reaching the auth gate is the proof the body was
// allowed through: 401 means the cap did not stop it.
func TestAnUploadSizedBodyStillReachesTheRoute(t *testing.T) {
	h := routerWithUploadCap(t)

	rec := post(t, h, "/api/v1/listings/11111111-1111-1111-1111-111111111111/images", testUploadCap)

	if rec.Code == http.StatusRequestEntityTooLarge {
		t.Fatalf("a legal upload was refused by the group cap - it must sit ABOVE MAX_UPLOAD_BYTES (%d)", testUploadCap)
	}
	if rec.Code != http.StatusUnauthorized {
		t.Errorf("status = %d, want 401 - the body passed the cap and met the auth gate", rec.Code)
	}
}

// Derived from config, never a literal: raising MAX_UPLOAD_BYTES past a
// hardcoded cap would break every upload, and the failure would be invisible
// to the handler tests.
func TestTheGroupCapFollowsTheUploadLimit(t *testing.T) {
	files, err := storage.NewLocal(t.TempDir())
	if err != nil {
		t.Fatalf("temporary upload dir: %v", err)
	}
	t.Cleanup(func() { _ = files.Close() })

	const raised = testUploadCap * 3

	h := NewRouter(
		slog.New(slog.NewTextHandler(io.Discard, nil)),
		&api{Files: files, DB: &database.DB{}, Upload: config.UploadConfig{MaxBytes: raised}},
	)

	rec := post(t, h, "/api/v1/listings/11111111-1111-1111-1111-111111111111/images", raised)

	if rec.Code == http.StatusRequestEntityTooLarge {
		t.Errorf("the cap did not follow MAX_UPLOAD_BYTES up to %d - it is hardcoded somewhere", raised)
	}
}
