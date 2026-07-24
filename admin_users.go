package main

import (
	"database/sql"
	"encoding/json"
	"net/http"
	"time"

	"github.com/gorilla/mux"
)

// Handler: List all users (admin only)
func handleAdminListUsers(w http.ResponseWriter, r *http.Request) {
	status := r.URL.Query().Get("status") // filter by status

	query := `SELECT id, google_id, email, name, avatar_url, status, 
		approved_by, approved_at, storage_limit_mb, max_file_duration_days, 
		created_at, last_login FROM users`
	
	args := []interface{}{}
	if status != "" {
		query += " WHERE status = ?"
		args = append(args, status)
	}
	query += " ORDER BY created_at DESC"

	rows, err := db.Query(query, args...)
	if err != nil {
		http.Error(w, `{"error":"Failed to fetch users"}`, http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	users := []User{}
	for rows.Next() {
		var u User
		rows.Scan(&u.ID, &u.GoogleID, &u.Email, &u.Name, &u.AvatarURL,
			&u.Status, &u.ApprovedBy, &u.ApprovedAt, &u.StorageLimitMB,
			&u.MaxFileDurationDays, &u.CreatedAt, &u.LastLogin)
		users = append(users, u)
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
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
	json.NewDecoder(r.Body).Decode(&req)

	// Check user exists
	var currentStatus string
	err := db.QueryRow("SELECT status FROM users WHERE id = ?", userID).Scan(&currentStatus)
	if err == sql.ErrNoRows {
		http.Error(w, `{"error":"User not found"}`, http.StatusNotFound)
		return
	}
	if err != nil {
		http.Error(w, `{"error":"Server error"}`, http.StatusInternalServerError)
		return
	}

	now := time.Now().Format(time.RFC3339)
	_, err = db.Exec(`UPDATE users 
		SET status = 'approved', approved_by = ?, approved_at = ?, 
		    storage_limit_mb = ?, max_file_duration_days = ?
		WHERE id = ?`,
		admin.ID, now, req.StorageLimitMB, req.MaxFileDurationDays, userID)

	if err != nil {
		http.Error(w, `{"error":"Failed to approve user"}`, http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
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
	json.NewDecoder(r.Body).Decode(&req)

	if req.Reason == "" {
		req.Reason = "Registration rejected by admin"
	}

	_, err := db.Exec(`UPDATE users 
		SET status = 'rejected', rejected_reason = ?
		WHERE id = ?`, req.Reason, userID)

	if err != nil {
		http.Error(w, `{"error":"Failed to reject user"}`, http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
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
	json.NewDecoder(r.Body).Decode(&req)

	_, err := db.Exec(`UPDATE users 
		SET status = 'suspended', rejected_reason = ?
		WHERE id = ?`, req.Reason, userID)

	if err != nil {
		http.Error(w, `{"error":"Failed to suspend user"}`, http.StatusInternalServerError)
		return
	}

	// Delete all user sessions
	db.Exec("DELETE FROM user_sessions WHERE user_id = ?", userID)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"success": true,
		"message": "User suspended",
	})
}

// Handler: Update user quotas
func handleAdminUpdateUserQuotas(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	userID := vars["id"]

	var req struct {
		StorageLimitMB      *int `json:"storage_limit_mb"`
		MaxFileDurationDays *int `json:"max_file_duration_days"`
	}
	json.NewDecoder(r.Body).Decode(&req)

	_, err := db.Exec(`UPDATE users 
		SET storage_limit_mb = ?, max_file_duration_days = ?
		WHERE id = ?`,
		req.StorageLimitMB, req.MaxFileDurationDays, userID)

	if err != nil {
		http.Error(w, `{"error":"Failed to update quotas"}`, http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"success": true,
		"message": "Quotas updated",
	})
}

// Handler: Get user stats
func handleAdminGetUserStats(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	userID := vars["id"]

	var user User
	var stats UserStats

	err := db.QueryRow(`SELECT id, email, name, status, storage_limit_mb, 
		max_file_duration_days, created_at, last_login 
		FROM users WHERE id = ?`, userID).
		Scan(&user.ID, &user.Email, &user.Name, &user.Status,
			&user.StorageLimitMB, &user.MaxFileDurationDays,
			&user.CreatedAt, &user.LastLogin)

	if err == sql.ErrNoRows {
		http.Error(w, `{"error":"User not found"}`, http.StatusNotFound)
		return
	}
	if err != nil {
		http.Error(w, `{"error":"Server error"}`, http.StatusInternalServerError)
		return
	}

	db.QueryRow(`SELECT user_id, total_rooms_created, total_files_uploaded, 
		total_storage_used_mb, last_upload_at 
		FROM user_stats WHERE user_id = ?`, userID).
		Scan(&stats.UserID, &stats.TotalRooms, &stats.TotalFiles,
			&stats.StorageUsedMB, &stats.LastUploadAt)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"success": true,
		"user":    user,
		"stats":   stats,
	})
}

// Handler: Get/Update system settings
func handleAdminSystemSettings(w http.ResponseWriter, r *http.Request) {
	if r.Method == "GET" {
		rows, err := db.Query(`SELECT key, value, description FROM system_settings`)
		if err != nil {
			http.Error(w, `{"error":"Failed to fetch settings"}`, http.StatusInternalServerError)
			return
		}
		defer rows.Close()

		settings := make(map[string]interface{})
		for rows.Next() {
			var key, value, desc string
			rows.Scan(&key, &value, &desc)
			settings[key] = map[string]string{"value": value, "description": desc}
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
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
		json.NewDecoder(r.Body).Decode(&req)

		now := time.Now().Format(time.RFC3339)
		_, err := db.Exec(`UPDATE system_settings 
			SET value = ?, updated_at = ?, updated_by = ?
			WHERE key = ?`,
			req.Value, now, admin.ID, req.Key)

		if err != nil {
			http.Error(w, `{"error":"Failed to update setting"}`, http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"success": true,
			"message": "Setting updated",
		})
	}
}
