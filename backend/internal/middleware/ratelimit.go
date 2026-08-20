package middleware

import (
	"context"
	"log/slog"
	"math"
	"net/http"
	"strconv"
	"sync"
	"time"
)

// maxTrackedKeys bounds the map, like presenceTracker's maxTrackedUsers.
const maxTrackedKeys = 10000

type bucket struct {
	tokens float64
	last   time.Time
}

type keyLimiter struct {
	capacity float64
	refill   float64 // tokens per second

	mu      sync.Mutex
	buckets map[int32]*bucket
}

// allow spends a token if one is available, and reports what to tell the client
// either way. The bucket refills lazily - a key that went quiet for an hour
// catches up in one subtraction, so there is no ticker to run.
func (l *keyLimiter) allow(id int32, now time.Time) (ok bool, remaining int, retryAfter time.Duration) {
	l.mu.Lock()
	defer l.mu.Unlock()

	b, seen := l.buckets[id]
	if !seen {
		l.evictFull(now)
		b = &bucket{tokens: l.capacity, last: now}
		l.buckets[id] = b
	} else {
		b.tokens = math.Min(l.capacity, b.tokens+now.Sub(b.last).Seconds()*l.refill)
		b.last = now
	}

	if b.tokens < 1 {
		// Round up and add a second: telling a client to retry in 0 seconds
		// invites an immediate retry that also fails.
		wait := time.Duration((1-b.tokens)/l.refill*float64(time.Second)) + time.Second
		return false, 0, wait.Round(time.Second)
	}

	b.tokens--
	// The floor, never more than we will honour.
	return true, int(b.tokens), 0
}

// evictFull drops buckets that have refilled completely: losing one costs the
// client nothing, since a bucket it has not used starts full anyway.
func (l *keyLimiter) evictFull(now time.Time) {
	if len(l.buckets) < maxTrackedKeys {
		return
	}
	for id, b := range l.buckets {
		if b.tokens+now.Sub(b.last).Seconds()*l.refill >= l.capacity {
			delete(l.buckets, id)
		}
	}
}

// RateLimitByKey throttles requests that authenticated with an API key. Browser
// sessions are not limited: a human's pace is not the problem this solves.
//
// The state is in memory, so it is lost on restart and not shared between
// instances - two servers give one client double its limit.
func RateLimitByKey(perMinute int) func(http.Handler) http.Handler {
	l := &keyLimiter{
		capacity: float64(perMinute),
		refill:   float64(perMinute) / 60,
		buckets:  make(map[int32]*bucket),
	}

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			id, ok := apiKeyID(r.Context())
			if !ok {
				next.ServeHTTP(w, r)
				return
			}

			allowed, remaining, retryAfter := l.allow(id, time.Now())

			w.Header().Set("X-RateLimit-Limit", strconv.Itoa(perMinute))
			w.Header().Set("X-RateLimit-Remaining", strconv.Itoa(remaining))

			if !allowed {
				w.Header().Set("Retry-After", strconv.Itoa(int(retryAfter.Seconds())))
				writeAuthzError(w, http.StatusTooManyRequests, "rate limit exceeded")
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}

type keyToucher interface {
	TouchKey(ctx context.Context, id int32) error
}

type keyUsageTracker struct {
	store    keyToucher
	interval time.Duration

	mu   sync.Mutex
	seen map[int32]time.Time
}

func (k *keyUsageTracker) shouldTouch(id int32, now time.Time) bool {
	k.mu.Lock()
	defer k.mu.Unlock()

	if last, ok := k.seen[id]; ok && now.Sub(last) < k.interval {
		return false
	}

	if len(k.seen) >= maxTrackedKeys {
		for key, at := range k.seen {
			if now.Sub(at) >= k.interval {
				delete(k.seen, key)
			}
		}
	}

	k.seen[id] = now
	return true
}

// TouchAPIKey records that a key was used, at most once per interval. Writing
// on every request would mean 60 updates a minute per key, at the limit, to a
// column nobody reads more than daily.
func TouchAPIKey(store keyToucher, interval time.Duration) func(http.Handler) http.Handler {
	tracker := &keyUsageTracker{
		store:    store,
		interval: interval,
		seen:     make(map[int32]time.Time),
	}

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			id, ok := apiKeyID(r.Context())
			if !ok {
				next.ServeHTTP(w, r)
				return
			}

			if tracker.shouldTouch(id, time.Now()) {
				if err := tracker.store.TouchKey(r.Context(), id); err != nil {
					slog.Error("could not record api key usage", "key_id", id, "error", err)
				}
			}

			next.ServeHTTP(w, r)
		})
	}
}
