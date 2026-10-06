package main

import (
	"crypto/rand"
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"math/big"
	"path/filepath"
	"time"

	"github.com/google/uuid"
	"github.com/gorilla/mux"
)


func generatePin() (string, error) {
	for {
		n, err := rand.Int(rand.Reader, big.NewInt(900000))
		if err != nil {
			return "", err
		}
		pin := fmt.Sprintf("%06d", n.Int64()+100000)
		
		var exists bool
		err = db.QueryRow("SELECT EXISTS(SELECT 1 FROM rooms WHERE pin = ?)", pin).Scan(&exists)
		if err != nil {
			return "", err
		}
		if !exists {
			return pin, nil
		}
	}
}


func createRoomHandler(w http.ResponseWriter, r *http.Request) {
	var req struct {
		ExpiryMinutes int `json:"expiry_minutes"`
	}
	json.NewDecoder(r.Body).Decode(&req)

	if req.ExpiryMinutes == 0 {
		req.ExpiryMinutes = 60
	}
	if req.ExpiryMinutes > 1440 {
		req.ExpiryMinutes = 1440
	}
	if req.ExpiryMinutes < 10 {
		req.ExpiryMinutes = 10
	}

	roomID := uuid.New().String()
	pin, err := generatePin()
	if err != nil {
		http.Error(w, `{"error":"Failed to generate PIN"}`, http.StatusInternalServerError)
		return
	}

	expiresAt := time.Now().Add(time.Duration(req.ExpiryMinutes) * time.Minute)

	// Tag room with logged-in user (if any) so it shows in their dashboard.
	// Rooms created by an admin bypass storage quotas (full access).
	var userID string
	noQuota := 0
	if _, err := validateAdminSession(r); err == nil {
		noQuota = 1
	} else if u, err := validateUserSession(r); err == nil && u != nil {
		userID = u.ID
	}

	_, err = db.Exec("INSERT INTO rooms (id, pin, expires_at, user_id, no_quota) VALUES (?, ?, ?, ?, ?)",
		roomID, pin, expiresAt.Format(time.RFC3339), userID, noQuota)
	if err != nil {
		http.Error(w, `{"error":"Failed to create room"}`, http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"success": true,
		"room_id": roomID,
		"pin":     pin,
	})
}


func accessRoomByPinHandler(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Pin string `json:"pin"`
	}
	json.NewDecoder(r.Body).Decode(&req)

	if req.Pin == "" {
		http.Error(w, `{"error":"PIN is empty"}`, http.StatusBadRequest)
		return
	}

	var room Room
	err := db.QueryRow("SELECT id, expires_at FROM rooms WHERE pin = ?", req.Pin).
		Scan(&room.ID, &room.ExpiresAt)
	if err == sql.ErrNoRows {
		http.Error(w, `{"error":"Invalid PIN or room expired"}`, http.StatusNotFound)
		return
	}
	if err != nil {
		http.Error(w, `{"error":"Server error"}`, http.StatusInternalServerError)
		return
	}

	if time.Now().After(room.ExpiresAt) {
		http.Error(w, `{"error":"Room expired"}`, http.StatusGone)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"success": true,
		"room_id": room.ID,
	})
}

// GET /api/stats — public statistics (total rooms created, total files uploaded)


func getRoomInfoHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	roomID := vars["id"]

	var room Room
	var permission sql.NullString
	var allowDelete sql.NullInt64
	err := db.QueryRow(`SELECT id, pin, created_at, expires_at, COALESCE(permission, 'both'), COALESCE(allow_delete, 1) FROM rooms WHERE id = ?`, roomID).
		Scan(&room.ID, &room.Pin, &room.CreatedAt, &room.ExpiresAt, &permission, &allowDelete)
	if err == sql.ErrNoRows {
		http.Error(w, `{"error":"Room not found"}`, http.StatusNotFound)
		return
	}
	if err != nil {
		http.Error(w, `{"error":"Server error"}`, http.StatusInternalServerError)
		return
	}

	if time.Now().After(room.ExpiresAt) {
		http.Error(w, `{"error":"Room expired"}`, http.StatusGone)
		return
	}

	// Get folder_id query parameter for filtering
	folderID := r.URL.Query().Get("folder_id")
	
	var rows *sql.Rows
	if folderID != "" {
		// Get files in specific folder
		rows, err = db.Query(`SELECT id, original_name, size, downloads, created_at 
			FROM files WHERE room_id = ? AND folder_id = ? ORDER BY created_at DESC`, roomID, folderID)
	} else {
		// Get files in root (no folder or NULL folder_id)
		rows, err = db.Query(`SELECT id, original_name, size, downloads, created_at 
			FROM files WHERE room_id = ? AND (folder_id IS NULL OR folder_id = '') ORDER BY created_at DESC`, roomID)
	}
	if err != nil {
		http.Error(w, `{"error":"Server error"}`, http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	files := []File{}
	for rows.Next() {
		var f File
		rows.Scan(&f.ID, &f.OriginalName, &f.Size, &f.Downloads, &f.CreatedAt)
		files = append(files, f)
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"room":        room,
		"files":       files,
		"quota_info":  getRoomQuotaInfo(roomID),
		"permission":  permission.String,
		"allow_delete": allowDelete.Int64 == 1,
	})
}

// PUT /api/room/{id}/settings — update room permission settings (owner)


func updateRoomSettingsHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	roomID := vars["id"]

	var req struct {
		Permission  string `json:"permission"`   // 'both', 'view', 'download'
		AllowDelete *bool  `json:"allow_delete"` // nil = no change
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, `{"error":"Data tidak valid"}`, http.StatusBadRequest)
		return
	}

	if req.Permission != "" && req.Permission != "both" && req.Permission != "view" && req.Permission != "download" {
		http.Error(w, `{"error":"Permission tidak valid"}`, http.StatusBadRequest)
		return
	}

	// Verify room exists
	var exists bool
	_ = db.QueryRow(`SELECT EXISTS(SELECT 1 FROM rooms WHERE id = ?)`, roomID).Scan(&exists)
	if !exists {
		http.Error(w, `{"error":"Room tidak ditemukan"}`, http.StatusNotFound)
		return
	}

	if req.Permission != "" {
		db.Exec(`UPDATE rooms SET permission = ? WHERE id = ?`, req.Permission, roomID)
	}
	if req.AllowDelete != nil {
		val := 0
		if *req.AllowDelete {
			val = 1
		}
		db.Exec(`UPDATE rooms SET allow_delete = ? WHERE id = ?`, val, roomID)
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{"success": true})
}

// getRoomQuotaInfo returns quota display info for a room


// DELETE /api/room/{id} — delete a room (owner or admin only)
func handleDeleteRoom(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	roomID := vars["id"]

	var userID sql.NullString
	var noQuota int
	err := db.QueryRow(`SELECT user_id, COALESCE(no_quota, 0) FROM rooms WHERE id = ?`, roomID).Scan(&userID, &noQuota)
	if err != nil {
		http.Error(w, `{"error":"Room tidak ditemukan"}`, http.StatusNotFound)
		return
	}

	// Authorization: admin can delete anything; user can delete their own rooms
	isAdmin := false
	if _, err := validateAdminSession(r); err == nil {
		isAdmin = true
	}
	if !isAdmin {
		u, err := validateUserSession(r)
		if err != nil || u == nil || !userID.Valid || userID.String != u.ID {
			http.Error(w, `{"error":"Tidak diizinkan"}`, http.StatusForbidden)
			return
		}
	}

	// Delete physical files
	fileRows, _ := db.Query(`SELECT stored_filename, size FROM files WHERE room_id = ?`, roomID)
	if fileRows != nil {
		for fileRows.Next() {
			var filename string
			var fsize int64
			fileRows.Scan(&filename, &fsize)
			os.Remove(filepath.Join(UploadDir, filename))
		}
		fileRows.Close()
	}

	// Delete from DB
	db.Exec(`DELETE FROM files WHERE room_id = ?`, roomID)
	db.Exec(`DELETE FROM rooms WHERE id = ?`, roomID)

	// Update stats
	db.Exec(`UPDATE system_settings SET value = CAST(value AS INTEGER) + 1 WHERE key = 'stats_deleted_rooms'`)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{"success": true})
}
