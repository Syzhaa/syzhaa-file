package apikeys

import (
	"database/sql"
	"github.com/syzhaa/file-server/internal/auth"
	"github.com/syzhaa/file-server/internal/db"
	"github.com/syzhaa/file-server/internal/httpx"
	"net/http"
	"time"

	"github.com/gorilla/mux"
)

// Unified API key handlers with role-based ownership.
// Admin: ownerColumn="admin_id", user: ownerColumn="user_id".
// Response formats are preserved per role for backward compatibility.

// listKeysByOwner returns API keys owned by the given ID.
func listKeysByOwner(ownerColumn, ownerID string) ([]map[string]interface{}, error) {
	rows, err := db.DB.Query(
		`SELECT id, name, expires_at, created_at, last_used_at, is_active
		 FROM api_keys WHERE `+ownerColumn+` = ? ORDER BY created_at DESC`, ownerID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	keys := []map[string]interface{}{}
	for rows.Next() {
		var id, name, createdAt string
		var expiresAt, lastUsed sql.NullString
		var isActive int
		rows.Scan(&id, &name, &expiresAt, &createdAt, &lastUsed, &isActive)
		k := map[string]interface{}{
			"id": id, "name": name, "created_at": createdAt,
			"is_active": isActive == 1,
		}
		if expiresAt.Valid {
			k["expires_at"] = expiresAt.String
		}
		if lastUsed.Valid {
			k["last_used_at"] = lastUsed.String
		}
		keys = append(keys, k)
	}
	return keys, nil
}

// deleteKeyByOwner deletes a key if owned by the given ID. Returns true if deleted.
func deleteKeyByOwner(ownerColumn, ownerID, keyID string) (bool, error) {
	res, err := db.DB.Exec(
		`DELETE FROM api_keys WHERE id = ? AND `+ownerColumn+` = ?`, keyID, ownerID)
	if err != nil {
		return false, err
	}
	n, _ := res.RowsAffected()
	return n > 0, nil
}

// --- Admin wrappers (preserve existing response format) ---

func HandleListAPIKeys(w http.ResponseWriter, r *http.Request) {
	admin := r.Context().Value("admin").(*auth.AdminUser)
	keys, err := listKeysByOwner("admin_id", admin.ID)
	if err != nil {
		httpx.WriteError(w, http.StatusInternalServerError, "Failed to fetch API keys")
		return
	}
	// Preserve admin response format: {"success":true,"keys":[...]}
	httpx.WriteJSON(w, http.StatusOK, map[string]interface{}{"success": true, "keys": keys})
}

func HandleDeleteAPIKey(w http.ResponseWriter, r *http.Request) {
	admin := r.Context().Value("admin").(*auth.AdminUser)
	keyID := mux.Vars(r)["id"]

	// Check existence first (preserve 404 behavior)
	var ownerID string
	err := db.DB.QueryRow("SELECT admin_id FROM api_keys WHERE id = ?", keyID).Scan(&ownerID)
	if err == sql.ErrNoRows {
		httpx.WriteError(w, http.StatusNotFound, "API key not found")
		return
	}
	if err != nil {
		httpx.WriteError(w, http.StatusInternalServerError, "Server error")
		return
	}
	if ownerID != admin.ID {
		httpx.WriteError(w, http.StatusForbidden, "Unauthorized")
		return
	}

	deleted, err := deleteKeyByOwner("admin_id", admin.ID, keyID)
	if err != nil || !deleted {
		httpx.WriteError(w, http.StatusInternalServerError, "Failed to delete API key")
		return
	}
	httpx.WriteJSON(w, http.StatusOK, map[string]interface{}{"success": true})
}

// --- auth.User wrappers (preserve existing response format) ---

func HandleUserListAPIKeys(w http.ResponseWriter, r *http.Request) {
	user := auth.UserFromContext(r.Context())
	if user == nil {
		httpx.WriteJSON(w, http.StatusUnauthorized,
			map[string]interface{}{"success": false, "error": "Login diperlukan"})
		return
	}
	keys, err := listKeysByOwner("user_id", user.ID)
	if err != nil {
		httpx.WriteJSON(w, http.StatusOK,
			map[string]interface{}{"success": true, "api_keys": []interface{}{}})
		return
	}
	// Preserve user response format: {"success":true,"api_keys":[...]}
	httpx.WriteJSON(w, http.StatusOK, map[string]interface{}{"success": true, "api_keys": keys})
}

func HandleUserDeleteAPIKey(w http.ResponseWriter, r *http.Request) {
	user := auth.UserFromContext(r.Context())
	if user == nil {
		httpx.WriteJSON(w, http.StatusUnauthorized,
			map[string]interface{}{"success": false, "error": "Login diperlukan"})
		return
	}
	keyID := mux.Vars(r)["id"]
	deleted, err := deleteKeyByOwner("user_id", user.ID, keyID)
	if err != nil {
		httpx.WriteJSON(w, http.StatusOK,
			map[string]interface{}{"success": false, "error": "Gagal menghapus"})
		return
	}
	if !deleted {
		httpx.WriteJSON(w, http.StatusNotFound,
			map[string]interface{}{"success": false, "error": "API key tidak ditemukan"})
		return
	}
	httpx.WriteJSON(w, http.StatusOK, map[string]interface{}{"success": true})
}

// generateAPIKey creates a new API key string.
func generateAPIKey() string {
	return "sfa_" + auth.GenerateRandomString(32)
}

// validateAPIKey checks if an API key is valid.
func ValidateAPIKey(keyString string) (*APIKey, error) {
	var k APIKey
	var expiresAt sql.NullString
	var isActive int
	err := db.DB.QueryRow(`
		SELECT id, admin_id, user_id, name, expires_at, created_at, is_active
		FROM api_keys WHERE key_hash = ?`, auth.HashString(keyString)).Scan(
		&k.ID, &k.AdminID, &k.UserID, &k.Name, &expiresAt, &k.CreatedAt, &isActive)
	if err != nil {
		return nil, err
	}
	if isActive != 1 {
		return nil, sql.ErrNoRows
	}
	if expiresAt.Valid {
		if t, _ := time.Parse(time.RFC3339, expiresAt.String); time.Now().After(t) {
			return nil, sql.ErrNoRows
		}
	}
	k.IsActive = true
	return &k, nil
}
