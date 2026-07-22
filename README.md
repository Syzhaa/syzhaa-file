# Syzhaa File Server (Go)

High-performance file sharing service with chunked upload support, built in Go.

## Features

- **Chunked Upload**: Handle large files (GB+) via 10MB chunks
- **Room-based Sharing**: Create temporary rooms with PIN access
- **Auto Cleanup**: Expired files deleted automatically (every 60s)
- **Orphan Detection**: Cleans up failed uploads (every 10min)
- **Concurrent Safe**: Goroutines for background workers
- **Low Memory**: ~8MB RAM footprint vs ~80MB for Node.js version

## Tech Stack

- **Go 1.23.5**: Compiled binary for max performance
- **SQLite3**: Zero-config embedded database
- **Gorilla Mux**: Fast HTTP router
- **PM2**: Process manager

## Performance

- Memory: 8MB (vs 83MB Node.js)
- Binary size: 12MB (includes all deps)
- Concurrency: Native goroutines
- Startup: <100ms

## API Endpoints

- `POST /api/room/create` - Create room
- `POST /api/room/pin` - Access by PIN
- `GET /api/room/{id}` - Room info
- `POST /api/upload/{roomId}` - Upload chunk
- `GET /d/{id}` - Download file
- `DELETE /api/file/{id}` - Delete file

## Deployment

```bash
go build -o file-server
pm2 start ecosystem.config.js
```

## Port

Default: `4006`
