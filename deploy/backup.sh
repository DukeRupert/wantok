#!/bin/bash
# Wantok database backup script
# Run daily via cron: 0 2 * * * /opt/wantok/backup.sh

set -euo pipefail

# Configuration
DB_PATH="${DATABASE_PATH:-/opt/wantok/data/wantok.db}"
BACKUP_DIR="${BACKUP_DIR:-/opt/wantok/backups}"
RETENTION_DAYS="${RETENTION_DAYS:-7}"

# Create backup directory if it doesn't exist
mkdir -p "$BACKUP_DIR"

# Generate backup filename with timestamp
TIMESTAMP=$(date +%Y%m%d_%H%M%S)
BACKUP_FILE="$BACKUP_DIR/wantok_$TIMESTAMP.db"

# Create backup using SQLite's backup command (safe for running database)
sqlite3 "$DB_PATH" ".backup '$BACKUP_FILE'"

# Compress the backup
gzip "$BACKUP_FILE"

echo "Backup created: ${BACKUP_FILE}.gz"

# Remove backups older than retention period
find "$BACKUP_DIR" -name "wantok_*.db.gz" -type f -mtime +$RETENTION_DAYS -delete

echo "Cleanup complete. Retained backups from last $RETENTION_DAYS days."
