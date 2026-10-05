package main

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
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
func generateAPIKey() string {
	return fmt.Sprintf("sfa_%s", generateRandomString(32))
}

// Handler: Create API Key
func handleCreateAPIKey(w http.ResponseWriter, r *http.Request) {
	admin := r.Context().Value("admin").(*AdminUser)

	var req CreateAPIKeyRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
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

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
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
func handleListAPIKeys(w http.ResponseWriter, r *http.Request) {
	admin := r.Context().Value("admin").(*AdminUser)

	rows, err := db.Query(`SELECT id, admin_id, name, expires_at, created_at, last_used_at, is_active
		FROM api_keys WHERE admin_id = ? ORDER BY created_at DESC`, admin.ID)
	if err != nil {
		http.Error(w, `{"error":"Failed to fetch API keys"}`, http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	keys := []APIKey{}
	for rows.Next() {
		var k APIKey
		var expiresAt, lastUsedAt sql.NullString
		var isActive int

		rows.Scan(&k.ID, &k.AdminID, &k.Name, &expiresAt, &k.CreatedAt, &lastUsedAt, &isActive)

		if expiresAt.Valid {
			t, _ := time.Parse(time.RFC3339, expiresAt.String)
			k.ExpiresAt = &t
		}
		if lastUsedAt.Valid {
			t, _ := time.Parse(time.RFC3339, lastUsedAt.String)
			k.LastUsedAt = &t
		}
		k.IsActive = isActive == 1

		keys = append(keys, k)
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"success": true,
		"keys":    keys,
	})
}

// Handler: Delete API Key
func handleDeleteAPIKey(w http.ResponseWriter, r *http.Request) {
	admin := r.Context().Value("admin").(*AdminUser)
	vars := mux.Vars(r)
	keyID := vars["id"]

	// Check ownership
	var ownerID string
	err := db.QueryRow("SELECT admin_id FROM api_keys WHERE id = ?", keyID).Scan(&ownerID)
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

	_, err = db.Exec("DELETE FROM api_keys WHERE id = ?", keyID)
	if err != nil {
		http.Error(w, `{"error":"Failed to delete API key"}`, http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{"success": true})
}

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

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"success":   true,
		"is_active": newStatus == 1,
	})
}

// Validate API Key
func validateAPIKey(keyString string) (*APIKey, error) {
	if !strings.HasPrefix(keyString, "sfa_") {
		return nil, fmt.Errorf("invalid API key format")
	}

	keyHash := hashString(keyString)

	var k APIKey
	var expiresAt, lastUsedAt sql.NullString
	var isActive int
	var userID sql.NullString

	err := db.QueryRow(`SELECT id, admin_id, user_id, name, expires_at, created_at, last_used_at, is_active
		FROM api_keys WHERE key_hash = ?`, keyHash).
		Scan(&k.ID, &k.AdminID, &userID, &k.Name, &expiresAt, &k.CreatedAt, &lastUsedAt, &isActive)

	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("invalid API key")
	}
	if err != nil {
		return nil, err
	}

	k.IsActive = isActive == 1
	if !k.IsActive {
		return nil, fmt.Errorf("API key is disabled")
	}
	if userID.Valid {
		k.UserID = userID.String
	}

	if expiresAt.Valid {
		t, _ := time.Parse(time.RFC3339, expiresAt.String)
		k.ExpiresAt = &t
		if time.Now().After(t) {
			return nil, fmt.Errorf("API key expired")
		}
	}

	// Update last used timestamp
	go db.Exec("UPDATE api_keys SET last_used_at = ? WHERE id = ?", time.Now().Format(time.RFC3339), k.ID)

	return &k, nil
}
