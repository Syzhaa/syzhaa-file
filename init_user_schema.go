package main

// Initialize user management schema
func initUserSchema() error {
	schema := `
	-- Users table
	CREATE TABLE IF NOT EXISTS users (
		id TEXT PRIMARY KEY,
		google_id TEXT UNIQUE NOT NULL,
		email TEXT UNIQUE NOT NULL,
		name TEXT NOT NULL,
		avatar_url TEXT,
		status TEXT DEFAULT 'pending',
		approved_by TEXT,
		approved_at DATETIME,
		rejected_reason TEXT,
		storage_limit_mb INTEGER,
		max_file_duration_days INTEGER,
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
		last_login DATETIME,
		FOREIGN KEY(approved_by) REFERENCES admin_users(id)
	);

	-- User stats
	CREATE TABLE IF NOT EXISTS user_stats (
		user_id TEXT PRIMARY KEY,
		total_rooms_created INTEGER DEFAULT 0,
		total_files_uploaded INTEGER DEFAULT 0,
		total_storage_used_mb REAL DEFAULT 0,
		last_upload_at DATETIME,
		FOREIGN KEY(user_id) REFERENCES users(id) ON DELETE CASCADE
	);

	-- User sessions
	CREATE TABLE IF NOT EXISTS user_sessions (
		id TEXT PRIMARY KEY,
		user_id TEXT NOT NULL,
		token_hash TEXT UNIQUE NOT NULL,
		expires_at DATETIME NOT NULL,
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
		FOREIGN KEY(user_id) REFERENCES users(id) ON DELETE CASCADE
	);

	-- System settings
	CREATE TABLE IF NOT EXISTS system_settings (
		key TEXT PRIMARY KEY,
		value TEXT NOT NULL,
		description TEXT,
		updated_at DATETIME DEFAULT CURRENT_TIMESTAMP,
		updated_by TEXT
	);
	`

	_, err := db.Exec(schema)
	if err != nil {
		return err
	}

	// Create indexes
	db.Exec("CREATE INDEX IF NOT EXISTS idx_users_status ON users(status)")
	db.Exec("CREATE INDEX IF NOT EXISTS idx_users_email ON users(email)")
	db.Exec("CREATE INDEX IF NOT EXISTS idx_user_sessions_user ON user_sessions(user_id)")
	db.Exec("CREATE INDEX IF NOT EXISTS idx_rooms_user ON rooms(user_id)")

	// Insert default settings
	db.Exec(`INSERT OR IGNORE INTO system_settings (key, value, description) VALUES
		('default_storage_limit_mb', '5120', 'Default storage limit per user (5GB)'),
		('default_max_duration_days', '7', 'Default max file duration (7 days)'),
		('require_approval', 'true', 'Require admin approval for new users')`)

	// Alter rooms table to add user_id if not exists
	db.Exec("ALTER TABLE rooms ADD COLUMN user_id TEXT")

	return nil
}
