package handler

import (
	"bytes"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/google/uuid"

	"github.com/IbnBaqqi/transcendence/internal/auth"
)

func TestUploadAvatarRequiresTheAvatarField(t *testing.T) {
	body := &bytes.Buffer{}
	form := multipart.NewWriter(body)

	part, err := form.CreateFormFile("image", "cat.png")
	if err != nil {
		t.Fatalf("building the multipart body failed: %v", err)
	}
	if _, err := part.Write([]byte("\x89PNG\r\n\x1a\n")); err != nil {
		t.Fatalf("writing the part failed: %v", err)
	}
	if err := form.Close(); err != nil {
		t.Fatalf("closing the form failed: %v", err)
	}

	req := httptest.NewRequest(http.MethodPost, "/api/v1/me/avatar", body)
	req.Header.Set("Content-Type", form.FormDataContentType())
	req = req.WithContext(auth.WithUser(req.Context(), auth.User{ID: uuid.New()}))

	rec := httptest.NewRecorder()
	New(nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, 5<<20, true, nil, "").UploadAvatar(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want %d (body: %s)", rec.Code, http.StatusBadRequest, rec.Body.String())
	}
}
