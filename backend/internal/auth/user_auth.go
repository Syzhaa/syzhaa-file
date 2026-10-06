package auth

import (
	"github.com/syzhaa/file-server/internal/httpx"
	"github.com/google/uuid"
	"database/sql"
	"context"
	"github.com/syzhaa/file-server/internal/db"
	"net/http"
	"time"

)

type userCtxKey struct{}

// User represents a registered user.
type User struct {
	ID                  string     `json:"id"`
	Email               string     `json:"email"`
	Name                string     `json:"name"`
	AvatarURL           string     `json:"avatar_url,omitempty"`
	Status              string     `json:"status"` // pending, approved, rejected, suspended
	ApprovedBy          *string    `json:"approved_by,omitempty"`
	ApprovedAt          *time.Time `json:"approved_at,omitempty"`
	RejectedReason      *string    `json:"rejected_reason,omitempty"`
	StorageLimitMB      *int       `json:"storage_limit_mb"`
	MaxFileDurationDays *int       `json:"max_file_duration_days"`
	CreatedAt           time.Time  `json:"created_at"`
	LastLogin           *time.Time `json:"last_login,omitempty"`
}

// contextWithUser stores the authenticated *User in the request context.
func contextWithUser(ctx context.Context, u *User) context.Context {
	return context.WithValue(ctx, userCtxKey{}, u)
}

// userFromContext retrieves the authenticated *User, or nil.
func UserFromContext(ctx context.Context) *User {
	if u, ok := ctx.Value(userCtxKey{}).(*User); ok {
		return u
	}
	return nil
}

// ---------------------------------------------------------------------------
// Schema migration for email/password user auth + user-owned API keys.
// Safe to run on every startup (IF NOT EXISTS / guarded ALTERs).
// ---------------------------------------------------------------------------

// ---------------------------------------------------------------------------
// Session helpers (mirror the admin session pattern)
// ---------------------------------------------------------------------------
type UserSession struct {
	ID        string
	UserID    string
	Token     string
	ExpiresAt time.Time
}

func CreateUserSession(userID string) (*UserSession, error) {
	sessionID := uuid.New().String()
	token := GenerateRandomString(48)
	tokenHash := HashPassword(token)
	expiresAt := time.Now().Add(7 * 24 * time.Hour)

	_, err := db.DB.Exec(`
		INSERT INTO user_sessions (id, user_id, token_hash, expires_at)
		VALUES (?, ?, ?, ?)
	`, sessionID, userID, tokenHash, expiresAt.Format(time.RFC3339))
	if err != nil {
		return nil, err
	}

	return &UserSession{ID: sessionID, UserID: userID, Token: token, ExpiresAt: expiresAt}, nil
}

func ValidateUserSession(r *http.Request) (*User, error) {
	userID, err := getSessionOwnerID(userSessionCfg, r)
	if err != nil {
		return nil, err
	}

	var u User
	var approvedBy, rejectedReason sql.NullString
	var approvedAt, lastLogin sql.NullString
	var storageLimit, maxDuration sql.NullInt64
	err = db.DB.QueryRow(`
		SELECT id, email, name, status, approved_by, approved_at,
		       rejected_reason, storage_limit_mb, max_file_duration_days,
		       created_at, last_login
		FROM users WHERE id = ?
	`, userID).Scan(&u.ID, &u.Email, &u.Name, &u.Status,
		&approvedBy, &approvedAt, &rejectedReason,
		&storageLimit, &maxDuration, &u.CreatedAt, &lastLogin)
	if err != nil {
		return nil, err
	}
	if approvedBy.Valid {
		u.ApprovedBy = &approvedBy.String
	}
	if rejectedReason.Valid {
		u.RejectedReason = &rejectedReason.String
	}
	if storageLimit.Valid {
		v := int(storageLimit.Int64)
		u.StorageLimitMB = &v
	}
	if maxDuration.Valid {
		v := int(maxDuration.Int64)
		u.MaxFileDurationDays = &v
	}
	if u.Status != "approved" && u.Status != "active" {
		return nil, sql.ErrNoRows
	}

	return &u, nil
}

// requireUserSession replaces the old stub middleware.
func RequireUserSession(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		user, err := ValidateUserSession(r)
		if err != nil {
			w.Header().Set("Content-Type", "application/json")
			httpx.WriteJSON(w, http.StatusUnauthorized, map[string]interface{}{
				"success": false,
				"error":   "Login diperlukan",
			})
			return
		}
		ctx := r.Context()
		// store *User in context under "user"
		next.ServeHTTP(w, r.WithContext(contextWithUser(ctx, user)))
	})
}

// ---------------------------------------------------------------------------
// Registration & login
// ---------------------------------------------------------------------------

// POST /auth/user/register
// POST /auth/user/register — DISABLED: user signup is Google-only now


// POST /auth/user/login — DISABLED: user login is Google-only now


// POST /auth/user/logout


// ---------------------------------------------------------------------------
// Authenticated user endpoints (replace stubs in users.go)
// ---------------------------------------------------------------------------

// GET /user/me


// roomOut is the JSON shape for /user/rooms
type roomOut struct {
	ID        string `json:"id"`
	PIN       string `json:"pin"`
	CreatedAt string `json:"created_at"`
	ExpiresAt string `json:"expires_at"`
}

// keyOut is the JSON shape for /user/api-keys (list)
type keyOut struct {
	ID        string  `json:"id"`
	Name      string  `json:"name"`
	ExpiresAt *string `json:"expires_at,omitempty"`
	CreatedAt string  `json:"created_at"`
	IsActive  bool    `json:"is_active"`
}

// GET /user/rooms


// ---------------------------------------------------------------------------
// User-owned API keys
// ---------------------------------------------------------------------------

// GET /user/api-keys

// POST /user/api-keys


// POST /user/api-keys/request — request admin approval for API key access


// DELETE /user/api-keys/{id}
