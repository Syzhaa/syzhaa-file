# Syzhaa File Server (Go)

High-performance file sharing service with **Admin Panel**, **API Keys**, and **Media Preview Cards**.

## 🚀 Features

### Core Features
- **Chunked Upload**: Handle large files (GB+) via 10MB chunks
- **Room-based Sharing**: Create temporary rooms with PIN access
- **Auto Cleanup**: Expired files deleted automatically (every 60s)
- **Orphan Detection**: Cleans up failed uploads (every 10min)
- **Media Preview**: Card layout with image thumbnails
- **Concurrent Safe**: Goroutines for background workers
- **Low Memory**: ~9MB RAM footprint vs ~80MB for Node.js

### Admin Features (NEW)
- **Google OAuth Login**: Secure admin authentication
- **API Key Management**: Create, disable, delete API keys with expiry
- **Dashboard**: Real-time stats (rooms, files, storage)
- **Session Management**: 7-day admin sessions

### API v1 (NEW)
- **Programmatic Access**: RESTful API with Bearer token auth
- **Room Management**: Create rooms, get links, list files
- **Bulk Download**: Download all room files as ZIP

## 🛠 Tech Stack

- **Go 1.25**: Compiled binary for max performance
- **SQLite3**: Zero-config embedded database
- **Gorilla Mux**: Fast HTTP router
- **OAuth2**: Google authentication
- **PM2**: Process manager

## 📊 Performance

- Memory: **9MB** (vs 83MB Node.js = **9.2x improvement**)
- Binary size: 14MB (includes all deps + OAuth)
- Concurrency: Native goroutines
- Startup: <100ms

## 🔐 Admin Setup

See [GOOGLE_OAUTH_SETUP.md](GOOGLE_OAUTH_SETUP.md) for complete setup guide.

### Quick Start

1. **Get Google OAuth credentials** from [Google Cloud Console](https://console.cloud.google.com/)
2. **Configure environment**:
   ```bash
   cp .env.example .env
   nano .env  # Add your GOOGLE_CLIENT_ID and GOOGLE_CLIENT_SECRET
   ```
3. **Restart server**:
   ```bash
   pm2 restart syzhaa-file-go --update-env
   ```
4. **Login**: Visit `https://file.syzhaa.my.id/admin/login.html`

## 📡 API Endpoints

### Public Endpoints
- `POST /api/room/create` - Create room (public)
- `POST /api/room/pin` - Access by PIN
- `GET /api/room/{id}` - Room info
- `POST /api/upload/{roomId}` - Upload chunk
- `GET /d/{id}` - Download file
- `DELETE /api/file/{id}` - Delete file

### Admin Endpoints (require session cookie)
- `GET /admin/me` - Current admin info
- `GET /admin/stats` - Dashboard stats
- `GET /admin/api-keys` - List API keys
- `POST /admin/api-keys` - Create API key
- `DELETE /admin/api-keys/{id}` - Delete API key
- `POST /admin/api-keys/{id}/toggle` - Enable/disable key

### API v1 Endpoints (require Bearer token)
- `POST /api/v1/room/create` - Create room
- `GET /api/v1/room/{id}/link` - Get room + PIN links
- `GET /api/v1/room/{id}/files` - List files with metadata
- `GET /api/v1/room/{id}/download-all` - Download all as ZIP

## 🔑 API Usage Example

```bash
# Create API key via admin dashboard first, then:

# Create room
curl -X POST https://file.syzhaa.my.id/api/v1/room/create \
  -H "Authorization: Bearer sfa_xxxxxxxxxxxxx" \
  -H "Content-Type: application/json" \
  -d '{"expiry_minutes": 120}'

# Get room link
curl https://file.syzhaa.my.id/api/v1/room/{room_id}/link \
  -H "Authorization: Bearer sfa_xxxxxxxxxxxxx"

# Download all files as ZIP
curl https://file.syzhaa.my.id/api/v1/room/{room_id}/download-all \
  -H "Authorization: Bearer sfa_xxxxxxxxxxxxx" \
  -o files.zip
```

## 📦 Deployment

```bash
go build -o file-server
pm2 start ecosystem.config.js
pm2 save
```

## 🎨 Frontend Features

- **Media Cards**: Grid layout with image previews
- **View Toggle**: Switch between card/list view
- **Responsive**: Mobile-friendly design
- **Drag & Drop**: Upload files by dragging
- **QR Code**: Share rooms via QR

## 🔒 Security

- Google OAuth for admin access
- SHA-256 hashed API keys
- HttpOnly secure session cookies
- CORS middleware
- API key expiry support
- Rate limiting ready (TODO)

## 📝 License

MIT

## 🚀 Live Demo

**Public Site**: https://file.syzhaa.my.id  
**Admin Login**: https://file.syzhaa.my.id/admin/login.html
