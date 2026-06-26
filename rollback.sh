#!/usr/bin/env bash
set -euo pipefail

# =====================================================
#  ZedProxy Rollback
#  Usage: sudo bash /opt/zedproxy/rollback.sh
# =====================================================

export LC_ALL=C.UTF-8
export LANG=C.UTF-8

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
  error "This script must be run as root. Please use sudo."
fi

echo -e "${CYAN}"
echo "╔══════════════════════════════════════════╗"
echo "║       ZedProxy Rollback Tool             ║"
echo "║   Restore previous ZedProxy binary       ║"
echo "╚══════════════════════════════════════════╝"
echo -e "${NC}"

step "Searching for backup binaries"

# List available backup binaries
echo -e "${WHITE}Available backups:${NC}"
ls -lht "${INSTALL_DIR}"/zedproxy.backup-* 2>/dev/null | head -10 | sed 's/^/  /' || true

# Find the most recent backup binary
LATEST=$(ls -t "${INSTALL_DIR}"/zedproxy.backup-* 2>/dev/null | head -1)
if [[ -z "$LATEST" ]]; then
  error "No backup binary found at ${INSTALL_DIR}/zedproxy.backup-*"
fi

echo ""
echo -e "  Restoring backup: ${CYAN}${LATEST}${NC}"
echo -e "  Current binary:   ${CYAN}${INSTALL_DIR}/zedproxy${NC}"
echo ""
read -rp "Proceed with rollback? (y/n): " CONFIRM
if [[ "$CONFIRM" != "y" && "$CONFIRM" != "Y" ]]; then
  warn "Rollback cancelled."
  exit 0
fi

step "Stopping service"
systemctl stop "$SERVICE_NAME" 2>/dev/null || true
sleep 1

step "Replacing binary"
cp "${LATEST}" "${INSTALL_DIR}/zedproxy"
chmod +x "${INSTALL_DIR}/zedproxy"
chown www-data:www-data "${INSTALL_DIR}/zedproxy" 2>/dev/null || true
info "Binary replaced"

step "Starting service"
systemctl start "$SERVICE_NAME"
sleep 3

if systemctl is-active --quiet "$SERVICE_NAME"; then
  info "Service $SERVICE_NAME is active"
else
  error "Service failed to start — check logs: journalctl -u $SERVICE_NAME -n 50"
fi

echo ""
echo -e "${GREEN}╔══════════════════════════════════════════════════╗${NC}"
echo -e "${GREEN}║       Rollback completed successfully!            ║${NC}"
echo -e "${GREEN}╚══════════════════════════════════════════════════╝${NC}"
echo ""
echo -e "${WHITE}Useful commands:${NC}"
echo -e "  sudo systemctl status ${SERVICE_NAME}    # Service status"
echo -e "  sudo journalctl -u ${SERVICE_NAME} -f   # Live logs"
echo ""
