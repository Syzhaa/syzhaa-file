# Path Anomalies Report
**Generated:** 2026-07-23  
**Project:** Syzhaa File Server

---

## 🚨 CRITICAL ISSUES

### 1. API Endpoint Mismatch (BREAKING BUG)
**Impact:** Room creation will fail with 404 error

**Frontend calls:**
- `public/index.html:406` → `fetch('/room/create')`
- `public/index-v2.html:404` → `fetch('/room/create')`

**Backend routes:**
- `main.go:543` → `r.HandleFunc("/api/room/create", ...)`

**Fix Required:**
```javascript
// Option A: Change frontend
fetch('/api/room/create', { ... })

// Option B: Change backend
r.HandleFunc("/room/create", createRoomHandler)
```

---

### 2. Missing Room Page Logic
**Impact:** After creating/joining room, users see blank page

**Current Flow:**
1. User creates room → redirects to `/?room={room_id}` (index.html:416)
2. User joins PIN → redirects to `/?pin={pin}` (index.html:435)
3. **Problem:** `index.html` has NO code to detect these query params
4. **Result:** User sees landing page instead of room interface

**Missing Code:**
```javascript
// index.html needs this on page load:
const urlParams = new URLSearchParams(window.location.search);
const roomId = urlParams.get('room');
const pin = urlParams.get('pin');

if (pin) {
  // Fetch room by PIN, then show room interface
}
if (roomId) {
  // Fetch room info, then show room interface
}
```

**Files affected:**
- `public/index.html` - no query param detection
- `public/index-v2.html` - no query param detection

---

### 3. Missing User Dashboard
**Impact:** Logged-in users have no home page

**Current Flow:**
- Admin login → `/admin/dashboard` ✅ (has dashboard)
- User login (approved) → `/` ❌ (just landing page)

**Backend:**
- `users.go:172` → `http.Redirect(w, r, "/", http.StatusTemporaryRedirect)`

**Problem:**
- No `/user/dashboard.html` exists
- Logged-in users can't tell they're authenticated
- No way for users to see their rooms, stats, or account info

**Missing Files:**
- `public/user/dashboard.html` (doesn't exist)
- User navbar/profile indicator on index.html

---

## ⚠️ MODERATE ISSUES

### 4. Inconsistent Path Conventions
**Impact:** Confusing, harder to maintain

**Mixed styles:**
```
WITH .html extension:
✓ /pending-approval.html (users.go:136)
✓ /registration-rejected.html (users.go:141)
✓ /account-suspended.html (users.go:146)
✓ /user-login.html
✓ /admin/login.html

WITHOUT .html extension:
✓ /admin/dashboard (auth.go:149, users.go:120)
  → relies on cleanURLMiddleware to find dashboard.html
```

**Recommendation:** 
- **Option A:** Always use `.html` extension (explicit, no middleware dependency)
- **Option B:** Never use `.html` extension (clean URLs, requires middleware)

---

### 5. Admin Dashboard Path Inconsistency
**Impact:** Works but relies on middleware

**Backend redirect:**
```go
// auth.go:149, users.go:120
http.Redirect(w, r, "/admin/dashboard", http.StatusTemporaryRedirect)
```

**Physical file:**
```
/public/admin/dashboard.html
```

**How it works:**
- `main.go:154-176` - `cleanURLMiddleware` appends `.html`
- Works, but creates inconsistency with other paths

---

## 📊 ROUTE MAPPING

### Public Routes (No Auth)
```
GET  /                          → public/index.html (landing)
GET  /user-login.html           → public/user-login.html
GET  /admin/login.html          → public/admin/login.html
GET  /pending-approval.html     → public/pending-approval.html
GET  /registration-rejected.html → public/registration-rejected.html
GET  /account-suspended.html    → public/account-suspended.html
```

### Authentication Routes
```
Admin Flow:
  GET  /auth/google/login       → OAuth start
  GET  /auth/google/callback    → OAuth return → redirect to /admin/dashboard
  POST /auth/logout             → Clear admin session

User Flow:
  GET  /auth/user/login         → OAuth start
  GET  /auth/user/callback      → OAuth return → redirect based on status:
                                  - pending: /pending-approval.html
                                  - rejected: /registration-rejected.html
                                  - suspended: /account-suspended.html
                                  - approved: / (ISSUE: no user dashboard)
                                  - admin detected: /admin/dashboard
  POST /auth/user/logout        → Clear user session
```

### API Routes (Public)
```
POST /api/room/create           → Create new room (main.go:543)
POST /api/room/pin              → Access room by PIN (main.go:544)
GET  /api/room/{id}             → Get room info (main.go:545)
POST /api/upload/{roomId}       → Upload file chunk (main.go:546)
DEL  /api/file/{id}             → Delete file (main.go:547)
GET  /d/{id}                    → Download file (main.go:548)
```

### Admin Routes (Require admin_session cookie)
```
GET  /admin/me                  → Current admin info
GET  /admin/stats               → System stats
GET  /admin/api-keys            → List API keys
POST /admin/api-keys            → Create API key
DEL  /admin/api-keys/{id}       → Delete API key
POST /admin/api-keys/{id}/toggle → Toggle API key
GET  /admin/users               → List all users
POST /admin/users/{id}/approve  → Approve user
POST /admin/users/{id}/reject   → Reject user
POST /admin/users/{id}/suspend  → Suspend user
PUT  /admin/users/{id}/quotas   → Update user quotas
GET  /admin/users/{id}/stats    → Get user stats
GET  /admin/settings            → Get system settings
POST /admin/settings            → Update system settings
```

### User Routes (Require user_session cookie)
```
GET  /user/me                   → Current user info (main.go:582)
```

### API v1 Routes (Require API key header)
```
POST /api/v1/room/create        → API create room
GET  /api/v1/room/{id}/link     → Get room link
GET  /api/v1/room/{id}/files    → List room files
GET  /api/v1/room/{id}/download-all → Download all files
```

---

## 🛠️ RECOMMENDED FIXES

### Priority 1: Fix API Endpoint (CRITICAL)
```javascript
// public/index.html:406
// public/index-v2.html:404
const response = await fetch('/api/room/create', {  // Add /api prefix
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({ expiry_minutes: expiryMinutes })
});
```

### Priority 2: Add Room Page Logic
Add to bottom of `index.html` and `index-v2.html`:
```javascript
// Detect room access via URL params
window.addEventListener('DOMContentLoaded', async () => {
    const urlParams = new URLSearchParams(window.location.search);
    const roomId = urlParams.get('room');
    const pin = urlParams.get('pin');
    
    if (pin) {
        // Convert PIN to room ID
        const response = await fetch('/api/room/pin', {
            method: 'POST',
            headers: { 'Content-Type': 'application/json' },
            body: JSON.stringify({ pin })
        });
        const data = await response.json();
        if (data.success) {
            // Load room interface with data.room_id
            loadRoomInterface(data.room_id);
        }
    } else if (roomId) {
        // Load room interface directly
        loadRoomInterface(roomId);
    }
});

async function loadRoomInterface(roomId) {
    // Hide landing page content
    // Show room interface (upload area, file list)
    // Fetch /api/room/{roomId} for room info
    // Set up file upload handlers
}
```

### Priority 3: Create User Dashboard
Create `public/user/dashboard.html` with:
- User profile info
- List of user's created rooms
- Storage usage stats
- Account settings link

Update `users.go:172`:
```go
// Redirect to user dashboard instead of /
http.Redirect(w, r, "/user/dashboard", http.StatusTemporaryRedirect)
```

### Priority 4: Standardize Path Convention
**Recommendation:** Use explicit `.html` extensions everywhere

```go
// auth.go:149
http.Redirect(w, r, "/admin/dashboard.html", http.StatusTemporaryRedirect)

// users.go:120
http.Redirect(w, r, "/admin/dashboard.html", http.StatusTemporaryRedirect)
```

---

## 📝 SUMMARY

**Total Anomalies Found:** 5

**Critical (breaks functionality):**
1. API endpoint mismatch - room creation fails
2. Missing room page logic - users can't use rooms

**Major (missing features):**
3. No user dashboard after login

**Minor (inconsistencies):**
4. Mixed path conventions
5. Admin dashboard path relies on middleware

**Files Requiring Changes:**
- `public/index.html` (fix API path, add room logic)
- `public/index-v2.html` (fix API path, add room logic)
- `public/user/dashboard.html` (create new)
- `users.go:172` (change redirect)
- `auth.go:149` (optional: add .html)
- `users.go:120` (optional: add .html)
