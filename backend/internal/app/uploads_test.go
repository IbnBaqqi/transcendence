package app

import (
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"github.com/go-chi/chi/v5"
)

func newUploadsRouter(t *testing.T) (http.Handler, string) {
	t.Helper()

	dir := t.TempDir()
	name := "3f1a7c2e-8b4d-4e91-9a5f-2c6d8e0b1a34.jpg"
	if err := os.WriteFile(filepath.Join(dir, name), []byte("photo bytes"), 0o644); err != nil {
		t.Fatalf("setup failed: %v", err)
	}

	r := chi.NewRouter()
	r.Handle("/uploads/*", uploadFileServer(dir))

	return r, name
}

func get(t *testing.T, h http.Handler, path string) *httptest.ResponseRecorder {
	t.Helper()

	req := httptest.NewRequest(http.MethodGet, path, nil)
	w := httptest.NewRecorder()
	h.ServeHTTP(w, req)

	return w
}

func TestUploadFileServerServesStoredFile(t *testing.T) {
	router, name := newUploadsRouter(t)

	w := get(t, router, "/uploads/"+name)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", w.Code, http.StatusOK)
	}
	if body := w.Body.String(); body != "photo bytes" {
		t.Errorf("body = %q, want %q", body, "photo bytes")
	}
	if got := w.Header().Get("X-Content-Type-Options"); got != "nosniff" {
		t.Errorf("X-Content-Type-Options = %q, want %q", got, "nosniff")
	}
}

func TestUploadFileServerHidesDirectory(t *testing.T) {
	router, _ := newUploadsRouter(t)

	paths := []string{
		"/uploads/",
		"/uploads",
		"/uploads/sub/x.jpg",
		"/uploads/../go.mod",
	}

	for _, path := range paths {
		t.Run(path, func(t *testing.T) {
			w := get(t, router, path)

			if w.Code != http.StatusNotFound {
				t.Errorf("status = %d, want %d (body: %q)", w.Code, http.StatusNotFound, w.Body.String())
			}
		})
	}
}
