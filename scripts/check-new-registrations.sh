#!/bin/bash
# Check for new AmbilFile user registrations AND new API-key access requests
# since last check.
# Prints lines, one per event:
#   REG|nama|email|waktu_daftar        (pendaftar baru)
#   API|nama|email|waktu_request       (permintaan API key baru, belum di-approve)
# Watermarks are maintained per event type.
set -u

DB="$HOME/workspace/syzhaa-file/backend/data/files.db"
WM_DIR="$HOME/workspace/syzhaa-file/hidden_files/user-reg-watch"
WM_REG="$WM_DIR/last_check"
WM_API="$WM_DIR/last_api_check"
mkdir -p "$WM_DIR"

NOW=$(date -u +"%Y-%m-%dT%H:%M:%SZ")
if [ -f "$WM_REG" ]; then LAST_REG=$(cat "$WM_REG"); else LAST_REG=$(date -u -d "5 minutes ago" +"%Y-%m-%dT%H:%M:%SZ"); fi
if [ -f "$WM_API" ]; then LAST_API=$(cat "$WM_API"); else LAST_API=$(date -u -d "5 minutes ago" +"%Y-%m-%dT%H:%M:%SZ"); fi

if [ -f "$DB" ]; then
    sqlite3 -separator '|' "$DB" \
        "SELECT 'REG', name, email, created_at FROM users WHERE created_at > '$LAST_REG' ORDER BY created_at ASC;"
    sqlite3 -separator '|' "$DB" \
        "SELECT 'API', name, email, api_requested_at FROM users WHERE api_requested_at > '$LAST_API' AND COALESCE(api_approved,0) = 0 ORDER BY api_requested_at ASC;"
fi

echo "$NOW" > "$WM_REG"
echo "$NOW" > "$WM_API"
