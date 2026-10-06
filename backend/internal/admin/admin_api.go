package admin

import (
	"database/sql"
	"github.com/syzhaa/file-server/internal/auth"
	"github.com/syzhaa/file-server/internal/db"
	"github.com/syzhaa/file-server/internal/httpx"
	"net/http"
	"os"
	"time"

)

// UploadDir is the file upload directory.
const UploadDir = "./uploads"

// getBaseURL returns the public base URL from BASE_URL env, with fallback.
func getBaseURL() string {
	if v := os.Getenv("BASE_URL"); v != "" {
		return v
	}
	return "https://ambilfile.web.id"
}

// Handler: Create Room via API


// Handler: Get Room Link via API


// Handler: Download All Files as Zip via API


// Handler: Get Room Files List via API


// Handler: Admin Stats
func HandleAdminStats(w http.ResponseWriter, r *http.Request) {
	admin := r.Context().Value("admin").(*auth.AdminUser)

	var totalRooms, activeRooms, totalFiles int64
	var totalSize int64
	var adminRooms int64

	db.DB.QueryRow("SELECT COUNT(*) FROM rooms WHERE expires_at > ?", time.Now().Format(time.RFC3339)).Scan(&totalRooms)
	db.DB.QueryRow("SELECT COUNT(*) FROM rooms WHERE expires_at > ?", time.Now().Format(time.RFC3339)).Scan(&activeRooms)
	db.DB.QueryRow("SELECT COUNT(*), COALESCE(SUM(size), 0) FROM files").Scan(&totalFiles, &totalSize)
	db.DB.QueryRow("SELECT COUNT(*) FROM rooms WHERE COALESCE(no_quota, 0) = 1").Scan(&adminRooms)

	var apiKeyCount int64
	db.DB.QueryRow("SELECT COUNT(*) FROM api_keys WHERE admin_id = ? AND is_active = 1", admin.ID).Scan(&apiKeyCount)

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
func HandleAdminListRooms(w http.ResponseWriter, r *http.Request) {
	rows, err := db.DB.Query(`
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
