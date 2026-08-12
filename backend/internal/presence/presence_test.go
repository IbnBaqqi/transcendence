package presence

import (
	"testing"
	"time"
)

func TestWindowExceedsInterval(t *testing.T) {
	if Window <= Interval {
		t.Fatalf("Window (%v) must exceed Interval (%v), or an active user appears offline "+
			"for %v out of every %v", Window, Interval, Interval-Window, Interval)
	}
}

func TestIsOnline(t *testing.T) {
	now := time.Now()

	tests := []struct {
		name     string
		lastSeen time.Time
		want     bool
	}{
		{"just now", now, true},
		{"one interval ago, the worst a live user can look", now.Add(-Interval), true},
		{"just inside the window", now.Add(-Window + time.Second), true},
		{"just outside the window", now.Add(-Window - time.Second), false},
		{"an hour ago", now.Add(-time.Hour), false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := IsOnline(tt.lastSeen, now); got != tt.want {
				t.Errorf("IsOnline() = %v, want %v", got, tt.want)
			}
		})
	}
}
