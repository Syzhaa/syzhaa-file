package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestRateLimiterAllow(t *testing.T) {
	rl := NewRateLimiter(3, 1*time.Minute)
	key := "test-ip-1"

	// First 3 should pass
	for i := 0; i < 3; i++ {
		if !rl.Allow(key) {
			t.Fatalf("request %d should be allowed", i+1)
		}
	}
	// 4th should be blocked
	if rl.Allow(key) {
		t.Fatal("4th request should be blocked")
	}
	// Different key should still pass
	if !rl.Allow("test-ip-2") {
		t.Fatal("different IP should be allowed")
	}
}

func TestClientIP(t *testing.T) {
	tests := []struct {
		name     string
		headers  map[string]string
		remote   string
		expected string
	}{
		{"CF-Connecting-IP", map[string]string{"CF-Connecting-IP": "1.2.3.4"}, "127.0.0.1:1234", "1.2.3.4"},
		{"X-Real-IP", map[string]string{"X-Real-IP": "5.6.7.8"}, "127.0.0.1:1234", "5.6.7.8"},
		{"X-Forwarded-For single", map[string]string{"X-Forwarded-For": "9.10.11.12"}, "127.0.0.1:1234", "9.10.11.12"},
		{"X-Forwarded-For multi", map[string]string{"X-Forwarded-For": "9.10.11.12, 1.1.1.1"}, "127.0.0.1:1234", "9.10.11.12"},
		{"RemoteAddr fallback", map[string]string{}, "192.168.1.1:5678", "192.168.1.1"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := httptest.NewRequest("GET", "/", nil)
			r.RemoteAddr = tt.remote
			for k, v := range tt.headers {
				r.Header.Set(k, v)
			}
			if got := ClientIP(r); got != tt.expected {
				t.Errorf("ClientIP() = %q, want %q", got, tt.expected)
			}
		})
	}
}

func TestRateLimitMiddleware(t *testing.T) {
	rl := NewRateLimiter(2, 1*time.Minute)
	handler := RateLimitMiddleware(rl)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	// First 2 pass
	for i := 0; i < 2; i++ {
		req := httptest.NewRequest("GET", "/", nil)
		req.RemoteAddr = "10.0.0.1:1234"
		rec := httptest.NewRecorder()
		handler.ServeHTTP(rec, req)
		if rec.Code != http.StatusOK {
			t.Fatalf("request %d: got %d, want 200", i+1, rec.Code)
		}
	}

	// 3rd blocked with 429 and Retry-After
	req := httptest.NewRequest("GET", "/", nil)
	req.RemoteAddr = "10.0.0.1:1234"
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	if rec.Code != http.StatusTooManyRequests {
		t.Fatalf("got %d, want 429", rec.Code)
	}
	if rec.Header().Get("Retry-After") == "" {
		t.Fatal("Retry-After header missing")
	}
}
