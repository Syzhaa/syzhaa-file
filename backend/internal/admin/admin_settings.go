package admin

import (
	"github.com/syzhaa/file-server/internal/auth"
	"github.com/syzhaa/file-server/internal/db"
	"github.com/syzhaa/file-server/internal/httpx"
	"log"
	"net/http"
	"time"
)

func HandleAdminSystemSettings(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	if r.Method == "GET" {
		rows, err := db.DB.Query(`SELECT key, value FROM system_settings`)
		if err != nil {
			httpx.WriteJSON(w, http.StatusOK, map[string]interface{}{
				"success": false,
				"error":   "Failed to fetch settings",
			})
			return
		}
		defer rows.Close()

		settings := make(map[string]string)
		for rows.Next() {
			var key, value string
			rows.Scan(&key, &value)
			settings[key] = value
		}

		httpx.WriteJSON(w, http.StatusOK, map[string]interface{}{
			"success":  true,
			"settings": settings,
		})
		return
	}

	if r.Method == "POST" {
		admin := r.Context().Value("admin").(*auth.AdminUser)

		var req struct {
			Key   string `json:"key"`
			Value string `json:"value"`
		}
		httpx.ReadJSON(r, &req)

		now := time.Now().Format(time.RFC3339)
		_, err := db.DB.Exec(`UPDATE system_settings 
			SET value = ?, updated_at = ?
			WHERE key = ?`,
			req.Value, now, req.Key)

		if err != nil {
			httpx.WriteJSON(w, http.StatusOK, map[string]interface{}{
				"success": false,
				"error":   "Failed to update setting",
			})
			return
		}

		log.Printf("✅ Admin %s updated setting: %s", admin.Email, req.Key)

		httpx.WriteJSON(w, http.StatusOK, map[string]interface{}{
			"success": true,
			"message": "Setting updated successfully",
		})
	}
}
