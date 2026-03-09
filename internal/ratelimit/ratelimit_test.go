package ratelimit

import (
	"testing"
	"time"
)

func TestLimiter_Allow(t *testing.T) {
	tests := []struct {
		name      string
		rate      float64
		burst     int
		requests  int
		wantAllow int // how many should be allowed
	}{
		{"burst allows initial", 1, 5, 5, 5},
		{"exceeds burst", 1, 5, 7, 5},
		{"single token", 1, 1, 3, 1},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			l := New(tt.rate, tt.burst)
			allowed := 0
			for range tt.requests {
				if l.Allow("user1") {
					allowed++
				}
			}
			if allowed != tt.wantAllow {
				t.Errorf("got %d allowed, want %d", allowed, tt.wantAllow)
			}
		})
	}
}

func TestLimiter_DifferentKeys(t *testing.T) {
	l := New(1, 2)

	// Each key gets its own bucket
	if !l.Allow("a") {
		t.Error("first request for 'a' should be allowed")
	}
	if !l.Allow("b") {
		t.Error("first request for 'b' should be allowed")
	}
}

func TestLimiter_Cleanup(t *testing.T) {
	l := New(1, 5)
	l.cleanup = 1 * time.Millisecond

	l.Allow("stale")
	time.Sleep(5 * time.Millisecond)
	l.Cleanup()

	l.mu.Lock()
	_, exists := l.buckets["stale"]
	l.mu.Unlock()

	if exists {
		t.Error("stale entry should have been cleaned up")
	}
}
