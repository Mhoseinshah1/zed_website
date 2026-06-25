#!/usr/bin/env bash
set -euo pipefail

# =====================================================
#  ZedProxy Website Updater
#  Tested on Ubuntu 20.04 / 22.04 / 24.04
#  Usage: sudo bash /opt/zedproxy/update.sh
# =====================================================

REPO_URL="https://github.com/mhoseinshah1/zed_website.git"
BRANCH="main"
INSTALL_DIR="/opt/zedproxy"
SERVICE_NAME="zedproxy"
GO_VERSION="1.22.4"
APP_PORT="8080"
TIMESTAMP="$(date +%Y%m%d-%H%M%S)"
BUILD_DIR="/tmp/zedproxy-update-${TIMESTAMP}"
BACKUP_DIR="${INSTALL_DIR}/backups"
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
detail()  { echo -e "    ${BLUE}→${NC} $1"; }

# ── Cleanup trap ────────────────────────────────────
# Called on EXIT; only removes the temp build dir.
# Never touches INSTALL_DIR data.
cleanup() {
  local exit_code=$?
  if [[ -d "$BUILD_DIR" ]]; then
    rm -rf "$BUILD_DIR"
    detail "پوشه موقت پاکسازی شد: $BUILD_DIR"
  fi
  if [[ $exit_code -ne 0 ]]; then
    echo -e "\n${RED}━━━ بازیابی پس از خطا ━━━${NC}"
    warn "خطا رخ داد. تلاش برای راه‌اندازی مجدد سرویس قدیمی..."
    systemctl start "$SERVICE_NAME" 2>/dev/null || true
    sleep 2
    if systemctl is-active --quiet "$SERVICE_NAME"; then
      info "سرویس قدیمی مجدداً راه‌اندازی شد"
    else
      warn "سرویس راه‌اندازی نشد. لاگ‌های اخیر:"
      journalctl -u "$SERVICE_NAME" -n 50 --no-pager 2>/dev/null || true
    fi
  fi
}
trap cleanup EXIT

banner() {
  echo -e "${CYAN}"
  echo "╔══════════════════════════════════════════╗"
  echo "║       ZedProxy Website Updater           ║"
  echo "║   بروزرسانی امن سایت ZedProxy           ║"
  echo "╚══════════════════════════════════════════╝"
  echo -e "${NC}"
  echo -e "  زمان:    ${WHITE}${TIMESTAMP}${NC}"
  echo -e "  مخزن:    ${CYAN}${REPO_URL}${NC}"
  echo -e "  شاخه:    ${CYAN}${BRANCH}${NC}"
  echo -e "  مسیر:    ${CYAN}${INSTALL_DIR}${NC}"
  echo ""
}

# ── Pre-flight checks ───────────────────────────────
check_root() {
  if [[ $EUID -ne 0 ]]; then
    echo -e "${RED}[✗]${NC} این اسکریپت باید با دسترسی root اجرا شود."
    echo -e "    اجرا کنید: ${WHITE}sudo bash /opt/zedproxy/update.sh${NC}"
    exit 1
  fi
}

check_installed() {
  if [[ ! -d "$INSTALL_DIR" ]]; then
    echo -e "${RED}[✗]${NC} ZedProxy نصب نشده است."
    echo -e "    ابتدا install.sh را اجرا کنید."
    exit 1
  fi
  if [[ ! -f "${OLD_BINARY}" ]]; then
    echo -e "${RED}[✗]${NC} فایل اجرایی ZedProxy یافت نشد: ${OLD_BINARY}"
    echo -e "    ابتدا install.sh را اجرا کنید."
    exit 1
  fi
}

check_env() {
  if [[ ! -f "$ENV_FILE" ]]; then
    warn "فایل .env یافت نشد: $ENV_FILE"
    warn "سرویس برای اجرا به SESSION_SECRET نیاز دارد."
    warn "مطمئن شوید که .env موجود است یا متغیر SESSION_SECRET تنظیم شده باشد."
    exit 1
  fi
  info "فایل .env موجود است"
}

check_go() {
  export PATH="/usr/local/go/bin:/usr/local/bin:$PATH"
  if ! command -v go &>/dev/null; then
    error "Go یافت نشد. لطفاً ابتدا install.sh را اجرا کنید تا Go نصب شود."
  fi
  local go_ver
  go_ver="$(go version | awk '{print $3}')"
  info "Go موجود است: ${go_ver}"
}

check_git() {
  if ! command -v git &>/dev/null; then
    error "git یافت نشد. اجرا کنید: apt-get install -y git"
  fi
}

# ── Backup ──────────────────────────────────────────
create_backups() {
  step "تهیه نسخه پشتیبان"
  mkdir -p "$BACKUP_DIR"

  # Backup database
  if [[ -f "$DB_FILE" ]]; then
    local db_backup="${BACKUP_DIR}/zedproxy-pre-update-${TIMESTAMP}.db"
    cp "$DB_FILE" "$db_backup"
    info "پشتیبان دیتابیس: $db_backup"
    detail "حجم: $(du -sh "$db_backup" | cut -f1)"
  else
    warn "فایل دیتابیس یافت نشد: $DB_FILE — رد شد"
  fi

  # Backup .env
  local env_backup="${BACKUP_DIR}/.env-pre-update-${TIMESTAMP}"
  cp "$ENV_FILE" "$env_backup"
  chmod 600 "$env_backup"
  info "پشتیبان .env: $env_backup"
}

# ── Clone ────────────────────────────────────────────
clone_repo() {
  step "دریافت آخرین کد از GitHub"
  detail "مخزن: $REPO_URL"
  detail "شاخه: $BRANCH"
  detail "پوشه موقت: $BUILD_DIR"

  git clone --depth=1 --branch "$BRANCH" "$REPO_URL" "$BUILD_DIR"
  info "کد با موفقیت دریافت شد"
  detail "کامیت: $(git -C "$BUILD_DIR" log --oneline -1)"
}

# ── Build ─────────────────────────────────────────────
build_binary() {
  step "بیلد برنامه Go"
  export PATH="/usr/local/go/bin:/usr/local/bin:$PATH"
  cd "$BUILD_DIR"

  detail "go mod download..."
  go mod download 2>&1 | sed 's/^/    /'

  detail "CGO_ENABLED=1 go build..."
  CGO_ENABLED=1 go build -ldflags="-s -w" -o "$NEW_BINARY" .

  if [[ ! -f "$NEW_BINARY" ]]; then
    error "بیلد ناموفق: فایل اجرایی جدید ساخته نشد"
  fi

  info "بیلد موفق"
  detail "حجم باینری: $(du -sh "$NEW_BINARY" | cut -f1)"
}

# ── Stop service ──────────────────────────────────────
stop_service() {
  step "توقف سرویس"
  if systemctl is-active --quiet "$SERVICE_NAME"; then
    systemctl stop "$SERVICE_NAME"
    sleep 2
    info "سرویس $SERVICE_NAME متوقف شد"
  else
    warn "سرویس $SERVICE_NAME در حال اجرا نبود"
  fi
}

# ── Deploy ────────────────────────────────────────────
deploy_binary() {
  step "جایگزینی باینری"
  # Keep a copy of old binary as fallback
  if [[ -f "$OLD_BINARY" ]]; then
    cp "$OLD_BINARY" "${OLD_BINARY}.backup-${TIMESTAMP}"
    detail "باینری قدیمی: ${OLD_BINARY}.backup-${TIMESTAMP}"
  fi
  cp "$NEW_BINARY" "$OLD_BINARY"
  chmod +x "$OLD_BINARY"
  info "باینری جدید جایگزین شد"
}

deploy_templates() {
  step "بروزرسانی قالب‌ها"
  if [[ ! -d "${BUILD_DIR}/templates" ]]; then
    warn "پوشه templates در کد جدید یافت نشد — رد شد"
    return
  fi
  # Sync templates completely (no user data lives here)
  rsync -a --delete "${BUILD_DIR}/templates/" "${INSTALL_DIR}/templates/"
  info "قالب‌ها بروزرسانی شدند"
}

deploy_static() {
  step "بروزرسانی فایل‌های استاتیک"
  if [[ ! -d "${BUILD_DIR}/static" ]]; then
    warn "پوشه static در کد جدید یافت نشد — رد شد"
    return
  fi
  # NEVER touch uploads — exclude it explicitly
  rsync -a \
    --exclude 'uploads/' \
    "${BUILD_DIR}/static/" "${INSTALL_DIR}/static/"

  # Ensure uploads directory still exists with correct permissions
  mkdir -p "$UPLOADS_DIR"
  info "فایل‌های استاتیک بروزرسانی شدند (uploads حفظ شد)"
}

deploy_update_script() {
  # Keep update.sh fresh on the server after each update
  if [[ -f "${BUILD_DIR}/update.sh" ]]; then
    cp "${BUILD_DIR}/update.sh" "${INSTALL_DIR}/update.sh"
    chmod +x "${INSTALL_DIR}/update.sh"
    chown root:root "${INSTALL_DIR}/update.sh"
    detail "update.sh بروزرسانی شد"
  fi
}

# ── Permissions ───────────────────────────────────────
fix_permissions() {
  step "تنظیم دسترسی‌ها"
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
  info "دسترسی‌ها تنظیم شدند"
}

# ── Start & verify ────────────────────────────────────
start_service() {
  step "راه‌اندازی سرویس"
  systemctl daemon-reload
  systemctl start "$SERVICE_NAME"
  sleep 3

  if ! systemctl is-active --quiet "$SERVICE_NAME"; then
    warn "سرویس راه‌اندازی نشد. لاگ‌ها:"
    journalctl -u "$SERVICE_NAME" -n 50 --no-pager || true
    error "راه‌اندازی سرویس ناموفق"
  fi
  info "سرویس $SERVICE_NAME فعال است"
  systemctl status "$SERVICE_NAME" --no-pager -l | head -20 | sed 's/^/    /'
}

health_check() {
  step "بررسی سلامت سایت"
  local url="http://127.0.0.1:${APP_PORT}/health"
  local attempts=5
  local wait=2

  for i in $(seq 1 $attempts); do
    local http_code
    http_code="$(curl -s -o /dev/null -w '%{http_code}' --connect-timeout 5 --max-time 10 "$url" 2>/dev/null || echo '000')"
    if [[ "$http_code" == "200" ]]; then
      info "Health check موفق (HTTP $http_code) — $url"
      return 0
    fi
    detail "تلاش $i/$attempts — HTTP $http_code — صبر ${wait}s..."
    sleep $wait
    wait=$((wait * 2))
  done

  # Fall back to root path if /health not available
  local root_code
  root_code="$(curl -s -o /dev/null -w '%{http_code}' --connect-timeout 5 --max-time 10 "http://127.0.0.1:${APP_PORT}/" 2>/dev/null || echo '000')"
  if [[ "$root_code" =~ ^(200|301|302)$ ]]; then
    info "Health check موفق (HTTP $root_code از صفحه اصلی)"
    return 0
  fi

  warn "Health check ناموفق (HTTP $root_code). سرویس ممکن است هنوز در حال راه‌اندازی باشد."
  warn "لاگ‌ها را بررسی کنید: journalctl -u $SERVICE_NAME -f"
}

# ── Print result ──────────────────────────────────────
print_result() {
  echo ""
  echo -e "${GREEN}╔══════════════════════════════════════════════════════╗${NC}"
  echo -e "${GREEN}║        ✅ بروزرسانی با موفقیت انجام شد!             ║${NC}"
  echo -e "${GREEN}╚══════════════════════════════════════════════════════╝${NC}"
  echo ""
  echo -e "${WHITE}خلاصه بروزرسانی:${NC}"
  echo -e "  📦  مخزن:           ${CYAN}${REPO_URL}${NC}"
  echo -e "  🌿  شاخه:           ${CYAN}${BRANCH}${NC}"
  echo -e "  🕐  زمان:           ${WHITE}${TIMESTAMP}${NC}"
  echo -e "  💾  پشتیبان DB:     ${WHITE}${BACKUP_DIR}/zedproxy-pre-update-${TIMESTAMP}.db${NC}"
  echo ""
  echo -e "${WHITE}دستورات مفید:${NC}"
  echo -e "  sudo systemctl status ${SERVICE_NAME}    # وضعیت سرویس"
  echo -e "  sudo systemctl restart ${SERVICE_NAME}   # ریستارت سرویس"
  echo -e "  sudo journalctl -u ${SERVICE_NAME} -f    # مشاهده لاگ زنده"
  echo -e "  sudo bash ${INSTALL_DIR}/update.sh       # بروزرسانی مجدد"
  echo ""
  echo -e "${YELLOW}داده‌های محفوظ:${NC}"
  echo -e "  ✅  دیتابیس:     ${DB_FILE}"
  echo -e "  ✅  آپلودها:     ${UPLOADS_DIR}"
  echo -e "  ✅  .env:        ${ENV_FILE}"
  echo ""
}

# ── Main ─────────────────────────────────────────────
banner
check_root
check_installed
check_env
check_go
check_git
create_backups
clone_repo
build_binary
stop_service
deploy_binary
deploy_templates
deploy_static
deploy_update_script
fix_permissions
start_service
health_check
print_result
