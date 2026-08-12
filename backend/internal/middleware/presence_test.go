package middleware

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
	"time"

	"github.com/IbnBaqqi/transcendence/internal/auth"
	"github.com/google/uuid"
)

type fakeStore struct {
	mu    sync.Mutex
	calls int
	err   error
}

func (f *fakeStore) TouchLastSeen(ctx context.Context, id uuid.UUID) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.calls++
	return f.err
}

func (f *fakeStore) count() int {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.calls
}

func serve(h http.Handler, user *auth.User) *httptest.ResponseRecorder {
	req := httptest.NewRequest(http.MethodGet, "/api/v1/listings", nil)
	if user != nil {
		req = req.WithContext(auth.WithUser(req.Context(), *user))
	}
	w := httptest.NewRecorder()
	h.ServeHTTP(w, req)
	return w
}

func okHandler() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})
}

func TestTouchLastSeenThrottles(t *testing.T) {
	store := &fakeStore{}
	h := TouchLastSeen(store, time.Minute)(okHandler())
	user := auth.User{ID: uuid.New()}

	for range 5 {
		serve(h, &user)
	}

	if got := store.count(); got != 1 {
		t.Errorf("writes = %d, want 1", got)
	}
}

func TestTouchLastSeenWritesPerUser(t *testing.T) {
	store := &fakeStore{}
	h := TouchLastSeen(store, time.Minute)(okHandler())
	first, second := auth.User{ID: uuid.New()}, auth.User{ID: uuid.New()}

	serve(h, &first)
	serve(h, &second)

	if got := store.count(); got != 2 {
		t.Errorf("writes = %d, want 2", got)
	}
}

func TestTouchLastSeenWritesAgainAfterInterval(t *testing.T) {
	store := &fakeStore{}
	h := TouchLastSeen(store, time.Millisecond)(okHandler())
	user := auth.User{ID: uuid.New()}

	serve(h, &user)
	time.Sleep(2 * time.Millisecond)
	serve(h, &user)

	if got := store.count(); got != 2 {
		t.Errorf("writes = %d, want 2", got)
	}
}

// The throttle's real workload: presence fires on every request, so many can
// hit shouldTouch at once. Run with -race — without the mutex around the map
// this fails, and without the "reserve the slot while locked" behaviour the
// count comes back higher than 1.
func TestTouchLastSeenThrottlesUnderConcurrency(t *testing.T) {
	store := &fakeStore{}
	h := TouchLastSeen(store, time.Minute)(okHandler())
	user := auth.User{ID: uuid.New()}

	var wg sync.WaitGroup
	for range 50 {
		wg.Add(1)
		go func() {
			defer wg.Done()
			serve(h, &user)
		}()
	}
	wg.Wait()

	if got := store.count(); got != 1 {
		t.Errorf("writes = %d, want 1 — 50 concurrent requests must still touch once", got)
	}
}

func TestTouchLastSeenIgnoresAnonymous(t *testing.T) {
	store := &fakeStore{}
	h := TouchLastSeen(store, time.Minute)(okHandler())

	serve(h, nil)

	if got := store.count(); got != 0 {
		t.Errorf("writes = %d, want 0", got)
	}
}

func TestTouchLastSeenServesDespiteError(t *testing.T) {
	store := &fakeStore{err: errors.New("db down")}
	h := TouchLastSeen(store, time.Minute)(okHandler())
	user := auth.User{ID: uuid.New()}

	if w := serve(h, &user); w.Code != http.StatusOK {
		t.Errorf("status = %d, want %d", w.Code, http.StatusOK)
	}
}
