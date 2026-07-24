# BUG FOUND: Orphaned File Records in Database
**Date:** 2026-07-23 07:41 UTC  
**Severity:** 🟡 MEDIUM  
**Status:** ❌ BUG CONFIRMED

---

## 🐛 Problem Description

**Symptom:**
- Database shows 6 files
- Upload folder has 0 files
- Files were uploaded, room expired, cleanup ran
- Physical files deleted ✅
- Database records NOT deleted ❌

**Evidence:**
```sql
-- Database has 6 file records
SELECT COUNT(*) FROM files;
-- Result: 6

-- But these files belong to rooms that don't exist anymore
SELECT f.id, f.original_name, f.room_id 
FROM files f 
LEFT JOIN rooms r ON f.room_id = r.id 
WHERE r.id IS NULL;
-- Result: 6 orphaned records
```

**Physical Reality:**
```bash
ls uploads/
# Empty - 0 files
```

---

## 🔍 Root Cause

**The Bug:** SQLite Foreign Keys NOT Enabled

In `main.go` database schema:
```sql
CREATE TABLE files (
    ...
    room_id TEXT NOT NULL,
    FOREIGN KEY(room_id) REFERENCES rooms(id) ON DELETE CASCADE
);
```

The `ON DELETE CASCADE` should auto-delete files when room is deleted.

**BUT:** SQLite has foreign keys **DISABLED by default**!

Check:
```sql
PRAGMA foreign_keys;
-- Result: 0 (disabled)
```

**Result:** When room is deleted, files records are NOT auto-deleted.

---

## 🔧 The Fix

### Option 1: Enable Foreign Keys (Recommended)
Add after opening database connection in `main.go`:

```go
func initDB() error {
    var err error
    os.MkdirAll("./data", 0755)
    db, err = sql.Open("sqlite3", DBPath)
    if err != nil {
        return err
    }

    // ✅ ENABLE FOREIGN KEYS
    _, err = db.Exec("PRAGMA foreign_keys = ON")
    if err != nil {
        return err
    }

    // ... rest of schema creation
}
```

### Option 2: Explicit Delete (Safest)
Update `autoCleanupWorker()` in `main.go`:

```go
for _, roomID := range expiredRooms {
    fileRows, _ := db.Query("SELECT filename FROM files WHERE room_id = ?", roomID)
    for fileRows.Next() {
        var filename string
        fileRows.Scan(&filename)
        filePath := filepath.Join(UploadDir, filename)
        os.Remove(filePath)
    }
    fileRows.Close()
    
    // ✅ EXPLICITLY DELETE FILES FROM DB
    db.Exec("DELETE FROM files WHERE room_id = ?", roomID)
    
    // Then delete room
    db.Exec("DELETE FROM rooms WHERE id = ?", roomID)
}
```

---

## 🧹 Clean Up Orphaned Records

**Immediate fix:**
```sql
-- Delete file records where room doesn't exist
DELETE FROM files 
WHERE room_id NOT IN (SELECT id FROM rooms);
```

**How many orphaned records:**
```sql
SELECT COUNT(*) 
FROM files f 
LEFT JOIN rooms r ON f.room_id = r.id 
WHERE r.id IS NULL;
-- Should show: 6
```

---

## 📊 Impact Assessment

**Current Impact:**
- 🟡 Medium severity
- Database slowly grows with orphaned records
- No immediate user-facing issues
- Storage space properly cleaned (physical files deleted)
- Only metadata pollution

**Long-term if not fixed:**
- Database will accumulate orphaned records
- Eventually database size grows unnecessarily
- Queries may slow down
- Disk space wasted on database file

**Good News:**
- ✅ Physical files ARE properly deleted
- ✅ Storage space IS freed
- ✅ No security risk
- ✅ No data leak

---

## ✅ Recommended Actions

### Immediate (Now):
1. Clean up current orphaned records
2. Apply fix to code
3. Rebuild & restart

### Testing:
1. Create test room with 1 minute expiry
2. Upload a file
3. Wait for expiry
4. Check database and uploads folder
5. Verify both are cleaned

---

## 📝 Implementation Steps

**Step 1: Clean current orphans**
```bash
cd backend
sqlite3 data/files.db "DELETE FROM files WHERE room_id NOT IN (SELECT id FROM rooms);"
```

**Step 2: Fix code (choose option 1 or 2)**

**Step 3: Rebuild**
```bash
cd backend
/usr/local/go/bin/go build -o file-server
pm2 restart syzhaa-file
```

**Step 4: Test**
- Create room with short expiry
- Upload file
- Wait for cleanup
- Verify complete deletion

---

## 🎯 Summary

**Problem:**
- CASCADE delete not working
- Foreign keys disabled in SQLite
- Orphaned records accumulate

**Solution:**
- Enable foreign keys: `PRAGMA foreign_keys = ON`
- OR explicit delete: `DELETE FROM files WHERE room_id = ?`

**Status:**
- Bug severity: Medium
- Fix difficulty: Easy (2 lines of code)
- Time to fix: 5 minutes

---

**Next:** Shall I implement the fix now?
