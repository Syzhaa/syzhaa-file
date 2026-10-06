package main

import (
	"github.com/syzhaa/file-server/internal/auth"
	"github.com/syzhaa/file-server/internal/db"
	"github.com/syzhaa/file-server/internal/httpx"
	"github.com/syzhaa/file-server/internal/middleware"
	"fmt"
	"log"
	"net/http"
	"os"
	"time"

	_ "github.com/mattn/go-sqlite3"
)

const (
	Port        = 4006
	ChunkDir    = "./chunks"
	UploadDir   = "./uploads"
	MaxMemory   = 100 << 20 // 100MB untuk buffer upload
)

type Room struct {
	ID        string    `json:"id"`
	Pin       string    `json:"pin"`
	CreatedAt time.Time `json:"created_at"`
	ExpiresAt time.Time `json:"expires_at"`
}

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

func getPublicStatsHandler(w http.ResponseWriter, r *http.Request) {
	var totalRooms, totalFiles, totalUsers int
	var totalBytes int64
	_ = db.DB.QueryRow(`SELECT COUNT(*) FROM rooms`).Scan(&totalRooms)
	_ = db.DB.QueryRow(`SELECT COUNT(*) FROM files`).Scan(&totalFiles)
	_ = db.DB.QueryRow(`SELECT COUNT(*) FROM users`).Scan(&totalUsers)
	_ = db.DB.QueryRow(`SELECT COALESCE(SUM(size), 0) FROM files`).Scan(&totalBytes)

	// Cumulative deleted stats
	var deletedRooms, deletedFiles int
	var deletedBytes int64
	_ = db.DB.QueryRow(`SELECT value FROM system_settings WHERE key = 'stats_deleted_rooms'`).Scan(&deletedRooms)
	_ = db.DB.QueryRow(`SELECT value FROM system_settings WHERE key = 'stats_deleted_files'`).Scan(&deletedFiles)
	_ = db.DB.QueryRow(`SELECT value FROM system_settings WHERE key = 'stats_deleted_bytes'`).Scan(&deletedBytes)

	httpx.WriteJSON(w, http.StatusOK, map[string]interface{}{
		"total_rooms":        totalRooms,
		"total_files":        totalFiles,
		"total_users":        totalUsers,
		"total_bytes":        totalBytes,
		"total_size_label":   formatBytes(totalBytes),
		"deleted_rooms":      deletedRooms,
		"deleted_files":      deletedFiles,
		"deleted_bytes":      deletedBytes,
		"deleted_size_label": formatBytes(deletedBytes),
	})
}


func main() {
	os.MkdirAll(ChunkDir, 0755)
	os.MkdirAll(UploadDir, 0755)

	if err := db.Init(); err != nil {
		log.Fatal("Failed to initialize database:", err)
	}
	defer db.Close()

	auth.InitGoogleOAuth()
	// Initialize admin password (use ADMIN_PASSWORD env or default)
	adminPassword := os.Getenv("ADMIN_PASSWORD")
	if adminPassword == "" {
		adminPassword = "admin"
	}
	auth.InitAdminDefaults(
		os.Getenv("ADMIN_EMAIL"),
		os.Getenv("ADMIN_NAME"),
		adminPassword,
	)

	// Initialize rate limiters
	middleware.InitRateLimiters()

	go autoCleanupWorker()
	go cleanOrphanedFiles()

	r := setupRoutes()
	addr := fmt.Sprintf(":%d", Port)
	log.Fatal(http.ListenAndServe(addr, r))
}
