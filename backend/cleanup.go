package main

import (
	"log"
	"os"
	"path/filepath"
	"time"
)


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
		var totalBytesDeleted int64

		for _, roomID := range expiredRooms {
			fileRows, err := db.Query("SELECT filename, size FROM files WHERE room_id = ?", roomID)
			if err != nil {
				log.Printf("⚠️  Auto-cleanup: Failed to query files for room %s: %v", roomID, err)
				continue
			}

			for fileRows.Next() {
				var filename string
				var fsize int64
				fileRows.Scan(&filename, &fsize)
				filePath := filepath.Join(UploadDir, filename)
				
				if err := os.Remove(filePath); err != nil {
					if !os.IsNotExist(err) {
						log.Printf("⚠️  Auto-cleanup: Failed to delete file %s: %v", filename, err)
						totalFilesFailed++
					}
				} else {
					log.Printf("🗑️  Auto-cleanup: Deleted file %s", filename)
					totalFilesDeleted++
					totalBytesDeleted += fsize
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

		// Update cumulative deletion stats
		if len(expiredRooms) > 0 {
			db.Exec(`UPDATE system_settings SET value = CAST(value AS INTEGER) + ? WHERE key = 'stats_deleted_rooms'`, len(expiredRooms))
			db.Exec(`UPDATE system_settings SET value = CAST(value AS INTEGER) + ? WHERE key = 'stats_deleted_files'`, totalFilesDeleted)
			db.Exec(`UPDATE system_settings SET value = CAST(value AS INTEGER) + ? WHERE key = 'stats_deleted_bytes'`, totalBytesDeleted)
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

