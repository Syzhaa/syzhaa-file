package main

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"time"

	"github.com/google/uuid"
	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
)

var (
	googleOAuthConfig *oauth2.Config
	oauthStateString  = generateRandomString(32)
)

type AdminUser struct {
	ID        string    `json:"id"`
	GoogleID  string    `json:"google_id"`
	Email     string    `json:"email"`
	Name      string    `json:"name"`
	AvatarURL string    `json:"avatar_url"`
	CreatedAt time.Time `json:"created_at"`
	LastLogin time.Time `json:"last_login"`
}

type AdminSession struct {
	ID        string    `json:"id"`
	AdminID   string    `json:"admin_id"`
	Token     string    `json:"token"`
	ExpiresAt time.Time `json:"expires_at"`
}

type GoogleUserInfo struct {
	ID      string `json:"id"`
	Email   string `json:"email"`
	Name    string `json:"name"`
	Picture string `json:"picture"`
}

func initGoogleOAuth(clientID, clientSecret, redirectURL string) {
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
}

func generateRandomString(length int) string {
	b := make([]byte, length)
	rand.Read(b)
	return hex.EncodeToString(b)
}

func hashString(s string) string {
	h := sha256.New()
	h.Write([]byte(s))
	return hex.EncodeToString(h.Sum(nil))
}

// Handler: Initiate Google OAuth (Admin)
func handleGoogleLogin(w http.ResponseWriter, r *http.Request) {
	url := googleOAuthConfig.AuthCodeURL(oauthStateString, oauth2.AccessTypeOffline)
	http.Redirect(w, r, url, http.StatusTemporaryRedirect)
}

// Handler: Initiate Google OAuth (User) - separate redirect URL
func handleUserGoogleLogin(w http.ResponseWriter, r *http.Request) {
	// Create temporary config with user callback URL
	userOAuthConfig := *googleOAuthConfig
	userOAuthConfig.RedirectURL = os.Getenv("BASE_URL") + "/auth/user/callback"
	
	url := userOAuthConfig.AuthCodeURL(oauthStateString, oauth2.AccessTypeOffline)
	http.Redirect(w, r, url, http.StatusTemporaryRedirect)
}

// Handler: Google OAuth Callback
func handleGoogleCallback(w http.ResponseWriter, r *http.Request) {
	state := r.FormValue("state")
	if state != oauthStateString {
		http.Error(w, "Invalid OAuth state", http.StatusBadRequest)
		return
	}

	code := r.FormValue("code")
	token, err := googleOAuthConfig.Exchange(context.Background(), code)
	if err != nil {
		http.Error(w, "Failed to exchange token", http.StatusInternalServerError)
		return
	}

	client := googleOAuthConfig.Client(context.Background(), token)
	resp, err := client.Get("https://www.googleapis.com/oauth2/v2/userinfo")
	if err != nil {
		http.Error(w, "Failed to get user info", http.StatusInternalServerError)
		return
	}
	defer resp.Body.Close()

	var googleUser GoogleUserInfo
	if err := json.NewDecoder(resp.Body).Decode(&googleUser); err != nil {
		http.Error(w, "Failed to decode user info", http.StatusInternalServerError)
		return
	}

	// Check or create admin user
	adminUser, err := getOrCreateAdminUser(googleUser)
	if err != nil {
		http.Error(w, "Failed to process user", http.StatusInternalServerError)
		return
	}

	// Create session
	session, err := createAdminSession(adminUser.ID)
	if err != nil {
		http.Error(w, "Failed to create session", http.StatusInternalServerError)
		return
	}

	// Update last login
	db.Exec("UPDATE admin_users SET last_login = ? WHERE id = ?", time.Now().Format(time.RFC3339), adminUser.ID)

	// Set cookie
	http.SetCookie(w, &http.Cookie{
		Name:     "admin_session",
		Value:    session.Token,
		Path:     "/",
		Expires:  session.ExpiresAt,
		HttpOnly: true,
		Secure:   true,
		SameSite: http.SameSiteStrictMode,
	})

	// Redirect to admin panel
	http.Redirect(w, r, "/admin/dashboard", http.StatusTemporaryRedirect)
}

func getOrCreateAdminUser(googleUser GoogleUserInfo) (*AdminUser, error) {
	var admin AdminUser
	err := db.QueryRow(`SELECT id, google_id, email, name, avatar_url, created_at, last_login 
		FROM admin_users WHERE google_id = ?`, googleUser.ID).
		Scan(&admin.ID, &admin.GoogleID, &admin.Email, &admin.Name, &admin.AvatarURL, &admin.CreatedAt, &admin.LastLogin)

	if err == sql.ErrNoRows {
		// Create new admin user
		admin.ID = uuid.New().String()
		admin.GoogleID = googleUser.ID
		admin.Email = googleUser.Email
		admin.Name = googleUser.Name
		admin.AvatarURL = googleUser.Picture
		admin.CreatedAt = time.Now()

		_, err = db.Exec(`INSERT INTO admin_users (id, google_id, email, name, avatar_url, created_at) 
			VALUES (?, ?, ?, ?, ?, ?)`,
			admin.ID, admin.GoogleID, admin.Email, admin.Name, admin.AvatarURL, admin.CreatedAt.Format(time.RFC3339))
		if err != nil {
			return nil, err
		}
		return &admin, nil
	}

	if err != nil {
		return nil, err
	}

	return &admin, nil
}

func createAdminSession(adminID string) (*AdminSession, error) {
	sessionID := uuid.New().String()
	token := generateRandomString(64)
	tokenHash := hashString(token)
	expiresAt := time.Now().Add(7 * 24 * time.Hour) // 7 days

	_, err := db.Exec(`INSERT INTO admin_sessions (id, admin_id, token_hash, expires_at) 
		VALUES (?, ?, ?, ?)`,
		sessionID, adminID, tokenHash, expiresAt.Format(time.RFC3339))
	if err != nil {
		return nil, err
	}

	return &AdminSession{
		ID:        sessionID,
		AdminID:   adminID,
		Token:     token,
		ExpiresAt: expiresAt,
	}, nil
}

func validateAdminSession(token string) (*AdminUser, error) {
	tokenHash := hashString(token)

	var admin AdminUser
	var expiresAt time.Time

	err := db.QueryRow(`SELECT u.id, u.google_id, u.email, u.name, u.avatar_url, s.expires_at
		FROM admin_users u
		JOIN admin_sessions s ON u.id = s.admin_id
		WHERE s.token_hash = ?`, tokenHash).
		Scan(&admin.ID, &admin.GoogleID, &admin.Email, &admin.Name, &admin.AvatarURL, &expiresAt)

	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("invalid session")
	}
	if err != nil {
		return nil, err
	}

	if time.Now().After(expiresAt) {
		return nil, fmt.Errorf("session expired")
	}

	return &admin, nil
}

// Handler: Logout
func handleAdminLogout(w http.ResponseWriter, r *http.Request) {
	cookie, err := r.Cookie("admin_session")
	if err == nil {
		tokenHash := hashString(cookie.Value)
		db.Exec("DELETE FROM admin_sessions WHERE token_hash = ?", tokenHash)
	}

	http.SetCookie(w, &http.Cookie{
		Name:     "admin_session",
		Value:    "",
		Path:     "/",
		MaxAge:   -1,
		HttpOnly: true,
	})

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{"success": true})
}

// Handler: Get current admin info
func handleAdminMe(w http.ResponseWriter, r *http.Request) {
	admin := r.Context().Value("admin").(*AdminUser)
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(admin)
}
