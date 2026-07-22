package main

import (
	"crypto/rand"
	"database/sql"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"math/big"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"time"

	"github.com/google/uuid"
	"github.com/gorilla/mux"
	_ "github.com/mattn/go-sqlite3"
)

const (
	Port        = 4006
	ChunkDir    = "./chunks"
	UploadDir   = "./uploads"
	DBPath      = "./data/files.db"
	MaxMemory   = 100 << 20 // 100MB untuk buffer upload
)

var db *sql.DB

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

func initDB() error {
	var err error
	os.MkdirAll("./data", 0755)
	db, err = sql.Open("sqlite3", DBPath)
	if err != nil {
		return err
	}

	schema := `
	CREATE TABLE IF NOT EXISTS rooms (
		id TEXT PRIMARY KEY,
		pin TEXT UNIQUE,
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
		expires_at DATETIME NOT NULL
	);
	
	CREATE TABLE IF NOT EXISTS files (
		id TEXT PRIMARY KEY,
		room_id TEXT NOT NULL,
		filename TEXT NOT NULL,
		original_name TEXT NOT NULL,
		mimetype TEXT NOT NULL,
		size INTEGER NOT NULL,
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
		downloads INTEGER DEFAULT 0,
		FOREIGN KEY(room_id) REFERENCES rooms(id) ON DELETE CASCADE
	);

	CREATE TABLE IF NOT EXISTS admin_users (
		id TEXT PRIMARY KEY,
		google_id TEXT UNIQUE NOT NULL,
		email TEXT UNIQUE NOT NULL,
		name TEXT NOT NULL,
		avatar_url TEXT,
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
		last_login DATETIME
	);

	CREATE TABLE IF NOT EXISTS api_keys (
		id TEXT PRIMARY KEY,
		key_hash TEXT UNIQUE NOT NULL,
		admin_id TEXT NOT NULL,
		name TEXT NOT NULL,
		expires_at DATETIME,
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
		last_used_at DATETIME,
		is_active INTEGER DEFAULT 1,
		FOREIGN KEY(admin_id) REFERENCES admin_users(id) ON DELETE CASCADE
	);

	CREATE TABLE IF NOT EXISTS admin_sessions (
		id TEXT PRIMARY KEY,
		admin_id TEXT NOT NULL,
		token_hash TEXT UNIQUE NOT NULL,
		expires_at DATETIME NOT NULL,
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
		FOREIGN KEY(admin_id) REFERENCES admin_users(id) ON DELETE CASCADE
	);
	`
	_, err = db.Exec(schema)
	if err != nil {
		return err
	}

	// Create indexes
	db.Exec("CREATE INDEX IF NOT EXISTS idx_api_keys_admin ON api_keys(admin_id)")
	db.Exec("CREATE INDEX IF NOT EXISTS idx_sessions_admin ON admin_sessions(admin_id)")
	db.Exec("CREATE INDEX IF NOT EXISTS idx_api_keys_active ON api_keys(is_active, expires_at)")

	return nil
}

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

func corsMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, DELETE, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
		if r.Method == "OPTIONS" {
			w.WriteHeader(http.StatusOK)
			return
		}
		next.ServeHTTP(w, r)
	})
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

	_, err = db.Exec("INSERT INTO rooms (id, pin, expires_at) VALUES (?, ?, ?)", 
		roomID, pin, expiresAt.Format(time.RFC3339))
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

func getRoomInfoHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	roomID := vars["id"]

	var room Room
	err := db.QueryRow("SELECT id, pin, created_at, expires_at FROM rooms WHERE id = ?", roomID).
		Scan(&room.ID, &room.Pin, &room.CreatedAt, &room.ExpiresAt)
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

	rows, err := db.Query(`SELECT id, original_name, size, downloads, created_at 
		FROM files WHERE room_id = ? ORDER BY created_at DESC`, roomID)
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
		"room":  room,
		"files": files,
	})
}

func uploadChunkHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	roomID := vars["roomId"]

	var exists bool
	err := db.QueryRow("SELECT EXISTS(SELECT 1 FROM rooms WHERE id = ?)", roomID).Scan(&exists)
	if err != nil || !exists {
		http.Error(w, `{"error":"Invalid room"}`, http.StatusNotFound)
		return
	}

	r.ParseMultipartForm(MaxMemory)
	file, _, err := r.FormFile("file")
	if err != nil {
		http.Error(w, `{"error":"No chunk received"}`, http.StatusBadRequest)
		return
	}
	defer file.Close()

	chunkIndex := r.FormValue("chunkIndex")
	totalChunks := r.FormValue("totalChunks")
	fileID := r.FormValue("fileId")
	originalName := r.FormValue("originalName")
	mimeType := r.FormValue("mimeType")
	totalSize := r.FormValue("totalSize")

	fileChunkDir := filepath.Join(ChunkDir, fileID)
	os.MkdirAll(fileChunkDir, 0755)

	chunkPath := filepath.Join(fileChunkDir, chunkIndex)
	out, err := os.Create(chunkPath)
	if err != nil {
		http.Error(w, `{"error":"Failed to save chunk"}`, http.StatusInternalServerError)
		return
	}
	defer out.Close()
	io.Copy(out, file)

	chunkIdx, _ := strconv.Atoi(chunkIndex)
	totalChunk, _ := strconv.Atoi(totalChunks)

	// Check if last chunk
	if chunkIdx == totalChunk-1 {
		ext := filepath.Ext(originalName)
		finalFileName := fileID + ext
		finalFilePath := filepath.Join(UploadDir, finalFileName)

		finalFile, err := os.Create(finalFilePath)
		if err != nil {
			http.Error(w, `{"error":"Failed to create final file"}`, http.StatusInternalServerError)
			return
		}
		defer finalFile.Close()

		// Merge all chunks
		for i := 0; i < totalChunk; i++ {
			chunkPath := filepath.Join(fileChunkDir, strconv.Itoa(i))
			chunkData, _ := os.ReadFile(chunkPath)
			finalFile.Write(chunkData)
			os.Remove(chunkPath)
		}
		os.RemoveAll(fileChunkDir)

		size, _ := strconv.ParseInt(totalSize, 10, 64)
		_, err = db.Exec(`INSERT INTO files (id, room_id, filename, original_name, mimetype, size) 
			VALUES (?, ?, ?, ?, ?, ?)`, fileID, roomID, finalFileName, originalName, mimeType, size)
		if err != nil {
			log.Printf("DB insert error: %v", err)
			http.Error(w, `{"error":"Failed to save metadata"}`, http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"success":   true,
			"id":        fileID,
			"completed": true,
		})
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"success":   true,
		"message":   fmt.Sprintf("Chunk %s received", chunkIndex),
		"completed": false,
	})
}

func downloadFileHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	fileID := vars["id"]

	var f File
	var expiresAt time.Time
	err := db.QueryRow(`SELECT f.id, f.filename, f.original_name, f.size, r.expires_at 
		FROM files f JOIN rooms r ON f.room_id = r.id WHERE f.id = ?`, fileID).
		Scan(&f.ID, &f.Filename, &f.OriginalName, &f.Size, &expiresAt)

	if err == sql.ErrNoRows {
		w.WriteHeader(http.StatusNotFound)
		fmt.Fprint(w, "<h1>File tidak ditemukan 💔</h1>")
		return
	}
	if err != nil {
		http.Error(w, "Server error", http.StatusInternalServerError)
		return
	}

	if time.Now().After(expiresAt) {
		w.WriteHeader(http.StatusGone)
		fmt.Fprint(w, "<h1>Link Kadaluarsa ⏳</h1>")
		return
	}

	filePath := filepath.Join(UploadDir, f.Filename)
	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		w.WriteHeader(http.StatusNotFound)
		fmt.Fprint(w, "<h1>File hilang 💔</h1>")
		return
	}

	db.Exec("UPDATE files SET downloads = downloads + 1 WHERE id = ?", fileID)

	w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=\"%s\"", f.OriginalName))
	w.Header().Set("Content-Type", "application/octet-stream")
	http.ServeFile(w, r, filePath)
}

func deleteFileHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	fileID := vars["id"]

	var filename string
	err := db.QueryRow("SELECT filename FROM files WHERE id = ?", fileID).Scan(&filename)
	if err == sql.ErrNoRows {
		http.Error(w, `{"error":"File not found"}`, http.StatusNotFound)
		return
	}
	if err != nil {
		http.Error(w, `{"error":"Server error"}`, http.StatusInternalServerError)
		return
	}

	filePath := filepath.Join(UploadDir, filename)
	if _, err := os.Stat(filePath); err == nil {
		os.Remove(filePath)
	}

	db.Exec("DELETE FROM files WHERE id = ?", fileID)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{"success": true})
}

func autoCleanupWorker() {
	ticker := time.NewTicker(60 * time.Second)
	defer ticker.Stop()

	for range ticker.C {
		now := time.Now().Format(time.RFC3339)
		rows, err := db.Query("SELECT id FROM rooms WHERE expires_at < ?", now)
		if err != nil {
			continue
		}

		expiredRooms := []string{}
		for rows.Next() {
			var roomID string
			rows.Scan(&roomID)
			expiredRooms = append(expiredRooms, roomID)
		}
		rows.Close()

		for _, roomID := range expiredRooms {
			fileRows, _ := db.Query("SELECT filename FROM files WHERE room_id = ?", roomID)
			for fileRows.Next() {
				var filename string
				fileRows.Scan(&filename)
				filePath := filepath.Join(UploadDir, filename)
				os.Remove(filePath)
			}
			fileRows.Close()
			db.Exec("DELETE FROM rooms WHERE id = ?", roomID)
		}

		if len(expiredRooms) > 0 {
			log.Printf("Cleaned up %d expired rooms", len(expiredRooms))
		}
	}
}

func cleanOrphanedFiles() {
	ticker := time.NewTicker(10 * time.Minute)
	defer ticker.Stop()

	for range ticker.C {
		files, err := os.ReadDir(UploadDir)
		if err != nil {
			continue
		}

		for _, file := range files {
			if file.IsDir() {
				continue
			}

			var exists bool
			db.QueryRow("SELECT EXISTS(SELECT 1 FROM files WHERE filename = ?)", file.Name()).Scan(&exists)
			if !exists {
				orphanPath := filepath.Join(UploadDir, file.Name())
				info, _ := os.Stat(orphanPath)
				if info != nil && time.Since(info.ModTime()) > 10*time.Minute {
					os.Remove(orphanPath)
					log.Printf("Removed orphaned file: %s", file.Name())
				}
			}
		}
	}
}

func main() {
	os.MkdirAll(ChunkDir, 0755)
	os.MkdirAll(UploadDir, 0755)

	if err := initDB(); err != nil {
		log.Fatal("Failed to initialize database:", err)
	}
	if err := initUserSchema(); err != nil {
		log.Fatal("Failed to initialize user schema:", err)
	}
	defer db.Close()

	// Initialize Google OAuth
	initGoogleOAuth(
		os.Getenv("GOOGLE_CLIENT_ID"),
		os.Getenv("GOOGLE_CLIENT_SECRET"),
		os.Getenv("GOOGLE_REDIRECT_URL"),
	)

	go autoCleanupWorker()
	go cleanOrphanedFiles()

	r := mux.NewRouter()
	r.Use(corsMiddleware)

	// Public routes (existing)
	r.HandleFunc("/api/room/create", createRoomHandler).Methods("POST", "OPTIONS")
	r.HandleFunc("/api/room/pin", accessRoomByPinHandler).Methods("POST", "OPTIONS")
	r.HandleFunc("/api/room/{id}", getRoomInfoHandler).Methods("GET", "OPTIONS")
	r.HandleFunc("/api/upload/{roomId}", uploadChunkHandler).Methods("POST", "OPTIONS")
	r.HandleFunc("/api/file/{id}", deleteFileHandler).Methods("DELETE", "OPTIONS")
	r.HandleFunc("/d/{id}", downloadFileHandler).Methods("GET")

	// Google OAuth routes
	r.HandleFunc("/auth/google/login", handleGoogleLogin).Methods("GET")
	r.HandleFunc("/auth/google/callback", handleGoogleCallback).Methods("GET")
	r.HandleFunc("/auth/logout", handleAdminLogout).Methods("POST", "OPTIONS")
	
	// User OAuth routes
	r.HandleFunc("/auth/user/login", handleGoogleLogin).Methods("GET")
	r.HandleFunc("/auth/user/callback", handleUserGoogleCallback).Methods("GET")
	r.HandleFunc("/auth/user/logout", handleUserLogout).Methods("POST", "OPTIONS")

	// Admin routes (require session)
	adminRouter := r.PathPrefix("/admin").Subrouter()
	adminRouter.Use(requireAdminSession)
	adminRouter.HandleFunc("/me", handleAdminMe).Methods("GET")
	adminRouter.HandleFunc("/stats", handleAdminStats).Methods("GET")
	adminRouter.HandleFunc("/api-keys", handleListAPIKeys).Methods("GET")
	adminRouter.HandleFunc("/api-keys", handleCreateAPIKey).Methods("POST")
	adminRouter.HandleFunc("/api-keys/{id}", handleDeleteAPIKey).Methods("DELETE")
	adminRouter.HandleFunc("/api-keys/{id}/toggle", handleToggleAPIKey).Methods("POST")
	
	// Admin user management routes
	adminRouter.HandleFunc("/users", handleAdminListUsers).Methods("GET")
	adminRouter.HandleFunc("/users/{id}/approve", handleAdminApproveUser).Methods("POST")
	adminRouter.HandleFunc("/users/{id}/reject", handleAdminRejectUser).Methods("POST")
	adminRouter.HandleFunc("/users/{id}/suspend", handleAdminSuspendUser).Methods("POST")
	adminRouter.HandleFunc("/users/{id}/quotas", handleAdminUpdateUserQuotas).Methods("PUT")
	adminRouter.HandleFunc("/users/{id}/stats", handleAdminGetUserStats).Methods("GET")
	adminRouter.HandleFunc("/settings", handleAdminSystemSettings).Methods("GET", "POST")

	// User routes (require user session)
	userRouter := r.PathPrefix("/user").Subrouter()
	userRouter.Use(requireUserSession)
	userRouter.HandleFunc("/me", handleUserMe).Methods("GET")

	// API v1 routes (require API key)
	apiRouter := r.PathPrefix("/api/v1").Subrouter()
	apiRouter.Use(requireAPIKey)
	apiRouter.HandleFunc("/room/create", handleAPICreateRoom).Methods("POST")
	apiRouter.HandleFunc("/room/{id}/link", handleAPIGetRoomLink).Methods("GET")
	apiRouter.HandleFunc("/room/{id}/files", handleAPIGetRoomFiles).Methods("GET")
	apiRouter.HandleFunc("/room/{id}/download-all", handleAPIDownloadAll).Methods("GET")

	r.PathPrefix("/").Handler(http.FileServer(http.Dir("./public")))

	addr := fmt.Sprintf(":%d", Port)
	log.Printf("🚀 Syzhaa File Server (Go) running on port %d", Port)
	log.Fatal(http.ListenAndServe(addr, r))
}
