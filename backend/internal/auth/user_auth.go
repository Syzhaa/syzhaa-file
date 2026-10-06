package auth

import (
	"context"
	"database/sql"
	"github.com/syzhaa/file-server/internal/db"
	"github.com/syzhaa/file-server/internal/httpx"
	"net/http"
	"strings"
	"time"

	"github.com/google/uuid"
)

type userCtxKey struct{}

// User represents a registered user.
type User struct {
	ID                  string     `json:"id"`
	Email               string     `json:"email"`
	Name                string     `json:"name"`
	AvatarURL           string     `json:"avatar_url,omitempty"`
	Status              string     `json:"status"` // pending, approved, rejected, suspended
	ApprovedBy          *string    `json:"approved_by,omitempty"`
	ApprovedAt          *time.Time `json:"approved_at,omitempty"`
	RejectedReason      *string    `json:"rejected_reason,omitempty"`
	StorageLimitMB      *int       `json:"storage_limit_mb"`
	MaxFileDurationDays *int       `json:"max_file_duration_days"`
	CreatedAt           time.Time  `json:"created_at"`
	LastLogin           *time.Time `json:"last_login,omitempty"`
}

// contextWithUser stores the authenticated *User in the request context.
func contextWithUser(ctx context.Context, u *User) context.Context {
	return context.WithValue(ctx, userCtxKey{}, u)
}

// userFromContext retrieves the authenticated *User, or nil.
func UserFromContext(ctx context.Context) *User {
	if u, ok := ctx.Value(userCtxKey{}).(*User); ok {
		return u
	}
	return nil
}

// ---------------------------------------------------------------------------
// Schema migration for email/password user auth + user-owned API keys.
// Safe to run on every startup (IF NOT EXISTS / guarded ALTERs).
// ---------------------------------------------------------------------------

// ---------------------------------------------------------------------------
// Session helpers (mirror the admin session pattern)
// ---------------------------------------------------------------------------
type UserSession struct {
	ID        string
	UserID    string
	Token     string
	ExpiresAt time.Time
}

func CreateUserSession(userID string) (*UserSession, error) {
	sessionID := uuid.New().String()
	token := GenerateRandomString(48)
	tokenHash := HashPassword(token)
	expiresAt := time.Now().Add(7 * 24 * time.Hour)

	_, err := db.DB.Exec(`
		INSERT INTO user_sessions (id, user_id, token_hash, expires_at)
		VALUES (?, ?, ?, ?)
	`, sessionID, userID, tokenHash, expiresAt.Format(time.RFC3339))
	if err != nil {
		return nil, err
	}

	return &UserSession{ID: sessionID, UserID: userID, Token: token, ExpiresAt: expiresAt}, nil
}

func ValidateUserSession(r *http.Request) (*User, error) {
	userID, err := getSessionOwnerID(userSessionCfg, r)
	if err != nil {
		return nil, err
	}

	var u User
	var approvedBy, rejectedReason sql.NullString
	var approvedAt, lastLogin sql.NullString
	var storageLimit, maxDuration sql.NullInt64
	err = db.DB.QueryRow(`
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
func RequireUserSession(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		user, err := ValidateUserSession(r)
		if err != nil {
			w.Header().Set("Content-Type", "application/json")
			httpx.WriteJSON(w, http.StatusUnauthorized, map[string]interface{}{
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
// POST /auth/user/register — DISABLED: user signup is Google-only now
func HandleUserRegister(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	httpx.WriteJSON(w, http.StatusGone, map[string]interface{}{
		"success": false,
		"error":   "Pendaftaran via email dinonaktifkan. Silakan daftar dengan Google.",
	})
}

// POST /auth/user/login — DISABLED: user login is Google-only now
func HandleUserLogin(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	httpx.WriteJSON(w, http.StatusGone, map[string]interface{}{
		"success": false,
		"error":   "Login via email dinonaktifkan. Silakan masuk dengan Google.",
	})
}

// POST /auth/user/logout
func HandleUserLogout(w http.ResponseWriter, r *http.Request) {
	logoutSession(userSessionCfg, w, r)
}

// ---------------------------------------------------------------------------
// Authenticated user endpoints (replace stubs in users.go)
// ---------------------------------------------------------------------------

// GET /user/me
func HandleUserMe(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	user := UserFromContext(r.Context())
	if user == nil {
		httpx.WriteJSON(w, http.StatusUnauthorized, map[string]interface{}{"success": false, "error": "Login diperlukan"})
		return
	}

	// Current storage usage across user's rooms
	var usedBytes int64
	_ = db.DB.QueryRow(`
		SELECT COALESCE(SUM(f.size), 0) FROM files f
		JOIN rooms r ON r.id = f.room_id
		WHERE r.user_id = ?`, user.ID).Scan(&usedBytes)

	// API key approval state
	var apiApproved int
	var apiRequestedAt sql.NullString
	_ = db.DB.QueryRow(`SELECT COALESCE(api_approved, 0), api_requested_at FROM users WHERE id = ?`,
		user.ID).Scan(&apiApproved, &apiRequestedAt)

	httpx.WriteJSON(w, http.StatusOK, map[string]interface{}{
		"success":            true,
		"user":               user,
		"storage_used_bytes": usedBytes,
		"storage_used_label": httpx.FormatBytesID(usedBytes),
		"api_approved":       apiApproved == 1,
		"api_requested":      apiRequestedAt.Valid,
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
func HandleUserRooms(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	user := UserFromContext(r.Context())
	if user == nil {
		httpx.WriteJSON(w, http.StatusUnauthorized, map[string]interface{}{"success": false, "error": "Login diperlukan"})
		return
	}

	rows, err := db.DB.Query(`
		SELECT id, pin, created_at, expires_at FROM rooms
		WHERE user_id = ? ORDER BY created_at DESC LIMIT 50
	`, user.ID)
	if err != nil {
		httpx.WriteJSON(w, http.StatusOK, map[string]interface{}{"success": true, "rooms": []roomOut{}})
		return
	}
	defer rows.Close()

	rooms := []roomOut{}
	for rows.Next() {
		var rm roomOut
		rows.Scan(&rm.ID, &rm.PIN, &rm.CreatedAt, &rm.ExpiresAt)
		rooms = append(rooms, rm)
	}
	httpx.WriteJSON(w, http.StatusOK, map[string]interface{}{"success": true, "rooms": rooms})
}

// ---------------------------------------------------------------------------
// User-owned API keys
// ---------------------------------------------------------------------------

// GET /user/api-keys

// POST /user/api-keys
func HandleUserCreateAPIKey(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	user := UserFromContext(r.Context())
	if user == nil {
		httpx.WriteJSON(w, http.StatusUnauthorized, map[string]interface{}{"success": false, "error": "Login diperlukan"})
		return
	}

	// API key creation requires admin approval
	var apiApproved int
	_ = db.DB.QueryRow(`SELECT COALESCE(api_approved, 0) FROM users WHERE id = ?`, user.ID).Scan(&apiApproved)
	if apiApproved != 1 {
		httpx.WriteJSON(w, http.StatusForbidden, map[string]interface{}{"success": false, "error": "Pembuatan API key perlu persetujuan admin. Minta persetujuan dulu ya."})
		return
	}

	var req struct {
		Name string `json:"name"`
	}
	httpx.ReadJSON(r, &req)
	if strings.TrimSpace(req.Name) == "" {
		req.Name = "API Key"
	}

	// Reuse the same key format as admin keys
	rawKey := "sfa_" + GenerateRandomString(32)
	keyID := uuid.New().String()

	// api_keys.admin_id has a FOREIGN KEY to admin_users (PRAGMA foreign_keys=ON),
	// so point it at the admin who approved this user (fallback: first admin).
	ownerAdminID := ""
	if user.ApprovedBy != nil && *user.ApprovedBy != "" {
		ownerAdminID = *user.ApprovedBy
	} else {
		_ = db.DB.QueryRow(`SELECT id FROM admin_users ORDER BY created_at LIMIT 1`).Scan(&ownerAdminID)
	}

	_, err := db.DB.Exec(`
		INSERT INTO api_keys (id, key_hash, admin_id, user_id, name, created_at, is_active)
		VALUES (?, ?, ?, ?, ?, ?, 1)
	`, keyID, HashPassword(rawKey), ownerAdminID, user.ID, strings.TrimSpace(req.Name), time.Now().Format(time.RFC3339))
	if err != nil {
		httpx.WriteJSON(w, http.StatusOK, map[string]interface{}{"success": false, "error": "Gagal membuat API key"})
		return
	}

	httpx.WriteJSON(w, http.StatusOK, map[string]interface{}{
		"success": true,
		"api_key": map[string]string{"id": keyID, "key": rawKey, "name": strings.TrimSpace(req.Name)},
		"warning": "Simpan key ini baik-baik, tidak akan ditampilkan lagi.",
	})
}

// POST /user/api-keys/request — request admin approval for API key access
func HandleUserRequestAPIAccess(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	user := UserFromContext(r.Context())
	if user == nil {
		httpx.WriteJSON(w, http.StatusUnauthorized, map[string]interface{}{"success": false, "error": "Login diperlukan"})
		return
	}

	var already int
	_ = db.DB.QueryRow(`SELECT COALESCE(api_approved, 0) FROM users WHERE id = ?`, user.ID).Scan(&already)
	if already == 1 {
		httpx.WriteJSON(w, http.StatusOK, map[string]interface{}{"success": true, "message": "Sudah disetujui"})
		return
	}

	_, err := db.DB.Exec(`UPDATE users SET api_requested_at = ? WHERE id = ?`,
		time.Now().Format(time.RFC3339), user.ID)
	if err != nil {
		httpx.WriteJSON(w, http.StatusInternalServerError, map[string]interface{}{"success": false, "error": "Gagal mengirim permintaan"})
		return
	}
	httpx.WriteJSON(w, http.StatusOK, map[string]interface{}{"success": true, "message": "Permintaan terkirim"})
}

// DELETE /user/api-keys/{id}
