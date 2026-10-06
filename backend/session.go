package main

import (
	"github.com/syzhaa/file-server/internal/httpx"
	"database/sql"
	"fmt"
	"net/http"
	"time"

	"github.com/google/uuid"
)

// sessionConfig holds the per-type differences between admin and user sessions.
type sessionConfig struct {
	table      string // admin_sessions | user_sessions
	idColumn   string // admin_id | user_id
	cookieName string // admin_session | user_session
	expiry     time.Duration
	sameSite   http.SameSite
}

var adminSessionCfg = sessionConfig{
	table:      "admin_sessions",
	idColumn:   "admin_id",
	cookieName: "admin_session",
	expiry:     24 * time.Hour,
	sameSite:   http.SameSiteStrictMode,
}

var userSessionCfg = sessionConfig{
	table:      "user_sessions",
	idColumn:   "user_id",
	cookieName: "user_session",
	expiry:     7 * 24 * time.Hour,
	sameSite:   http.SameSiteLaxMode,
}

// createSession creates a new session and returns the raw token.
func createSession(cfg sessionConfig, ownerID string) (token string, err error) {
	sessionID := uuid.New().String()
	token = generateRandomString(64)
	tokenHash := hashPassword(token)
	expiresAt := time.Now().Add(cfg.expiry)

	_, err = db.Exec(
		fmt.Sprintf("INSERT INTO %s (id, %s, token_hash, expires_at) VALUES (?, ?, ?, ?)",
			cfg.table, cfg.idColumn),
		sessionID, ownerID, tokenHash, expiresAt.Format(time.RFC3339))
	if err != nil {
		return "", err
	}
	return token, nil
}

// getSessionOwnerID validates the session token and returns the owner ID.
func getSessionOwnerID(cfg sessionConfig, r *http.Request) (string, error) {
	var token string
	if c, err := r.Cookie(cfg.cookieName); err == nil {
		token = c.Value
	} else if h := r.Header.Get("Authorization"); len(h) > 7 && h[:7] == "Bearer " {
		t := h[7:]
		if len(t) < 4 || t[:4] != "sfa_" {
			token = t
		}
	}
	if token == "" {
		return "", sql.ErrNoRows
	}

	tokenHash := hashPassword(token)
	var ownerID string
	err := db.QueryRow(
		fmt.Sprintf("SELECT %s FROM %s WHERE token_hash = ? AND expires_at > ?",
			cfg.idColumn, cfg.table),
		tokenHash, time.Now().Format(time.RFC3339)).Scan(&ownerID)
	if err != nil {
		return "", err
	}
	return ownerID, nil
}

// setSessionCookie writes the session cookie to the response.
func setSessionCookie(cfg sessionConfig, w http.ResponseWriter, token string, maxAge int) {
	http.SetCookie(w, &http.Cookie{
		Name:     cfg.cookieName,
		Value:    token,
		Path:     "/",
		MaxAge:   maxAge,
		HttpOnly: true,
		Secure:   true,
		SameSite: cfg.sameSite,
	})
}

// clearSessionCookie expires the session cookie.
func clearSessionCookie(cfg sessionConfig, w http.ResponseWriter) {
	setSessionCookie(cfg, w, "", -1)
}

// logoutSession deletes the session from DB and clears the cookie.
func logoutSession(cfg sessionConfig, w http.ResponseWriter, r *http.Request) {
	if c, err := r.Cookie(cfg.cookieName); err == nil && c.Value != "" {
		db.Exec(fmt.Sprintf("DELETE FROM %s WHERE token_hash = ?",
			cfg.table), hashPassword(c.Value))
	}
	clearSessionCookie(cfg, w)
	httpx.WriteJSON(w, http.StatusOK, map[string]interface{}{
		"success": true,
		"message": "Logged out successfully",
	})
}
