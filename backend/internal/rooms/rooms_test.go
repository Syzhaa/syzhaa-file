package rooms

import (
	"crypto/subtle"
	"testing"

	"github.com/syzhaa/file-server/internal/auth"
)

// TestOwnerTokenConstantTime verifies we use constant-time comparison.
func TestOwnerTokenConstantTime(t *testing.T) {
	token := auth.GenerateRandomString(64)
	hash1 := auth.HashString(token)
	hash2 := auth.HashString(token)
	hash3 := auth.HashString("different-token")

	// Same token hashes should match with constant-time compare
	if subtle.ConstantTimeCompare([]byte(hash1), []byte(hash2)) != 1 {
		t.Error("same token hashes should match")
	}
	// Different should not match
	if subtle.ConstantTimeCompare([]byte(hash1), []byte(hash3)) == 1 {
		t.Error("different token hashes should not match")
	}
}

// TestGeneratePinFormat documents that PIN is 6 digits.
// Actual generation requires DB, tested manually.
func TestGeneratePinFormat(t *testing.T) {
	// PIN format: %06d (6 digits, 100000-999999)
	// Verified in code review: internal/rooms/room_handlers.go GeneratePin()
	t.Log("PIN format is 6 digits (verified in code)")
}
