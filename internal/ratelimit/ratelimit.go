// Package ratelimit provides per-key token bucket rate limiting.
package ratelimit

import (
	"sync"
	"time"
)

// Limiter provides per-key (user ID) rate limiting using a token bucket.
type Limiter struct {
	mu      sync.Mutex
	buckets map[string]*bucket
	rate    float64       // tokens per second
	burst   int           // max tokens
	cleanup time.Duration // remove stale entries after this
}

type bucket struct {
	tokens    float64
	lastCheck time.Time
}

// New creates a new per-key rate limiter.
// rate is tokens per second, burst is the maximum tokens allowed.
func New(rate float64, burst int) *Limiter {
	return &Limiter{
		buckets: make(map[string]*bucket),
		rate:    rate,
		burst:   burst,
		cleanup: 10 * time.Minute,
	}
}

// Allow checks whether a request for the given key is allowed.
// Returns true if allowed, false if rate limit exceeded.
func (l *Limiter) Allow(key string) bool {
	l.mu.Lock()
	defer l.mu.Unlock()

	now := time.Now()
	b, ok := l.buckets[key]
	if !ok {
		l.buckets[key] = &bucket{
			tokens:    float64(l.burst) - 1,
			lastCheck: now,
		}
		return true
	}

	// Refill tokens based on elapsed time
	elapsed := now.Sub(b.lastCheck).Seconds()
	b.tokens += elapsed * l.rate
	if b.tokens > float64(l.burst) {
		b.tokens = float64(l.burst)
	}
	b.lastCheck = now

	if b.tokens < 1 {
		return false
	}

	b.tokens--
	return true
}

// Cleanup removes entries that haven't been accessed within the cleanup duration.
func (l *Limiter) Cleanup() {
	l.mu.Lock()
	defer l.mu.Unlock()

	cutoff := time.Now().Add(-l.cleanup)
	for key, b := range l.buckets {
		if b.lastCheck.Before(cutoff) {
			delete(l.buckets, key)
		}
	}
}
