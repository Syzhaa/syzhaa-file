package auth

import (
	"fmt"
	"github.com/google/uuid"
	"crypto/sha256"
	"encoding/hex"
	"github.com/syzhaa/file-server/internal/db"
	"log"
	"net/http"
	"time"

)

type AdminUser struct {
	ID           string     `json:"id"`
	Email        string     `json:"email"`
	Name         string     `json:"name"`
	CreatedAt    time.Time  `json:"created_at"`
	LastLogin    *time.Time `json:"last_login,omitempty"`
	IsSuperAdmin int        `json:"is_super_admin"`
}

type AdminSession struct {
	ID        string    `json:"id"`
	AdminID   string    `json:"admin_id"`
	Token     string    `json:"token"`
	ExpiresAt time.Time `json:"expires_at"`
}

// Hash password using SHA256
func HashPassword(password string) string {
	h := sha256.New()
	h.Write([]byte(password))
	return hex.EncodeToString(h.Sum(nil))
}

// Alias for compatibility with api_keys.go
func HashString(s string) string {
	return HashPassword(s)
}

// Initialize admin user with default password on first run
func InitAdminDefaults(email, name, password string) {
	if email == "" {
		email = "admin@ambilfile.local"
	}
	if name == "" {
		name = "Administrator"
	}
	if password == "" {
		password = "admin"
	}

	passwordHash := HashPassword(password)

	// Check if admin already exists
	var exists bool
	err := db.DB.QueryRow("SELECT EXISTS(SELECT 1 FROM admin_users WHERE email = ?)", email).Scan(&exists)

	if err != nil {
		log.Printf("❌ Failed to check admin existence: %v", err)
		return
	}

	if !exists {
		// Create default admin user
		adminID := uuid.New().String()
		_, err := db.DB.Exec(`
			INSERT INTO admin_users (id, email, name, password_hash, created_at, is_super_admin)
			VALUES (?, ?, ?, ?, ?, 1)
		`, adminID, email, name, passwordHash, time.Now().Format(time.RFC3339))

		if err != nil {
			log.Printf("❌ Failed to create default admin: %v", err)
			return
		}
		log.Printf("✅ Default admin created: %s (password: %s)", email, password)
	}
}

// Handler: Admin login with email and password


// Create admin session
func CreateAdminSession(adminID string) (*AdminSession, error) {
	token, err := createSession(adminSessionCfg, adminID)
	if err != nil {
		return nil, err
	}
	return &AdminSession{
		ID:      uuid.New().String(),
		AdminID: adminID,
		Token:   token,
	}, nil
}

// Handler: Admin logout


// Generate random string helper
func GenerateRandomString(length int) string {
	const charset = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
	b := make([]byte, length)
	for i := range b {
		b[i] = charset[uuid.New().ID()%uint32(len(charset))]
	}
	return string(b)
}

// Validate admin session from cookie or Authorization header
// Validate admin session from cookie or Authorization header
func ValidateAdminSession(r *http.Request) (*AdminUser, error) {
	adminID, err := getSessionOwnerID(adminSessionCfg, r)
	if err != nil {
		return nil, fmt.Errorf("invalid or expired session")
	}

	// Get admin user
	var admin AdminUser
	err = db.DB.QueryRow(`
		SELECT id, email, name, created_at, last_login, is_super_admin
		FROM admin_users
		WHERE id = ?
	`, adminID).Scan(&admin.ID, &admin.Email, &admin.Name,
		&admin.CreatedAt, &admin.LastLogin, &admin.IsSuperAdmin)
	if err != nil {
		return nil, err
	}
	return &admin, nil
}

// Handler: Get current admin info


// Handler: Update own admin account (email and/or password).
// Requires current password verification.

