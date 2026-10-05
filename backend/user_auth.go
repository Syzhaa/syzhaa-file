package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"net/http"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/gorilla/mux"
)

type userCtxKey struct{}

// contextWithUser stores the authenticated *User in the request context.
func contextWithUser(ctx context.Context, u *User) context.Context {
	return context.WithValue(ctx, userCtxKey{}, u)
}

// userFromContext retrieves the authenticated *User, or nil.
func userFromContext(ctx context.Context) *User {
	if u, ok := ctx.Value(userCtxKey{}).(*User); ok {
		return u
	}
	return nil
}

// ---------------------------------------------------------------------------
// Schema migration for email/password user auth + user-owned API keys.
// Safe to run on every startup (IF NOT EXISTS / guarded ALTERs).
// ---------------------------------------------------------------------------
func initUserAuthSchema() error {
	// password_hash for email/password login (users table predates it)
	_, _ = db.Exec(`ALTER TABLE users ADD COLUMN password_hash TEXT`)

	// user_id on api_keys so approved users can own keys (admin keys use admin_id)
	_, _ = db.Exec(`ALTER TABLE api_keys ADD COLUMN user_id TEXT`)

	return nil
}

// ---------------------------------------------------------------------------
// Session helpers (mirror the admin session pattern)
// ---------------------------------------------------------------------------
type UserSession struct {
	ID        string
	UserID    string
	Token     string
	ExpiresAt time.Time
}

func createUserSession(userID string) (*UserSession, error) {
	sessionID := uuid.New().String()
	token := generateRandomString(48)
	tokenHash := hashPassword(token)
	expiresAt := time.Now().Add(7 * 24 * time.Hour)

	_, err := db.Exec(`
		INSERT INTO user_sessions (id, user_id, token_hash, expires_at)
		VALUES (?, ?, ?, ?)
	`, sessionID, userID, tokenHash, expiresAt.Format(time.RFC3339))
	if err != nil {
		return nil, err
	}

	return &UserSession{ID: sessionID, UserID: userID, Token: token, ExpiresAt: expiresAt}, nil
}

func validateUserSession(r *http.Request) (*User, error) {
	var token string

	if cookie, err := r.Cookie("user_session"); err == nil {
		token = cookie.Value
	} else if h := r.Header.Get("Authorization"); strings.HasPrefix(h, "Bearer ") {
		// Only treat as session token if it is NOT an API key (sfa_ prefix)
		t := strings.TrimPrefix(h, "Bearer ")
		if !strings.HasPrefix(t, "sfa_") {
			token = t
		}
	}

	if token == "" {
		return nil, sql.ErrNoRows
	}

	tokenHash := hashPassword(token)

	var userID string
	err := db.QueryRow(`
		SELECT user_id FROM user_sessions
		WHERE token_hash = ? AND expires_at > ?
	`, tokenHash, time.Now().Format(time.RFC3339)).Scan(&userID)
	if err != nil {
		return nil, err
	}

	var u User
	var approvedBy, rejectedReason sql.NullString
	var approvedAt, lastLogin sql.NullString
	var storageLimit, maxDuration sql.NullInt64
	err = db.QueryRow(`
		SELECT id, email, name, status, approved_by, approved_at,
		       rejected_reason, storage_limit_mb, max_file_duration_days,
		       created_at, last_login
		FROM users WHERE id = ?
	`, userID).Scan(&u.ID, &u.Email, &u.Name, &u.Status,
		&approvedBy, &approvedAt, &rejectedReason,
		&storageLimit, &maxDuration, &u.CreatedAt, &lastLogin)
	if err != nil {
		return nil, err
	}
	if approvedBy.Valid {
		u.ApprovedBy = &approvedBy.String
	}
	if rejectedReason.Valid {
		u.RejectedReason = &rejectedReason.String
	}
	if storageLimit.Valid {
		v := int(storageLimit.Int64)
		u.StorageLimitMB = &v
	}
	if maxDuration.Valid {
		v := int(maxDuration.Int64)
		u.MaxFileDurationDays = &v
	}
	if u.Status != "approved" && u.Status != "active" {
		return nil, sql.ErrNoRows
	}

	return &u, nil
}

// requireUserSession replaces the old stub middleware.
func requireUserSession(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		user, err := validateUserSession(r)
		if err != nil {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusUnauthorized)
			json.NewEncoder(w).Encode(map[string]interface{}{
				"success": false,
				"error":   "Login diperlukan",
			})
			return
		}
		ctx := r.Context()
		// store *User in context under "user"
		next.ServeHTTP(w, r.WithContext(contextWithUser(ctx, user)))
	})
}

// ---------------------------------------------------------------------------
// Registration & login
// ---------------------------------------------------------------------------

// POST /auth/user/register
func handleUserRegister(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	var req struct {
		Name     string `json:"name"`
		Email    string `json:"email"`
		Password string `json:"password"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		json.NewEncoder(w).Encode(map[string]interface{}{"success": false, "error": "Data tidak valid"})
		return
	}

	req.Name = strings.TrimSpace(req.Name)
	req.Email = strings.TrimSpace(strings.ToLower(req.Email))

	if req.Name == "" || req.Email == "" || req.Password == "" {
		json.NewEncoder(w).Encode(map[string]interface{}{"success": false, "error": "Nama, email, dan password wajib diisi"})
		return
	}
	if !strings.Contains(req.Email, "@") {
		json.NewEncoder(w).Encode(map[string]interface{}{"success": false, "error": "Format email tidak valid"})
		return
	}
	if len(req.Password) < 6 {
		json.NewEncoder(w).Encode(map[string]interface{}{"success": false, "error": "Password minimal 6 karakter"})
		return
	}

	var exists bool
	err := db.QueryRow(`SELECT EXISTS(SELECT 1 FROM users WHERE email = ?)`, req.Email).Scan(&exists)
	if err == nil && exists {
		json.NewEncoder(w).Encode(map[string]interface{}{"success": false, "error": "Email sudah terdaftar"})
		return
	}
	var adminExists bool
	_ = db.QueryRow(`SELECT EXISTS(SELECT 1 FROM admin_users WHERE email = ?)`, req.Email).Scan(&adminExists)
	if adminExists {
		json.NewEncoder(w).Encode(map[string]interface{}{"success": false, "error": "Email sudah terdaftar"})
		return
	}

	userID := uuid.New().String()

	// Default storage quota for new users (2GB, from system settings)
	var defaultLimit int = 2048
	_ = db.QueryRow(`SELECT value FROM system_settings WHERE key = 'default_storage_limit_mb'`).Scan(&defaultLimit)
	if defaultLimit <= 0 {
		defaultLimit = 2048
	}

	// google_id uses a unique placeholder for password users (column is UNIQUE NOT NULL)
	// New users are active immediately; API key creation needs admin approval separately.
	_, err = db.Exec(`
		INSERT INTO users (id, google_id, email, name, password_hash, status, storage_limit_mb, api_approved, created_at)
		VALUES (?, ?, ?, ?, ?, 'active', ?, 0, ?)
	`, userID, "pwd_"+userID, req.Email, req.Name, hashPassword(req.Password), defaultLimit, time.Now().Format(time.RFC3339))
	if err != nil {
		json.NewEncoder(w).Encode(map[string]interface{}{"success": false, "error": "Gagal mendaftar, coba lagi"})
		return
	}

	json.NewEncoder(w).Encode(map[string]interface{}{
		"success": true,
		"message": "Pendaftaran berhasil! Silakan masuk.",
	})
}

// POST /auth/user/login
func handleUserLogin(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	var req struct {
		Email    string `json:"email"`
		Password string `json:"password"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		json.NewEncoder(w).Encode(map[string]interface{}{"success": false, "error": "Data tidak valid"})
		return
	}
	req.Email = strings.TrimSpace(strings.ToLower(req.Email))

	var userID, storedHash, status, name string
	err := db.QueryRow(`SELECT id, password_hash, status, name FROM users WHERE email = ?`,
		req.Email).Scan(&userID, &storedHash, &status, &name)
	if err != nil || storedHash == "" || hashPassword(req.Password) != storedHash {
		json.NewEncoder(w).Encode(map[string]interface{}{"success": false, "error": "Email atau password salah"})
		return
	}

	switch status {
	case "rejected":
		json.NewEncoder(w).Encode(map[string]interface{}{"success": false, "error": "Pendaftaranmu ditolak admin"})
		return
	case "suspended":
		json.NewEncoder(w).Encode(map[string]interface{}{"success": false, "error": "Akunmu dinonaktifkan"})
		return
	case "approved", "active", "pending":
		// ok — login allowed; API keys need separate approval
	default:
		json.NewEncoder(w).Encode(map[string]interface{}{"success": false, "error": "Status akun tidak valid"})
		return
	}

	sess, err := createUserSession(userID)
	if err != nil {
		json.NewEncoder(w).Encode(map[string]interface{}{"success": false, "error": "Gagal membuat sesi"})
		return
	}

	db.Exec(`UPDATE users SET last_login = ? WHERE id = ?`, time.Now().Format(time.RFC3339), userID)

	http.SetCookie(w, &http.Cookie{
		Name:     "user_session",
		Value:    sess.Token,
		Path:     "/",
		MaxAge:   7 * 24 * 3600,
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
	})

	json.NewEncoder(w).Encode(map[string]interface{}{
		"success": true,
		"user":    map[string]string{"id": userID, "email": req.Email, "name": name},
		"token":   sess.Token,
	})
}

// POST /auth/user/logout
func handleUserLogout(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	if cookie, err := r.Cookie("user_session"); err == nil {
		db.Exec(`DELETE FROM user_sessions WHERE token_hash = ?`, hashPassword(cookie.Value))
	}
	http.SetCookie(w, &http.Cookie{Name: "user_session", Value: "", Path: "/", MaxAge: -1, HttpOnly: true})

	json.NewEncoder(w).Encode(map[string]interface{}{"success": true})
}

// ---------------------------------------------------------------------------
// Authenticated user endpoints (replace stubs in users.go)
// ---------------------------------------------------------------------------

// GET /user/me
func handleUserMe(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	user := userFromContext(r.Context())
	if user == nil {
		w.WriteHeader(http.StatusUnauthorized)
		json.NewEncoder(w).Encode(map[string]interface{}{"success": false, "error": "Login diperlukan"})
		return
	}

	// Current storage usage across user's rooms
	var usedBytes int64
	_ = db.QueryRow(`
		SELECT COALESCE(SUM(f.size), 0) FROM files f
		JOIN rooms r ON r.id = f.room_id
		WHERE r.user_id = ?`, user.ID).Scan(&usedBytes)

	// API key approval state
	var apiApproved int
	var apiRequestedAt sql.NullString
	_ = db.QueryRow(`SELECT COALESCE(api_approved, 0), api_requested_at FROM users WHERE id = ?`,
		user.ID).Scan(&apiApproved, &apiRequestedAt)

	json.NewEncoder(w).Encode(map[string]interface{}{
		"success":             true,
		"user":                user,
		"storage_used_bytes":  usedBytes,
		"storage_used_label":  formatBytesID(usedBytes),
		"api_approved":        apiApproved == 1,
		"api_requested":       apiRequestedAt.Valid,
	})
}

// roomOut is the JSON shape for /user/rooms
type roomOut struct {
	ID        string `json:"id"`
	PIN       string `json:"pin"`
	CreatedAt string `json:"created_at"`
	ExpiresAt string `json:"expires_at"`
}

// keyOut is the JSON shape for /user/api-keys (list)
type keyOut struct {
	ID        string  `json:"id"`
	Name      string  `json:"name"`
	ExpiresAt *string `json:"expires_at,omitempty"`
	CreatedAt string  `json:"created_at"`
	IsActive  bool    `json:"is_active"`
}

// GET /user/rooms
func handleUserRooms(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	user := userFromContext(r.Context())
	if user == nil {
		w.WriteHeader(http.StatusUnauthorized)
		json.NewEncoder(w).Encode(map[string]interface{}{"success": false, "error": "Login diperlukan"})
		return
	}

	rows, err := db.Query(`
		SELECT id, pin, created_at, expires_at FROM rooms
		WHERE user_id = ? ORDER BY created_at DESC LIMIT 50
	`, user.ID)
	if err != nil {
		json.NewEncoder(w).Encode(map[string]interface{}{"success": true, "rooms": []roomOut{}})
		return
	}
	defer rows.Close()

	rooms := []roomOut{}
	for rows.Next() {
		var rm roomOut
		rows.Scan(&rm.ID, &rm.PIN, &rm.CreatedAt, &rm.ExpiresAt)
		rooms = append(rooms, rm)
	}
	json.NewEncoder(w).Encode(map[string]interface{}{"success": true, "rooms": rooms})
}

// ---------------------------------------------------------------------------
// User-owned API keys
// ---------------------------------------------------------------------------

// GET /user/api-keys
func handleUserListAPIKeys(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	user := userFromContext(r.Context())
	if user == nil {
		w.WriteHeader(http.StatusUnauthorized)
		json.NewEncoder(w).Encode(map[string]interface{}{"success": false, "error": "Login diperlukan"})
		return
	}

	rows, err := db.Query(`
		SELECT id, name, expires_at, created_at, last_used_at, is_active
		FROM api_keys WHERE user_id = ? ORDER BY created_at DESC
	`, user.ID)
	if err != nil {
		json.NewEncoder(w).Encode(map[string]interface{}{"success": true, "api_keys": []keyOut{}})
		return
	}
	defer rows.Close()

	keys := []keyOut{}
	for rows.Next() {
		var k keyOut
		var expiresAt sql.NullString
		var isActive int
		var lastUsed sql.NullString
		rows.Scan(&k.ID, &k.Name, &expiresAt, &k.CreatedAt, &lastUsed, &isActive)
		if expiresAt.Valid {
			k.ExpiresAt = &expiresAt.String
		}
		k.IsActive = isActive == 1
		keys = append(keys, k)
	}
	json.NewEncoder(w).Encode(map[string]interface{}{"success": true, "api_keys": keys})
}

// POST /user/api-keys
func handleUserCreateAPIKey(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	user := userFromContext(r.Context())
	if user == nil {
		w.WriteHeader(http.StatusUnauthorized)
		json.NewEncoder(w).Encode(map[string]interface{}{"success": false, "error": "Login diperlukan"})
		return
	}

	// API key creation requires admin approval
	var apiApproved int
	_ = db.QueryRow(`SELECT COALESCE(api_approved, 0) FROM users WHERE id = ?`, user.ID).Scan(&apiApproved)
	if apiApproved != 1 {
		w.WriteHeader(http.StatusForbidden)
		json.NewEncoder(w).Encode(map[string]interface{}{"success": false, "error": "Pembuatan API key perlu persetujuan admin. Minta persetujuan dulu ya."})
		return
	}

	var req struct {
		Name string `json:"name"`
	}
	json.NewDecoder(r.Body).Decode(&req)
	if strings.TrimSpace(req.Name) == "" {
		req.Name = "API Key"
	}

	// Reuse the same key format as admin keys
	rawKey := "sfa_" + generateRandomString(32)
	keyID := uuid.New().String()

	// api_keys.admin_id has a FOREIGN KEY to admin_users (PRAGMA foreign_keys=ON),
	// so point it at the admin who approved this user (fallback: first admin).
	ownerAdminID := ""
	if user.ApprovedBy != nil && *user.ApprovedBy != "" {
		ownerAdminID = *user.ApprovedBy
	} else {
		_ = db.QueryRow(`SELECT id FROM admin_users ORDER BY created_at LIMIT 1`).Scan(&ownerAdminID)
	}

	_, err := db.Exec(`
		INSERT INTO api_keys (id, key_hash, admin_id, user_id, name, created_at, is_active)
		VALUES (?, ?, ?, ?, ?, ?, 1)
	`, keyID, hashPassword(rawKey), ownerAdminID, user.ID, strings.TrimSpace(req.Name), time.Now().Format(time.RFC3339))
	if err != nil {
		json.NewEncoder(w).Encode(map[string]interface{}{"success": false, "error": "Gagal membuat API key"})
		return
	}

	json.NewEncoder(w).Encode(map[string]interface{}{
		"success": true,
		"api_key": map[string]string{"id": keyID, "key": rawKey, "name": strings.TrimSpace(req.Name)},
		"warning": "Simpan key ini baik-baik, tidak akan ditampilkan lagi.",
	})
}

// POST /user/api-keys/request — request admin approval for API key access
func handleUserRequestAPIAccess(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	user := userFromContext(r.Context())
	if user == nil {
		w.WriteHeader(http.StatusUnauthorized)
		json.NewEncoder(w).Encode(map[string]interface{}{"success": false, "error": "Login diperlukan"})
		return
	}

	var already int
	_ = db.QueryRow(`SELECT COALESCE(api_approved, 0) FROM users WHERE id = ?`, user.ID).Scan(&already)
	if already == 1 {
		json.NewEncoder(w).Encode(map[string]interface{}{"success": true, "message": "Sudah disetujui"})
		return
	}

	_, err := db.Exec(`UPDATE users SET api_requested_at = ? WHERE id = ?`,
		time.Now().Format(time.RFC3339), user.ID)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]interface{}{"success": false, "error": "Gagal mengirim permintaan"})
		return
	}
	json.NewEncoder(w).Encode(map[string]interface{}{"success": true, "message": "Permintaan terkirim"})
}

// DELETE /user/api-keys/{id}
func handleUserDeleteAPIKey(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	user := userFromContext(r.Context())
	if user == nil {
		w.WriteHeader(http.StatusUnauthorized)
		json.NewEncoder(w).Encode(map[string]interface{}{"success": false, "error": "Login diperlukan"})
		return
	}

	keyID := mux.Vars(r)["id"]
	res, err := db.Exec(`DELETE FROM api_keys WHERE id = ? AND user_id = ?`, keyID, user.ID)
	if err != nil {
		json.NewEncoder(w).Encode(map[string]interface{}{"success": false, "error": "Gagal menghapus"})
		return
	}
	if n, _ := res.RowsAffected(); n == 0 {
		w.WriteHeader(http.StatusNotFound)
		json.NewEncoder(w).Encode(map[string]interface{}{"success": false, "error": "API key tidak ditemukan"})
		return
	}
	json.NewEncoder(w).Encode(map[string]interface{}{"success": true})
}
