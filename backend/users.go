package main

import (
	"github.com/syzhaa/file-server/internal/db"
	"github.com/syzhaa/file-server/internal/httpx"
	"database/sql"
	"log"
	"net/http"
	"time"

	"github.com/gorilla/mux"
)

type User struct {
	ID                   string     `json:"id"`
	Email                string     `json:"email"`
	Name                 string     `json:"name"`
	AvatarURL            string     `json:"avatar_url,omitempty"`
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

// NOTE: handleUserMe, handleUserRooms, and requireUserSession are now
// implemented in user_auth.go (email/password user auth).

// Handler: List all users (admin only)
func handleAdminListUsers(w http.ResponseWriter, r *http.Request) {
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

	users := []User{}
	for rows.Next() {
		var u User
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
func handleAdminApproveUser(w http.ResponseWriter, r *http.Request) {
	admin := r.Context().Value("admin").(*AdminUser)
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
		http.Error(w, `{"error":"User not found"}`, http.StatusNotFound)
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
		"message": "User approved successfully",
	})
}

// Handler: Reject user
func handleAdminRejectUser(w http.ResponseWriter, r *http.Request) {
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
		"message": "User rejected",
	})
}

// Handler: Suspend user
func handleAdminSuspendUser(w http.ResponseWriter, r *http.Request) {
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

	httpx.WriteJSON(w, http.StatusOK, map[string]interface{}{
		"success": true,
		"message": "User suspended",
	})
}

// Handler: Get user stats
func handleAdminGetUserStats(w http.ResponseWriter, r *http.Request) {
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
func handleAdminUpdateUserQuotas(w http.ResponseWriter, r *http.Request) {
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
func handleAdminApproveUserAPI(w http.ResponseWriter, r *http.Request) {
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
func handleAdminRevokeUserAPI(w http.ResponseWriter, r *http.Request) {
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

// NOTE: requireUserSession is now implemented in user_auth.go.

// Handler: Admin system settings
func handleAdminSystemSettings(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	if r.Method == "GET" {
		rows, err := db.DB.Query(`SELECT key, value FROM system_settings`)
		if err != nil {
			httpx.WriteJSON(w, http.StatusOK, map[string]interface{}{
				"success": false,
				"error":   "Failed to fetch settings",
			})
			return
		}
		defer rows.Close()

		settings := make(map[string]string)
		for rows.Next() {
			var key, value string
			rows.Scan(&key, &value)
			settings[key] = value
		}

		httpx.WriteJSON(w, http.StatusOK, map[string]interface{}{
			"success":  true,
			"settings": settings,
		})
		return
	}

	if r.Method == "POST" {
		admin := r.Context().Value("admin").(*AdminUser)

		var req struct {
			Key   string `json:"key"`
			Value string `json:"value"`
		}
		httpx.ReadJSON(r, &req)

		now := time.Now().Format(time.RFC3339)
		_, err := db.DB.Exec(`UPDATE system_settings 
			SET value = ?, updated_at = ?
			WHERE key = ?`,
			req.Value, now, req.Key)

		if err != nil {
			httpx.WriteJSON(w, http.StatusOK, map[string]interface{}{
				"success": false,
				"error":   "Failed to update setting",
			})
			return
		}

		log.Printf("✅ Admin %s updated setting: %s", admin.Email, req.Key)

		httpx.WriteJSON(w, http.StatusOK, map[string]interface{}{
			"success": true,
			"message": "Setting updated successfully",
		})
	}
}
