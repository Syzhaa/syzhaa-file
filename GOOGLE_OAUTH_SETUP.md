# Google OAuth Setup - Syzhaa File Server

## Step 1: Buat Google OAuth App

1. Buka **Google Cloud Console**: https://console.cloud.google.com/
2. Pilih/bikin project baru
3. Ke **APIs & Services** → **Credentials**
4. Klik **Create Credentials** → **OAuth 2.0 Client ID**
5. Pilih **Application type**: Web application
6. **Name**: Syzhaa File Admin
7. **Authorized redirect URIs**:
   - Add: `https://file.syzhaa.my.id/auth/google/callback`
   - Add: `http://localhost:4006/auth/google/callback` (untuk testing)

8. Klik **Create**
9. Copy **Client ID** dan **Client Secret**

## Step 2: Configure Environment

```bash
cd /www/wwwroot/file-go
cp .env.example .env
nano .env
```

Isi dengan credentials dari Google:
```env
GOOGLE_CLIENT_ID=xxxxxxxxxxxx.apps.googleusercontent.com
GOOGLE_CLIENT_SECRET=GOCSPX-xxxxxxxxxxxxxxxxxxxxxxxx
GOOGLE_REDIRECT_URL=https://file.syzhaa.my.id/auth/google/callback
```

## Step 3: Restart Server

```bash
pm2 restart syzhaa-file-go
pm2 save
```

## Step 4: Test Login

Buka: https://file.syzhaa.my.id/auth/google/login

## API Endpoints

### Admin Endpoints (require session cookie)
- `GET /admin/me` - Current admin info
- `GET /admin/stats` - Stats dashboard
- `GET /admin/api-keys` - List API keys
- `POST /admin/api-keys` - Create API key
- `DELETE /admin/api-keys/{id}` - Delete API key
- `POST /admin/api-keys/{id}/toggle` - Enable/disable key

### API v1 (require Bearer token)
- `POST /api/v1/room/create` - Create room
- `GET /api/v1/room/{id}/link` - Get room link
- `GET /api/v1/room/{id}/files` - List files with metadata
- `GET /api/v1/room/{id}/download-all` - Download as ZIP

### Auth Endpoints
- `GET /auth/google/login` - Initiate Google login
- `GET /auth/google/callback` - OAuth callback
- `POST /auth/logout` - Logout

## API Key Format

API keys format: `sfa_xxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxx`

Example API call:
```bash
curl -X POST https://file.syzhaa.my.id/api/v1/room/create \
  -H "Authorization: Bearer sfa_xxxxxxxxxxxxx" \
  -H "Content-Type: application/json" \
  -d '{"expiry_minutes": 120}'
```
