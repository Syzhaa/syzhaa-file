package main

import (
	"database/sql"
	"encoding/json"
	"net/http"
	"os"
	"path/filepath"

	"github.com/gorilla/mux"
)

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
