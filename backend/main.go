package main

import (
	"crypto/rand"
	"database/sql"
	"encoding/base64"
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

	// Enable foreign key constraints
	_, err = db.Exec("PRAGMA foreign_keys = ON")
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
		folder_id TEXT,
		filename TEXT NOT NULL,
		original_name TEXT NOT NULL,
		mimetype TEXT NOT NULL,
		size INTEGER NOT NULL,
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
		downloads INTEGER DEFAULT 0,
		salt TEXT,
		nonce TEXT,
		is_encrypted INTEGER DEFAULT 0,
		original_size INTEGER,
		FOREIGN KEY(room_id) REFERENCES rooms(id) ON DELETE CASCADE,
		FOREIGN KEY(folder_id) REFERENCES folders(id) ON DELETE CASCADE
	);

	CREATE TABLE IF NOT EXISTS folders (
		id TEXT PRIMARY KEY,
		room_id TEXT NOT NULL,
		parent_id TEXT,
		name TEXT NOT NULL,
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
		FOREIGN KEY(room_id) REFERENCES rooms(id) ON DELETE CASCADE,
		FOREIGN KEY(parent_id) REFERENCES folders(id) ON DELETE CASCADE
	);

	CREATE TABLE IF NOT EXISTS admin_users (
		id TEXT PRIMARY KEY,
		email TEXT UNIQUE NOT NULL,
		name TEXT NOT NULL,
		password_hash TEXT NOT NULL,
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
		last_login DATETIME,
		is_super_admin INTEGER DEFAULT 1
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
		origin := r.Header.Get("Origin")
		allowedOrigins := []string{
			"https://file.syzhaa.my.id",
			"http://localhost:3000",
			"http://localhost:4006",
		}
		// Also allow the configured public base URL (for the user's own domain).
		if baseURL := os.Getenv("BASE_URL"); baseURL != "" {
			allowedOrigins = append(allowedOrigins, baseURL)
		}
		
		// Check if origin is allowed
		isAllowed := false
		for _, allowed := range allowedOrigins {
			if origin == allowed {
				isAllowed = true
				break
			}
		}
		
		if isAllowed {
			w.Header().Set("Access-Control-Allow-Origin", origin)
			w.Header().Set("Access-Control-Allow-Credentials", "true")
		}
		
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, DELETE, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, X-CSRF-Token, X-Session-ID")
		
		if r.Method == "OPTIONS" {
			w.WriteHeader(http.StatusOK)
			return
		}
		next.ServeHTTP(w, r)
	})
}

// Security Headers Middleware
func securityHeadersMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Security Headers
		w.Header().Set("X-Content-Type-Options", "nosniff")
		w.Header().Set("X-Frame-Options", "DENY")
		w.Header().Set("X-XSS-Protection", "1; mode=block")
		w.Header().Set("Referrer-Policy", "strict-origin-when-cross-origin")
		w.Header().Set("Permissions-Policy", "geolocation=(), microphone=(), camera=()")
		w.Header().Set("Strict-Transport-Security", "max-age=31536000; includeSubDomains; preload")
		
		// Content Security Policy
		csp := "default-src 'self'; " +
			"script-src 'self' 'unsafe-inline' https://accounts.google.com https://apis.google.com https://cdnjs.cloudflare.com https://static.cloudflareinsights.com; " +
			"style-src 'self' 'unsafe-inline' https://fonts.googleapis.com; " +
			"font-src 'self' https://fonts.gstatic.com; " +
			"img-src 'self' data: https:; " +
			"connect-src 'self'; " +
			"frame-src https://accounts.google.com; " +
			"upgrade-insecure-requests;"
		w.Header().Set("Content-Security-Policy", csp)
		
		next.ServeHTTP(w, r)
	})
}

func cleanURLMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		path := r.URL.Path
		
		if filepath.Ext(path) == "" && path != "/" {
			filePath := filepath.Join("../frontend", path)
			
			htmlPath := filePath + ".html"
			if _, err := os.Stat(htmlPath); err == nil {
				http.ServeFile(w, r, htmlPath)
				return
			}
			
			indexPath := filepath.Join(filePath, "index.html")
			if _, err := os.Stat(indexPath); err == nil {
				http.ServeFile(w, r, indexPath)
				return
			}
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
		"room":       room,
		"files":      files,
		"quota_info": getRoomQuotaInfo(roomID),
	})
}

// getRoomQuotaInfo returns quota display info for a room
func getRoomQuotaInfo(roomID string) map[string]interface{} {
	var noQuota int
	var userID sql.NullString
	_ = db.QueryRow(`SELECT COALESCE(no_quota, 0), user_id FROM rooms WHERE id = ?`, roomID).Scan(&noQuota, &userID)

	if noQuota == 1 {
		return map[string]interface{}{"label": "Tanpa batas", "unlimited": true}
	}
	if userID.Valid && userID.String != "" {
		var lim sql.NullInt64
		_ = db.QueryRow(`SELECT storage_limit_mb FROM users WHERE id = ?`, userID.String).Scan(&lim)
		mb := lim.Int64
		if !lim.Valid || mb <= 0 {
			mb = 2048
		}
		return map[string]interface{}{"label": formatBytesID(mb * 1024 * 1024), "unlimited": false}
	}
	var anonMB int64 = 1024
	_ = db.QueryRow(`SELECT value FROM system_settings WHERE key = 'anonymous_storage_limit_mb'`).Scan(&anonMB)
	if anonMB <= 0 {
		anonMB = 1024
	}
	return map[string]interface{}{"label": formatBytesID(anonMB * 1024 * 1024), "unlimited": false}
}

// checkStorageQuota verifies the incoming file fits within the applicable
// storage limit. Returns "" if OK, otherwise an Indonesian error message.
//   - Room owned by a logged-in user: user's storage_limit_mb (default 2GB),
//     usage counted across all of the user's rooms.
//   - Anonymous room: anonymous_storage_limit_mb (default 1GB) per room.
func checkStorageQuota(roomID string, incomingBytes int64) string {
	// Admin-created rooms have no quota limit (full access)
	var noQuota int
	_ = db.QueryRow(`SELECT COALESCE(no_quota, 0) FROM rooms WHERE id = ?`, roomID).Scan(&noQuota)
	if noQuota == 1 {
		return ""
	}

	var userID sql.NullString
	_ = db.QueryRow(`SELECT user_id FROM rooms WHERE id = ?`, roomID).Scan(&userID)

	var limitMB int64
	var usedBytes int64

	if userID.Valid && userID.String != "" {
		var lim sql.NullInt64
		_ = db.QueryRow(`SELECT storage_limit_mb FROM users WHERE id = ?`, userID.String).Scan(&lim)
		limitMB = lim.Int64
		if !lim.Valid || limitMB <= 0 {
			_ = db.QueryRow(`SELECT value FROM system_settings WHERE key = 'default_storage_limit_mb'`).Scan(&limitMB)
			if limitMB <= 0 {
				limitMB = 2048
			}
		}
		_ = db.QueryRow(`
			SELECT COALESCE(SUM(f.size), 0) FROM files f
			JOIN rooms r ON r.id = f.room_id
			WHERE r.user_id = ?`, userID.String).Scan(&usedBytes)
	} else {
		_ = db.QueryRow(`SELECT value FROM system_settings WHERE key = 'anonymous_storage_limit_mb'`).Scan(&limitMB)
		if limitMB <= 0 {
			limitMB = 1024
		}
		_ = db.QueryRow(`SELECT COALESCE(SUM(size), 0) FROM files WHERE room_id = ?`, roomID).Scan(&usedBytes)
	}

	limitBytes := limitMB * 1024 * 1024
	if usedBytes+incomingBytes > limitBytes {
		limitLabel := formatBytesID(limitBytes)
		return "Batas penyimpanan terlampaui (maks " + limitLabel + "). Hapus file lama atau hubungi admin."
	}
	return ""
}

func formatBytesID(b int64) string {
	const gb = 1024 * 1024 * 1024
	const mb = 1024 * 1024
	if b >= gb {
		return strconv.FormatFloat(float64(b)/float64(gb), 'f', 1, 64) + " GB"
	}
	return strconv.FormatInt(b/mb, 10) + " MB"
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
		// Enforce storage quota before assembling the file
		totalBytes, _ := strconv.ParseInt(totalSize, 10, 64)
		if qErr := checkStorageQuota(roomID, totalBytes); qErr != "" {
			os.RemoveAll(fileChunkDir)
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusRequestEntityTooLarge)
			json.NewEncoder(w).Encode(map[string]interface{}{"success": false, "error": qErr})
			return
		}

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
		
		// Check if encryption is requested
		encryptPassphrase := r.FormValue("encryptPassphrase")
		folderID := r.FormValue("folderId")
		var saltB64, nonceB64 string
		isEncrypted := 0
		originalSize := size

		if encryptPassphrase != "" {
			// Encrypt the file
			meta, err := EncryptFileInPlace(finalFileName, encryptPassphrase)
			if err != nil {
				log.Printf("❌ Encryption failed: %v", err)
				http.Error(w, `{"error":"Encryption failed"}`, http.StatusInternalServerError)
				return
			}
			saltB64 = base64.StdEncoding.EncodeToString(meta.Salt)
			nonceB64 = base64.StdEncoding.EncodeToString(meta.Nonce)
			isEncrypted = 1
			size = meta.EncryptedSize
			log.Printf("🔒 File encrypted: %s (original: %d bytes, encrypted: %d bytes)", finalFileName, originalSize, size)
		}

		_, err = db.Exec(`INSERT INTO files (id, room_id, folder_id, filename, original_name, mimetype, size, salt, nonce, is_encrypted, original_size) 
			VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`, 
			fileID, roomID, sql.NullString{String: folderID, Valid: folderID != ""}, finalFileName, originalName, mimeType, size, saltB64, nonceB64, isEncrypted, originalSize)
		if err != nil {
			log.Printf("❌ DB insert error: %v", err)
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
	var saltB64, nonceB64 sql.NullString
	var isEncrypted sql.NullInt64
	var originalSize sql.NullInt64
	
	var permission string
	err := db.QueryRow(`SELECT f.id, f.filename, f.original_name, f.size, r.expires_at, 
		f.salt, f.nonce, f.is_encrypted, f.original_size, COALESCE(r.permission, 'both')
		FROM files f JOIN rooms r ON f.room_id = r.id WHERE f.id = ?`, fileID).
		Scan(&f.ID, &f.Filename, &f.OriginalName, &f.Size, &expiresAt, 
			&saltB64, &nonceB64, &isEncrypted, &originalSize, &permission)

	if err == sql.ErrNoRows {
		w.WriteHeader(http.StatusNotFound)
		fmt.Fprint(w, "<h1>File tidak ditemukan 💔</h1>")
		return
	}
	if err != nil {
		log.Printf("❌ Download query error: %v", err)
		http.Error(w, "Server error", http.StatusInternalServerError)
		return
	}

	// Enforce room permission: 'view' only blocks downloads
	if permission == "view" {
		w.WriteHeader(http.StatusForbidden)
		fmt.Fprint(w, "<h1>Download dinonaktifkan</h1><p>Pemilik room hanya mengizinkan melihat file.</p>")
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

	// Check if file is encrypted
	if isEncrypted.Valid && isEncrypted.Int64 == 1 {
		// File is encrypted, need passphrase to decrypt
		passphrase := r.URL.Query().Get("passphrase")
		if passphrase == "" {
			passphrase = r.Header.Get("X-Decrypt-Passphrase")
		}
		
		if passphrase == "" {
			w.WriteHeader(http.StatusForbidden)
			fmt.Fprint(w, "<h1>🔒 File Terenkripsi</h1><p>Passphrase diperlukan untuk download</p>")
			return
		}

		// Decrypt file
		salt, _ := base64.StdEncoding.DecodeString(saltB64.String)
		nonce, _ := base64.StdEncoding.DecodeString(nonceB64.String)
		
		metadata := &EncryptedFileMetadata{
			Salt:        salt,
			Nonce:       nonce,
			IsEncrypted: true,
			OriginalSize: originalSize.Int64,
			EncryptedSize: f.Size,
		}

		decryptedData, err := DecryptFile(filePath, passphrase, metadata)
		if err != nil {
			log.Printf("❌ Decryption failed for file %s: %v", fileID, err)
			w.WriteHeader(http.StatusForbidden)
			fmt.Fprint(w, "<h1>🔒 Dekripsi Gagal</h1><p>Passphrase salah atau file corrupt</p>")
			return
		}

		log.Printf("🔓 File decrypted successfully: %s (%d bytes)", f.Filename, len(decryptedData))
		
		w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=\"%s\"", f.OriginalName))
		w.Header().Set("Content-Type", "application/octet-stream")
		w.Header().Set("Content-Length", fmt.Sprintf("%d", len(decryptedData)))
		w.Write(decryptedData)
		return
	}

	// File not encrypted, serve directly
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

	// Delete physical file from disk
	filePath := filepath.Join(UploadDir, filename)
	if _, err := os.Stat(filePath); err == nil {
		if err := os.Remove(filePath); err != nil {
			log.Printf("⚠️  Failed to delete file from disk: %s (error: %v)", filename, err)
			// Continue anyway to remove from DB
		} else {
			log.Printf("🗑️  Deleted file from disk: %s", filename)
		}
	} else {
		log.Printf("⚠️  File not found on disk (already deleted?): %s", filename)
	}

	// Delete from database
	if _, err := db.Exec("DELETE FROM files WHERE id = ?", fileID); err != nil {
		log.Printf("❌ Failed to delete file from database: %v", err)
		http.Error(w, `{"error":"Failed to delete file from database"}`, http.StatusInternalServerError)
		return
	}

	log.Printf("✅ File deleted successfully: %s (ID: %s)", filename, fileID)
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{"success": true})
}

func createFolderHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	roomID := vars["roomId"]
	
	// Verify room exists
	var exists bool
	err := db.QueryRow("SELECT EXISTS(SELECT 1 FROM rooms WHERE id = ?)", roomID).Scan(&exists)
	if err != nil || !exists {
		http.Error(w, `{"error":"Invalid room"}`, http.StatusNotFound)
		return
	}
	
	// Parse request
	var req struct {
		Name     string `json:"name"`
		ParentID string `json:"parent_id"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, `{"error":"Invalid request"}`, http.StatusBadRequest)
		return
	}
	
	// Validate folder name
	if req.Name == "" || len(req.Name) > 255 {
		http.Error(w, `{"error":"Invalid folder name"}`, http.StatusBadRequest)
		return
	}
	
	// Generate folder ID
	folderID := uuid.New().String()
	
	// Insert folder
	_, err = db.Exec(`INSERT INTO folders (id, room_id, parent_id, name) VALUES (?, ?, ?, ?)`,
		folderID, roomID, sql.NullString{String: req.ParentID, Valid: req.ParentID != ""}, req.Name)
	if err != nil {
		log.Printf("❌ Failed to create folder: %v", err)
		http.Error(w, `{"error":"Failed to create folder"}`, http.StatusInternalServerError)
		return
	}
	
	log.Printf("📁 Folder created: %s in room %s", req.Name, roomID)
	
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"success": true,
		"folder": map[string]interface{}{
			"id":        folderID,
			"name":      req.Name,
			"parent_id": req.ParentID,
		},
	})
}

func listFoldersHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	roomID := vars["roomId"]
	parentID := r.URL.Query().Get("parent_id")
	
	// Build query based on parent_id
	var rows *sql.Rows
	var err error
	
	if parentID == "" {
		// Get root folders (no parent)
		rows, err = db.Query(`SELECT id, name, created_at FROM folders 
			WHERE room_id = ? AND parent_id IS NULL 
			ORDER BY created_at DESC`, roomID)
	} else {
		// Get folders in specific parent
		rows, err = db.Query(`SELECT id, name, created_at FROM folders 
			WHERE room_id = ? AND parent_id = ? 
			ORDER BY created_at DESC`, roomID, parentID)
	}
	
	if err != nil {
		log.Printf("❌ Failed to list folders: %v", err)
		http.Error(w, `{"error":"Failed to list folders"}`, http.StatusInternalServerError)
		return
	}
	defer rows.Close()
	
	folders := []map[string]interface{}{}
	for rows.Next() {
		var id, name, createdAt string
		if err := rows.Scan(&id, &name, &createdAt); err != nil {
			continue
		}
		folders = append(folders, map[string]interface{}{
			"id":         id,
			"name":       name,
			"created_at": createdAt,
		})
	}
	
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"success": true,
		"folders": folders,
	})
}

func deleteFolderHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	folderID := vars["id"]
	
	// Verify folder exists
	var name string
	err := db.QueryRow("SELECT name FROM folders WHERE id = ?", folderID).Scan(&name)
	if err == sql.ErrNoRows {
		http.Error(w, `{"error":"Folder not found"}`, http.StatusNotFound)
		return
	}
	if err != nil {
		http.Error(w, `{"error":"Server error"}`, http.StatusInternalServerError)
		return
	}
	
	// Delete folder (CASCADE will delete child folders and files)
	_, err = db.Exec("DELETE FROM folders WHERE id = ?", folderID)
	if err != nil {
		log.Printf("❌ Failed to delete folder: %v", err)
		http.Error(w, `{"error":"Failed to delete folder"}`, http.StatusInternalServerError)
		return
	}
	
	log.Printf("🗑️  Folder deleted: %s (ID: %s)", name, folderID)
	
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"success": true,
	})
}

func autoCleanupWorker() {
	ticker := time.NewTicker(60 * time.Second)
	defer ticker.Stop()

	for range ticker.C {
		now := time.Now().Format(time.RFC3339)
		rows, err := db.Query("SELECT id FROM rooms WHERE expires_at < ?", now)
		if err != nil {
			log.Printf("⚠️  Auto-cleanup: Failed to query expired rooms: %v", err)
			continue
		}

		expiredRooms := []string{}
		for rows.Next() {
			var roomID string
			rows.Scan(&roomID)
			expiredRooms = append(expiredRooms, roomID)
		}
		rows.Close()

		if len(expiredRooms) == 0 {
			continue
		}

		log.Printf("🧹 Auto-cleanup: Found %d expired rooms", len(expiredRooms))
		
		totalFilesDeleted := 0
		totalFilesFailed := 0

		for _, roomID := range expiredRooms {
			fileRows, err := db.Query("SELECT filename FROM files WHERE room_id = ?", roomID)
			if err != nil {
				log.Printf("⚠️  Auto-cleanup: Failed to query files for room %s: %v", roomID, err)
				continue
			}

			for fileRows.Next() {
				var filename string
				fileRows.Scan(&filename)
				filePath := filepath.Join(UploadDir, filename)
				
				if err := os.Remove(filePath); err != nil {
					if !os.IsNotExist(err) {
						log.Printf("⚠️  Auto-cleanup: Failed to delete file %s: %v", filename, err)
						totalFilesFailed++
					}
				} else {
					log.Printf("🗑️  Auto-cleanup: Deleted file %s", filename)
					totalFilesDeleted++
				}
			}
			fileRows.Close()

			// Delete files from database first
			if _, err := db.Exec("DELETE FROM files WHERE room_id = ?", roomID); err != nil {
				log.Printf("⚠️  Auto-cleanup: Failed to delete files from DB for room %s: %v", roomID, err)
			}
			
			// Then delete the room
			if _, err := db.Exec("DELETE FROM rooms WHERE id = ?", roomID); err != nil {
				log.Printf("⚠️  Auto-cleanup: Failed to delete room %s: %v", roomID, err)
			} else {
				log.Printf("🧹 Auto-cleanup: Deleted room %s", roomID)
			}
		}

		log.Printf("✅ Auto-cleanup complete: %d rooms, %d files deleted, %d files failed", 
			len(expiredRooms), totalFilesDeleted, totalFilesFailed)
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
	if err := initUserAuthSchema(); err != nil {
		log.Fatal("Failed to initialize user auth schema:", err)
	}
	defer db.Close()

	initGoogleOAuth()
	// Initialize admin password (use ADMIN_PASSWORD env or default)
	adminPassword := os.Getenv("ADMIN_PASSWORD")
	if adminPassword == "" {
		adminPassword = "admin"
	}
	initAdminDefaults(
		os.Getenv("ADMIN_EMAIL"),
		os.Getenv("ADMIN_NAME"),
		adminPassword,
	)

	// Initialize rate limiters
	initRateLimiters()

	go autoCleanupWorker()
	go cleanOrphanedFiles()

	r := mux.NewRouter()
	r.Use(corsMiddleware)
	r.Use(securityHeadersMiddleware)
	// r.Use(csrfMiddleware) // Temporarily disabled - TODO: Implement proper CSRF token flow in frontend
	r.Use(rateLimitMiddleware(generalLimiter))

	// Public routes (existing)
	r.HandleFunc("/api/room/create", createRoomHandler).Methods("POST", "OPTIONS")
	r.HandleFunc("/api/room/pin", accessRoomByPinHandler).Methods("POST", "OPTIONS")
	r.HandleFunc("/api/room/{id}", getRoomInfoHandler).Methods("GET", "OPTIONS")
	r.HandleFunc("/api/upload/{roomId}", uploadChunkHandler).Methods("POST", "OPTIONS")
	r.HandleFunc("/api/file/{id}", deleteFileHandler).Methods("DELETE", "OPTIONS")
	r.HandleFunc("/api/folder/create/{roomId}", createFolderHandler).Methods("POST", "OPTIONS")
	r.HandleFunc("/api/folders/{roomId}", listFoldersHandler).Methods("GET", "OPTIONS")
	r.HandleFunc("/api/folder/{id}", deleteFolderHandler).Methods("DELETE", "OPTIONS")
	r.HandleFunc("/d/{id}", downloadFileHandler).Methods("GET")

	// Admin email/password auth routes
	r.HandleFunc("/auth/login", handleAdminLogin).Methods("POST", "OPTIONS")
	r.HandleFunc("/auth/logout", handleAdminLogout).Methods("POST", "OPTIONS")

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
	adminRouter.HandleFunc("/users/{id}/api-approve", handleAdminApproveUserAPI).Methods("PUT")
	adminRouter.HandleFunc("/users/{id}/api-revoke", handleAdminRevokeUserAPI).Methods("PUT")
	adminRouter.HandleFunc("/users/{id}/stats", handleAdminGetUserStats).Methods("GET")
	adminRouter.HandleFunc("/settings", handleAdminSystemSettings).Methods("GET", "POST")
	adminRouter.HandleFunc("/account", handleAdminUpdateAccount).Methods("POST", "PUT")

	// User routes (require user session)
	userRouter := r.PathPrefix("/user").Subrouter()
	userRouter.Use(requireUserSession)
	userRouter.HandleFunc("/me", handleUserMe).Methods("GET")
	userRouter.HandleFunc("/rooms", handleUserRooms).Methods("GET")
	userRouter.HandleFunc("/api-keys", handleUserListAPIKeys).Methods("GET")
	userRouter.HandleFunc("/api-keys", handleUserCreateAPIKey).Methods("POST")
	userRouter.HandleFunc("/api-keys/request", handleUserRequestAPIAccess).Methods("POST")
	userRouter.HandleFunc("/api-keys/{id}", handleUserDeleteAPIKey).Methods("DELETE")

	// User auth (public)
	r.HandleFunc("/auth/user/register", handleUserRegister).Methods("POST", "OPTIONS")
	r.HandleFunc("/auth/user/login", handleUserLogin).Methods("POST", "OPTIONS")
	r.HandleFunc("/auth/user/logout", handleUserLogout).Methods("POST", "OPTIONS")
	r.HandleFunc("/auth/user/google", handleUserGoogleLogin).Methods("GET")
	r.HandleFunc("/auth/user/google/callback", handleUserGoogleCallback).Methods("GET")

	// API v1 routes (require API key)
	apiRouter := r.PathPrefix("/api/v1").Subrouter()
	apiRouter.Use(requireAPIKey)
	apiRouter.HandleFunc("/room/create", handleAPICreateRoom).Methods("POST")
	apiRouter.HandleFunc("/room/{id}/link", handleAPIGetRoomLink).Methods("GET")
	apiRouter.HandleFunc("/room/{id}/files", handleAPIGetRoomFiles).Methods("GET")
	apiRouter.HandleFunc("/room/{id}/download-all", handleAPIDownloadAll).Methods("GET")

	r.PathPrefix("/").Handler(cleanURLMiddleware(http.FileServer(http.Dir("./frontend"))))

	addr := fmt.Sprintf(":%d", Port)
	log.Printf("🚀 AmbilFile Server (Go) running on port %d", Port)
	log.Fatal(http.ListenAndServe(addr, r))
}
