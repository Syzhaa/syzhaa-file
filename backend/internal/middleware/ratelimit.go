package middleware

import (
	"net"
	"net/http"
	"strings"
	"sync"
	"time"
)

type RateLimiter struct {
	requests map[string][]time.Time
	mu       sync.Mutex
	limit    int
	window   time.Duration
}

func NewRateLimiter(limit int, window time.Duration) *RateLimiter {
	rl := &RateLimiter{
		requests: make(map[string][]time.Time),
		limit:    limit,
		window:   window,
	}

	// Start cleanup goroutine
	go rl.cleanup()

	return rl
}

func (rl *RateLimiter) cleanup() {
	ticker := time.NewTicker(1 * time.Minute)
	defer ticker.Stop()

	for range ticker.C {
		rl.mu.Lock()
		now := time.Now()
		for key, reqs := range rl.requests {
			var validReqs []time.Time
			for _, t := range reqs {
				if now.Sub(t) < rl.window {
					validReqs = append(validReqs, t)
				}
			}
			if len(validReqs) == 0 {
				delete(rl.requests, key)
			} else {
				rl.requests[key] = validReqs
			}
		}
		rl.mu.Unlock()
	}
}

func (rl *RateLimiter) Allow(key string) bool {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	now := time.Now()
	cutoff := now.Add(-rl.window)

	// Clean old requests
	reqs := rl.requests[key]
	var validReqs []time.Time
	for _, t := range reqs {
		if t.After(cutoff) {
			validReqs = append(validReqs, t)
		}
	}

	if len(validReqs) >= rl.limit {
		return false
	}

	validReqs = append(validReqs, now)
	rl.requests[key] = validReqs
	return true
}

var (
	GeneralLimiter *RateLimiter
	PinLimiter     *RateLimiter
	UploadLimiter  *RateLimiter
	LoginLimiter   *RateLimiter
)

func InitRateLimiters() {
	GeneralLimiter = NewRateLimiter(100, 1*time.Minute)
	PinLimiter = NewRateLimiter(5, 15*time.Minute)
	// Chunked uploads hit /api/upload/{roomId} once per chunk (5MB each),
	// so a 800MB file = ~160 requests. 10/min broke every upload >50MB.
	// 120/min still blocks floods while letting real uploads through.
	UploadLimiter = NewRateLimiter(120, 1*time.Minute)
	LoginLimiter = NewRateLimiter(5, 10*time.Minute)
}

// ClientIP extracts the real client IP, trusting Cloudflare/proxy headers.
// Priority: CF-Connecting-IP > X-Real-IP > X-Forwarded-For (first) > RemoteAddr.
func ClientIP(r *http.Request) string {
	if ip := r.Header.Get("CF-Connecting-IP"); ip != "" {
		return strings.TrimSpace(ip)
	}
	if ip := r.Header.Get("X-Real-IP"); ip != "" {
		return strings.TrimSpace(ip)
	}
	if xff := r.Header.Get("X-Forwarded-For"); xff != "" {
		if i := strings.Index(xff, ","); i >= 0 {
			return strings.TrimSpace(xff[:i])
		}
		return strings.TrimSpace(xff)
	}
	if host, _, err := net.SplitHostPort(r.RemoteAddr); err == nil {
		return host
	}
	return r.RemoteAddr
}

func RateLimitMiddleware(limiter *RateLimiter) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// /api/upload/* has its own dedicated UploadLimiter (per-chunk
			// requests); don't double-limit it with the general limiter.
			if limiter == GeneralLimiter && strings.HasPrefix(r.URL.Path, "/api/upload/") {
				next.ServeHTTP(w, r)
				return
			}
			ip := ClientIP(r)

			if !limiter.Allow(ip) {
				w.Header().Set("Retry-After", "60")
				http.Error(w, `{"error":"Rate limit exceeded. Please try again later."}`, http.StatusTooManyRequests)
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}
