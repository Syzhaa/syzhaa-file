#!/bin/bash
# Check for new AmbilFile API key access requests since last check.
# Prints new requests (name | email | api_requested_at), one per line.
# Updates the watermark file on every run.
set -u

DB="$HOME/workspace/syzhaa-file/backend/data/files.db"
WM_DIR="$HOME/workspace/syzhaa-file/hidden_files/user-reg-watch"
WM="$WM_DIR/last_check"
mkdir -p "$WM_DIR"

NOW=$(date -u +"%Y-%m-%dT%H:%M:%SZ")
if [ -f "$WM" ]; then
    LAST=$(cat "$WM")
else
    # First run: only look back 5 minutes to avoid spamming old rows
    LAST=$(date -u -d "5 minutes ago" +"%Y-%m-%dT%H:%M:%SZ")
fi

if [ ! -f "$DB" ]; then
    echo "$NOW" > "$WM"
    exit 0
fi

sqlite3 -separator '|' "$DB" \
    "SELECT name, email, api_requested_at FROM users WHERE api_requested_at IS NOT NULL AND api_requested_at > '$LAST' AND COALESCE(api_approved, 0) = 0 ORDER BY api_requested_at ASC;"

echo "$NOW" > "$WM"
