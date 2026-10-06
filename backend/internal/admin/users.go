package admin

import (
	"database/sql"
	"github.com/syzhaa/file-server/internal/auth"
	"github.com/syzhaa/file-server/internal/db"
	"github.com/syzhaa/file-server/internal/httpx"
	"net/http"
	"time"

	"github.com/gorilla/mux"
)

type UserStats struct {
	UserID        string     `json:"user_id"`
	TotalRooms    int        `json:"total_rooms_created"`
	TotalFiles    int        `json:"total_files_uploaded"`
	StorageUsedMB float64    `json:"total_storage_used_mb"`
	LastUploadAt  *time.Time `json:"last_upload_at,omitempty"`
}

// NOTE: auth.HandleUserMe, handleUserRooms, and auth.RequireUserSession are now
// implemented in user_auth.go (email/password user auth).

// Handler: List all users (admin only)
func HandleAdminListUsers(w http.ResponseWriter, r *http.Request) {
	status := r.URL.Query().Get("status") // filter by status

	query := `SELECT id, email, name, status, 
		approved_by, approved_at, storage_limit_mb, max_file_duration_days, 
		created_at, last_login FROM users`

	args := []interface{}{}
	if status != "" {
		query += " WHERE status = ?"
		args = append(args, status)
	}
	query += " ORDER BY created_at DESC"

	rows, err := db.DB.Query(query, args...)
	if err != nil {
		http.Error(w, `{"error":"Failed to fetch users"}`, http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	users := []auth.User{}
	for rows.Next() {
		var u auth.User
		rows.Scan(&u.ID, &u.Email, &u.Name,
			&u.Status, &u.ApprovedBy, &u.ApprovedAt, &u.StorageLimitMB,
			&u.MaxFileDurationDays, &u.CreatedAt, &u.LastLogin)
		users = append(users, u)
	}

	httpx.WriteJSON(w, http.StatusOK, map[string]interface{}{
		"success": true,
		"users":   users,
		"count":   len(users),
	})
}

// Handler: Approve user
func HandleAdminApproveUser(w http.ResponseWriter, r *http.Request) {
	admin := r.Context().Value("admin").(*auth.AdminUser)
	vars := mux.Vars(r)
	userID := vars["id"]

	var req struct {
		StorageLimitMB      *int `json:"storage_limit_mb"`
		MaxFileDurationDays *int `json:"max_file_duration_days"`
	}
	httpx.ReadJSON(r, &req)

	// Check user exists
	var currentStatus string
	err := db.DB.QueryRow("SELECT status FROM users WHERE id = ?", userID).Scan(&currentStatus)
	if err == sql.ErrNoRows {
		http.Error(w, `{"error":"auth.User not found"}`, http.StatusNotFound)
		return
	}
	if err != nil {
		http.Error(w, `{"error":"Server error"}`, http.StatusInternalServerError)
		return
	}

	now := time.Now().Format(time.RFC3339)
	_, err = db.DB.Exec(`UPDATE users 
		SET status = 'approved', approved_by = ?, approved_at = ?, 
		    storage_limit_mb = ?, max_file_duration_days = ?
		WHERE id = ?`,
		admin.ID, now, req.StorageLimitMB, req.MaxFileDurationDays, userID)

	if err != nil {
		http.Error(w, `{"error":"Failed to approve user"}`, http.StatusInternalServerError)
		return
	}

	httpx.WriteJSON(w, http.StatusOK, map[string]interface{}{
		"success": true,
		"message": "auth.User approved successfully",
	})
}

// Handler: Reject user
func HandleAdminRejectUser(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	userID := vars["id"]

	var req struct {
		Reason string `json:"reason"`
	}
	httpx.ReadJSON(r, &req)

	if req.Reason == "" {
		req.Reason = "Registration rejected by admin"
	}

	_, err := db.DB.Exec(`UPDATE users 
		SET status = 'rejected', rejected_reason = ?
		WHERE id = ?`, req.Reason, userID)

	if err != nil {
		http.Error(w, `{"error":"Failed to reject user"}`, http.StatusInternalServerError)
		return
	}

	httpx.WriteJSON(w, http.StatusOK, map[string]interface{}{
		"success": true,
		"message": "auth.User rejected",
	})
}

// Handler: Suspend user
func HandleAdminSuspendUser(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	userID := vars["id"]

	var req struct {
		Reason string `json:"reason"`
	}
	httpx.ReadJSON(r, &req)

	_, err := db.DB.Exec(`UPDATE users 
		SET status = 'suspended', rejected_reason = ?
		WHERE id = ?`, req.Reason, userID)

	if err != nil {
		http.Error(w, `{"error":"Failed to suspend user"}`, http.StatusInternalServerError)
		return
	}

	// Delete all user sessions
	db.DB.Exec("DELETE FROM user_sessions WHERE user_id = ?", userID)

	// Deactivate all user API keys (suspended users must not retain API access)
	db.DB.Exec("UPDATE api_keys SET is_active = 0 WHERE user_id = ?", userID)

	httpx.WriteJSON(w, http.StatusOK, map[string]interface{}{
		"success": true,
		"message": "auth.User suspended",
	})
}

// Handler: Get user stats
func HandleAdminGetUserStats(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	userID := vars["id"]

	var stats UserStats
	stats.UserID = userID

	// Get total rooms created by user
	db.DB.QueryRow(`SELECT COUNT(*) FROM rooms WHERE created_by = ?`, userID).Scan(&stats.TotalRooms)

	// Get total files uploaded by user
	db.DB.QueryRow(`SELECT COUNT(*) FROM files WHERE created_by = ?`, userID).Scan(&stats.TotalFiles)

	// Get total storage used
	db.DB.QueryRow(`SELECT COALESCE(SUM(size), 0) / 1024 / 1024 FROM files WHERE created_by = ?`, userID).Scan(&stats.StorageUsedMB)

	httpx.WriteJSON(w, http.StatusOK, map[string]interface{}{
		"success": true,
		"stats":   stats,
	})
}

// Handler: Update user quotas
func HandleAdminUpdateUserQuotas(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	userID := vars["id"]

	var req struct {
		StorageLimitMB      int `json:"storage_limit_mb"`
		MaxFileDurationDays int `json:"max_file_duration_days"`
	}
	httpx.ReadJSON(r, &req)

	_, err := db.DB.Exec(`UPDATE users 
		SET storage_limit_mb = ?, max_file_duration_days = ?
		WHERE id = ?`,
		req.StorageLimitMB, req.MaxFileDurationDays, userID)

	if err != nil {
		http.Error(w, `{"error":"Failed to update quotas"}`, http.StatusInternalServerError)
		return
	}

	httpx.WriteJSON(w, http.StatusOK, map[string]interface{}{
		"success": true,
		"message": "Quotas updated successfully",
	})
}

// Handler: Approve user's API key access
func HandleAdminApproveUserAPI(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	userID := vars["id"]

	_, err := db.DB.Exec(`UPDATE users SET api_approved = 1, api_requested_at = NULL WHERE id = ?`, userID)
	if err != nil {
		http.Error(w, `{"error":"Failed to approve API access"}`, http.StatusInternalServerError)
		return
	}

	httpx.WriteJSON(w, http.StatusOK, map[string]interface{}{
		"success": true,
		"message": "Akses API key disetujui",
	})
}

// Handler: Revoke user's API key access
func HandleAdminRevokeUserAPI(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	userID := vars["id"]

	_, err := db.DB.Exec(`UPDATE users SET api_approved = 0 WHERE id = ?`, userID)
	if err != nil {
		http.Error(w, `{"error":"Failed to revoke API access"}`, http.StatusInternalServerError)
		return
	}

	httpx.WriteJSON(w, http.StatusOK, map[string]interface{}{
		"success": true,
		"message": "Akses API key dicabut",
	})
}

// NOTE: auth.RequireUserSession is now implemented in user_auth.go.

// Handler: Admin system settings
