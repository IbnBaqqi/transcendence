package middleware

import (
	"context"
	"log/slog"
	"net/http"
	"sync"
	"time"

	"github.com/IbnBaqqi/transcendence/internal/auth"
	"github.com/google/uuid"
)

type presenceStore interface {
	TouchLastSeen(ctx context.Context, id uuid.UUID) error
}

const maxTrackedUsers = 10000

type presenceTracker struct {
	store    presenceStore
	interval time.Duration

	mu   sync.Mutex
	seen map[uuid.UUID]time.Time
}

func (p *presenceTracker) shouldTouch(id uuid.UUID, now time.Time) bool {
	p.mu.Lock()
	defer p.mu.Unlock()

	if last, ok := p.seen[id]; ok && now.Sub(last) < p.interval {
		return false
	}

	if len(p.seen) >= maxTrackedUsers {
		for key, at := range p.seen {
			if now.Sub(at) >= p.interval {
				delete(p.seen, key)
			}
		}
	}

	p.seen[id] = now
	return true
}

func TouchLastSeen(store presenceStore, interval time.Duration) func(http.Handler) http.Handler {
	tracker := &presenceTracker{
		store:    store,
		interval: interval,
		seen:     make(map[uuid.UUID]time.Time),
	}

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			user, ok := auth.UserFromContext(r.Context())
			if !ok {
				next.ServeHTTP(w, r)
				return
			}

			if tracker.shouldTouch(user.ID, time.Now()) {
				if err := tracker.store.TouchLastSeen(r.Context(), user.ID); err != nil {
					slog.Error("could not update last seen", "user_id", user.ID, "error", err)
				}
			}

			next.ServeHTTP(w, r)
		})
	}
}
