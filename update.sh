#!/usr/bin/env bash
set -euo pipefail

# =====================================================
#  ZedProxy Website Updater
#  Tested on Ubuntu 20.04 / 22.04 / 24.04
#  Usage: sudo bash /opt/zedproxy/update.sh
# =====================================================

export LC_ALL=C.UTF-8
export LANG=C.UTF-8

REPO_URL="https://github.com/mhoseinshah1/zed_website.git"
BRANCH="main"
INSTALL_DIR="/opt/zedproxy"
SERVICE_NAME="zedproxy"
GO_VERSION="1.22.4"
APP_PORT="8080"
TIMESTAMP="$(date +%Y%m%d-%H%M%S)"
BUILD_DIR="/tmp/zedproxy-update-${TIMESTAMP}"
BACKUP_DIR="${INSTALL_DIR}/data/backups"
LOG_FILE="${INSTALL_DIR}/logs/update-${TIMESTAMP}.log"
DB_FILE="${INSTALL_DIR}/data/zedproxy.db"
ENV_FILE="${INSTALL_DIR}/.env"
UPLOADS_DIR="${INSTALL_DIR}/static/uploads"
OLD_BINARY="${INSTALL_DIR}/zedproxy"
NEW_BINARY="${BUILD_DIR}/zedproxy_new"

# ── Colors ─────────────────────────────────────────
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
CYAN='\033[0;36m'
WHITE='\033[1;37m'
BLUE='\033[0;34m'
NC='\033[0m'

info()    { echo -e "${GREEN}[✓]${NC} $1"; }
warn()    { echo -e "${YELLOW}[!]${NC} $1"; }
error()   { echo -e "${RED}[✗]${NC} $1"; exit 1; }
step()    { echo -e "\n${WHITE}━━━ $1 ━━━${NC}"; }
detail()  { echo -e "    ${BLUE}->  ${NC}$1"; }

tg_notify() {
  # Best-effort Telegram notification — never allowed to fail the update
  local title="$1"
  local msg="${2:-}"
  local cat="${3:-updates}"
  "${INSTALL_DIR}/zedproxy" \
    --db="${DB_FILE}" \
    --telegram-notify-title="$title" \
    --telegram-notify-msg="$msg" \
    --telegram-notify-cat="$cat" 2>/dev/null || true
}

# ── Cleanup trap ────────────────────────────────────
# Called on EXIT; only removes the temp build dir.
# Never touches INSTALL_DIR data.
cleanup() {
  local exit_code=$?
  if [[ -d "$BUILD_DIR" ]]; then
    rm -rf "$BUILD_DIR"
    detail "Temporary build directory removed: $BUILD_DIR"
  fi
  if [[ $exit_code -ne 0 ]]; then
    echo -e "\n${RED}━━━ Error Recovery ━━━${NC}"
    warn "An error occurred. Attempting to restart the previous service..."
    systemctl start "$SERVICE_NAME" 2>/dev/null || true
    sleep 2
    if systemctl is-active --quiet "$SERVICE_NAME"; then
      info "Previous service restarted successfully"
    else
      warn "Service failed to restart. Recent logs:"
      journalctl -u "$SERVICE_NAME" -n 50 --no-pager 2>/dev/null || true
    fi
  fi
}
trap cleanup EXIT

setup_logging() {
  mkdir -p "${INSTALL_DIR}/logs"
  exec > >(tee -a "$LOG_FILE") 2>&1
  echo "=== ZedProxy Update Log: $TIMESTAMP ==="
}

banner() {
  echo -e "${CYAN}"
  echo "╔══════════════════════════════════════════╗"
  echo "║       ZedProxy Website Updater           ║"
  echo "║   Secure update for ZedProxy website     ║"
  echo "╚══════════════════════════════════════════╝"
  echo -e "${NC}"
  echo -e "  Time:       ${WHITE}${TIMESTAMP}${NC}"
  echo -e "  Repository: ${CYAN}${REPO_URL}${NC}"
  echo -e "  Branch:     ${CYAN}${BRANCH}${NC}"
  echo -e "  Directory:  ${CYAN}${INSTALL_DIR}${NC}"
  echo ""
}

# ── Pre-flight checks ───────────────────────────────
check_root() {
  if [[ $EUID -ne 0 ]]; then
    echo -e "${RED}[✗]${NC} This script must be run as root."
    echo -e "    Run: ${WHITE}sudo bash /opt/zedproxy/update.sh${NC}"
    exit 1
  fi
}

check_installed() {
  if [[ ! -d "$INSTALL_DIR" ]]; then
    echo -e "${RED}[✗]${NC} ZedProxy is not installed."
    echo -e "    Please run install.sh first."
    exit 1
  fi
  if [[ ! -f "${OLD_BINARY}" ]]; then
    echo -e "${RED}[✗]${NC} ZedProxy binary not found: ${OLD_BINARY}"
    echo -e "    Please run install.sh first."
    exit 1
  fi
}

check_env() {
  if [[ ! -f "$ENV_FILE" ]]; then
    warn ".env file not found: $ENV_FILE"
    warn "The service requires SESSION_SECRET to run."
    warn "Make sure .env exists or the SESSION_SECRET variable is set."
    exit 1
  fi
  info ".env file found"
}

check_go() {
  export PATH="/usr/local/go/bin:/usr/local/bin:$PATH"
  if ! command -v go &>/dev/null; then
    error "Go not found. Please run install.sh first to install Go."
  fi
  local go_ver
  go_ver="$(go version | awk '{print $3}')"
  info "Go available: ${go_ver}"
}

check_git() {
  if ! command -v git &>/dev/null; then
    error "git not found. Run: apt-get install -y git"
  fi
}

# ── Backup ──────────────────────────────────────────
create_backups() {
  step "Creating backups"
  mkdir -p "$BACKUP_DIR"

  local zip_backup="${BACKUP_DIR}/zedproxy-pre-update-${TIMESTAMP}.zip"
  local tmp_zip_dir="/tmp/zedproxy-backup-${TIMESTAMP}"
  mkdir -p "$tmp_zip_dir"

  # Copy database (never include secrets in ZIP)
  if [[ -f "$DB_FILE" ]]; then
    cp "$DB_FILE" "$tmp_zip_dir/zedproxy.db"
  else
    warn "Database file not found: $DB_FILE — skipped from backup"
  fi

  # Copy binary
  if [[ -f "$OLD_BINARY" ]]; then
    cp "$OLD_BINARY" "$tmp_zip_dir/zedproxy"
  fi

  # Copy templates
  if [[ -d "${INSTALL_DIR}/templates" ]]; then
    cp -r "${INSTALL_DIR}/templates" "$tmp_zip_dir/templates"
  fi

  # Copy static (exclude uploads — user content)
  if [[ -d "${INSTALL_DIR}/static" ]]; then
    rsync -a --exclude 'uploads/' "${INSTALL_DIR}/static/" "$tmp_zip_dir/static/"
  fi

  # Create ZIP (no .env, no tokens, no secrets)
  if command -v zip &>/dev/null; then
    (cd "$tmp_zip_dir" && zip -r "$zip_backup" . -x "*.env" 2>/dev/null) || true
  elif command -v python3 &>/dev/null; then
    python3 -c "
import zipfile, os, sys
src = sys.argv[1]; dst = sys.argv[2]
with zipfile.ZipFile(dst, 'w', zipfile.ZIP_DEFLATED) as z:
    for root, dirs, files in os.walk(src):
        for f in files:
            fp = os.path.join(root, f)
            z.write(fp, os.path.relpath(fp, src))
" "$tmp_zip_dir" "$zip_backup"
  else
    warn "Neither zip nor python3 found — using tar fallback"
    tar -czf "${zip_backup%.zip}.tar.gz" -C "$tmp_zip_dir" . 2>/dev/null || true
    zip_backup="${zip_backup%.zip}.tar.gz"
  fi

  rm -rf "$tmp_zip_dir"

  if [[ -f "$zip_backup" ]]; then
    chmod 600 "$zip_backup"
    chown root:root "$zip_backup"
    info "Backup created: $zip_backup"
    detail "Size: $(du -sh "$zip_backup" | cut -f1)"
  else
    warn "Backup creation may have failed — check $BACKUP_DIR"
  fi

  # Separate .env backup (root-only, never in ZIP)
  local env_backup="${BACKUP_DIR}/.env-pre-update-${TIMESTAMP}"
  if [[ -f "$ENV_FILE" ]]; then
    cp "$ENV_FILE" "$env_backup"
    chmod 600 "$env_backup"
    chown root:root "$env_backup"
    info ".env backup: $env_backup"
  fi
}

# ── Clone ────────────────────────────────────────────
notify_update_start() {
  tg_notify "🔄 Update started" "Update process started on $(hostname)" "updates"
}

notify_update_done() {
  tg_notify "✅ Update completed" "ZedProxy updated successfully on $(hostname)" "updates"
}

notify_update_fail() {
  tg_notify "❌ Update failed" "Update failed on $(hostname). Check logs: ${LOG_FILE}" "critical_alerts"
}

clone_repo() {
  step "Fetching latest code from GitHub"
  detail "Repository: $REPO_URL"
  detail "Branch: $BRANCH"
  detail "Build directory: $BUILD_DIR"

  git clone --depth=1 --branch "$BRANCH" "$REPO_URL" "$BUILD_DIR"
  info "Code fetched successfully"
  detail "Commit: $(git -C "$BUILD_DIR" log --oneline -1)"
}

# ── Build ─────────────────────────────────────────────
build_binary() {
  step "Building Go binary"
  export PATH="/usr/local/go/bin:/usr/local/bin:$PATH"
  cd "$BUILD_DIR"

  detail "go mod download..."
  go mod download 2>&1 | sed 's/^/    /'

  detail "CGO_ENABLED=1 go build..."
  _VER="$(git -C "$BUILD_DIR" describe --tags --always 2>/dev/null || echo 'unknown')"
  _COMMIT="$(git -C "$BUILD_DIR" rev-parse --short HEAD 2>/dev/null || echo 'unknown')"
  _DATE="$(date -u +%Y%m%dT%H%M%SZ)"
  CGO_ENABLED=1 go build \
    -ldflags="-s -w -X main.Version=${_VER} -X main.GitCommit=${_COMMIT} -X main.BuildDate=${_DATE}" \
    -o "$NEW_BINARY" .

  if [[ ! -f "$NEW_BINARY" ]]; then
    error "Build failed: new binary was not created"
  fi

  info "Build successful"
  detail "Binary size: $(du -sh "$NEW_BINARY" | cut -f1)"
}

# ── Stop service ──────────────────────────────────────
stop_service() {
  step "Stopping service"
  systemctl stop "$SERVICE_NAME" 2>/dev/null || true
  systemctl reset-failed "$SERVICE_NAME" 2>/dev/null || true
  sleep 1

  for i in $(seq 1 10); do
    if ! ss -tlnp 2>/dev/null | grep -q ':8080 '; then
      break
    fi
    PID="$(ss -tlnp 2>/dev/null | grep ':8080 ' | grep -oP 'pid=\K[0-9]+' | head -1 || true)"
    if [[ -n "$PID" ]]; then
      PROC="$(cat /proc/$PID/comm 2>/dev/null || echo unknown)"
      if [[ "$PROC" == "zedproxy" ]]; then
        kill "$PID" 2>/dev/null || true
        detail "Killed stale zedproxy process (pid $PID)"
      else
        error "Port 8080 is held by $PROC (pid $PID), not zedproxy. Aborting."
      fi
    fi
    sleep 1
  done

  info "Service $SERVICE_NAME stopped"
}

# ── Ensure required directories exist ─────────────────
ensure_dirs() {
  mkdir -p "${INSTALL_DIR}/static/uploads"
  mkdir -p "${INSTALL_DIR}/static/uploads/images"
  mkdir -p "${INSTALL_DIR}/static/uploads/videos"
  mkdir -p "${INSTALL_DIR}/static/uploads/plans"
  mkdir -p "${INSTALL_DIR}/static/uploads/blog"
  mkdir -p "${INSTALL_DIR}/static/uploads/pages"
  mkdir -p "${INSTALL_DIR}/static/uploads/tmp"
  mkdir -p "${INSTALL_DIR}/logs"
  mkdir -p "${INSTALL_DIR}/data/backups"
  mkdir -p "${INSTALL_DIR}/data"
  detail "Required directories ensured"
}

# ── Deploy ────────────────────────────────────────────
deploy_binary() {
  step "Replacing binary"
  # Keep a copy of old binary as fallback
  if [[ -f "$OLD_BINARY" ]]; then
    cp "$OLD_BINARY" "${OLD_BINARY}.backup-${TIMESTAMP}"
    detail "Old binary saved: ${OLD_BINARY}.backup-${TIMESTAMP}"
  fi
  cp "$NEW_BINARY" "$OLD_BINARY"
  chmod +x "$OLD_BINARY"
  info "New binary deployed"
}

deploy_templates() {
  step "Updating templates"
  if [[ ! -d "${BUILD_DIR}/templates" ]]; then
    warn "templates directory not found in new code — skipped"
    return
  fi
  # Sync templates completely (no user data lives here)
  rsync -a --delete "${BUILD_DIR}/templates/" "${INSTALL_DIR}/templates/"
  info "Templates updated"
}

deploy_static() {
  step "Updating static files"
  if [[ ! -d "${BUILD_DIR}/static" ]]; then
    warn "static directory not found in new code — skipped"
    return
  fi
  # NEVER touch uploads — exclude it explicitly
  rsync -a \
    --exclude 'uploads/' \
    "${BUILD_DIR}/static/" "${INSTALL_DIR}/static/"

  # Ensure uploads directory still exists with correct permissions
  mkdir -p "$UPLOADS_DIR"
  info "Static files updated (uploads preserved)"
}

deploy_manage_sh() {
  if [[ -f "${BUILD_DIR}/manage.sh" ]]; then
    cp "${BUILD_DIR}/manage.sh" "${INSTALL_DIR}/manage.sh"
    chmod +x "${INSTALL_DIR}/manage.sh"
    chown root:root "${INSTALL_DIR}/manage.sh"
    detail "manage.sh refreshed"
  fi
  # Refresh global wrapper scripts
  cat > /usr/local/bin/zedproxy-doctor << 'WRAPPER'
#!/bin/bash
exec /opt/zedproxy/zedproxy doctor "$@"
WRAPPER
  chmod +x /usr/local/bin/zedproxy-doctor
  cat > /usr/local/bin/zedproxy-repair << 'WRAPPER'
#!/bin/bash
exec /opt/zedproxy/zedproxy repair "$@"
WRAPPER
  chmod +x /usr/local/bin/zedproxy-repair
  detail "zedproxy-doctor and zedproxy-repair wrappers refreshed"
}

deploy_rollback_sh() {
  if [[ -f "${BUILD_DIR}/rollback.sh" ]]; then
    cp "${BUILD_DIR}/rollback.sh" "${INSTALL_DIR}/rollback.sh"
    chmod +x "${INSTALL_DIR}/rollback.sh"
    chown root:root "${INSTALL_DIR}/rollback.sh"
    detail "rollback.sh refreshed"
  fi
}

deploy_update_script() {
  # Keep update.sh fresh on the server after each update
  if [[ -f "${BUILD_DIR}/update.sh" ]]; then
    cp "${BUILD_DIR}/update.sh" "${INSTALL_DIR}/update.sh"
    chmod +x "${INSTALL_DIR}/update.sh"
    chown root:root "${INSTALL_DIR}/update.sh"
    detail "update.sh refreshed"
  fi
}

# ── Permissions ───────────────────────────────────────
fix_permissions() {
  step "Setting permissions"
  chown -R www-data:www-data "${INSTALL_DIR}/templates" 2>/dev/null || true
  chown -R www-data:www-data "${INSTALL_DIR}/static" 2>/dev/null || true
  chown -R www-data:www-data "${INSTALL_DIR}/data" 2>/dev/null || true
  chown www-data:www-data "${OLD_BINARY}" 2>/dev/null || true
  chmod 755 "${OLD_BINARY}"
  chmod 600 "${ENV_FILE}"
  chmod -R 755 "${INSTALL_DIR}/templates" 2>/dev/null || true
  chmod -R 755 "${INSTALL_DIR}/static" 2>/dev/null || true
  chmod -R 775 "${INSTALL_DIR}/static/uploads" 2>/dev/null || true
  chmod -R 775 "${INSTALL_DIR}/data" 2>/dev/null || true
  # backups: root-owned, not world-readable
  chown -R root:root "${BACKUP_DIR}" 2>/dev/null || true
  chmod -R 700 "${BACKUP_DIR}" 2>/dev/null || true
  info "Permissions set"
}

# ── Start & verify ────────────────────────────────────
start_service() {
  step "Starting service"
  systemctl daemon-reload
  systemctl start "$SERVICE_NAME"
  sleep 3

  if ! systemctl is-active --quiet "$SERVICE_NAME"; then
    warn "Service failed to start. Logs:"
    journalctl -u "$SERVICE_NAME" -n 50 --no-pager || true
    error "Failed to start ZedProxy service."
  fi
  info "Service $SERVICE_NAME is active"
  systemctl status "$SERVICE_NAME" --no-pager -l | head -20 | sed 's/^/    /'
}

run_self_test() {
  step "Self-test"
  if "${OLD_BINARY}" \
      --db="${DB_FILE}" \
      --templates="${INSTALL_DIR}/templates" \
      --static="${INSTALL_DIR}/static" \
      --self-test 2>&1 | sed 's/^/    /'; then
    info "Self-test passed"
  else
    error "Self-test FAILED — rolling back"
  fi
}

rollback_deployment() {
  warn "Rolling back deployment..."
  systemctl stop "$SERVICE_NAME" 2>/dev/null || true

  local old_bin_backup="${OLD_BINARY}.backup-${TIMESTAMP}"
  if [[ -f "$old_bin_backup" ]]; then
    cp "$old_bin_backup" "$OLD_BINARY"
    chmod +x "$OLD_BINARY"
    info "Previous binary restored"
  fi

  systemctl start "$SERVICE_NAME" 2>/dev/null || true
  sleep 2
  if systemctl is-active --quiet "$SERVICE_NAME"; then
    info "Previous service restored and running"
  else
    warn "Service failed to start after rollback — check logs:"
    journalctl -u "$SERVICE_NAME" -n 30 --no-pager || true
  fi
}

health_check() {
  step "Health check"
  local base="http://127.0.0.1:${APP_PORT}"
  local attempts=5
  local wait=2
  local passed=false

  for i in $(seq 1 $attempts); do
    local http_code
    http_code="$(curl -s -o /dev/null -w '%{http_code}' --connect-timeout 5 --max-time 10 "${base}/" 2>/dev/null || echo '000')"
    if [[ "$http_code" =~ ^(200|301|302)$ ]]; then
      info "Root health check passed (HTTP $http_code)"
      passed=true
      break
    fi
    detail "Attempt $i/$attempts — HTTP $http_code — waiting ${wait}s..."
    sleep $wait
    wait=$((wait * 2))
  done

  if [[ "$passed" != "true" ]]; then
    rollback_deployment
    error "Health check failed — rolled back to previous version"
  fi

  # Verify critical admin routes (404 is unacceptable)
  local admin_routes=("/zed-admin/integrations/telegram" "/zed-admin/settings/appearance")
  local route_fail=false
  for route in "${admin_routes[@]}"; do
    local code
    code="$(curl -s -o /dev/null -w '%{http_code}' --connect-timeout 5 --max-time 10 "${base}${route}" 2>/dev/null || echo '000')"
    if [[ "$code" =~ ^(200|301|302)$ ]]; then
      info "Route check OK: $route (HTTP $code)"
    else
      warn "Route check FAILED: $route returned HTTP $code"
      route_fail=true
    fi
  done

  if [[ "$route_fail" == "true" ]]; then
    rollback_deployment
    error "Admin route check failed — rolled back to previous version"
  fi

  # Post-update verification: confirm new templates are deployed
  step "Post-update template verification"
  grep -rq "integrations/telegram" "${INSTALL_DIR}/templates" && \
    info "  [OK] integrations/telegram found in templates" || \
    warn "  [!] integrations/telegram not found in templates — may be missing"
  grep -rq "settings/appearance" "${INSTALL_DIR}/templates" && \
    info "  [OK] settings/appearance found in templates" || \
    warn "  [!] settings/appearance not found in templates — may be missing"
  grep -rq "admin-accent" "${INSTALL_DIR}/templates" && \
    info "  [OK] CSS variable admin-accent found in templates" || \
    warn "  [!] CSS variable admin-accent not found in templates"
}

# ── Print result ──────────────────────────────────────
print_result() {
  echo ""
  echo -e "${GREEN}╔══════════════════════════════════════════════════════╗${NC}"
  echo -e "${GREEN}║         Update completed successfully!               ║${NC}"
  echo -e "${GREEN}╚══════════════════════════════════════════════════════╝${NC}"
  echo ""
  echo -e "${WHITE}Update summary:${NC}"
  echo -e "  Repository:   ${CYAN}${REPO_URL}${NC}"
  echo -e "  Branch:       ${CYAN}${BRANCH}${NC}"
  echo -e "  Time:         ${WHITE}${TIMESTAMP}${NC}"
  echo -e "  Backup:       ${WHITE}${BACKUP_DIR}/zedproxy-pre-update-${TIMESTAMP}.zip${NC}"
  echo -e "  Log file:     ${WHITE}${LOG_FILE}${NC}"
  echo ""
  echo -e "${WHITE}Useful commands:${NC}"
  echo -e "  sudo systemctl status ${SERVICE_NAME}    # Service status"
  echo -e "  sudo systemctl restart ${SERVICE_NAME}   # Restart service"
  echo -e "  sudo journalctl -u ${SERVICE_NAME} -f    # Live logs"
  echo -e "  sudo bash ${INSTALL_DIR}/update.sh       # Run update again"
  echo ""
  echo -e "${YELLOW}Data preserved:${NC}"
  echo -e "  Database:  ${DB_FILE}"
  echo -e "  Uploads:   ${UPLOADS_DIR}"
  echo -e "  .env:      ${ENV_FILE}"
  echo ""
}

# ── Main ─────────────────────────────────────────────
setup_logging
banner
check_root
# Register Telegram failure notification
trap 'notify_update_fail' ERR
check_installed
check_env
check_go
check_git
create_backups
notify_update_start
clone_repo
build_binary
ensure_dirs
stop_service
deploy_binary
deploy_templates
deploy_static
deploy_manage_sh
deploy_rollback_sh
deploy_update_script
fix_permissions
run_self_test
start_service
health_check
notify_update_done
print_result
