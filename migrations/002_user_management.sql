-- User Management Schema
-- Users are regular app users (not admins)

CREATE TABLE IF NOT EXISTS users (
  id TEXT PRIMARY KEY,
  google_id TEXT UNIQUE NOT NULL,
  email TEXT UNIQUE NOT NULL,
  name TEXT NOT NULL,
  avatar_url TEXT,
  
  -- Approval status
  status TEXT DEFAULT 'pending', -- pending, approved, rejected, suspended
  approved_by TEXT, -- admin_id who approved
  approved_at DATETIME,
  rejected_reason TEXT,
  
  -- Quotas (NULL = use system default)
  storage_limit_mb INTEGER, -- Max storage in MB
  max_file_duration_days INTEGER, -- Max days file can be active
  
  created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
  last_login DATETIME,
  
  FOREIGN KEY(approved_by) REFERENCES admin_users(id)
);

-- User statistics
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

-- Default settings
INSERT OR IGNORE INTO system_settings (key, value, description) VALUES
  ('default_storage_limit_mb', '5120', 'Default storage limit per user (5GB)'),
  ('default_max_duration_days', '7', 'Default max file duration (7 days)'),
  ('require_approval', 'true', 'Require admin approval for new users');

-- Indexes
CREATE INDEX IF NOT EXISTS idx_users_status ON users(status);
CREATE INDEX IF NOT EXISTS idx_users_email ON users(email);
CREATE INDEX IF NOT EXISTS idx_user_sessions_user ON user_sessions(user_id);

-- Update rooms table to track owner
ALTER TABLE rooms ADD COLUMN user_id TEXT;
CREATE INDEX IF NOT EXISTS idx_rooms_user ON rooms(user_id);
