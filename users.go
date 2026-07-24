package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/google/uuid"
)

type User struct {
	ID                   string     `json:"id"`
	GoogleID             string     `json:"google_id"`
	Email                string     `json:"email"`
	Name                 string     `json:"name"`
	AvatarURL            string     `json:"avatar_url"`
	Status               string     `json:"status"` // pending, approved, rejected, suspended
	ApprovedBy           *string    `json:"approved_by,omitempty"`
	ApprovedAt           *time.Time `json:"approved_at,omitempty"`
	RejectedReason       *string    `json:"rejected_reason,omitempty"`
	StorageLimitMB       *int       `json:"storage_limit_mb"`
	MaxFileDurationDays  *int       `json:"max_file_duration_days"`
	CreatedAt            time.Time  `json:"created_at"`
	LastLogin            *time.Time `json:"last_login,omitempty"`
}

type UserStats struct {
	UserID            string     `json:"user_id"`
	TotalRooms        int        `json:"total_rooms_created"`
	TotalFiles        int        `json:"total_files_uploaded"`
	StorageUsedMB     float64    `json:"total_storage_used_mb"`
	LastUploadAt      *time.Time `json:"last_upload_at,omitempty"`
}

// Handler: User registration (Google OAuth callback for users)
func handleUserGoogleCallback(w http.ResponseWriter, r *http.Request) {
	log.Printf("🔍 User callback: method=%s url=%s", r.Method, r.URL.String())
	log.Printf("🔍 Query params: %v", r.URL.Query())
	log.Printf("🔍 Form values: state=%s code=%s", r.FormValue("state"), r.FormValue("code"))
	
	state := r.FormValue("state")
	if state != oauthStateString {
		log.Printf("❌ User OAuth state mismatch: got='%s' want='%s'", state, oauthStateString)
		http.Error(w, "Invalid OAuth state", http.StatusBadRequest)
		return
	}
	log.Printf("✅ User OAuth state valid")

	code := r.FormValue("code")
	
	// Create temporary config with user callback URL (use existing Google Console URI)
	userOAuthConfig := *googleOAuthConfig
	userOAuthConfig.RedirectURL = os.Getenv("GOOGLE_REDIRECT_URL") // Use existing /auth/google/callback
	
	token, err := userOAuthConfig.Exchange(context.Background(), code)
	if err != nil {
		http.Error(w, "Failed to exchange token", http.StatusInternalServerError)
		return
	}

	client := userOAuthConfig.Client(context.Background(), token)
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

	// Check if email is in admin whitelist
	allowedEmails := os.Getenv("ADMIN_EMAILS")
	if allowedEmails != "" {
		emailList := strings.Split(allowedEmails, ",")
		for _, email := range emailList {
			if strings.TrimSpace(email) == googleUser.Email {
				// This is an admin - redirect to admin flow
				log.Printf("✅ Admin detected: %s", googleUser.Email)
				
				// Process as admin user
				adminUser, err := getOrCreateAdminUser(googleUser)
				if err != nil {
					log.Printf("❌ Failed to create admin: %v", err)
					http.Error(w, "Failed to process admin user", http.StatusInternalServerError)
					return
				}
				
				// Create admin session
				adminSession, err := createAdminSession(adminUser.ID)
				if err != nil {
					http.Error(w, "Failed to create admin session", http.StatusInternalServerError)
					return
				}
				
				// Update last login
				db.Exec("UPDATE admin_users SET last_login = ? WHERE id = ?", time.Now().Format(time.RFC3339), adminUser.ID)
				
				// Set admin cookie
				http.SetCookie(w, &http.Cookie{
					Name:     "admin_session",
					Value:    adminSession.Token,
					Path:     "/",
					Expires:  adminSession.ExpiresAt,
					HttpOnly: true,
					Secure:   true,
					SameSite: http.SameSiteStrictMode,
				})
				
				// Redirect to admin dashboard
				http.Redirect(w, r, "/admin/dashboard-v2.html", http.StatusTemporaryRedirect)
				return
			}
		}
	}

	// Not admin - continue with normal user flow
	// Check or create user
	user, err := getOrCreateUser(googleUser)
	if err != nil {
		http.Error(w, "Failed to process user", http.StatusInternalServerError)
		return
	}

	// Check approval status
	if user.Status == "pending" {
		http.Redirect(w, r, "/pending-approval.html", http.StatusTemporaryRedirect)
		return
	}

	if user.Status == "rejected" {
		http.Redirect(w, r, "/registration-rejected.html", http.StatusTemporaryRedirect)
		return
	}

	if user.Status == "suspended" {
		http.Redirect(w, r, "/account-suspended.html", http.StatusTemporaryRedirect)
		return
	}

	// Create user session
	session, err := createUserSession(user.ID)
	if err != nil {
		http.Error(w, "Failed to create session", http.StatusInternalServerError)
		return
	}

	// Update last login
	db.Exec("UPDATE users SET last_login = ? WHERE id = ?", time.Now().Format(time.RFC3339), user.ID)

	// Set cookie
	http.SetCookie(w, &http.Cookie{
		Name:     "user_session",
		Value:    session.Token,
		Path:     "/",
		Expires:  session.ExpiresAt,
		HttpOnly: true,
		Secure:   true,
		SameSite: http.SameSiteStrictMode,
	})

	// Redirect to main app
	http.Redirect(w, r, "/", http.StatusTemporaryRedirect)
}

func getOrCreateUser(googleUser GoogleUserInfo) (*User, error) {
	var user User
	err := db.QueryRow(`SELECT id, google_id, email, name, avatar_url, status, 
		storage_limit_mb, max_file_duration_days, created_at, last_login 
		FROM users WHERE google_id = ?`, googleUser.ID).
		Scan(&user.ID, &user.GoogleID, &user.Email, &user.Name, &user.AvatarURL,
			&user.Status, &user.StorageLimitMB, &user.MaxFileDurationDays,
			&user.CreatedAt, &user.LastLogin)

	if err == sql.ErrNoRows {
		// Check if approval required
		var requireApproval string
		db.QueryRow("SELECT value FROM system_settings WHERE key = 'require_approval'").Scan(&requireApproval)

		status := "approved"
		if requireApproval == "true" {
			status = "pending"
		}

		// Create new user
		user.ID = uuid.New().String()
		user.GoogleID = googleUser.ID
		user.Email = googleUser.Email
		user.Name = googleUser.Name
		user.AvatarURL = googleUser.Picture
		user.Status = status
		user.CreatedAt = time.Now()

		_, err = db.Exec(`INSERT INTO users (id, google_id, email, name, avatar_url, status, created_at) 
			VALUES (?, ?, ?, ?, ?, ?, ?)`,
			user.ID, user.GoogleID, user.Email, user.Name, user.AvatarURL, user.Status,
			user.CreatedAt.Format(time.RFC3339))
		if err != nil {
			return nil, err
		}

		// Create stats entry
		db.Exec(`INSERT INTO user_stats (user_id) VALUES (?)`, user.ID)

		return &user, nil
	}

	if err != nil {
		return nil, err
	}

	return &user, nil
}

func createUserSession(userID string) (*AdminSession, error) {
	sessionID := uuid.New().String()
	token := generateRandomString(64)
	tokenHash := hashString(token)
	expiresAt := time.Now().Add(7 * 24 * time.Hour) // 7 days

	_, err := db.Exec(`INSERT INTO user_sessions (id, user_id, token_hash, expires_at) 
		VALUES (?, ?, ?, ?)`,
		sessionID, userID, tokenHash, expiresAt.Format(time.RFC3339))
	if err != nil {
		return nil, err
	}

	return &AdminSession{
		ID:        sessionID,
		AdminID:   userID, // reuse struct
		Token:     token,
		ExpiresAt: expiresAt,
	}, nil
}

func validateUserSession(token string) (*User, error) {
	tokenHash := hashString(token)

	var user User
	var expiresAt time.Time

	err := db.QueryRow(`SELECT u.id, u.google_id, u.email, u.name, u.avatar_url, u.status,
		u.storage_limit_mb, u.max_file_duration_days, s.expires_at
		FROM users u
		JOIN user_sessions s ON u.id = s.user_id
		WHERE s.token_hash = ?`, tokenHash).
		Scan(&user.ID, &user.GoogleID, &user.Email, &user.Name, &user.AvatarURL,
			&user.Status, &user.StorageLimitMB, &user.MaxFileDurationDays, &expiresAt)

	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("invalid session")
	}
	if err != nil {
		return nil, err
	}

	if time.Now().After(expiresAt) {
		return nil, fmt.Errorf("session expired")
	}

	if user.Status != "approved" {
		return nil, fmt.Errorf("user not approved")
	}

	return &user, nil
}

// Handler: User logout
func handleUserLogout(w http.ResponseWriter, r *http.Request) {
	cookie, err := r.Cookie("user_session")
	if err == nil {
		tokenHash := hashString(cookie.Value)
		db.Exec("DELETE FROM user_sessions WHERE token_hash = ?", tokenHash)
	}

	http.SetCookie(w, &http.Cookie{
		Name:     "user_session",
		Value:    "",
		Path:     "/",
		MaxAge:   -1,
		HttpOnly: true,
	})

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{"success": true})
}

// Handler: Get current user info
func handleUserMe(w http.ResponseWriter, r *http.Request) {
	user := r.Context().Value("user").(*User)

	// Get stats
	var stats UserStats
	db.QueryRow(`SELECT user_id, total_rooms_created, total_files_uploaded, 
		total_storage_used_mb, last_upload_at 
		FROM user_stats WHERE user_id = ?`, user.ID).
		Scan(&stats.UserID, &stats.TotalRooms, &stats.TotalFiles,
			&stats.StorageUsedMB, &stats.LastUploadAt)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"user":  user,
		"stats": stats,
	})
}
