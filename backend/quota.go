package main

import (
	"database/sql"
	"fmt"
	"strconv"
)


func formatBytes(b int64) string {
	const unit = 1024
	if b < unit {
		return fmt.Sprintf("%d B", b)
	}
	div, exp := int64(unit), 0
	for n := b / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f %cB", float64(b)/float64(div), "KMGTPE"[exp])
}


func formatBytesID(b int64) string {
	const gb = 1024 * 1024 * 1024
	const mb = 1024 * 1024
	if b >= gb {
		return strconv.FormatFloat(float64(b)/float64(gb), 'f', 1, 64) + " GB"
	}
	return strconv.FormatInt(b/mb, 10) + " MB"
}


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

