#!/usr/bin/env bash
set -euo pipefail

# =====================================================
#  ZedProxy Rollback
#  Usage: sudo bash /opt/zedproxy/rollback.sh
# =====================================================

INSTALL_DIR="/opt/zedproxy"
SERVICE_NAME="zedproxy"

RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
WHITE='\033[1;37m'
CYAN='\033[0;36m'
NC='\033[0m'

info()   { echo -e "${GREEN}[✓]${NC} $1"; }
warn()   { echo -e "${YELLOW}[!]${NC} $1"; }
error()  { echo -e "${RED}[✗]${NC} $1"; exit 1; }
step()   { echo -e "\n${WHITE}━━━ $1 ━━━${NC}"; }

if [[ $EUID -ne 0 ]]; then
  error "این اسکریپت باید با دسترسی root اجرا شود. از sudo استفاده کنید."
fi

echo -e "${CYAN}"
echo "╔══════════════════════════════════════════╗"
echo "║       ZedProxy Rollback Tool             ║"
echo "║   بازگشت به نسخه قبلی ZedProxy          ║"
echo "╚══════════════════════════════════════════╝"
echo -e "${NC}"

step "جستجوی باینری پشتیبان"

# Find the most recent backup binary
LATEST=$(ls -t "${INSTALL_DIR}"/zedproxy.backup-* 2>/dev/null | head -1)
if [[ -z "$LATEST" ]]; then
  error "هیچ باینری پشتیبانی یافت نشد در ${INSTALL_DIR}/zedproxy.backup-*"
fi

echo -e "  باینری برگشت: ${CYAN}${LATEST}${NC}"
echo -e "  باینری فعلی:  ${CYAN}${INSTALL_DIR}/zedproxy${NC}"
echo ""
read -rp "آیا می‌خواهید Rollback انجام شود؟ (y/n): " CONFIRM
if [[ "$CONFIRM" != "y" && "$CONFIRM" != "Y" ]]; then
  warn "Rollback لغو شد."
  exit 0
fi

step "توقف سرویس"
systemctl stop "$SERVICE_NAME" 2>/dev/null || true
sleep 1

step "جایگزینی باینری"
cp "${LATEST}" "${INSTALL_DIR}/zedproxy"
chmod +x "${INSTALL_DIR}/zedproxy"
chown www-data:www-data "${INSTALL_DIR}/zedproxy" 2>/dev/null || true
info "باینری جایگزین شد"

step "راه‌اندازی سرویس"
systemctl start "$SERVICE_NAME"
sleep 3

if systemctl is-active --quiet "$SERVICE_NAME"; then
  info "سرویس $SERVICE_NAME فعال است"
else
  error "راه‌اندازی سرویس ناموفق — لاگ‌ها را بررسی کنید: journalctl -u $SERVICE_NAME -n 50"
fi

echo ""
echo -e "${GREEN}╔══════════════════════════════════════════════════╗${NC}"
echo -e "${GREEN}║        ✅ Rollback با موفقیت انجام شد!           ║${NC}"
echo -e "${GREEN}╚══════════════════════════════════════════════════╝${NC}"
echo ""
echo -e "${WHITE}دستورات مفید:${NC}"
echo -e "  sudo systemctl status ${SERVICE_NAME}    # وضعیت سرویس"
echo -e "  sudo journalctl -u ${SERVICE_NAME} -f   # مشاهده لاگ زنده"
echo ""
