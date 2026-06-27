#!/usr/bin/env bash
set -euo pipefail

# =====================================================
#  ZedProxy Server Manager
#  Usage: sudo zedproxy-manager
#    or:  sudo bash /opt/zedproxy/manage.sh
# =====================================================

export LC_ALL=C.UTF-8
export LANG=C.UTF-8

INSTALL_DIR="/opt/zedproxy"
SERVICE_NAME="zedproxy"
DB_FILE="${INSTALL_DIR}/data/zedproxy.db"
BIN="${INSTALL_DIR}/zedproxy"
LOG_DIR="${INSTALL_DIR}/logs"

RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
CYAN='\033[0;36m'
WHITE='\033[1;37m'
BLUE='\033[0;34m'
NC='\033[0m'

info()  { echo -e "${GREEN}[+]${NC} $1"; }
warn()  { echo -e "${YELLOW}[!]${NC} $1"; }
err()   { echo -e "${RED}[x]${NC} $1"; }
ok()    { echo -e "${GREEN}[OK]${NC} $1"; }
sep()   { echo -e "${BLUE}────────────────────────────────────────${NC}"; }

# ── Guards ────────────────────────────────────────────

check_root() {
  if [[ $EUID -ne 0 ]]; then
    err "This script must be run as root: sudo zedproxy-manager"
    exit 1
  fi
}

check_bin() {
  if [[ ! -f "$BIN" ]]; then
    err "ZedProxy binary not found: $BIN"
    err "Please run install.sh first."
    exit 1
  fi
}

check_db() {
  if [[ ! -f "$DB_FILE" ]]; then
    err "Database not found: $DB_FILE"
    return 1
  fi
}

# ── CLI wrapper ───────────────────────────────────────

run_cli() {
  "$BIN" --db="$DB_FILE" "$@"
}

press_enter() {
  echo ""
  read -rp "Press Enter to return to menu..."
}

confirm() {
  local msg="${1:-Are you sure?}"
  read -rp "$msg [y/N]: " ans
  [[ "${ans,,}" == "y" ]]
}

# ── Credential helpers ────────────────────────────────

gen_username() {
  if command -v openssl &>/dev/null; then
    echo "admin_$(openssl rand -hex 3)"
  else
    echo "admin_$(date +%s | tail -c 7)"
  fi
}

gen_password() {
  if command -v openssl &>/dev/null; then
    openssl rand -base64 32 | tr -dc 'a-zA-Z0-9!@#%+=_' | head -c 20
  else
    tr -dc 'a-zA-Z0-9!@#%+=_' < /dev/urandom | head -c 20
  fi
}

validate_username() {
  local u="$1"
  [[ ${#u} -ge 3 && ${#u} -le 32 && "$u" =~ ^[a-zA-Z0-9._-]+$ ]]
}

validate_password() {
  local p="$1"
  [[ ${#p} -ge 8 && ${#p} -le 128 ]]
}

prompt_username() {
  local label="${1:-Admin username}"
  local result
  read -rp "$label (Enter to auto-generate): " result
  if [[ -z "$result" ]]; then
    result="$(gen_username)"
    info "Auto-generated username: $result"
  elif ! validate_username "$result"; then
    warn "Invalid username. Auto-generating instead."
    result="$(gen_username)"
    info "Auto-generated username: $result"
  fi
  echo "$result"
}

prompt_password() {
  local label="${1:-Admin password}"
  local result
  read -rsp "$label (Enter to auto-generate): " result
  echo ""
  if [[ -z "$result" ]]; then
    result="$(gen_password)"
    info "Auto-generated password (shown once): $result"
  elif ! validate_password "$result"; then
    warn "Password too short (min 8 chars). Auto-generating instead."
    result="$(gen_password)"
    info "Auto-generated password (shown once): $result"
  fi
  echo "$result"
}

# ── Banner ────────────────────────────────────────────

banner() {
  clear
  local svc_state
  svc_state="$(systemctl is-active "$SERVICE_NAME" 2>/dev/null || echo 'unknown')"
  local svc_color="${RED}"
  [[ "$svc_state" == "active" ]] && svc_color="${GREEN}"

  echo -e "${CYAN}"
  echo "╔══════════════════════════════════════════════════╗"
  echo "║           ZedProxy Server Manager               ║"
  echo "╚══════════════════════════════════════════════════╝"
  echo -e "${NC}"
  echo -e "  Service: ${svc_color}${svc_state}${NC}   |   $(date '+%Y-%m-%d %H:%M:%S')"
  echo ""
}

show_menu() {
  banner
  echo -e "${WHITE}  Service & System${NC}"
  echo "   1) Show System Status"
  echo "   2) Emergency Recovery"
  echo "   3) Restart Service"
  echo "   4) Start Service"
  echo "   5) Stop Service"
  echo ""
  echo -e "${WHITE}  Admin Accounts${NC}"
  echo "   6) Reset Admin Username / Password"
  echo "   7) Create New Owner Admin"
  echo ""
  echo -e "${WHITE}  Maintenance Mode${NC}"
  echo "   8) Maintenance Mode Status"
  echo "   9) Maintenance Mode ON"
  echo "  10) Maintenance Mode OFF / Emergency Disable"
  echo ""
  echo -e "${WHITE}  Updates & Backup${NC}"
  echo "  11) Run Website Update"
  echo "  12) Repair Missing update.sh"
  echo "  13) Rollback to Previous Release"
  echo "  14) Create ZIP DB Backup"
  echo "  14t) Create ZIP Backup + Send to Telegram"
  echo "  14f) Create Full ZIP Backup (db+uploads)"
  echo "  15) List Backups"
  echo "  16) Restore Backup"
  echo ""
  echo -e "${WHITE}  Diagnostics${NC}"
  echo "  17) Run Self-Test"
  echo "  17d) Run Doctor (full health check)"
  echo "  17r) Run Repair (fix dirs + restart)"
  echo "  18) Show Health Check"
  echo "  19) Check SQLite Database"
  echo "  20) Repair SQLite Database"
  echo "  21) Vacuum SQLite Database"
  echo "  22) View Recent Logs"
  echo "  23) Follow Live Logs"
  echo "  24) Export Diagnostic Report"
  echo ""
  echo -e "${WHITE}  Configuration${NC}"
  echo "  25) Fix File Permissions"
  echo "  26) Show App Version"
  echo "  27) Show Domain and URLs"
  echo "  28) Change Domain"
  echo "  29) Check Nginx Config"
  echo "  30) Reload Nginx"
  echo "  31) Renew SSL Certificate"
  echo "  32) Show Disk Usage"
  echo "  33) Clean Temporary Files"
  echo ""
  echo -e "${WHITE}  Telegram Admin Reporter${NC}"
  echo "  34) Telegram Bot Status"
  echo "  35) Test Bot Connection"
  echo "  36) Send Test Message"
  echo "  37) Create Group Topics"
  echo "  38) Enable Telegram Alerts"
  echo "  39) Disable Telegram Alerts"
  echo "  40) Send Daily Report Now"
  echo ""
  echo -e "${CYAN}  Configure Telegram token and chat ID from the admin panel:${NC}"
  echo -e "  Admin panel -> /zed-admin/integrations/telegram"
  echo ""
  echo -e "${WHITE}  Danger Zone${NC}"
  echo "  41) Uninstall ZedProxy"
  echo "  42) Exit"
  echo ""
}

# ── Menu Actions ──────────────────────────────────────

action_system_status() {
  sep
  echo "=== System Status ==="
  sep
  echo ""
  info "Service:"
  systemctl status "$SERVICE_NAME" --no-pager -l 2>/dev/null | head -20 || err "Service not found"
  echo ""
  info "Disk:"
  df -h "$INSTALL_DIR" 2>/dev/null || df -h / | tail -2
  echo ""
  info "Memory:"
  free -h 2>/dev/null || true
  echo ""
  info "Load:"
  uptime 2>/dev/null || true
}

action_emergency_recovery() {
  sep
  warn "Emergency Recovery"
  sep
  echo ""
  warn "This will:"
  echo "  1. Disable maintenance mode"
  echo "  2. Restart the service"
  echo "  3. Reload Nginx"
  echo ""
  confirm "Proceed with emergency recovery?" || return
  run_cli --maintenance-off 2>/dev/null || true
  systemctl restart "$SERVICE_NAME" 2>/dev/null || systemctl start "$SERVICE_NAME" 2>/dev/null || true
  sleep 2
  systemctl reload nginx 2>/dev/null || true
  if systemctl is-active --quiet "$SERVICE_NAME"; then
    ok "Service is running"
  else
    err "Service failed to start. Check: journalctl -u $SERVICE_NAME -n 50"
  fi
}

action_restart() {
  confirm "Restart $SERVICE_NAME?" || return
  systemctl restart "$SERVICE_NAME" && ok "Service restarted" || err "Restart failed"
}

action_start() {
  systemctl start "$SERVICE_NAME" && ok "Service started" || err "Start failed"
}

action_stop() {
  confirm "Stop $SERVICE_NAME?" || return
  systemctl stop "$SERVICE_NAME" && ok "Service stopped" || err "Stop failed"
}

action_reset_admin() {
  sep
  echo "=== Reset Admin Credentials ==="
  sep
  echo ""

  read -rp "Enter admin username to reset: " TARGET_USER
  if [[ -z "$TARGET_USER" ]]; then
    err "Username is required."
    return
  fi

  NEW_USER="$(prompt_username "New username")"
  NEW_PASS="$(prompt_password "New password")"

  confirm "Reset credentials for '$TARGET_USER'?" || return

  if run_cli --reset-admin --admin-user="$TARGET_USER" --admin-pass="$NEW_PASS" 2>&1; then
    ok "Password reset for: $TARGET_USER"
    if [[ "$NEW_USER" != "$TARGET_USER" ]]; then
      warn "Username rename is not supported via CLI. Use admin panel to change username."
    fi
  else
    err "Failed to reset admin. Check the username is correct."
  fi
  echo ""
  echo -e "${YELLOW}New password (shown once): ${WHITE}${NEW_PASS}${NC}"
}

action_create_admin() {
  sep
  echo "=== Create New Owner Admin ==="
  sep
  echo ""

  NEW_USER="$(prompt_username "New admin username")"
  read -rp "Admin email (Enter for admin@zedproxy.com): " NEW_EMAIL
  [[ -z "$NEW_EMAIL" ]] && NEW_EMAIL="admin@zedproxy.com"
  NEW_PASS="$(prompt_password "New admin password")"

  confirm "Create admin '$NEW_USER' with role 'owner'?" || return

  if run_cli --create-admin --admin-user="$NEW_USER" --admin-email="$NEW_EMAIL" --admin-pass="$NEW_PASS" --role="owner" 2>&1; then
    ok "Admin created: $NEW_USER"
    echo ""
    echo -e "${WHITE}Username: ${CYAN}${NEW_USER}${NC}"
    echo -e "${WHITE}Email:    ${CYAN}${NEW_EMAIL}${NC}"
    echo -e "${WHITE}Password: ${RED}${NEW_PASS}${NC}"
    echo -e "${YELLOW}Save these credentials securely — password shown once only.${NC}"
  else
    err "Failed to create admin. Username may already exist."
  fi
}

action_maintenance_status() {
  run_cli --maintenance-status 2>&1 || err "Failed to check maintenance status"
}

action_maintenance_on() {
  confirm "Enable maintenance mode? Site will show maintenance page." || return
  run_cli --maintenance-on && ok "Maintenance mode enabled" || err "Failed"
}

action_maintenance_off() {
  run_cli --maintenance-off && ok "Maintenance mode disabled" || err "Failed"
  systemctl restart "$SERVICE_NAME" 2>/dev/null || true
  ok "Service restarted to ensure maintenance is lifted"
}

action_update() {
  if [[ ! -f "$INSTALL_DIR/update.sh" ]]; then
    err "update.sh not found. Run option 12 to repair it."
    return
  fi
  confirm "Run website update? Service will restart." || return
  bash "$INSTALL_DIR/update.sh"
}

action_repair_update_sh() {
  warn "Downloading update.sh from GitHub..."
  curl -fsSL "https://raw.githubusercontent.com/mhoseinshah1/zed_website/main/update.sh" \
    -o "$INSTALL_DIR/update.sh" || { err "Download failed. Check internet connection."; return; }
  chmod +x "$INSTALL_DIR/update.sh"
  chown root:root "$INSTALL_DIR/update.sh"
  bash -n "$INSTALL_DIR/update.sh" && ok "update.sh downloaded and validated" || err "Script has syntax error"
}

action_rollback() {
  if [[ ! -f "$INSTALL_DIR/rollback.sh" ]]; then
    warn "rollback.sh not found. Attempting manual rollback..."
    local latest
    latest="$(ls -t "${INSTALL_DIR}"/zedproxy.backup-* 2>/dev/null | head -1)"
    if [[ -z "$latest" ]]; then
      err "No backup binary found at ${INSTALL_DIR}/zedproxy.backup-*"
      return
    fi
    echo "Latest backup: $latest"
    confirm "Restore this binary?" || return
    systemctl stop "$SERVICE_NAME" 2>/dev/null || true
    cp "$latest" "$BIN"
    chmod +x "$BIN"
    systemctl start "$SERVICE_NAME" && ok "Rollback complete" || err "Service failed to start"
    return
  fi
  bash "$INSTALL_DIR/rollback.sh"
}

action_backup() {
  ok "Creating ZIP database backup..."
  "${INSTALL_DIR}/zedproxy" \
    --db="$DB_FILE" \
    --backups="${INSTALL_DIR}/data/backups" \
    --create-db-backup 2>&1 || {
      warn "ZIP backup failed, falling back to raw copy..."
      local ts
      ts="$(date +%Y%m%d-%H%M%S)"
      local dest="${INSTALL_DIR}/data/backups/manual-${ts}.db"
      mkdir -p "${INSTALL_DIR}/data/backups"
      cp "$DB_FILE" "$dest" && ok "Raw backup created: $dest" || err "Backup failed"
    }
}

action_backup_telegram() {
  ok "Creating ZIP backup and sending to Telegram..."
  "${INSTALL_DIR}/zedproxy" \
    --db="$DB_FILE" \
    --backups="${INSTALL_DIR}/data/backups" \
    --send-db-backup-telegram 2>&1 && ok "Backup sent to Telegram" || warn "Telegram send failed (local backup kept)"
}

action_backup_full() {
  ok "Creating full ZIP backup (db + uploads)..."
  "${INSTALL_DIR}/zedproxy" \
    --db="$DB_FILE" \
    --backups="${INSTALL_DIR}/data/backups" \
    --uploads="${INSTALL_DIR}/static/uploads" \
    --create-full-backup 2>&1 && ok "Full backup created" || err "Full backup failed"
}

action_list_backups() {
  sep
  echo "=== Available Backups ==="
  sep
  local bdir="${INSTALL_DIR}/data/backups"
  if [[ -d "$bdir" ]]; then
    ls -lht "$bdir"/ 2>/dev/null | head -20 || warn "No backups in $bdir"
  fi
  echo ""
  echo "Binary backups (for rollback):"
  ls -lht "${INSTALL_DIR}"/zedproxy.backup-* 2>/dev/null | head -10 || warn "No binary backups found"
}

action_restore_backup() {
  local bdir="${INSTALL_DIR}/data/backups"
  if [[ ! -d "$bdir" ]] || [[ -z "$(ls "$bdir"/*.db 2>/dev/null)" ]]; then
    err "No database backups found in $bdir"
    return
  fi
  echo "Available database backups:"
  ls -lht "$bdir"/*.db 2>/dev/null | head -15
  echo ""
  read -rp "Enter backup filename (just the name, not path): " BNAME
  local bpath="${bdir}/${BNAME}"
  if [[ ! -f "$bpath" ]]; then
    err "File not found: $bpath"
    return
  fi
  confirm "Restore $BNAME? This will overwrite the current database." || return
  systemctl stop "$SERVICE_NAME" 2>/dev/null || true
  cp "$DB_FILE" "${DB_FILE}.pre-restore-$(date +%Y%m%d-%H%M%S)" && info "Current DB backed up"
  cp "$bpath" "$DB_FILE"
  chown www-data:www-data "$DB_FILE" 2>/dev/null || true
  systemctl start "$SERVICE_NAME" && ok "Database restored and service restarted" || err "Service failed to start"
}

action_self_test() {
  run_cli --self-test \
    --templates="${INSTALL_DIR}/templates" \
    --static="${INSTALL_DIR}/static" \
    --uploads="${INSTALL_DIR}/static/uploads" 2>&1 || err "Self-test failed"
}

action_health_check() {
  local code
  code="$(curl -s -o /dev/null -w '%{http_code}' --max-time 5 "http://127.0.0.1:8080/health" 2>/dev/null || echo '000')"
  if [[ "$code" == "200" ]]; then
    ok "Health check passed (HTTP $code)"
    curl -s "http://127.0.0.1:8080/health" 2>/dev/null | python3 -m json.tool 2>/dev/null || true
  else
    err "Health check returned HTTP $code"
  fi
}

action_check_db() {
  if ! check_db; then return; fi
  echo "Running SQLite integrity check..."
  local result
  result="$(sqlite3 "$DB_FILE" 'PRAGMA integrity_check;' 2>&1)"
  echo "$result"
  if [[ "$result" == "ok" ]]; then
    ok "Database integrity: OK"
  else
    err "Database integrity issues found"
  fi
  echo ""
  info "Database size: $(du -sh "$DB_FILE" | cut -f1)"
  echo "Tables:"
  sqlite3 "$DB_FILE" ".tables" 2>/dev/null || true
}

action_repair_db() {
  if ! check_db; then return; fi
  confirm "Attempt SQLite repair? A backup will be created first." || return
  local bak="${DB_FILE}.pre-repair-$(date +%Y%m%d-%H%M%S)"
  cp "$DB_FILE" "$bak" && info "Backup: $bak"
  systemctl stop "$SERVICE_NAME" 2>/dev/null || true
  sqlite3 "$DB_FILE" "PRAGMA wal_checkpoint(TRUNCATE);" 2>/dev/null || true
  local tmp="${DB_FILE}.repaired"
  sqlite3 "$DB_FILE" ".dump" | sqlite3 "$tmp" && \
    mv "$tmp" "$DB_FILE" && ok "Database repaired" || err "Repair failed; original preserved at $bak"
  chown www-data:www-data "$DB_FILE" 2>/dev/null || true
  systemctl start "$SERVICE_NAME" 2>/dev/null || true
}

action_vacuum_db() {
  if ! check_db; then return; fi
  confirm "Vacuum SQLite database? Service will be stopped briefly." || return
  systemctl stop "$SERVICE_NAME" 2>/dev/null || true
  sqlite3 "$DB_FILE" "VACUUM;" && ok "Vacuum complete" || err "Vacuum failed"
  info "New size: $(du -sh "$DB_FILE" | cut -f1)"
  systemctl start "$SERVICE_NAME" 2>/dev/null || true
}

action_recent_logs() {
  journalctl -u "$SERVICE_NAME" -n 100 --no-pager 2>/dev/null || \
    tail -100 "${LOG_DIR}/update-*.log" 2>/dev/null || err "No logs found"
}

action_live_logs() {
  info "Showing live logs (Ctrl+C to exit)..."
  journalctl -u "$SERVICE_NAME" -f 2>/dev/null || err "Cannot follow logs"
}

action_diagnostic_report() {
  local report_file="/tmp/zedproxy-diag-$(date +%Y%m%d-%H%M%S).txt"
  {
    echo "=== ZedProxy Diagnostic Report ==="
    echo "Date: $(date)"
    echo ""
    echo "--- Service Status ---"
    systemctl status "$SERVICE_NAME" --no-pager -l 2>/dev/null | head -30 || true
    echo ""
    echo "--- Recent Logs ---"
    journalctl -u "$SERVICE_NAME" -n 50 --no-pager 2>/dev/null || true
    echo ""
    echo "--- Disk Usage ---"
    df -h 2>/dev/null || true
    echo ""
    echo "--- Memory ---"
    free -h 2>/dev/null || true
    echo ""
    echo "--- App Version ---"
    "$BIN" --version --db="$DB_FILE" 2>/dev/null || true
    echo ""
    echo "--- DB Integrity ---"
    sqlite3 "$DB_FILE" 'PRAGMA integrity_check;' 2>/dev/null || true
    echo ""
    echo "--- Files ---"
    ls -lh "$INSTALL_DIR"/ 2>/dev/null || true
  } > "$report_file" 2>&1
  ok "Diagnostic report saved: $report_file"
  echo "View with: cat $report_file"
}

action_fix_permissions() {
  confirm "Fix file permissions? (chown + chmod)" || return
  chown -R www-data:www-data "$INSTALL_DIR" 2>/dev/null || true
  chmod -R 755 "$INSTALL_DIR" 2>/dev/null || true
  chmod 600 "$INSTALL_DIR/.env" 2>/dev/null || true
  chmod 755 "$BIN" 2>/dev/null || true
  chmod -R 775 "$INSTALL_DIR/data" 2>/dev/null || true
  chmod -R 775 "$INSTALL_DIR/static/uploads" 2>/dev/null || true
  for f in update.sh manage.sh rollback.sh; do
    [[ -f "$INSTALL_DIR/$f" ]] && chown root:root "$INSTALL_DIR/$f" && chmod 755 "$INSTALL_DIR/$f" || true
  done
  ok "Permissions fixed"
}

action_version() {
  run_cli --version 2>&1 || "$BIN" --version 2>&1 || info "Binary: $(ls -lh "$BIN" 2>/dev/null)"
}

action_show_domain() {
  sep
  echo "=== Domain and URLs ==="
  sep
  local site_url
  site_url="$(sqlite3 "$DB_FILE" "SELECT value FROM settings WHERE key='site_url';" 2>/dev/null || echo 'unknown')"
  echo ""
  info "Site URL:    $site_url"
  info "Admin:       $site_url/zed-admin"
  info "Health:      $site_url/health"
  info "Nginx conf:  /etc/nginx/sites-available/zedproxy"
  echo ""
  echo "Nginx config:"
  cat /etc/nginx/sites-available/zedproxy 2>/dev/null | head -15 || warn "Nginx config not found"
}

action_change_domain() {
  echo ""
  read -rp "Enter new domain (example.com): " NEW_DOMAIN
  if [[ -z "$NEW_DOMAIN" ]]; then
    err "Domain cannot be empty."
    return
  fi
  NEW_DOMAIN="${NEW_DOMAIN,,}"
  confirm "Change domain to $NEW_DOMAIN?" || return
  sqlite3 "$DB_FILE" \
    "INSERT INTO settings (key, value) VALUES ('site_url', 'https://$NEW_DOMAIN') ON CONFLICT(key) DO UPDATE SET value='https://$NEW_DOMAIN';" \
    2>/dev/null && ok "site_url updated in database" || err "Failed to update database"
  warn "Also update your Nginx config: /etc/nginx/sites-available/zedproxy"
  warn "Then run: sudo nginx -t && sudo systemctl reload nginx"
}

action_nginx_check() {
  nginx -t 2>&1 && ok "Nginx config is valid" || err "Nginx config has errors"
}

action_nginx_reload() {
  nginx -t && systemctl reload nginx && ok "Nginx reloaded" || err "Nginx reload failed"
}

action_ssl_renew() {
  confirm "Renew SSL certificate with Certbot?" || return
  certbot renew --nginx 2>&1 && ok "SSL renewed" || err "SSL renewal failed"
  systemctl reload nginx 2>/dev/null || true
}

action_disk_usage() {
  sep
  echo "=== Disk Usage ==="
  sep
  df -h 2>/dev/null || true
  echo ""
  du -sh "${INSTALL_DIR}"/* 2>/dev/null | sort -h | head -20 || true
}

action_clean_tmp() {
  confirm "Remove temporary build files and old binary backups (keep 5 newest)?" || return
  rm -rf /tmp/zedproxy-* 2>/dev/null || true
  rm -rf /tmp/go-build* 2>/dev/null || true
  ls -t "${INSTALL_DIR}"/zedproxy.backup-* 2>/dev/null | tail -n +6 | xargs rm -f 2>/dev/null || true
  ok "Temporary files cleaned"
}

# ── Telegram actions ──────────────────────────────────

action_tg_status() {
  run_cli --telegram-status 2>&1 || err "Failed to get Telegram status"
}

action_tg_test() {
  run_cli --telegram-test 2>&1 || err "Connection test failed. Check token and Chat ID."
}

action_tg_send_test() {
  run_cli --telegram-send-test 2>&1 && ok "Test message sent" || err "Failed to send test message"
}

action_tg_create_topics() {
  info "Creating forum topics in Telegram group..."
  info "Note: The group must be a supergroup with Topics enabled."
  run_cli --telegram-create-topics 2>&1 || err "Some topics may have failed — check Telegram group"
}

action_tg_enable() {
  run_cli --telegram-enable && ok "Telegram bot enabled" || err "Failed"
}

action_tg_disable() {
  confirm "Disable Telegram alerts?" || return
  run_cli --telegram-disable && ok "Telegram bot disabled" || err "Failed"
}

action_tg_daily_report() {
  run_cli --send-daily-report 2>&1 && ok "Daily report sent" || err "Failed to send daily report"
}

# ── Uninstall ─────────────────────────────────────────

action_uninstall() {
  sep
  warn "UNINSTALL ZEDPROXY"
  sep
  echo ""
  warn "This will:"
  echo "  - Stop and disable the service"
  echo "  - Remove /opt/zedproxy (including database, uploads)"
  echo "  - Remove systemd service"
  echo "  - Remove Nginx config"
  echo "  - Remove /usr/local/bin/zedproxy-manager symlink"
  echo ""
  warn "This action CANNOT be undone."
  read -rp "Type 'uninstall' to confirm: " CONFIRM_TEXT
  if [[ "$CONFIRM_TEXT" != "uninstall" ]]; then
    info "Uninstall cancelled."
    return
  fi

  systemctl stop "$SERVICE_NAME" 2>/dev/null || true
  systemctl disable "$SERVICE_NAME" 2>/dev/null || true
  rm -f "/etc/systemd/system/${SERVICE_NAME}.service"
  systemctl daemon-reload 2>/dev/null || true
  rm -f /etc/nginx/sites-enabled/zedproxy
  rm -f /etc/nginx/sites-available/zedproxy
  systemctl reload nginx 2>/dev/null || true
  rm -f /usr/local/bin/zedproxy-manager
  rm -rf "$INSTALL_DIR"
  ok "ZedProxy has been uninstalled."
}

# ── Main loop ─────────────────────────────────────────

main() {
  check_root
  check_bin

  while true; do
    show_menu
    read -rp "Select option [1-42]: " CHOICE
    echo ""
    case "$CHOICE" in
      1)  action_system_status ;;
      2)  action_emergency_recovery ;;
      3)  action_restart ;;
      4)  action_start ;;
      5)  action_stop ;;
      6)  action_reset_admin ;;
      7)  action_create_admin ;;
      8)  action_maintenance_status ;;
      9)  action_maintenance_on ;;
      10) action_maintenance_off ;;
      11) action_update ;;
      12) action_repair_update_sh ;;
      13) action_rollback ;;
      14) action_backup ;;
      14t) action_backup_telegram ;;
      14f) action_backup_full ;;
      15) action_list_backups ;;
      16) action_restore_backup ;;
      17) action_self_test ;;
      17d) run_cli --doctor 2>&1 ;;
      17r) confirm "Run repair?" && run_cli --repair 2>&1 ;;
      18) action_health_check ;;
      19) action_check_db ;;
      20) action_repair_db ;;
      21) action_vacuum_db ;;
      22) action_recent_logs ;;
      23) action_live_logs ;;
      24) action_diagnostic_report ;;
      25) action_fix_permissions ;;
      26) action_version ;;
      27) action_show_domain ;;
      28) action_change_domain ;;
      29) action_nginx_check ;;
      30) action_nginx_reload ;;
      31) action_ssl_renew ;;
      32) action_disk_usage ;;
      33) action_clean_tmp ;;
      34) action_tg_status ;;
      35) action_tg_test ;;
      36) action_tg_send_test ;;
      37) action_tg_create_topics ;;
      38) action_tg_enable ;;
      39) action_tg_disable ;;
      40) action_tg_daily_report ;;
      41) action_uninstall ;;
      42) echo "Goodbye."; exit 0 ;;
      "q"|"Q"|"exit") echo "Goodbye."; exit 0 ;;
      *) warn "Invalid option: $CHOICE" ;;
    esac
    press_enter
  done
}

main "$@"
