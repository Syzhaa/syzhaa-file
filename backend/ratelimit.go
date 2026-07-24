package main

import (
	"net/http"
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
	generalLimiter *RateLimiter
	pinLimiter     *RateLimiter
	uploadLimiter  *RateLimiter
)

func initRateLimiters() {
	generalLimiter = NewRateLimiter(100, 1*time.Minute)
	pinLimiter = NewRateLimiter(5, 15*time.Minute)
	uploadLimiter = NewRateLimiter(10, 1*time.Minute)
}

func rateLimitMiddleware(limiter *RateLimiter) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			ip := r.RemoteAddr
			
			if !limiter.Allow(ip) {
				w.Header().Set("Retry-After", "60")
				http.Error(w, `{"error":"Rate limit exceeded. Please try again later."}`, http.StatusTooManyRequests)
				return
			}
			
			next.ServeHTTP(w, r)
		})
	}
}
