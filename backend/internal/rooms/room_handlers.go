package rooms

import (
	"github.com/syzhaa/file-server/internal/auth"
	"github.com/syzhaa/file-server/internal/db"
	"github.com/syzhaa/file-server/internal/httpx"
	"crypto/rand"
	"database/sql"
	"fmt"
	"net/http"
	"os"
	"math/big"
	"path/filepath"
	"time"

	"github.com/google/uuid"
	"github.com/gorilla/mux"
)

type File struct {
	ID           string    `json:"id"`
	RoomID       string    `json:"room_id"`
	Filename     string    `json:"filename"`
	OriginalName string    `json:"original_name"`
	MimeType     string    `json:"mimetype"`
	Size         int64     `json:"size"`
	CreatedAt    time.Time `json:"created_at"`
	Downloads    int       `json:"downloads"`
}

type Room struct {
	ID        string    `json:"id"`
	Pin       string    `json:"pin"`
	CreatedAt time.Time `json:"created_at"`
	ExpiresAt time.Time `json:"expires_at"`
}


func GeneratePin() (string, error) {
	for {
		n, err := rand.Int(rand.Reader, big.NewInt(900000))
		if err != nil {
			return "", err
		}
		pin := fmt.Sprintf("%06d", n.Int64()+100000)
		
		var exists bool
		err = db.DB.QueryRow("SELECT EXISTS(SELECT 1 FROM rooms WHERE pin = ?)", pin).Scan(&exists)
		if err != nil {
			return "", err
		}
		if !exists {
			return pin, nil
		}
	}
}


func CreateRoomHandler(w http.ResponseWriter, r *http.Request) {
	var req struct {
		ExpiryMinutes int `json:"expiry_minutes"`
	}
	httpx.ReadJSON(r, &req)

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
	pin, err := GeneratePin()
	if err != nil {
		httpx.WriteError(w, http.StatusInternalServerError, "Failed to generate PIN")
		return
	}

	expiresAt := time.Now().Add(time.Duration(req.ExpiryMinutes) * time.Minute)

	// Tag room with logged-in user (if any) so it shows in their dashboard.
	// Rooms created by an admin bypass storage quotas (full access).
	var userID string
	noQuota := 0
	if _, err := auth.ValidateAdminSession(r); err == nil {
		noQuota = 1
	} else if u, err := auth.ValidateUserSession(r); err == nil && u != nil {
		userID = u.ID
	}

	_, err = db.DB.Exec("INSERT INTO rooms (id, pin, expires_at, user_id, no_quota) VALUES (?, ?, ?, ?, ?)",
		roomID, pin, expiresAt.Format(time.RFC3339), userID, noQuota)
	if err != nil {
		httpx.WriteError(w, http.StatusInternalServerError, "Failed to create room")
		return
	}

	httpx.WriteJSON(w, http.StatusOK, map[string]interface{}{
		"success": true,
		"room_id": roomID,
		"pin":     pin,
	})
}


func AccessRoomByPinHandler(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Pin string `json:"pin"`
	}
	httpx.ReadJSON(r, &req)

	if req.Pin == "" {
		httpx.WriteError(w, http.StatusBadRequest, "PIN is empty")
		return
	}

	var room Room
	err := db.DB.QueryRow("SELECT id, expires_at FROM rooms WHERE pin = ?", req.Pin).
		Scan(&room.ID, &room.ExpiresAt)
	if err == sql.ErrNoRows {
		httpx.WriteError(w, http.StatusNotFound, "Invalid PIN or room expired")
		return
	}
	if err != nil {
		httpx.WriteError(w, http.StatusInternalServerError, "Server error")
		return
	}

	if time.Now().After(room.ExpiresAt) {
		httpx.WriteError(w, http.StatusGone, "Room expired")
		return
	}

	httpx.WriteJSON(w, http.StatusOK, map[string]interface{}{
		"success": true,
		"room_id": room.ID,
	})
}

// GET /api/stats — public statistics (total rooms created, total files uploaded)


func GetRoomInfoHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	roomID := vars["id"]

	var room Room
	var permission sql.NullString
	var allowDelete sql.NullInt64
	err := db.DB.QueryRow(`SELECT id, pin, created_at, expires_at, COALESCE(permission, 'both'), COALESCE(allow_delete, 1) FROM rooms WHERE id = ?`, roomID).
		Scan(&room.ID, &room.Pin, &room.CreatedAt, &room.ExpiresAt, &permission, &allowDelete)
	if err == sql.ErrNoRows {
		httpx.WriteError(w, http.StatusNotFound, "Room not found")
		return
	}
	if err != nil {
		httpx.WriteError(w, http.StatusInternalServerError, "Server error")
		return
	}

	if time.Now().After(room.ExpiresAt) {
		httpx.WriteError(w, http.StatusGone, "Room expired")
		return
	}

	// Get folder_id query parameter for filtering
	folderID := r.URL.Query().Get("folder_id")
	
	var rows *sql.Rows
	if folderID != "" {
		// Get files in specific folder
		rows, err = db.DB.Query(`SELECT id, original_name, size, downloads, created_at 
			FROM files WHERE room_id = ? AND folder_id = ? ORDER BY created_at DESC`, roomID, folderID)
	} else {
		// Get files in root (no folder or NULL folder_id)
		rows, err = db.DB.Query(`SELECT id, original_name, size, downloads, created_at 
			FROM files WHERE room_id = ? AND (folder_id IS NULL OR folder_id = '') ORDER BY created_at DESC`, roomID)
	}
	if err != nil {
		httpx.WriteError(w, http.StatusInternalServerError, "Server error")
		return
	}
	defer rows.Close()

	files := []File{}
	for rows.Next() {
		var f File
		rows.Scan(&f.ID, &f.OriginalName, &f.Size, &f.Downloads, &f.CreatedAt)
		files = append(files, f)
	}

	httpx.WriteJSON(w, http.StatusOK, map[string]interface{}{
		"room":        room,
		"files":       files,
		"quota_info":  GetRoomQuotaInfo(roomID),
		"permission":  permission.String,
		"allow_delete": allowDelete.Int64 == 1,
	})
}

// PUT /api/room/{id}/settings — update room permission settings (owner)


func UpdateRoomSettingsHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	roomID := vars["id"]

	var req struct {
		Permission  string `json:"permission"`   // 'both', 'view', 'download'
		AllowDelete *bool  `json:"allow_delete"` // nil = no change
	}
	if err := httpx.ReadJSON(r, &req); err != nil {
		httpx.WriteError(w, http.StatusBadRequest, "Data tidak valid")
		return
	}

	if req.Permission != "" && req.Permission != "both" && req.Permission != "view" && req.Permission != "download" {
		httpx.WriteError(w, http.StatusBadRequest, "Permission tidak valid")
		return
	}

	// Verify room exists
	var exists bool
	_ = db.DB.QueryRow(`SELECT EXISTS(SELECT 1 FROM rooms WHERE id = ?)`, roomID).Scan(&exists)
	if !exists {
		httpx.WriteError(w, http.StatusNotFound, "Room tidak ditemukan")
		return
	}

	if req.Permission != "" {
		db.DB.Exec(`UPDATE rooms SET permission = ? WHERE id = ?`, req.Permission, roomID)
	}
	if req.AllowDelete != nil {
		val := 0
		if *req.AllowDelete {
			val = 1
		}
		db.DB.Exec(`UPDATE rooms SET allow_delete = ? WHERE id = ?`, val, roomID)
	}

	httpx.WriteJSON(w, http.StatusOK, map[string]interface{}{"success": true})
}

// getRoomQuotaInfo returns quota display info for a room


// DELETE /api/room/{id} — delete a room (owner or admin only)
func HandleDeleteRoom(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	roomID := vars["id"]

	var userID sql.NullString
	var noQuota int
	err := db.DB.QueryRow(`SELECT user_id, COALESCE(no_quota, 0) FROM rooms WHERE id = ?`, roomID).Scan(&userID, &noQuota)
	if err != nil {
		httpx.WriteError(w, http.StatusNotFound, "Room tidak ditemukan")
		return
	}

	// Authorization: admin can delete anything; user can delete their own rooms
	isAdmin := false
	if _, err := auth.ValidateAdminSession(r); err == nil {
		isAdmin = true
	}
	if !isAdmin {
		u, err := auth.ValidateUserSession(r)
		if err != nil || u == nil || !userID.Valid || userID.String != u.ID {
			httpx.WriteError(w, http.StatusForbidden, "Tidak diizinkan")
			return
		}
	}

	// Delete physical files
	fileRows, _ := db.DB.Query(`SELECT stored_filename, size FROM files WHERE room_id = ?`, roomID)
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
	db.DB.Exec(`DELETE FROM files WHERE room_id = ?`, roomID)
	db.DB.Exec(`DELETE FROM rooms WHERE id = ?`, roomID)

	// Update stats
	db.DB.Exec(`UPDATE system_settings SET value = CAST(value AS INTEGER) + 1 WHERE key = 'stats_deleted_rooms'`)

	httpx.WriteJSON(w, http.StatusOK, map[string]interface{}{"success": true})
}
