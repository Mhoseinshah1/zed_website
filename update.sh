#!/bin/bash
set -euo pipefail

SERVICE="zedproxy"
INSTALL_DIR="/opt/zedproxy"
REPO_DIR="$(cd "$(dirname "$0")" && pwd)"
DATA_DIR="$INSTALL_DIR/data"
BACKUP_DIR="$DATA_DIR/backups"
DB_FILE="$DATA_DIR/zedproxy.db"
UPLOADS_DIR="$INSTALL_DIR/uploads"
ENV_FILE="$INSTALL_DIR/.env"

echo "==> ZedProxy Update Script"
echo "    Install dir: $INSTALL_DIR"
echo "    Source dir:  $REPO_DIR"

# Backup DB before update
if [ -f "$DB_FILE" ]; then
    BACKUP_NAME="pre-update-$(date +%Y%m%d-%H%M%S).db"
    mkdir -p "$BACKUP_DIR"
    cp "$DB_FILE" "$BACKUP_DIR/$BACKUP_NAME"
    echo "==> DB backed up to $BACKUP_DIR/$BACKUP_NAME"
fi

# Stop service
echo "==> Stopping service..."
systemctl stop "$SERVICE" || true

# Build new binary
echo "==> Building..."
cd "$REPO_DIR"
CGO_ENABLED=1 go build -o zedproxy . 2>&1

# Copy binary
cp zedproxy "$INSTALL_DIR/zedproxy"
chmod +x "$INSTALL_DIR/zedproxy"

# Copy templates and static (preserve uploads)
echo "==> Copying templates and static files..."
rsync -a --delete "$REPO_DIR/templates/" "$INSTALL_DIR/templates/"
rsync -a --delete \
    --exclude 'uploads/' \
    "$REPO_DIR/static/" "$INSTALL_DIR/static/" 2>/dev/null || true

echo "==> Starting service..."
systemctl start "$SERVICE"
systemctl status "$SERVICE" --no-pager -l

echo ""
echo "✓ Update complete!"
