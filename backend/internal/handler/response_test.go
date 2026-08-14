package handler

import (
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/IbnBaqqi/transcendence/internal/service"
)

func TestServiceErrorPassesTypedMessagesThrough(t *testing.T) {
	tests := []struct {
		name       string
		err        error
		wantStatus int
	}{
		{"validation", &service.ValidationError{Message: "price must be greater than 0"}, http.StatusBadRequest},
		{"not found", &service.NotFoundError{Message: "listing not found"}, http.StatusNotFound},
		{"forbidden", &service.ForbiddenError{Message: "you do not own this listing"}, http.StatusForbidden},
		{"conflict", &service.ConflictError{Message: "listing has orders"}, http.StatusConflict},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rec := httptest.NewRecorder()
			req := httptest.NewRequest(http.MethodGet, "/api/v1/listings/1", nil)

			respondWithServiceError(rec, req, tt.err)

			if rec.Code != tt.wantStatus {
				t.Errorf("status = %d, want %d", rec.Code, tt.wantStatus)
			}
			if got := decodeError(t, rec); got != tt.err.Error() {
				t.Errorf("message = %q, want %q", got, tt.err.Error())
			}
		})
	}
}

func TestServiceErrorHidesUnexpectedDetail(t *testing.T) {
	leaky := errors.New("pq: numeric field overflow, a field with preccision 10, scale 2")

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/listings", nil)

	respondWithServiceError(rec, req, leaky)

	if rec.Code != http.StatusInternalServerError {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusInternalServerError)
	}

	body := rec.Body.String()
	for _, leaked := range []string{"pq", "numeric", "precision", "scale"} {
		if strings.Contains(body, leaked) {
			t.Errorf("response leaked %q: %s", leaked, body)
		}
	}
	if got := decodeError(t, rec); got != "something went wrong" {
		t.Errorf("message = %q, want the fixed string", got)
	}
}

func TestRespondWithJSONFailsSafely(t *testing.T) {
	rec := httptest.NewRecorder()

	respondWithJSON(rec, http.StatusOK, map[string]any{"bad": make(chan int)})

	if rec.Code != http.StatusInternalServerError {
		t.Errorf("status = %d, want %d", rec.Code, http.StatusInternalServerError)
	}
	if ct := rec.Header().Get("Content-Type"); ct != "application/json" {
		t.Errorf("content-type = %q, want application/json", ct)
	}
	if got := decodeError(t, rec); got != "something went wrong" {
		t.Errorf("body = %q, want a parseable error object", got)
	}
}

func decodeError(t *testing.T, rec *httptest.ResponseRecorder) string {
	t.Helper()

	var body errorResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatalf("response was not valid JSON (%v): %s", err, rec.Body.String())
	}
	return body.Error
}
