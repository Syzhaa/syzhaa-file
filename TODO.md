# TODO - Fix Path Anomalies
**Created:** 2026-07-23  
**Priority:** 🔴 CRITICAL  
**Estimated Time:** 60 minutes

---

## 🔴 PHASE 1: CRITICAL FIXES (30 min)

### [ ] Fix 1: API Endpoint Path
**File:** `public/index.html` line 406  
**File:** `public/index-v2.html` line 404

**Change:**
```javascript
// FROM:
fetch('/room/create', {

// TO:
fetch('/api/room/create', {
```

**Test:** Click "Mulai Berbagi" button - should not get 404 error

---

### [ ] Fix 2: Add Room Interface Logic
**File:** `public/index.html` after line 447 (before `</script>`)  
**File:** `public/index-v2.html` after line 445 (before `</script>`)

**Action:** Copy entire room interface code from `QUICK_FIX_GUIDE.md` section "Fix 1.2"

**Contains:**
- Room detection on page load
- PIN to room conversion
- Room interface HTML generation
- File upload functionality
- File list display
- Delete file function

**Test:** 
- Create room → should show room interface with PIN
- Upload file → should show in list
- Download file → should work

---

## 🟡 PHASE 2: USER DASHBOARD (30 min)

### [ ] Fix 3: Create User Dashboard
**Action:** Create new file `public/user/dashboard.html`

**Steps:**
```bash
mkdir -p public/user
# Copy code from QUICK_FIX_GUIDE.md section "Fix 2.1"
```

**Test:** Visit `/user/dashboard` → should show dashboard page

---

### [ ] Fix 4: Update User Login Redirect
**File:** `users.go` line 172

**Change:**
```go
// FROM:
http.Redirect(w, r, "/", http.StatusTemporaryRedirect)

// TO:
http.Redirect(w, r, "/user/dashboard", http.StatusTemporaryRedirect)
```

**Test:** Login as user → should redirect to dashboard, not landing page

---

### [ ] Fix 5: Rebuild & Restart
**Commands:**
```bash
# Backup first
cp public/index.html public/index.html.backup
cp users.go users.go.backup

# Rebuild Go application
go build -o file-server

# Restart
pm2 restart file-server
# OR: ./file-server
```

---

## ✅ TESTING CHECKLIST

### Room Creation Flow
- [ ] Go to landing page
- [ ] Click "Mulai Berbagi"
- [ ] Select expiry time
- [ ] Click "Buat Ruangan"
- [ ] Should redirect to `/?room={id}`
- [ ] Should see room interface (not landing page)
- [ ] Should see PIN displayed
- [ ] Should see "Upload File" area

### File Upload Flow
- [ ] In room, click "Pilih File" or drag file
- [ ] Should show upload progress
- [ ] After upload, file should appear in list
- [ ] Click "Download" → file should download
- [ ] Click "Hapus" → file should be deleted

### PIN Join Flow
- [ ] Go to landing page
- [ ] Click "Masuk dengan PIN"
- [ ] Enter valid 6-digit PIN
- [ ] Should load room interface
- [ ] Should see files in that room

### User Login Flow
- [ ] Go to `/user-login.html`
- [ ] Click "Login with Google"
- [ ] Complete OAuth
- [ ] Should redirect to `/user/dashboard`
- [ ] Should see user name
- [ ] Should see stats (even if 0)
- [ ] Click "Logout" → should go back to home

### Admin Login Flow (verify still works)
- [ ] Go to `/admin/login.html`
- [ ] Login with admin email
- [ ] Should redirect to `/admin/dashboard`
- [ ] Should see admin panel

---

## 🔍 VERIFICATION

After all fixes:

### Browser Console Check
```
Open DevTools (F12) → Console tab
Should see NO red errors

If you see errors, check:
- Are all functions defined?
- Are fetch paths correct?
- Any typos in variable names?
```

### Network Tab Check
```
Open DevTools (F12) → Network tab

When creating room, should see:
✅ POST /api/room/create → 200 OK

When uploading file, should see:
✅ POST /api/upload/{roomId} → 200 OK

NOT:
❌ POST /room/create → 404 Not Found
```

### Backend Logs Check
```bash
# Check for errors
pm2 logs file-server --lines 50

# Or if running directly
./file-server
# Watch for errors in console
```

---

## 🐛 TROUBLESHOOTING

### Issue: Room creation still fails
**Check:**
- [ ] Did you change both index.html AND index-v2.html?
- [ ] Is the path exactly `/api/room/create`?
- [ ] Did you hard refresh browser (Ctrl+Shift+R)?
- [ ] Check browser console for errors

### Issue: Room interface doesn't show
**Check:**
- [ ] Did you add the entire room interface code?
- [ ] Is it inside the `<script>` tag?
- [ ] Did you paste it before the closing `</script>`?
- [ ] Check browser console for JavaScript errors

### Issue: User dashboard 404
**Check:**
- [ ] Does `/public/user/dashboard.html` exist?
- [ ] Did you create the `/public/user/` directory?
- [ ] File permissions correct? (644)
- [ ] Did you restart the server after creating file?

### Issue: Still redirects to landing page
**Check:**
- [ ] Did you modify `users.go` line 172?
- [ ] Did you rebuild? `go build -o file-server`
- [ ] Did you restart server?
- [ ] Check backend logs for errors

---

## 📚 REFERENCE DOCS

- **Detailed analysis:** `PATH_ANOMALIES_REPORT.md`
- **Flow diagrams:** `ROUTING_FLOW_DIAGRAM.md`
- **Code snippets:** `QUICK_FIX_GUIDE.md`
- **Overview:** `EXECUTIVE_SUMMARY.md`

---

## 🎯 COMPLETION CRITERIA

All done when:
- [x] Users can create rooms
- [x] Room interface displays properly
- [x] Files can be uploaded
- [x] Files can be downloaded
- [x] PIN join works
- [x] Users get dashboard after login
- [x] No console errors
- [x] No 404 errors in network tab

---

## 📝 NOTES

**Time tracking:**
- Phase 1 started: ___:___
- Phase 1 completed: ___:___
- Phase 2 started: ___:___
- Phase 2 completed: ___:___
- Testing completed: ___:___
- **Total time:** ___ minutes

**Issues encountered:**
```
(Write any problems you faced here)
```

**Additional changes made:**
```
(Note any extra fixes or improvements)
```

---

**Status:** ⬜ Not Started | 🟡 In Progress | ✅ Completed
