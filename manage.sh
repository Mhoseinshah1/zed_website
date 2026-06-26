#!/usr/bin/env bash
# ZedProxy Manager - Interactive management menu
# Usage: sudo bash /opt/zedproxy/manage.sh
# Or:    sudo zedproxy-manager

set -euo pipefail
export LC_ALL=C.UTF-8
export LANG=C.UTF-8

INSTALL_DIR="/opt/zedproxy"
SERVICE_NAME="zedproxy"
DB_FILE="${INSTALL_DIR}/data/zedproxy.db"
BIN="${INSTALL_DIR}/zedproxy"
LOG_FILE="${INSTALL_DIR}/logs/manager.log"

RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
CYAN='\033[0;36m'
WHITE='\033[1;37m'
NC='\033[0m'

info()  { echo -e "${GREEN}[+]${NC} $1"; }
warn()  { echo -e "${YELLOW}[!]${NC} $1"; }
error() { echo -e "${RED}[x]${NC} $1"; }
ok()    { echo -e "${GREEN}[OK]${NC} $1"; }

check_root() {
  if [[ $EUID -ne 0 ]]; then
    error "Run as root: sudo bash $0"
    exit 1
  fi
}

check_installed() {
  if [[ ! -f "$BIN" ]]; then
    error "ZedProxy binary not found at $BIN"
    error "Run install.sh first."
    exit 1
  fi
}

run_cli() {
  "$BIN" --db="$DB_FILE" "$@"
}

press_enter() {
  echo ""
  read -rp "Press Enter to continue..."
}

# ── Service Management ────────────────────────────────

service_status()  { systemctl status "$SERVICE_NAME" --no-pager -l | head -30; }
service_start()   { systemctl start "$SERVICE_NAME" && ok "Service started"; }
service_stop()    { systemctl stop "$SERVICE_NAME" && ok "Service stopped"; }
service_restart() { systemctl restart "$SERVICE_NAME" && ok "Service restarted"; }
service_enable()  { systemctl enable "$SERVICE_NAME" && ok "Service enabled on boot"; }
service_disable() { systemctl disable "$SERVICE_NAME" && ok "Service disabled from boot"; }
service_logs()    { journalctl -u "$SERVICE_NAME" -n 80 --no-pager; }
service_logs_live() { journalctl -u "$SERVICE_NAME" -f; }

# ── Maintenance ────────────────────────────────────────

maintenance_on()     { run_cli --maintenance-on; }
maintenance_off()    { run_cli --maintenance-off; }
maintenance_status() { run_cli --maintenance-status; }

# ── Backup & Update ────────────────────────────────────

run_update()   { bash "${INSTALL_DIR}/update.sh"; }
run_rollback() { bash "${INSTALL_DIR}/rollback.sh"; }

backup_db() {
  TS="$(date +%Y%m%d-%H%M%S)"
  DEST="${INSTALL_DIR}/data/backups/manual-${TS}.db"
  mkdir -p "${INSTALL_DIR}/data/backups"
  cp "$DB_FILE" "$DEST"
  ok "Database backed up: $DEST"
}

list_backups() {
  echo "Available backups in ${INSTALL_DIR}/data/backups/:"
  ls -lh "${INSTALL_DIR}/data/backups/" 2>/dev/null || warn "No backups found."
}

# ── Self-test ──────────────────────────────────────────

run_self_test() {
  run_cli --self-test \
    --templates="${INSTALL_DIR}/templates" \
    --static="${INSTALL_DIR}/static" \
    --uploads="${INSTALL_DIR}/static/uploads"
}

# ── Health Check ───────────────────────────────────────

health_check() {
  local code
  code=$(curl -s -o /dev/null -w '%{http_code}' --max-time 5 http://127.0.0.1:8080/health 2>/dev/null || echo "000")
  if [[ "$code" == "200" ]]; then
    ok "Health check passed (HTTP 200)"
  else
    warn "Health check HTTP $code"
  fi
}

# ── Telegram ───────────────────────────────────────────

tg_status()         { run_cli --telegram-status; }
tg_test()           { run_cli --telegram-test; }
tg_send_test()      { run_cli --telegram-send-test; }
tg_create_topics()  { run_cli --telegram-create-topics; }
tg_daily_report()   { run_cli --send-daily-report; }
tg_enable()         { run_cli --telegram-enable; }
tg_disable()        { run_cli --telegram-disable; }

tg_set_token() {
  echo ""
  read -rsp "Enter Telegram bot token: " TOKEN
  echo ""
  if [[ -z "$TOKEN" ]]; then
    warn "No token entered."
    return
  fi
  run_cli --telegram-set-token="$TOKEN"
  ok "Token updated (not displayed for security)"
}

tg_set_chat_id() {
  echo ""
  read -rp "Enter Telegram group Chat ID: " CHATID
  if [[ -z "$CHATID" ]]; then
    warn "No Chat ID entered."
    return
  fi
  run_cli --telegram-set-chat-id="$CHATID"
}

# ── Firewall ────────────────────────────────────────────

fw_status() { ufw status verbose 2>/dev/null || warn "ufw not installed"; }

# ── Nginx ───────────────────────────────────────────────

nginx_status() { systemctl status nginx --no-pager | head -15; }
nginx_reload()  { nginx -t && systemctl reload nginx && ok "Nginx reloaded"; }
nginx_logs()    { tail -50 /var/log/nginx/error.log 2>/dev/null || warn "Nginx log not found"; }

# ── Banner ─────────────────────────────────────────────

banner() {
  clear
  echo -e "${CYAN}"
  echo "╔══════════════════════════════════════════════════╗"
  echo "║          ZedProxy Manager v1.0                   ║"
  echo "╚══════════════════════════════════════════════════╝"
  echo -e "${NC}"
  local svc_color="${RED}"
  if systemctl is-active --quiet "$SERVICE_NAME" 2>/dev/null; then
    svc_color="${GREEN}"
  fi
  echo -e "  Service: ${svc_color}$(systemctl is-active "$SERVICE_NAME" 2>/dev/null || echo 'unknown')${NC}"
  echo ""
}

# ── Menus ───────────────────────────────────────────────

show_menu() {
  banner
  echo -e "${WHITE}--- Service Management ---${NC}"
  echo "  1)  Service status"
  echo "  2)  Start service"
  echo "  3)  Stop service"
  echo "  4)  Restart service"
  echo "  5)  Enable on boot"
  echo "  6)  Disable on boot"
  echo "  7)  View logs (last 80 lines)"
  echo "  8)  Live logs (Ctrl+C to exit)"
  echo ""
  echo -e "${WHITE}--- Website ---${NC}"
  echo "  9)  Health check"
  echo "  10) Self-test"
  echo "  11) Maintenance ON"
  echo "  12) Maintenance OFF"
  echo "  13) Maintenance status"
  echo ""
  echo -e "${WHITE}--- Backup & Update ---${NC}"
  echo "  14) Run update"
  echo "  15) Run rollback"
  echo "  16) Manual DB backup"
  echo "  17) List backups"
  echo ""
  echo -e "${WHITE}--- Nginx ---${NC}"
  echo "  18) Nginx status"
  echo "  19) Nginx reload"
  echo "  20) Nginx error logs"
  echo ""
  echo -e "${WHITE}--- Firewall ---${NC}"
  echo "  21) Firewall status"
  echo ""
  echo -e "${WHITE}--- Telegram Integration ---${NC}"
  echo "  34) Telegram status"
  echo "  35) Test connection (getMe)"
  echo "  36) Send test message"
  echo "  37) Create forum topics"
  echo "  38) Send daily report now"
  echo "  39) Enable Telegram bot"
  echo "  40) Disable Telegram bot"
  echo "  41) Set bot token"
  echo "  42) Set group Chat ID"
  echo ""
  echo "  0)  Exit"
  echo ""
}

main() {
  check_root
  check_installed

  while true; do
    show_menu
    read -rp "Select option: " choice
    echo ""
    case "$choice" in
      1)  service_status ;;
      2)  service_start ;;
      3)  service_stop ;;
      4)  service_restart ;;
      5)  service_enable ;;
      6)  service_disable ;;
      7)  service_logs ;;
      8)  service_logs_live ;;
      9)  health_check ;;
      10) run_self_test ;;
      11) maintenance_on ;;
      12) maintenance_off ;;
      13) maintenance_status ;;
      14) run_update ;;
      15) run_rollback ;;
      16) backup_db ;;
      17) list_backups ;;
      18) nginx_status ;;
      19) nginx_reload ;;
      20) nginx_logs ;;
      21) fw_status ;;
      34) tg_status ;;
      35) tg_test ;;
      36) tg_send_test ;;
      37) tg_create_topics ;;
      38) tg_daily_report ;;
      39) tg_enable ;;
      40) tg_disable ;;
      41) tg_set_token ;;
      42) tg_set_chat_id ;;
      0)  echo "Goodbye."; exit 0 ;;
      *)  warn "Unknown option: $choice" ;;
    esac
    press_enter
  done
}

main "$@"
