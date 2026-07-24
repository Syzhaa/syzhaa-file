# Quick Reference - Struktur Baru
**Updated:** 2026-07-23

---

## 📂 Struktur Sekarang

```
/www/wwwroot/file.syzhaa.my.id/
│
├── backend/              (14 MB)
│   ├── *.go             ← 9 file Go
│   ├── file-server      ← Binary executable
│   ├── data/            ← Database SQLite
│   ├── uploads/         ← File yang diupload user
│   └── chunks/          ← Temporary chunks
│
└── frontend/             (124 KB)
    ├── *.html           ← Landing page, login, dll
    ├── *.js             ← JavaScript files
    └── admin/           ← Admin dashboard
```

---

## ⚡ Command Cepat

### Restart Aplikasi
```bash
pm2 restart syzhaa-file
```

### Lihat Logs
```bash
pm2 logs syzhaa-file
```

### Rebuild (setelah edit Go code)
```bash
cd /www/wwwroot/file.syzhaa.my.id/backend
/usr/local/go/bin/go build -o file-server
pm2 restart syzhaa-file
```

### Edit Frontend (HTML/JS)
```bash
# Edit langsung, tidak perlu rebuild
vim frontend/index.html
# Refresh browser saja
```

---

## ✅ Status

- **Backend:** ✅ Running di port 4006
- **Frontend:** ✅ Served via backend
- **Database:** ✅ backend/data/files.db
- **PM2:** ✅ Saved & auto-restart enabled

---

## 🐛 Masalah yang Masih Ada

**PENTING:** Aplikasi jalan tapi ada **3 bug kritikal** yang sudah saya analisis:

1. ❌ API endpoint salah (`/room/create` → harus `/api/room/create`)
2. ❌ Room interface tidak ada (user tidak bisa upload file)
3. ❌ User dashboard tidak ada

**Solusi lengkap ada di:** `QUICK_FIX_GUIDE.md` (~60 menit)

---

## 📚 Dokumentasi

- `RESTRUCTURE_COMPLETE.md` - Detail restrukturisasi (ini baru saja)
- `QUICK_FIX_GUIDE.md` - Perbaikan bug (BACA INI NEXT!)
- `PATH_ANOMALIES_REPORT.md` - Analisis lengkap masalah
- `TODO.md` - Checklist perbaikan

---

**Server:** file.syzhaa.my.id  
**Port:** 4006  
**Status:** 🟢 ONLINE
