-- Admin Users Table
CREATE TABLE IF NOT EXISTS admin_users (
  id TEXT PRIMARY KEY,
  google_id TEXT UNIQUE NOT NULL,
  email TEXT UNIQUE NOT NULL,
  name TEXT NOT NULL,
  avatar_url TEXT,
  created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
  last_login DATETIME
);

-- API Keys Table
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

-- Admin Sessions Table
CREATE TABLE IF NOT EXISTS admin_sessions (
  id TEXT PRIMARY KEY,
  admin_id TEXT NOT NULL,
  token_hash TEXT UNIQUE NOT NULL,
  expires_at DATETIME NOT NULL,
  created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
  FOREIGN KEY(admin_id) REFERENCES admin_users(id) ON DELETE CASCADE
);

-- Index for performance
CREATE INDEX IF NOT EXISTS idx_api_keys_admin ON api_keys(admin_id);
CREATE INDEX IF NOT EXISTS idx_sessions_admin ON admin_sessions(admin_id);
CREATE INDEX IF NOT EXISTS idx_api_keys_active ON api_keys(is_active, expires_at);
