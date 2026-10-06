package admin

import (
	"archive/zip"
	"database/sql"
	"fmt"
	"github.com/syzhaa/file-server/internal/apikeys"
	"github.com/syzhaa/file-server/internal/db"
	"github.com/syzhaa/file-server/internal/httpx"
	"github.com/syzhaa/file-server/internal/rooms"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"time"

	"github.com/google/uuid"
	"github.com/gorilla/mux"
)

// apiKeyMayAccessRoom reports whether the API key in the request context may
// access the given room. Admin-owned keys (no user_id) have full access;
// user-owned keys may only access rooms they own. Anonymous rooms are never
// accessible via a user-owned key.
func apiKeyMayAccessRoom(r *http.Request, roomID string) bool {
	apiKey, ok := r.Context().Value("api_key").(*apikeys.APIKey)
	if !ok || apiKey == nil {
		return false
	}
	if apiKey.UserID == "" {
		return true // admin-owned key: full access
	}
	var owner sql.NullString
	if err := db.DB.QueryRow("SELECT user_id FROM rooms WHERE id = ?", roomID).Scan(&owner); err != nil {
		return false
	}
	return owner.Valid && owner.String != "" && owner.String == apiKey.UserID
}

func HandleAPIGetRoomFiles(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	roomID := vars["id"]

	// Ownership scope: user-owned API keys may only access their own rooms.
	if !apiKeyMayAccessRoom(r, roomID) {
		http.Error(w, `{"error":"Tidak diizinkan"}`, http.StatusForbidden)
		return
	}

	var expiresAt time.Time
	err := db.DB.QueryRow("SELECT expires_at FROM rooms WHERE id = ?", roomID).Scan(&expiresAt)

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

	rows, err := db.DB.Query(`SELECT id, original_name, mimetype, size, downloads, created_at 
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
func HandleAPIDownloadAll(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	roomID := vars["id"]

	// Ownership scope: user-owned API keys may only access their own rooms.
	if !apiKeyMayAccessRoom(r, roomID) {
		http.Error(w, `{"error":"Tidak diizinkan"}`, http.StatusForbidden)
		return
	}

	var expiresAt time.Time
	err := db.DB.QueryRow("SELECT expires_at FROM rooms WHERE id = ?", roomID).Scan(&expiresAt)

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

	rows, err := db.DB.Query("SELECT id, filename, original_name FROM files WHERE room_id = ?", roomID)
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
		db.DB.Exec("UPDATE files SET downloads = downloads + 1 WHERE id = ?", f.ID)
	}
}
func HandleAPIGetRoomLink(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	roomID := vars["id"]

	// Ownership scope: user-owned API keys may only access their own rooms.
	if !apiKeyMayAccessRoom(r, roomID) {
		http.Error(w, `{"error":"Tidak diizinkan"}`, http.StatusForbidden)
		return
	}

	var pin string
	var expiresAt time.Time
	err := db.DB.QueryRow("SELECT pin, expires_at FROM rooms WHERE id = ?", roomID).Scan(&pin, &expiresAt)

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
func HandleAPICreateRoom(w http.ResponseWriter, r *http.Request) {
	apiKey := r.Context().Value("api_key").(*apikeys.APIKey)

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
	pin, err := rooms.GeneratePin()
	if err != nil {
		http.Error(w, `{"error":"Failed to generate PIN"}`, http.StatusInternalServerError)
		return
	}

	expiresAt := time.Now().Add(time.Duration(req.ExpiryMinutes) * time.Minute)

	// Admin-owned keys (not user keys) create quota-free rooms (full access).
	// auth.User-owned keys tag the room with the user for quota tracking.
	noQuota := 0
	roomUserID := ""
	if apiKey.UserID == "" {
		noQuota = 1
	} else {
		roomUserID = apiKey.UserID
	}

	_, err = db.DB.Exec("INSERT INTO rooms (id, pin, expires_at, user_id, no_quota) VALUES (?, ?, ?, ?, ?)",
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
