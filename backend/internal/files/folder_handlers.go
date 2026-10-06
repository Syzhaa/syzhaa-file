package files

import (
	"database/sql"
	"github.com/syzhaa/file-server/internal/db"
	"github.com/syzhaa/file-server/internal/httpx"
	"log"
	"net/http"

	"github.com/google/uuid"
	"github.com/gorilla/mux"
)

func CreateFolderHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	roomID := vars["roomId"]

	// Verify room exists
	var exists bool
	err := db.DB.QueryRow("SELECT EXISTS(SELECT 1 FROM rooms WHERE id = ?)", roomID).Scan(&exists)
	if err != nil || !exists {
		httpx.WriteError(w, http.StatusNotFound, "Invalid room")
		return
	}

	// Parse request
	var req struct {
		Name     string `json:"name"`
		ParentID string `json:"parent_id"`
	}
	if err := httpx.ReadJSON(r, &req); err != nil {
		httpx.WriteError(w, http.StatusBadRequest, "Invalid request")
		return
	}

	// Validate folder name
	if req.Name == "" || len(req.Name) > 255 {
		httpx.WriteError(w, http.StatusBadRequest, "Invalid folder name")
		return
	}

	// Generate folder ID
	folderID := uuid.New().String()

	// Insert folder
	_, err = db.DB.Exec(`INSERT INTO folders (id, room_id, parent_id, name) VALUES (?, ?, ?, ?)`,
		folderID, roomID, sql.NullString{String: req.ParentID, Valid: req.ParentID != ""}, req.Name)
	if err != nil {
		log.Printf("❌ Failed to create folder: %v", err)
		httpx.WriteError(w, http.StatusInternalServerError, "Failed to create folder")
		return
	}

	log.Printf("📁 Folder created: %s in room %s", req.Name, roomID)

	httpx.WriteJSON(w, http.StatusOK, map[string]interface{}{
		"success": true,
		"folder": map[string]interface{}{
			"id":        folderID,
			"name":      req.Name,
			"parent_id": req.ParentID,
		},
	})
}

func ListFoldersHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	roomID := vars["roomId"]
	parentID := r.URL.Query().Get("parent_id")

	// Build query based on parent_id
	var rows *sql.Rows
	var err error

	if parentID == "" {
		// Get root folders (no parent)
		rows, err = db.DB.Query(`SELECT id, name, created_at FROM folders 
			WHERE room_id = ? AND parent_id IS NULL 
			ORDER BY created_at DESC`, roomID)
	} else {
		// Get folders in specific parent
		rows, err = db.DB.Query(`SELECT id, name, created_at FROM folders 
			WHERE room_id = ? AND parent_id = ? 
			ORDER BY created_at DESC`, roomID, parentID)
	}

	if err != nil {
		log.Printf("❌ Failed to list folders: %v", err)
		httpx.WriteError(w, http.StatusInternalServerError, "Failed to list folders")
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

	httpx.WriteJSON(w, http.StatusOK, map[string]interface{}{
		"success": true,
		"folders": folders,
	})
}

func DeleteFolderHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	folderID := vars["id"]

	// Verify folder exists
	var name string
	err := db.DB.QueryRow("SELECT name FROM folders WHERE id = ?", folderID).Scan(&name)
	if err == sql.ErrNoRows {
		httpx.WriteError(w, http.StatusNotFound, "Folder not found")
		return
	}
	if err != nil {
		httpx.WriteError(w, http.StatusInternalServerError, "Server error")
		return
	}

	// Delete folder (CASCADE will delete child folders and files)
	_, err = db.DB.Exec("DELETE FROM folders WHERE id = ?", folderID)
	if err != nil {
		log.Printf("❌ Failed to delete folder: %v", err)
		httpx.WriteError(w, http.StatusInternalServerError, "Failed to delete folder")
		return
	}

	log.Printf("🗑️  Folder deleted: %s (ID: %s)", name, folderID)

	httpx.WriteJSON(w, http.StatusOK, map[string]interface{}{
		"success": true,
	})
}
