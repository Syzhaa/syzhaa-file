package auth

import (
	"database/sql"
	"github.com/syzhaa/file-server/internal/db"
	"github.com/syzhaa/file-server/internal/httpx"
	"net/http"
	"strings"
	"time"

	"github.com/google/uuid"
)

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
func HandleUserLogout(w http.ResponseWriter, r *http.Request) {
	logoutSession(userSessionCfg, w, r)
}
func HandleUserLogin(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	httpx.WriteJSON(w, http.StatusGone, map[string]interface{}{
		"success": false,
		"error":   "Login via email dinonaktifkan. Silakan masuk dengan Google.",
	})
}
func HandleUserRegister(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	httpx.WriteJSON(w, http.StatusGone, map[string]interface{}{
		"success": false,
		"error":   "Pendaftaran via email dinonaktifkan. Silakan daftar dengan Google.",
	})
}
