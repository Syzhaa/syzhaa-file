# Routing Flow Diagram
**Generated:** 2026-07-23  
**Status:** 🔴 BROKEN - Critical routing issues detected

---

## 🔴 CRITICAL: Room Interface Missing

```
USER FLOW (BROKEN):
┌──────────────────┐
│   Landing Page   │
│  (index.html)    │
└────────┬─────────┘
         │
         ├─── [Create Room Button Clicked]
         │                 │
         │                 ▼
         │        ❌ fetch('/room/create')  ← WRONG PATH! Should be /api/room/create
         │                 │
         │                 ▼ (404 ERROR - Backend expects /api/room/create)
         │        ✗ No room created
         │
         └─── [Join PIN Button Clicked]
                      │
                      ▼
             ❌ Redirects to /?pin=123456
                      │
                      ▼
             🚫 NO CODE TO HANDLE THIS!
             🚫 User just sees landing page again
             🚫 Room interface DOES NOT EXIST


EXPECTED FLOW (Not Implemented):
┌──────────────────┐
│   Landing Page   │
└────────┬─────────┘
         │
         ├─── Create Room
         │         │
         │         ▼
         │    POST /api/room/create ← Should use correct path
         │         │
         │         ▼
         │    Redirect to /?room={id}
         │         │
         │         ▼
         │    ✅ Detect URL params
         │    ✅ Fetch room data
         │    ✅ Show room interface (MISSING!)
         │    ✅ Allow file uploads
         │    ✅ Display file list
         │
         └─── Join PIN
                   │
                   ▼
              Redirect to /?pin={pin}
                   │
                   ▼
              ✅ Detect PIN param (MISSING!)
              ✅ Convert PIN to room ID (MISSING!)
              ✅ Show room interface (MISSING!)
```

---

## 🟡 Admin Login Flow (WORKS, with inconsistency)

```
ADMIN LOGIN:
┌──────────────────────┐
│ /admin/login.html    │
│ [Login with Google]  │
└──────────┬───────────┘
           │
           ▼
   /auth/google/login
           │
           ▼
   Google OAuth Dialog
           │
           ▼
   /auth/google/callback
           │
           ├─── Check email whitelist
           │
           ├─── ✅ Whitelisted
           │         │
           │         ▼
           │    Create admin_session cookie
           │         │
           │         ▼
           │    Redirect to /admin/dashboard
           │         │
           │         ▼
           │    ⚠️  cleanURLMiddleware appends .html
           │         │
           │         ▼
           │    Serves /admin/dashboard.html ✅
           │
           └─── ❌ Not whitelisted
                     │
                     ▼
                 Error: "email not authorized"
```

---

## 🟡 User Login Flow (BROKEN - No Dashboard)

```
USER LOGIN:
┌──────────────────────┐
│ /user-login.html     │
│ [Login with Google]  │
└──────────┬───────────┘
           │
           ▼
   /auth/user/login
           │
           ▼
   Google OAuth Dialog
           │
           ▼
   /auth/user/callback
           │
           ├─── Check if email in admin whitelist
           │         │
           │         ├─── ✅ Yes → Redirect to /admin/dashboard (works)
           │         │
           │         └─── ❌ No → Continue as regular user
           │
           ├─── Check user status in DB
           │
           ├─── Status: pending
           │         │
           │         ▼
           │    /pending-approval.html ✅
           │
           ├─── Status: rejected
           │         │
           │         ▼
           │    /registration-rejected.html ✅
           │
           ├─── Status: suspended
           │         │
           │         ▼
           │    /account-suspended.html ✅
           │
           └─── Status: approved
                     │
                     ▼
                Create user_session cookie
                     │
                     ▼
                Redirect to / (root)
                     │
                     ▼
                🚫 PROBLEM: Just shows landing page!
                🚫 No /user/dashboard.html exists
                🚫 No way to tell user is logged in
                🚫 No profile, rooms, or settings UI
```

---

## 📊 Route Architecture

```
PUBLIC LAYER (No Auth):
├─── / (index.html)                          Landing page
├─── /user-login.html                        User login page
├─── /admin/login.html                       Admin login page
├─── /pending-approval.html                  Status page
├─── /registration-rejected.html             Status page
├─── /account-suspended.html                 Status page
└─── 🚫 /room/{id} or /?room={id}           MISSING!

API LAYER (Public):
├─── POST /api/room/create                   Create room
├─── POST /api/room/pin                      Access by PIN
├─── GET  /api/room/{id}                     Room info
├─── POST /api/upload/{roomId}               Upload file
├─── DELETE /api/file/{id}                   Delete file
└─── GET  /d/{id}                            Download file

ADMIN LAYER (Requires admin_session):
├─── /admin/dashboard.html                   Admin panel ✅
├─── GET  /admin/me                          Admin info
├─── GET  /admin/stats                       System stats
├─── GET  /admin/users                       User management
├─── POST /admin/users/{id}/approve          Approve user
├─── GET  /admin/api-keys                    API key management
└─── [... 10+ more admin routes]

USER LAYER (Requires user_session):
├─── 🚫 /user/dashboard.html                 MISSING!
└─── GET  /user/me                           User info (API only)

API v1 LAYER (Requires API key header):
├─── POST /api/v1/room/create
├─── GET  /api/v1/room/{id}/link
├─── GET  /api/v1/room/{id}/files
└─── GET  /api/v1/room/{id}/download-all
```

---

## 🐛 Broken Paths Detail

### 1. Frontend calls wrong API endpoint
```
❌ CURRENT (index.html:406):
fetch('/room/create', { ... })

✅ SHOULD BE:
fetch('/api/room/create', { ... })

Backend Route (main.go:543):
r.HandleFunc("/api/room/create", createRoomHandler)
```

### 2. No room display logic
```
❌ CURRENT (index.html:416):
window.location.href = `/?room=${data.room_id}`;
// Then nothing! No code to handle this URL parameter

✅ NEEDS:
// On page load:
const params = new URLSearchParams(window.location.search);
const roomId = params.get('room');
if (roomId) {
    loadRoomInterface(roomId);  // Function doesn't exist!
}
```

### 3. No user dashboard
```
❌ CURRENT (users.go:172):
http.Redirect(w, r, "/", http.StatusTemporaryRedirect)
// Redirects to landing page - user can't tell they're logged in

✅ NEEDS:
http.Redirect(w, r, "/user/dashboard", ...)
// But /public/user/dashboard.html doesn't exist!
```

### 4. media-preview.js orphaned
```
❌ CURRENT (media-preview.js:150):
if (window.currentRoomFiles) {
    window.renderFiles(window.currentRoomFiles);
}
// These variables/functions are never defined anywhere!

✅ NEEDS:
// Must be called from room interface:
window.currentRoomFiles = filesArray;
window.renderFiles = function(files) { ... };
```

---

## 🎯 Missing Files

### Critical:
1. **Room Interface** - No page exists to show room content
   - Should display room info (PIN, expiry, etc.)
   - Should have drag-and-drop upload area
   - Should show file list with download buttons
   - Should integrate with media-preview.js

2. **User Dashboard** - `/public/user/dashboard.html` doesn't exist
   - Should show user profile
   - Should list user's created rooms
   - Should show storage usage
   - Should have logout button

### Implementation Options:

#### Option A: Single Page Application (Recommended)
```
index.html becomes dynamic:
- Default: Shows landing page (hero, features, etc.)
- If ?room=xxx: Hides landing, shows room interface
- If ?pin=xxx: Converts PIN to room, shows room interface
- If user_session exists: Shows user navbar with profile
```

#### Option B: Separate Pages
```
Create new files:
- /public/room.html → Room interface
- /public/user/dashboard.html → User dashboard

Update redirects:
- After create room: redirect to /room?id=xxx
- After login: redirect to /user/dashboard
```

---

## 🔧 Fix Priority

### 🔴 P0 - CRITICAL (Blocks core functionality):
1. Fix API endpoint mismatch (`/room/create` → `/api/room/create`)
2. Add room display logic to index.html (detect ?room= and ?pin= params)
3. Implement room interface UI (upload area, file list)

### 🟡 P1 - HIGH (Missing features):
4. Create user dashboard (`/user/dashboard.html`)
5. Update user login redirect to go to dashboard
6. Add user session indicator on landing page (show user is logged in)

### 🟢 P2 - MEDIUM (Improvements):
7. Standardize path conventions (.html vs no extension)
8. Fix media-preview.js integration (define missing functions)
9. Add proper error handling for expired/invalid rooms
10. Add loading states during room creation/join

---

## 📝 File Modification Checklist

```
✅ READ:
[✓] main.go                  (understand backend routes)
[✓] auth.go                  (understand admin login flow)
[✓] users.go                 (understand user login flow)
[✓] public/index.html        (understand frontend structure)
[✓] public/index-v2.html     (duplicate of index.html?)
[✓] public/admin/dashboard.* (admin interface exists)
[✓] public/media-preview.js  (orphaned functionality)

🔧 NEEDS MODIFICATION:
[ ] public/index.html        (fix API path, add room logic)
[ ] public/index-v2.html     (same fixes as above)
[ ] users.go:172             (change redirect path)
[ ] auth.go:149              (optional: add .html extension)

📝 NEEDS CREATION:
[ ] public/user/dashboard.html
[ ] public/user/dashboard.js
[ ] Room interface logic (either in index.html or separate file)

❓ CLARIFICATION NEEDED:
[ ] Is index-v2.html actually used? (seems identical to index.html)
[ ] Should rooms be in same page or separate?
[ ] User approval - is this feature actually needed?
[ ] Admin dashboard route - keep /admin/dashboard or change to .html?
```

---

## 🚀 Recommended Implementation Order

### Phase 1: Make rooms work (Critical)
```bash
1. Fix API endpoint in index.html:406
   - Change '/room/create' to '/api/room/create'

2. Add room detection logic to index.html
   - Detect ?room= and ?pin= URL parameters
   - Fetch room data from backend
   - Hide landing page, show room interface

3. Create room interface HTML/CSS
   - Upload area with drag-and-drop
   - File list display
   - Download buttons
   - Room info (PIN, expiry timer)

4. Integrate media-preview.js
   - Define window.renderFiles()
   - Define window.currentRoomFiles
   - Call from room interface
```

### Phase 2: Add user dashboard
```bash
5. Create /public/user/dashboard.html
   - User profile section
   - List of created rooms
   - Storage usage stats
   - Account settings

6. Update users.go:172
   - Redirect to /user/dashboard instead of /

7. Add user session indicator to index.html
   - Show user avatar in navbar when logged in
   - Add dropdown with dashboard link + logout
```

### Phase 3: Polish
```bash
8. Standardize path conventions
9. Add error handling
10. Add loading states
11. Add animations/transitions
```

---

## 💡 Architecture Recommendation

**Current State:**
- Broken: Users can't actually use the core room feature
- Inconsistent: Mixed routing conventions
- Incomplete: Missing critical pages

**Recommended Fix:**
1. **Keep index.html as SPA (Single Page App)**
   - Default state: Landing page
   - Room state: Shows room interface
   - Dynamically switch between states based on URL

2. **Create separate user dashboard**
   - Dedicated page for logged-in users
   - Shows room history and account management

3. **Standardize all paths**
   - Backend: Always redirect to paths without .html
   - Let cleanURLMiddleware handle .html appending
   - OR: Always use .html in redirects (simpler, more explicit)

This matches modern web app patterns and keeps the codebase maintainable.
