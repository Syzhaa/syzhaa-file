package auth

import (
	"context"
	"crypto/rand"
	"database/sql"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"github.com/syzhaa/file-server/internal/db"
	"log"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/google/uuid"
	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
)

var googleOAuthConfig *oauth2.Config
var googleOAuthEnabled bool

type GoogleUserInfo struct {
	ID      string `json:"id"`
	Email   string `json:"email"`
	Name    string `json:"name"`
	Picture string `json:"picture"`
}

func InitGoogleOAuth() {
	clientID := os.Getenv("GOOGLE_CLIENT_ID")
	clientSecret := os.Getenv("GOOGLE_CLIENT_SECRET")
	if clientID == "" || clientSecret == "" {
		log.Println("ℹ️  Google OAuth disabled (GOOGLE_CLIENT_ID / GOOGLE_CLIENT_SECRET not set)")
		return
	}
	redirectURL := os.Getenv("GOOGLE_REDIRECT_URL")
	if redirectURL == "" {
		base := os.Getenv("BASE_URL")
		if base == "" {
			base = "https://ambilfile.web.id"
		}
		redirectURL = base + "/auth/user/google/callback"
	}
	googleOAuthConfig = &oauth2.Config{
		ClientID:     clientID,
		ClientSecret: clientSecret,
		RedirectURL:  redirectURL,
		Scopes: []string{
			"https://www.googleapis.com/auth/userinfo.email",
			"https://www.googleapis.com/auth/userinfo.profile",
		},
		Endpoint: google.Endpoint,
	}
	googleOAuthEnabled = true
	log.Printf("✅ Google OAuth enabled (redirect: %s)", redirectURL)
}

func googleOAuthState() (string, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return base64.URLEncoding.EncodeToString(b), nil
}

// GET /auth/user/google — redirect to Google
func HandleUserGoogleLogin(w http.ResponseWriter, r *http.Request) {
	if !googleOAuthEnabled {
		http.Error(w, `{"error":"Google login belum dikonfigurasi"}`, http.StatusServiceUnavailable)
		return
	}
	state, err := googleOAuthState()
	if err != nil {
		http.Error(w, `{"error":"Gagal membuat sesi login"}`, http.StatusInternalServerError)
		return
	}
	// Jalur aplikasi: tandai state agar callback tidak ke halaman web.
	// ?app=1 -> deep link ambilfile:// (Android/iOS)
	// ?app=desktop -> halaman HTML berisi token untuk di-copy (Windows/Linux/macOS)
	// Alur web tidak berubah sama sekali.
	if r.FormValue("app") == "desktop" {
		state += ".desktop"
	} else if r.FormValue("app") == "1" {
		state += ".mobile"
	}
	http.SetCookie(w, &http.Cookie{
		Name:     "oauth_state",
		Value:    state,
		Path:     "/",
		MaxAge:   600,
		HttpOnly: true,
		Secure:   true,
		SameSite: http.SameSiteLaxMode,
	})
	url := googleOAuthConfig.AuthCodeURL(state, oauth2.AccessTypeOnline)
	http.Redirect(w, r, url, http.StatusTemporaryRedirect)
}

// GET /auth/user/google/callback — Google redirects back here
func HandleUserGoogleCallback(w http.ResponseWriter, r *http.Request) {
	if !googleOAuthEnabled {
		http.Error(w, "Google login belum dikonfigurasi", http.StatusServiceUnavailable)
		return
	}

	// Verify state
	stateCookie, err := r.Cookie("oauth_state")
	if err != nil || stateCookie.Value == "" || stateCookie.Value != r.FormValue("state") {
		http.Error(w, "Sesi login tidak valid, silakan coba lagi", http.StatusBadRequest)
		return
	}
	// Clear state cookie
	http.SetCookie(w, &http.Cookie{Name: "oauth_state", Value: "", Path: "/", MaxAge: -1})

	code := r.FormValue("code")
	if code == "" {
		http.Redirect(w, r, "/user-login.html?error=cancelled", http.StatusTemporaryRedirect)
		return
	}

	token, err := googleOAuthConfig.Exchange(context.Background(), code)
	if err != nil {
		log.Printf("❌ Google token exchange failed: %v", err)
		http.Redirect(w, r, "/user-login.html?error=oauth", http.StatusTemporaryRedirect)
		return
	}

	// Fetch user profile
	client := googleOAuthConfig.Client(context.Background(), token)
	resp, err := client.Get("https://www.googleapis.com/oauth2/v2/userinfo")
	if err != nil {
		log.Printf("❌ Google userinfo failed: %v", err)
		http.Redirect(w, r, "/user-login.html?error=oauth", http.StatusTemporaryRedirect)
		return
	}
	defer resp.Body.Close()

	var gUser GoogleUserInfo
	if err := json.NewDecoder(resp.Body).Decode(&gUser); err != nil || gUser.Email == "" {
		log.Printf("❌ Google userinfo decode failed: %v", err)
		http.Redirect(w, r, "/user-login.html?error=oauth", http.StatusTemporaryRedirect)
		return
	}

	user := findOrCreateGoogleUser(&gUser)
	if user == nil {
		http.Redirect(w, r, "/user-login.html?error=server", http.StatusTemporaryRedirect)
		return
	}

	switch user.Status {
	case "approved", "active", "pending":
		sess, err := CreateUserSession(user.ID)
		if err != nil {
			http.Redirect(w, r, "/user-login.html?error=server", http.StatusTemporaryRedirect)
			return
		}
		// Aplikasi desktop: tampilkan token untuk di-copy manual.
		if strings.HasSuffix(r.FormValue("state"), ".desktop") {
			w.Header().Set("Content-Type", "text/html; charset=utf-8")
			fmt.Fprintf(w, `<!DOCTYPE html><html><head><meta charset="utf-8"><meta name="viewport" content="width=device-width,initial-scale=1"><title>Login Berhasil</title>
<style>body{font-family:system-ui,sans-serif;display:flex;align-items:center;justify-content:center;min-height:100vh;margin:0;background:#f9fafb}.box{background:#fff;border-radius:16px;padding:32px;max-width:420px;box-shadow:0 4px 24px rgba(0,0,0,.08);text-align:center}h2{color:#1f2937;margin:0 0 8px}p{color:#6b7280;font-size:14px}.token{background:#1f2937;color:#fff;font-family:monospace;font-size:12px;padding:12px;border-radius:8px;word-break:break-all;user-select:all;margin:16px 0}</style></head>
<body><div class="box"><h2>Login Berhasil ✅</h2><p>Copy token di bawah ini lalu paste di aplikasi AmbilFile Desktop:</p><div class="token">%s</div><p>Kamu bisa tutup halaman ini.</p></div></body></html>`, sess.Token)
			return
		}
		// Aplikasi Android: kembalikan token lewat deep link, bukan cookie web.
		if strings.HasSuffix(r.FormValue("state"), ".mobile") {
			http.Redirect(w, r, "ambilfile://login?token="+sess.Token, http.StatusTemporaryRedirect)
			return
		}
		http.SetCookie(w, &http.Cookie{
			Name:     "user_session",
			Value:    sess.Token,
			Path:     "/",
			MaxAge:   30 * 24 * 3600,
			HttpOnly: true,
			Secure:   true,
			SameSite: http.SameSiteLaxMode,
		})
		http.Redirect(w, r, "/user", http.StatusTemporaryRedirect)
	case "rejected":
		http.Redirect(w, r, "/user-login.html?error=rejected", http.StatusTemporaryRedirect)
	case "suspended":
		http.Redirect(w, r, "/user-login.html?error=suspended", http.StatusTemporaryRedirect)
	default:
		http.Redirect(w, r, "/user-login.html?error=status", http.StatusTemporaryRedirect)
	}
}

// findOrCreateGoogleUser links or creates a user from Google profile.
// New Google users start as "pending" (need admin approval), same as email signup.
func findOrCreateGoogleUser(g *GoogleUserInfo) *User {
	// 1. Existing Google-linked account
	var u User
	err := db.DB.QueryRow(`SELECT id, email, name, status FROM users WHERE google_id = ?`, g.ID).
		Scan(&u.ID, &u.Email, &u.Name, &u.Status)
	if err == nil {
		db.DB.Exec(`UPDATE users SET last_login = ?, avatar_url = ? WHERE id = ?`,
			time.Now().Format(time.RFC3339), g.Picture, u.ID)
		u.AvatarURL = g.Picture
		return &u
	}

	// 2. Existing email account (password signup) — link Google to it
	var existingID, existingStatus string
	err = db.DB.QueryRow(`SELECT id, status FROM users WHERE email = ?`, g.Email).
		Scan(&existingID, &existingStatus)
	if err == nil {
		db.DB.Exec(`UPDATE users SET google_id = ?, avatar_url = ?, last_login = ? WHERE id = ?`,
			g.ID, g.Picture, time.Now().Format(time.RFC3339), existingID)
		return &User{ID: existingID, Email: g.Email, Status: existingStatus}
	}

	// 3. Brand new user — active immediately, default 2GB quota.
	// API key creation needs admin approval separately.
	var defaultLimit int = 2048
	_ = db.DB.QueryRow(`SELECT value FROM system_settings WHERE key = 'default_storage_limit_mb'`).Scan(&defaultLimit)
	if defaultLimit <= 0 {
		defaultLimit = 2048
	}
	newID := uuid.New().String()
	_, err = db.DB.Exec(`
		INSERT INTO users (id, google_id, email, name, avatar_url, status, storage_limit_mb, api_approved, created_at)
		VALUES (?, ?, ?, ?, ?, 'active', ?, 0, ?)
	`, newID, g.ID, g.Email, g.Name, g.Picture, defaultLimit, time.Now().Format(time.RFC3339))
	if err != nil {
		log.Printf("❌ Failed to create Google user: %v", err)
		return nil
	}
	lim := defaultLimit
	return &User{ID: newID, Email: g.Email, Name: g.Name, Status: "active", StorageLimitMB: &lim}
}

// Ensure User struct has AvatarURL (added if missing)
var _ = sql.ErrNoRows
