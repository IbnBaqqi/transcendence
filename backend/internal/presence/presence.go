package presence

import "time"

const (
	Interval = time.Minute

	Window = 2 * Interval
)

// IsOnline reports whether a last-seen timestamp is fresh enough to count.
func IsOnline(lastSeen, now time.Time) bool {
	return now.Sub(lastSeen) < Window
}
