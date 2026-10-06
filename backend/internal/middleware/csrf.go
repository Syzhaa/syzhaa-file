package middleware

import (
	"crypto/rand"
	"encoding/base64"
	"net/http"
	"sync"
	"time"
)

type CSRFToken struct {
	Token     string
	ExpiresAt time.Time
}

var (
	csrfTokens = make(map[string]CSRFToken)
	csrfMutex  sync.RWMutex
)

func GenerateCSRFToken() string {
	b := make([]byte, 32)
	rand.Read(b)
	return base64.URLEncoding.EncodeToString(b)
}

func CreateCSRFToken(sessionID string) string {
	token := GenerateCSRFToken()

	csrfMutex.Lock()
	csrfTokens[sessionID] = CSRFToken{
		Token:     token,
		ExpiresAt: time.Now().Add(1 * time.Hour),
	}
	csrfMutex.Unlock()

	// Cleanup expired tokens
	go CleanupExpiredCSRFTokens()

	return token
}

func ValidateCSRFToken(sessionID, token string) bool {
	csrfMutex.RLock()
	defer csrfMutex.RUnlock()

	storedToken, exists := csrfTokens[sessionID]
	if !exists || time.Now().After(storedToken.ExpiresAt) {
		return false
	}

	return storedToken.Token == token
}

func CleanupExpiredCSRFTokens() {
	csrfMutex.Lock()
	defer csrfMutex.Unlock()

	now := time.Now()
	for sessionID, token := range csrfTokens {
		if now.After(token.ExpiresAt) {
			delete(csrfTokens, sessionID)
		}
	}
}

func CsrfMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Skip CSRF check for GET, HEAD, OPTIONS
		if r.Method == "GET" || r.Method == "HEAD" || r.Method == "OPTIONS" {
			next.ServeHTTP(w, r)
			return
		}

		// Skip for download endpoint
		if r.URL.Path[:3] == "/d/" {
			next.ServeHTTP(w, r)
			return
		}

		sessionID := r.Header.Get("X-Session-ID")
		csrfToken := r.Header.Get("X-CSRF-Token")

		if sessionID == "" || csrfToken == "" {
			http.Error(w, `{"error":"Missing CSRF token"}`, http.StatusForbidden)
			return
		}

		if !ValidateCSRFToken(sessionID, csrfToken) {
			http.Error(w, `{"error":"Invalid CSRF token"}`, http.StatusForbidden)
			return
		}

		next.ServeHTTP(w, r)
	})
}
