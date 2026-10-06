package main

import (
	"database/sql"
	"os"
)

func initDB() error {
	var err error
	os.MkdirAll("./data", 0755)
	db, err = sql.Open("sqlite3", DBPath)
	if err != nil {
		return err
	}

	// Enable foreign key constraints
	_, err = db.Exec("PRAGMA foreign_keys = ON")
	if err != nil {
		return err
	}

	schema := `
	CREATE TABLE IF NOT EXISTS rooms (
		id TEXT PRIMARY KEY,
		pin TEXT UNIQUE,
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
		expires_at DATETIME NOT NULL
	);
	
	CREATE TABLE IF NOT EXISTS files (
		id TEXT PRIMARY KEY,
		room_id TEXT NOT NULL,
		folder_id TEXT,
		filename TEXT NOT NULL,
		original_name TEXT NOT NULL,
		mimetype TEXT NOT NULL,
		size INTEGER NOT NULL,
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
		downloads INTEGER DEFAULT 0,
		salt TEXT,
		nonce TEXT,
		is_encrypted INTEGER DEFAULT 0,
		original_size INTEGER,
		FOREIGN KEY(room_id) REFERENCES rooms(id) ON DELETE CASCADE,
		FOREIGN KEY(folder_id) REFERENCES folders(id) ON DELETE CASCADE
	);

	CREATE TABLE IF NOT EXISTS folders (
		id TEXT PRIMARY KEY,
		room_id TEXT NOT NULL,
		parent_id TEXT,
		name TEXT NOT NULL,
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
		FOREIGN KEY(room_id) REFERENCES rooms(id) ON DELETE CASCADE,
		FOREIGN KEY(parent_id) REFERENCES folders(id) ON DELETE CASCADE
	);

	CREATE TABLE IF NOT EXISTS admin_users (
		id TEXT PRIMARY KEY,
		email TEXT UNIQUE NOT NULL,
		name TEXT NOT NULL,
		password_hash TEXT NOT NULL,
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
		last_login DATETIME,
		is_super_admin INTEGER DEFAULT 1
	);

	CREATE TABLE IF NOT EXISTS api_keys (
		id TEXT PRIMARY KEY,
		key_hash TEXT UNIQUE NOT NULL,
		admin_id TEXT NOT NULL,
		name TEXT NOT NULL,
		expires_at DATETIME,
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
		last_used_at DATETIME,
		is_active INTEGER DEFAULT 1,
		FOREIGN KEY(admin_id) REFERENCES admin_users(id) ON DELETE CASCADE
	);

	CREATE TABLE IF NOT EXISTS admin_sessions (
		id TEXT PRIMARY KEY,
		admin_id TEXT NOT NULL,
		token_hash TEXT UNIQUE NOT NULL,
		expires_at DATETIME NOT NULL,
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
		FOREIGN KEY(admin_id) REFERENCES admin_users(id) ON DELETE CASCADE
	);
	`
	_, err = db.Exec(schema)
	if err != nil {
		return err
	}

	// Create indexes
	db.Exec("CREATE INDEX IF NOT EXISTS idx_api_keys_admin ON api_keys(admin_id)")
	db.Exec("CREATE INDEX IF NOT EXISTS idx_sessions_admin ON admin_sessions(admin_id)")
	db.Exec("CREATE INDEX IF NOT EXISTS idx_api_keys_active ON api_keys(is_active, expires_at)")

	return nil
}
