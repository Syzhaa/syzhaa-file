package main

import (
	"crypto/rand"
	"encoding/hex"
	"net/http"
	"strings"
	"time"
)

// Session token handling shared by admin and user sessions.
// Cookie names, tables, and expiry differ per type — those stay
// in auth.go / user_auth.go. Only the token mechanics are shared.

// extractSessionToken reads the session token from cookie or Authorization header.
func extractSessionToken(r *http.Request, cookieName string) string {
	if c, err := r.Cookie(cookieName); err == nil {
		return c.Value
	}
	if h := r.Header.Get("Authorization"); strings.HasPrefix(h, "Bearer ") {
		t := strings.TrimPrefix(h, "Bearer ")
		if !strings.HasPrefix(t, "sfa_") {
			return t
		}
	}
	return ""
}

// newSessionToken generates a cryptographically random session token.
func newSessionToken() (token string, err error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return hex.EncodeToString(b), nil
}

// sessionExpiry returns the expiry time for a session type.
func sessionExpiry(isAdmin bool) time.Time {
	if isAdmin {
		return time.Now().Add(24 * time.Hour)
	}
	return time.Now().Add(7 * 24 * time.Hour)
}
