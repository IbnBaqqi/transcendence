package middleware

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

const testCap = 1 << 10

// Reads the body, so the reader's limit is what answers rather than the
// handler never touching it.
func drain(t *testing.T) (http.Handler, *int64) {
	t.Helper()

	var read int64
	return MaxBody(testCap)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		n, err := io.Copy(io.Discard, r.Body)
		read = n
		if err != nil {
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		w.WriteHeader(http.StatusOK)
	})), &read
}

func TestADeclaredOversizeBodyIsRefusedBeforeItIsRead(t *testing.T) {
	h, read := drain(t)

	req := httptest.NewRequest(http.MethodPost, "/", strings.NewReader(strings.Repeat("x", testCap+1)))
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)

	if rec.Code != http.StatusRequestEntityTooLarge {
		t.Errorf("status = %d, want 413", rec.Code)
	}
	if *read != 0 {
		t.Errorf("read %d bytes, want 0 - the point is to refuse before allocating", *read)
	}

	var body map[string]string
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatalf("the error body is not JSON: %v", err)
	}
	if body["error"] == "" {
		t.Errorf("body = %v, want an error message clients can show", body)
	}
}

// A chunked body has no declared length, so the header check cannot see it and
// the reader is what stops it. The handler reports the failure instead, which
// is why this bounds the read without producing a 413.
func TestAnUndeclaredOversizeBodyIsStoppedByTheReader(t *testing.T) {
	h, read := drain(t)

	req := httptest.NewRequest(http.MethodPost, "/", strings.NewReader(strings.Repeat("x", testCap*4)))
	req.ContentLength = -1
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)

	if *read > testCap {
		t.Errorf("read %d bytes, want no more than %d", *read, testCap)
	}
	if rec.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want the handler's 400 once the read fails", rec.Code)
	}
}

func TestABodyAtTheLimitIsUntouched(t *testing.T) {
	h, read := drain(t)

	req := httptest.NewRequest(http.MethodPost, "/", strings.NewReader(strings.Repeat("x", testCap)))
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("status = %d, want 200 - a body exactly at the limit is legal", rec.Code)
	}
	if *read != testCap {
		t.Errorf("read %d bytes, want all %d", *read, testCap)
	}
}
