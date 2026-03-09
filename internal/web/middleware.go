package web

import (
	"net/http"
	"strings"

	"github.com/justestif/go-spotify-era-organizer/internal/ratelimit"
)

// RateLimitMiddleware returns middleware that rate limits requests per user (or per IP if unauthenticated).
func RateLimitMiddleware(limiter *ratelimit.Limiter, sessions SessionManager) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Determine rate limit key
			key := rateLimitKey(r, sessions)

			if !limiter.Allow(key) {
				w.Header().Set("Retry-After", "1")
				http.Error(w, "Too Many Requests", http.StatusTooManyRequests)
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}

// rateLimitKey returns the user ID if authenticated, otherwise the client IP.
func rateLimitKey(r *http.Request, sessions SessionManager) string {
	if session := sessions.GetFromRequest(r); session != nil {
		return "user:" + session.UserID
	}

	// Use IP for unauthenticated requests
	ip := r.RemoteAddr
	if fwd := r.Header.Get("X-Forwarded-For"); fwd != "" {
		ip = strings.SplitN(fwd, ",", 2)[0]
	}
	return "ip:" + strings.TrimSpace(ip)
}
