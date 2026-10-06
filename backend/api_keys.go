package main

import (
	"github.com/syzhaa/file-server/internal/httpx"
	"database/sql"
	"net/http"
	"time"

	"github.com/google/uuid"
	"github.com/gorilla/mux"
)

type APIKey struct {
	ID         string     `json:"id"`
	Key        string     `json:"key,omitempty"` // Only shown once on creation
	AdminID    string     `json:"admin_id"`
	UserID     string     `json:"user_id,omitempty"`
	Name       string     `json:"name"`
	ExpiresAt  *time.Time `json:"expires_at"`
	CreatedAt  time.Time  `json:"created_at"`
	LastUsedAt *time.Time `json:"last_used_at"`
	IsActive   bool       `json:"is_active"`
}

type CreateAPIKeyRequest struct {
	Name           string `json:"name"`
	ExpiryDays     *int   `json:"expiry_days"` // null = never expires
}

// Generate API Key (format: sfa_xxxxxxxxxxxxxxxxxxxxx)

// Handler: Create API Key
func handleCreateAPIKey(w http.ResponseWriter, r *http.Request) {
	admin := r.Context().Value("admin").(*AdminUser)

	var req CreateAPIKeyRequest
	if err := httpx.ReadJSON(r, &req); err != nil {
		http.Error(w, `{"error":"Invalid request"}`, http.StatusBadRequest)
		return
	}

	if req.Name == "" {
		http.Error(w, `{"error":"Name is required"}`, http.StatusBadRequest)
		return
	}

	key := generateAPIKey()
	keyHash := hashString(key)
	keyID := uuid.New().String()

	var expiresAt *string
	if req.ExpiryDays != nil && *req.ExpiryDays > 0 {
		expiry := time.Now().AddDate(0, 0, *req.ExpiryDays)
		expiryStr := expiry.Format(time.RFC3339)
		expiresAt = &expiryStr
	}

	_, err := db.Exec(`INSERT INTO api_keys (id, key_hash, admin_id, name, expires_at, created_at, is_active)
		VALUES (?, ?, ?, ?, ?, ?, 1)`,
		keyID, keyHash, admin.ID, req.Name, expiresAt, time.Now().Format(time.RFC3339))

	if err != nil {
		http.Error(w, `{"error":"Failed to create API key"}`, http.StatusInternalServerError)
		return
	}

	httpx.WriteJSON(w, http.StatusOK, map[string]interface{}{
		"success": true,
		"api_key": APIKey{
			ID:        keyID,
			Key:       key, // Only shown once
			AdminID:   admin.ID,
			Name:      req.Name,
			CreatedAt: time.Now(),
			IsActive:  true,
		},
	})
}

// Handler: List API Keys

// Handler: Delete API Key

// Handler: Toggle API Key
func handleToggleAPIKey(w http.ResponseWriter, r *http.Request) {
	admin := r.Context().Value("admin").(*AdminUser)
	vars := mux.Vars(r)
	keyID := vars["id"]

	var ownerID string
	var isActive int
	err := db.QueryRow("SELECT admin_id, is_active FROM api_keys WHERE id = ?", keyID).Scan(&ownerID, &isActive)
	if err == sql.ErrNoRows {
		http.Error(w, `{"error":"API key not found"}`, http.StatusNotFound)
		return
	}
	if err != nil {
		http.Error(w, `{"error":"Server error"}`, http.StatusInternalServerError)
		return
	}

	if ownerID != admin.ID {
		http.Error(w, `{"error":"Unauthorized"}`, http.StatusForbidden)
		return
	}

	newStatus := 0
	if isActive == 0 {
		newStatus = 1
	}

	_, err = db.Exec("UPDATE api_keys SET is_active = ? WHERE id = ?", newStatus, keyID)
	if err != nil {
		http.Error(w, `{"error":"Failed to update API key"}`, http.StatusInternalServerError)
		return
	}

	httpx.WriteJSON(w, http.StatusOK, map[string]interface{}{
		"success":   true,
		"is_active": newStatus == 1,
	})
}

// Validate API Key
