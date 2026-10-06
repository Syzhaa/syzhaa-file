package main

import (
	"database/sql"
	"encoding/json"
	"encoding/base64"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"time"

	"github.com/gorilla/mux"
)


func uploadChunkHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	roomID := vars["roomId"]

	var exists bool
	err := db.QueryRow("SELECT EXISTS(SELECT 1 FROM rooms WHERE id = ?)", roomID).Scan(&exists)
	if err != nil || !exists {
		errJSON(w, http.StatusNotFound, "Invalid room")
		return
	}

	r.ParseMultipartForm(MaxMemory)
	file, _, err := r.FormFile("file")
	if err != nil {
		errJSON(w, http.StatusBadRequest, "No chunk received")
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
		errJSON(w, http.StatusInternalServerError, "Failed to save chunk")
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
			errJSON(w, http.StatusInternalServerError, "Failed to create final file")
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
				errJSON(w, http.StatusInternalServerError, "Encryption failed")
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
			errJSON(w, http.StatusInternalServerError, "Failed to save metadata")
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
	var allowDelete int
	err := db.QueryRow(`SELECT f.filename, COALESCE(r.allow_delete, 1) FROM files f JOIN rooms r ON f.room_id = r.id WHERE f.id = ?`, fileID).Scan(&filename, &allowDelete)
	if err == sql.ErrNoRows {
		errJSON(w, http.StatusNotFound, "File not found")
		return
	}
	if err != nil {
		errJSON(w, http.StatusInternalServerError, "Server error")
		return
	}

	// Owner can disable delete
	if allowDelete == 0 {
		errJSON(w, http.StatusForbidden, "Hapus dinonaktifkan oleh pemilik room")
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
		errJSON(w, http.StatusInternalServerError, "Failed to delete file from database")
		return
	}

	log.Printf("✅ File deleted successfully: %s (ID: %s)", filename, fileID)
	okJSON(w, map[string]interface{}{"success": true})
}

