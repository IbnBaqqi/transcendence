package handler

import (
	"bytes"
	"context"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"net/textproto"
	"testing"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"

	"github.com/IbnBaqqi/transcendence/internal/auth"
)

func TestDetectImageExt(t *testing.T) {
	tests := []struct {
		name    string
		head    []byte
		wantExt string
		wantOK  bool
	}{
		{"png", append([]byte("\x89PNG\r\n\x1a\n"), make([]byte, 16)...), ".png", true},
		{"jpeg", append([]byte("\xff\xd8\xff\xe0"), make([]byte, 16)...), ".jpg", true},
		{"webp", []byte("RIFF\x00\x00\x00\x00WEBPVP8 "), ".webp", true},
		{"git is not allowed", []byte("GIF87a\x00\x00"), "", false},
		{"pdf is not allowed", []byte("%PDF-1.7\n"), "", false},
		{"text is not an imgae", []byte("hello, this is not an image"), "", false},
		{"empty upload", []byte{}, "", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ext, ok := detectImageExt(tt.head)
			if ok != tt.wantOK {
				t.Fatalf("ok = %v, want %v", ok, tt.wantOK)
			}
			if ext != tt.wantExt {
				t.Errorf("ext = %q, want %q", ext, tt.wantExt)
			}
		})
	}
}

func TestUploadRejectsLyingContentType(t *testing.T) {
	body := &bytes.Buffer{}
	form := multipart.NewWriter(body)

	partHeader := make(textproto.MIMEHeader)
	partHeader.Set("Content-Disposition", `form-data; name="image"; filename="cat.png"`)
	partHeader.Set("Content-Type", "image/png")

	part, err := form.CreatePart(partHeader)
	if err != nil {
		t.Fatalf("building the multipart body failed: %v", err)
	}
	if _, err := part.Write([]byte("<script>alert(1)</script>")); err != nil {
		t.Fatalf("writing the part failed: %v", err)
	}
	if err := form.Close(); err != nil {
		t.Fatalf("closing the form failed: %v", err)
	}

	req := httptest.NewRequest(http.MethodPost, "/api/v1/listings/1/images", body)
	req.Header.Set("Content-Type", form.FormDataContentType())
	req = req.WithContext(auth.WithUser(req.Context(), auth.User{ID: uuid.New()}))

	routeCtx := chi.NewRouteContext()
	routeCtx.URLParams.Add("id", "1")
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, routeCtx))

	rec := httptest.NewRecorder()
	New(nil, nil, nil, nil, nil, nil, nil, nil, nil, 5<<20).UploadListingImage(rec, req)

	if rec.Code != http.StatusUnsupportedMediaType {
		t.Errorf("status = %d, want %d (body: %s)", rec.Code, http.StatusUnsupportedMediaType, rec.Body.String())
	}
}
