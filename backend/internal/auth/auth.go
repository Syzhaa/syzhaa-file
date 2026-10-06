package auth

import (
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"fmt"
	"github.com/syzhaa/file-server/internal/db"
	"github.com/syzhaa/file-server/internal/httpx"
	"log"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/google/uuid"
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
func HandleAdminLogin(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	var req struct {
		Email    string `json:"email"`
		Password string `json:"password"`
	}

	if err := httpx.ReadJSON(r, &req); err != nil {
		httpx.WriteJSON(w, http.StatusOK, map[string]interface{}{
			"success": false,
			"error":   "Invalid request body",
		})
		return
	}

	// Trim whitespace
	req.Email = strings.TrimSpace(req.Email)
	req.Password = strings.TrimSpace(req.Password)

	if req.Email == "" || req.Password == "" {
		httpx.WriteJSON(w, http.StatusOK, map[string]interface{}{
			"success": false,
			"error":   "Email and password required",
		})
		return
	}

	// Check admin email whitelist (env) OR existing admin in DB.
	// The env allowlist bootstraps initial access; once an admin changes
	// their email via /admin/account, the DB becomes the source of truth.
	allowedEmails := os.Getenv("ADMIN_EMAILS")
	if allowedEmails == "" {
		allowedEmails = os.Getenv("ADMIN_EMAIL")
	}

	isWhitelisted := false
	if allowedEmails != "" {
		emailList := strings.Split(allowedEmails, ",")
		for _, e := range emailList {
			if strings.TrimSpace(e) == req.Email {
				isWhitelisted = true
				break
			}
		}
	}
	if !isWhitelisted {
		var inDB bool
		_ = db.DB.QueryRow(`SELECT EXISTS(SELECT 1 FROM admin_users WHERE email = ?)`, req.Email).Scan(&inDB)
		isWhitelisted = inDB
	}

	if !isWhitelisted {
		httpx.WriteJSON(w, http.StatusUnauthorized, map[string]interface{}{
			"success": false,
			"error":   "Email not authorized",
		})
		log.Printf("⚠️ Unauthorized login attempt: %s", req.Email)
		return
	}

	// Verify password
	passwordHash := HashPassword(req.Password)
	var admin AdminUser
	err := db.DB.QueryRow(`
		SELECT id, email, name, created_at, last_login, is_super_admin
		FROM admin_users
		WHERE email = ? AND password_hash = ?
	`, req.Email, passwordHash).Scan(
		&admin.ID, &admin.Email, &admin.Name,
		&admin.CreatedAt, &admin.LastLogin, &admin.IsSuperAdmin)

	if err == sql.ErrNoRows {
		httpx.WriteJSON(w, http.StatusUnauthorized, map[string]interface{}{
			"success": false,
			"error":   "Invalid email or password",
		})
		log.Printf("❌ Failed login: %s", req.Email)
		return
	}

	if err != nil {
		http.Error(w, `{"error":"Server error"}`, http.StatusInternalServerError)
		log.Printf("❌ Database error: %v", err)
		return
	}

	// Create session
	session, err := CreateAdminSession(admin.ID)
	if err != nil {
		http.Error(w, `{"error":"Failed to create session"}`, http.StatusInternalServerError)
		return
	}

	// Update last login
	db.DB.Exec("UPDATE admin_users SET last_login = ? WHERE id = ?",
		time.Now().Format(time.RFC3339), admin.ID)

	// Set secure cookie
	http.SetCookie(w, &http.Cookie{
		Name:     "admin_session",
		Value:    session.Token,
		Path:     "/",
		Expires:  session.ExpiresAt,
		HttpOnly: true,
		Secure:   true,
		SameSite: http.SameSiteStrictMode,
	})

	log.Printf("✅ Admin login successful: %s", req.Email)

	httpx.WriteJSON(w, http.StatusOK, map[string]interface{}{
		"success": true,
		"token":   session.Token,
		"admin":   admin,
	})
}

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
func HandleAdminLogout(w http.ResponseWriter, r *http.Request) {
	logoutSession(adminSessionCfg, w, r)
}

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
func HandleAdminMe(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	admin := r.Context().Value("admin").(*AdminUser)

	httpx.WriteJSON(w, http.StatusOK, map[string]interface{}{
		"success": true,
		"admin":   admin,
	})
}

// Handler: Update own admin account (email and/or password).
// Requires current password verification.
func HandleAdminUpdateAccount(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	admin := r.Context().Value("admin").(*AdminUser)

	var req struct {
		Email           string `json:"email"`
		CurrentPassword string `json:"current_password"`
		NewPassword     string `json:"new_password"`
	}
	if err := httpx.ReadJSON(r, &req); err != nil {
		httpx.WriteJSON(w, http.StatusOK, map[string]interface{}{"success": false, "error": "Data tidak valid"})
		return
	}

	req.Email = strings.TrimSpace(strings.ToLower(req.Email))

	// Verify current password
	var storedHash string
	err := db.DB.QueryRow(`SELECT password_hash FROM admin_users WHERE id = ?`, admin.ID).Scan(&storedHash)
	if err != nil || HashPassword(req.CurrentPassword) != storedHash {
		httpx.WriteJSON(w, http.StatusOK, map[string]interface{}{"success": false, "error": "Password saat ini salah"})
		return
	}

	// Update email if changed
	if req.Email != "" && req.Email != admin.Email {
		if !strings.Contains(req.Email, "@") {
			httpx.WriteJSON(w, http.StatusOK, map[string]interface{}{"success": false, "error": "Format email tidak valid"})
			return
		}
		var taken bool
		_ = db.DB.QueryRow(`SELECT EXISTS(SELECT 1 FROM admin_users WHERE email = ? AND id != ?)`, req.Email, admin.ID).Scan(&taken)
		if taken {
			httpx.WriteJSON(w, http.StatusOK, map[string]interface{}{"success": false, "error": "Email sudah dipakai admin lain"})
			return
		}
		if _, err := db.DB.Exec(`UPDATE admin_users SET email = ? WHERE id = ?`, req.Email, admin.ID); err != nil {
			httpx.WriteJSON(w, http.StatusOK, map[string]interface{}{"success": false, "error": "Gagal mengganti email"})
			return
		}
		admin.Email = req.Email
	}

	// Update password if provided
	if req.NewPassword != "" {
		if len(req.NewPassword) < 8 {
			httpx.WriteJSON(w, http.StatusOK, map[string]interface{}{"success": false, "error": "Password baru minimal 8 karakter"})
			return
		}
		if _, err := db.DB.Exec(`UPDATE admin_users SET password_hash = ? WHERE id = ?`, HashPassword(req.NewPassword), admin.ID); err != nil {
			httpx.WriteJSON(w, http.StatusOK, map[string]interface{}{"success": false, "error": "Gagal mengganti password"})
			return
		}
	}

	httpx.WriteJSON(w, http.StatusOK, map[string]interface{}{
		"success": true,
		"message": "Akun berhasil diperbarui",
		"admin":   map[string]string{"email": admin.Email, "name": admin.Name},
	})
}
