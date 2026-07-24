# Running Web Applications - System Report
**Date:** 2026-07-23  
**Server:** /www/wwwroot/file.syzhaa.my.id  

---

## 🖥️ Server Summary

**Total Web Applications Running:** 6 applications  
**Go Applications in This Folder:** 1 (file-server)  
**Node.js Applications (Other Folders):** 5  

---

## ✅ Go Application (THIS FOLDER)

### **Syzhaa File Server** 
**Status:** 🟢 ONLINE

```
Name:           syzhaa-file
Location:       /www/wwwroot/file.syzhaa.my.id/
Binary:         ./file-server (13.4 MB)
PID:            1288988
Port:           4006
URL:            https://file.syzhaa.my.id
Protocol:       HTTP/HTTPS
Uptime:         16 minutes
Restarts:       20 times
Memory:         11.8 MB
PM2 ID:         1
```

**Composed of 9 Go files:**
```
main.go               17 KB    Main application & routes
auth.go               7.3 KB   Admin Google OAuth
users.go              9.2 KB   User authentication & registration
admin_api.go          7.0 KB   Admin dashboard API
admin_users.go        7.1 KB   Admin user management
api_keys.go           6.2 KB   API key generation & management
middleware.go         1.5 KB   CORS & clean URL middleware
user_middleware.go    1.3 KB   User session validation
init_user_schema.go   2.2 KB   Database schema initialization
```

**Total Lines of Code:** ~1,500 lines

**Database:**
```
Type:     SQLite3
Location: ./data/files.db
Tables:   - rooms (file sharing rooms)
          - files (uploaded files)
          - admin_users (admin accounts)
          - users (regular user accounts)
          - api_keys (API access tokens)
          - admin_sessions (admin login sessions)
          - user_sessions (user login sessions)
          - system_settings (app configuration)
```

**Features:**
- ✅ File room creation with PIN
- ✅ Multi-file upload with chunking (5MB chunks)
- ✅ File download with counter
- ✅ Auto-cleanup expired rooms
- ✅ Admin panel with Google OAuth
- ✅ User registration with approval system
- ✅ API key management
- ✅ RESTful API (v1)

**Routes Summary:**
```
Public:     7 routes (room create, upload, download, etc.)
Admin:      13 routes (user management, API keys, stats)
User:       1 route (profile info)
API v1:     4 routes (programmatic access)
OAuth:      4 routes (Google authentication)
Static:     File server for /public/*
```

**Environment Variables:**
```
GOOGLE_CLIENT_ID:      974513868394-cil*****.apps.googleusercontent.com
GOOGLE_CLIENT_SECRET:  GOCSPX-l1BTfjzK****
GOOGLE_REDIRECT_URL:   https://file.syzhaa.my.id/auth/google/callback
BASE_URL:              https://file.syzhaa.my.id
ADMIN_EMAILS:          syzhaadigital@gmail.com
```

---

## 🌐 Other Web Applications (Different Folders)

### 1. **Cam Backend** (Node.js)
```
Port:       4005
Location:   /www/wwwroot/cam.syzhaa.my.id/
PM2 ID:     4
Memory:     98.5 MB
Status:     🟢 Online
```

### 2. **Cam Frontend** (Node.js)
```
PM2 ID:     5
Location:   /www/wwwroot/cam.syzhaa.my.id/
Memory:     75.4 MB
Status:     🟢 Online
```

### 3. **GitHub Panel** (Node.js)
```
PM2 ID:     6
Location:   /www/wwwroot/github.syzhaa.my.id/
Memory:     75.3 MB
Status:     🟢 Online
```

### 4. **Quizz Backend** (Node.js)
```
PM2 ID:     2
Location:   /www/wwwroot/quizz.syzhaa.my.id/
Memory:     95.0 MB
Status:     🟢 Online
```

### 5. **Serbaapps Backend** (Node.js)
```
Port:       5000
Location:   /www/wwwroot/serbaapps.biz.id/
PM2 ID:     3
Memory:     103.2 MB
Status:     🟢 Online
```

---

## 📁 Directory Structure

```
/www/wwwroot/
├─ file.syzhaa.my.id/               ← YOU ARE HERE (Go Application)
│  ├─ file-server                   ← Compiled binary (13.4 MB)
│  ├─ main.go                       ← Main application
│  ├─ auth.go                       ← Admin authentication
│  ├─ users.go                      ← User authentication
│  ├─ admin_api.go                  ← Admin API
│  ├─ admin_users.go                ← User management
│  ├─ api_keys.go                   ← API key management
│  ├─ middleware.go                 ← HTTP middleware
│  ├─ user_middleware.go            ← User middleware
│  ├─ init_user_schema.go           ← DB schema
│  ├─ go.mod, go.sum                ← Go dependencies
│  ├─ ecosystem.config.js           ← PM2 config (has wrong path!)
│  ├─ public/                       ← Frontend files
│  │  ├─ index.html
│  │  ├─ user-login.html
│  │  ├─ admin/
│  │  │  ├─ dashboard.html
│  │  │  ├─ login.html
│  │  │  └─ users.html
│  │  └─ (other HTML files)
│  ├─ data/                         ← SQLite database
│  │  └─ files.db
│  ├─ uploads/                      ← Uploaded files storage
│  └─ chunks/                       ← Temporary upload chunks
│
├─ cam.syzhaa.my.id/                ← Node.js Application
├─ github.syzhaa.my.id/             ← Node.js Application
├─ quizz.syzhaa.my.id/              ← Node.js Application
├─ serbaapps.biz.id/                ← Node.js Application
└─ (other directories)
```

---

## 🔧 How It Works

### Compilation & Execution
```bash
# All 9 Go files are compiled into ONE binary
go build -o file-server

# Binary includes:
# - All Go code
# - All route handlers
# - Database logic
# - Authentication
# - Middleware
# - API endpoints

# When you run:
./file-server

# It starts ONE web server on port 4006
# serving ALL routes (public, admin, user, API)
```

### PM2 Management
```bash
# Currently running as:
pm2 start file-server --name syzhaa-file

# Can be managed with:
pm2 restart syzhaa-file
pm2 stop syzhaa-file
pm2 logs syzhaa-file
pm2 delete syzhaa-file
```

---

## ⚠️ Configuration Issue Found

**ecosystem.config.js has WRONG path:**
```javascript
cwd: '/www/wwwroot/file-go',  // ← WRONG! This path doesn't exist
```

**Should be:**
```javascript
cwd: '/www/wwwroot/file.syzhaa.my.id',  // ← CORRECT
```

**However:** The app is currently running correctly from the right location because PM2 is using the actual binary path, not the ecosystem.config.js.

**To fix (optional):**
```bash
# Stop current process
pm2 stop syzhaa-file
pm2 delete syzhaa-file

# Edit ecosystem.config.js line 5:
# Change: cwd: '/www/wwwroot/file-go',
# To:     cwd: '/www/wwwroot/file.syzhaa.my.id',

# Restart with config
pm2 start ecosystem.config.js
pm2 save
```

---

## 📊 Port Allocation

```
Port 4005:  cam-backend (Node.js)
Port 4006:  file-server (Go) ← THIS FOLDER
Port 5000:  serbaapps-backend (Node.js)
```

---

## 🎯 Summary

**In This Folder (`/www/wwwroot/file.syzhaa.my.id/`):**
- ✅ **1 Go application** (file-server)
- ✅ Runs on port 4006
- ✅ Compiled from 9 Go source files
- ✅ Serves file sharing platform
- ✅ Currently ONLINE and running

**In Other Folders:**
- ✅ 5 Node.js applications
- ✅ Running on different ports
- ✅ All managed by PM2

**Answer to Your Question:**
> "Bisa kamu cek aplikasi yang web jalan apa aja ini di apke file go apakah di folder ini apa bukan"

**Answer:** 
Di folder ini (`/www/wwwroot/file.syzhaa.my.id/`) hanya ada **SATU aplikasi Go** yang jalan, yaitu:
- **`file-server`** (port 4006)
- Binary hasil compile dari 9 file Go
- Melayani semua endpoint (public, admin, user, API)
- Ini adalah aplikasi file sharing yang sedang kita analisis

Tidak ada aplikasi Go lain di folder ini. Semua aplikasi lain (cam, github, quizz, serbaapps) adalah aplikasi Node.js yang ada di folder berbeda.

---

**Last Updated:** 2026-07-23 07:02 UTC  
**System Uptime:** 27 minutes  
**Server Load:** CPU 14%, RAM 47.4%
