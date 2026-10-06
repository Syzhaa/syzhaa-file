package main

import (
	"github.com/syzhaa/file-server/internal/httpx"
	"archive/zip"
	"database/sql"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"time"

	"github.com/google/uuid"
	"github.com/gorilla/mux"
)

// getBaseURL returns the public base URL from BASE_URL env, with fallback.
func getBaseURL() string {
	if v := os.Getenv("BASE_URL"); v != "" {
		return v
	}
	return "https://ambilfile.web.id"
}

// Handler: Create Room via API
func handleAPICreateRoom(w http.ResponseWriter, r *http.Request) {
	apiKey := r.Context().Value("api_key").(*APIKey)

	var req struct {
		ExpiryMinutes int `json:"expiry_minutes"`
	}
	httpx.ReadJSON(r, &req)

	if req.ExpiryMinutes == 0 {
		req.ExpiryMinutes = 60
	}
	if req.ExpiryMinutes > 10080 { // 7 days max
		req.ExpiryMinutes = 10080
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

	// Admin-owned keys (not user keys) create quota-free rooms (full access).
	// User-owned keys tag the room with the user for quota tracking.
	noQuota := 0
	roomUserID := ""
	if apiKey.UserID == "" {
		noQuota = 1
	} else {
		roomUserID = apiKey.UserID
	}

	_, err = db.Exec("INSERT INTO rooms (id, pin, expires_at, user_id, no_quota) VALUES (?, ?, ?, ?, ?)",
		roomID, pin, expiresAt.Format(time.RFC3339), roomUserID, noQuota)
	if err != nil {
		http.Error(w, `{"error":"Failed to create room"}`, http.StatusInternalServerError)
		return
	}

	httpx.WriteJSON(w, http.StatusOK, map[string]interface{}{
		"success":        true,
		"room_id":        roomID,
		"pin":            pin,
		"expires_at":     expiresAt.Format(time.RFC3339),
		"expiry_minutes": req.ExpiryMinutes,
		"api_key_id":     apiKey.ID,
	})
}

// Handler: Get Room Link via API
func handleAPIGetRoomLink(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	roomID := vars["id"]

	var pin string
	var expiresAt time.Time
	err := db.QueryRow("SELECT pin, expires_at FROM rooms WHERE id = ?", roomID).Scan(&pin, &expiresAt)

	if err == sql.ErrNoRows {
		http.Error(w, `{"error":"Room not found"}`, http.StatusNotFound)
		return
	}
	if err != nil {
		http.Error(w, `{"error":"Server error"}`, http.StatusInternalServerError)
		return
	}

	if time.Now().After(expiresAt) {
		http.Error(w, `{"error":"Room expired"}`, http.StatusGone)
		return
	}

	// Generate shareable links
	baseURL := getBaseURL()
	roomLink := fmt.Sprintf("%s/?room=%s", baseURL, roomID)
	pinLink := fmt.Sprintf("%s/?pin=%s", baseURL, pin)

	httpx.WriteJSON(w, http.StatusOK, map[string]interface{}{
		"success":    true,
		"room_id":    roomID,
		"pin":        pin,
		"room_link":  roomLink,
		"pin_link":   pinLink,
		"expires_at": expiresAt.Format(time.RFC3339),
	})
}

// Handler: Download All Files as Zip via API
func handleAPIDownloadAll(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	roomID := vars["id"]

	var expiresAt time.Time
	err := db.QueryRow("SELECT expires_at FROM rooms WHERE id = ?", roomID).Scan(&expiresAt)

	if err == sql.ErrNoRows {
		http.Error(w, `{"error":"Room not found"}`, http.StatusNotFound)
		return
	}
	if err != nil {
		http.Error(w, `{"error":"Server error"}`, http.StatusInternalServerError)
		return
	}

	if time.Now().After(expiresAt) {
		http.Error(w, `{"error":"Room expired"}`, http.StatusGone)
		return
	}

	rows, err := db.Query("SELECT id, filename, original_name FROM files WHERE room_id = ?", roomID)
	if err != nil {
		http.Error(w, `{"error":"Failed to fetch files"}`, http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	type fileInfo struct {
		ID           string
		Filename     string
		OriginalName string
	}

	files := []fileInfo{}
	for rows.Next() {
		var f fileInfo
		rows.Scan(&f.ID, &f.Filename, &f.OriginalName)
		files = append(files, f)
	}

	if len(files) == 0 {
		http.Error(w, `{"error":"No files in room"}`, http.StatusNotFound)
		return
	}

	// Stream zip
	zipFilename := fmt.Sprintf("ambilfile_files_%s.zip", roomID[:8])
	w.Header().Set("Content-Type", "application/zip")
	w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=\"%s\"", zipFilename))

	zipWriter := zip.NewWriter(w)
	defer zipWriter.Close()

	for _, f := range files {
		filePath := filepath.Join(UploadDir, f.Filename)
		if _, err := os.Stat(filePath); os.IsNotExist(err) {
			continue
		}

		fileToZip, err := os.Open(filePath)
		if err != nil {
			continue
		}

		zipFileWriter, err := zipWriter.Create(f.OriginalName)
		if err != nil {
			fileToZip.Close()
			continue
		}

		io.Copy(zipFileWriter, fileToZip)
		fileToZip.Close()

		// Update download counter
		db.Exec("UPDATE files SET downloads = downloads + 1 WHERE id = ?", f.ID)
	}
}

// Handler: Get Room Files List via API
func handleAPIGetRoomFiles(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	roomID := vars["id"]

	var expiresAt time.Time
	err := db.QueryRow("SELECT expires_at FROM rooms WHERE id = ?", roomID).Scan(&expiresAt)

	if err == sql.ErrNoRows {
		http.Error(w, `{"error":"Room not found"}`, http.StatusNotFound)
		return
	}
	if err != nil {
		http.Error(w, `{"error":"Server error"}`, http.StatusInternalServerError)
		return
	}

	if time.Now().After(expiresAt) {
		http.Error(w, `{"error":"Room expired"}`, http.StatusGone)
		return
	}

	rows, err := db.Query(`SELECT id, original_name, mimetype, size, downloads, created_at 
		FROM files WHERE room_id = ? ORDER BY created_at DESC`, roomID)
	if err != nil {
		http.Error(w, `{"error":"Failed to fetch files"}`, http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	type fileEntry struct {
		ID           string    `json:"id"`
		OriginalName string    `json:"original_name"`
		MimeType     string    `json:"mimetype"`
		Size         int64     `json:"size"`
		Downloads    int       `json:"downloads"`
		CreatedAt    time.Time `json:"created_at"`
		DownloadURL  string    `json:"download_url"`
	}

	files := []fileEntry{}
	for rows.Next() {
		var f fileEntry
		rows.Scan(&f.ID, &f.OriginalName, &f.MimeType, &f.Size, &f.Downloads, &f.CreatedAt)
		f.DownloadURL = fmt.Sprintf("%s/d/%s", getBaseURL(), f.ID)
		files = append(files, f)
	}

	httpx.WriteJSON(w, http.StatusOK, map[string]interface{}{
		"success":    true,
		"room_id":    roomID,
		"expires_at": expiresAt.Format(time.RFC3339),
		"files":      files,
		"count":      len(files),
	})
}

// Handler: Admin Stats
func handleAdminStats(w http.ResponseWriter, r *http.Request) {
	admin := r.Context().Value("admin").(*AdminUser)

	var totalRooms, activeRooms, totalFiles int64
	var totalSize int64
	var adminRooms int64

	db.QueryRow("SELECT COUNT(*) FROM rooms WHERE expires_at > ?", time.Now().Format(time.RFC3339)).Scan(&totalRooms)
	db.QueryRow("SELECT COUNT(*) FROM rooms WHERE expires_at > ?", time.Now().Format(time.RFC3339)).Scan(&activeRooms)
	db.QueryRow("SELECT COUNT(*), COALESCE(SUM(size), 0) FROM files").Scan(&totalFiles, &totalSize)
	db.QueryRow("SELECT COUNT(*) FROM rooms WHERE COALESCE(no_quota, 0) = 1").Scan(&adminRooms)

	var apiKeyCount int64
	db.QueryRow("SELECT COUNT(*) FROM api_keys WHERE admin_id = ? AND is_active = 1", admin.ID).Scan(&apiKeyCount)

	httpx.WriteJSON(w, http.StatusOK, map[string]interface{}{
		"success": true,
		"stats": map[string]interface{}{
			"total_rooms":     totalRooms,
			"active_rooms":    activeRooms,
			"admin_rooms":     adminRooms,
			"total_files":     totalFiles,
			"total_size":      totalSize,
			"api_keys_active": apiKeyCount,
		},
	})
}

// GET /admin/rooms — list all rooms (admin)
func handleAdminListRooms(w http.ResponseWriter, r *http.Request) {
	rows, err := db.Query(`
		SELECT r.id, r.pin, r.created_at, r.expires_at, COALESCE(r.no_quota, 0),
		       (SELECT COUNT(*) FROM files f WHERE f.room_id = r.id) as file_count,
		       u.email as owner_email
		FROM rooms r LEFT JOIN users u ON r.user_id = u.id
		ORDER BY r.created_at DESC LIMIT 100
	`)
	if err != nil {
		httpx.WriteJSON(w, http.StatusOK, map[string]interface{}{"success": true, "rooms": []interface{}{}})
		return
	}
	defer rows.Close()

	rooms := []map[string]interface{}{}
	for rows.Next() {
		var id, pin, createdAt, expiresAt string
		var noQuota, fileCount int
		var ownerEmail sql.NullString
		rows.Scan(&id, &pin, &createdAt, &expiresAt, &noQuota, &fileCount, &ownerEmail)
		rooms = append(rooms, map[string]interface{}{
			"id":          id,
			"pin":         pin,
			"created_at":  createdAt,
			"expires_at":  expiresAt,
			"is_admin":    noQuota == 1,
			"file_count":  fileCount,
			"owner_email": ownerEmail.String,
		})
	}
	httpx.WriteJSON(w, http.StatusOK, map[string]interface{}{"success": true, "rooms": rooms})
}
