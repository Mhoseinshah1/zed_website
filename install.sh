#!/usr/bin/env bash
set -e

# =====================================================
#  ZedProxy Website Installer
#  Tested on Ubuntu 20.04 / 22.04 / 24.04
# =====================================================

RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
CYAN='\033[0;36m'
WHITE='\033[1;37m'
NC='\033[0m'

INSTALL_DIR="/opt/zedproxy"
SERVICE_NAME="zedproxy"
APP_PORT="8080"
REPO_URL="https://github.com/mhoseinshah1/zed_website"
GO_VERSION="1.22.4"
ADMIN_USERNAME="admin"
ADMIN_EMAIL="admin@zedproxy.com"
TELEGRAM_BOT="https://t.me/zedproxy_bot"

banner() {
  echo -e "${CYAN}"
  echo "╔═══════════════════════════════════════╗"
  echo "║        ZedProxy Website Installer     ║"
  echo "║     Premium Persian RTL VPN Website   ║"
  echo "╚═══════════════════════════════════════╝"
  echo -e "${NC}"
}

info()    { echo -e "${GREEN}[✓]${NC} $1"; }
warn()    { echo -e "${YELLOW}[!]${NC} $1"; }
error()   { echo -e "${RED}[✗]${NC} $1"; exit 1; }
step()    { echo -e "\n${WHITE}━━━ $1 ━━━${NC}"; }

check_root() {
  if [[ $EUID -ne 0 ]]; then
    error "این اسکریپت باید با دسترسی root اجرا شود. از sudo استفاده کنید."
  fi
}

get_domain() {
  echo ""
  echo -e "${CYAN}لطفاً دامنه سایت خود را وارد کنید (مثال: zedproxy.com):${NC}"
  read -rp "Domain: " DOMAIN
  if [[ -z "$DOMAIN" ]]; then
    error "دامنه نمی‌تواند خالی باشد"
  fi
  DOMAIN=$(echo "$DOMAIN" | tr '[:upper:]' '[:lower:]' | sed 's|https\?://||')
  info "دامنه: $DOMAIN"
}

generate_secrets() {
  SESSION_SECRET=$(openssl rand -base64 48 | tr -dc 'a-zA-Z0-9' | head -c 64)
  ADMIN_PASSWORD=$(openssl rand -base64 18 | tr -dc 'a-zA-Z0-9' | head -c 16)
  info "رمز عبور ادمین تولید شد"
}

install_packages() {
  step "نصب پیش‌نیازها"
  apt-get update -qq
  apt-get install -y -qq \
    nginx \
    certbot \
    python3-certbot-nginx \
    sqlite3 \
    git \
    curl \
    wget \
    unzip \
    build-essential \
    gcc \
    libsqlite3-dev \
    ufw 2>/dev/null || true
  info "پکیج‌های مورد نیاز نصب شدند"
}

install_go() {
  step "نصب Go $GO_VERSION"
  if command -v go &>/dev/null; then
    CURRENT_GO=$(go version | awk '{print $3}' | sed 's/go//')
    if [[ "$CURRENT_GO" == "$GO_VERSION" ]]; then
      info "Go $GO_VERSION قبلاً نصب شده"
      return
    fi
  fi

  cd /tmp
  GO_ARCH="amd64"
  if [[ "$(uname -m)" == "aarch64" ]]; then GO_ARCH="arm64"; fi
  GO_TAR="go${GO_VERSION}.linux-${GO_ARCH}.tar.gz"

  wget -q "https://go.dev/dl/${GO_TAR}" -O "/tmp/${GO_TAR}"
  rm -rf /usr/local/go
  tar -C /usr/local -xzf "/tmp/${GO_TAR}"
  rm "/tmp/${GO_TAR}"

  export PATH="/usr/local/go/bin:$PATH"
  echo 'export PATH="/usr/local/go/bin:$PATH"' > /etc/profile.d/go.sh
  chmod +x /etc/profile.d/go.sh

  info "Go $(go version | awk '{print $3}') نصب شد"
}

setup_directory() {
  step "ایجاد ساختار پوشه‌ها"
  mkdir -p "$INSTALL_DIR"/{data/backups,templates,static/uploads,logs}
  info "پوشه‌ها ایجاد شدند: $INSTALL_DIR"
}

clone_or_copy() {
  step "دریافت فایل‌های پروژه"
  SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"

  if [[ -f "$SCRIPT_DIR/main.go" ]]; then
    info "استفاده از فایل‌های موجود در $SCRIPT_DIR"
    BUILD_DIR="$SCRIPT_DIR"
  else
    info "دریافت کد از GitHub..."
    CLONE_DIR="/tmp/zedproxy-build-$$"
    git clone "$REPO_URL" "$CLONE_DIR"
    BUILD_DIR="$CLONE_DIR"
  fi
}

build_app() {
  step "کامپایل برنامه Go"
  export PATH="/usr/local/go/bin:$PATH"
  cd "$BUILD_DIR"

  go mod download
  CGO_ENABLED=1 go build -ldflags="-s -w" -o "$INSTALL_DIR/zedproxy" .

  info "برنامه با موفقیت کامپایل شد"
  ls -lh "$INSTALL_DIR/zedproxy"
}

copy_assets() {
  step "کپی فایل‌های قالب و استاتیک"
  cp -r "$BUILD_DIR/templates/"* "$INSTALL_DIR/templates/"
  # Preserve uploaded files; only copy non-uploads static content
  rsync -a --exclude 'uploads/' "$BUILD_DIR/static/" "$INSTALL_DIR/static/" 2>/dev/null || \
    cp -r "$BUILD_DIR/static/"* "$INSTALL_DIR/static/" 2>/dev/null || true
  info "فایل‌های قالب کپی شدند"
}

install_update_script() {
  step "نصب اسکریپت بروزرسانی"
  if [[ -f "$BUILD_DIR/update.sh" ]]; then
    cp "$BUILD_DIR/update.sh" "$INSTALL_DIR/update.sh"
    chmod +x "$INSTALL_DIR/update.sh"
    chown root:root "$INSTALL_DIR/update.sh"
    info "update.sh نصب شد: $INSTALL_DIR/update.sh"
  else
    warn "update.sh در کد پروژه یافت نشد — رد شد"
  fi
}

seed_database() {
  step "مقداردهی اولیه پایگاه داده"
  export PATH="/usr/local/go/bin:$PATH"
  cd "$INSTALL_DIR"

  "$INSTALL_DIR/zedproxy" \
    --db="$INSTALL_DIR/data/zedproxy.db" \
    --templates="$INSTALL_DIR/templates" \
    --static="$INSTALL_DIR/static" \
    --uploads="$INSTALL_DIR/static/uploads" \
    --seed \
    --admin-user="$ADMIN_USERNAME" \
    --admin-email="$ADMIN_EMAIL" \
    --admin-pass="$ADMIN_PASSWORD" \
    --secret="$SESSION_SECRET"

  # Update site URL in settings
  sqlite3 "$INSTALL_DIR/data/zedproxy.db" \
    "INSERT INTO settings (key, value) VALUES ('site_url', 'https://$DOMAIN') ON CONFLICT(key) DO UPDATE SET value='https://$DOMAIN';"

  info "پایگاه داده مقداردهی اولیه شد"
}

create_env() {
  step "ایجاد فایل تنظیمات محیطی"
  cat > "$INSTALL_DIR/.env" <<EOF
SESSION_SECRET=${SESSION_SECRET}
GIN_MODE=release
EOF
  chmod 600 "$INSTALL_DIR/.env"
  info "فایل .env ایجاد شد"
}

setup_systemd() {
  step "تنظیم سرویس systemd"
  cat > "/etc/systemd/system/${SERVICE_NAME}.service" <<EOF
[Unit]
Description=ZedProxy Website
After=network.target

[Service]
Type=simple
User=www-data
WorkingDirectory=${INSTALL_DIR}
ExecStart=${INSTALL_DIR}/zedproxy \\
  --addr=127.0.0.1:${APP_PORT} \\
  --db=${INSTALL_DIR}/data/zedproxy.db \\
  --templates=${INSTALL_DIR}/templates \\
  --static=${INSTALL_DIR}/static \\
  --uploads=${INSTALL_DIR}/static/uploads \\
  --backups=${INSTALL_DIR}/data/backups \\
  --secret=\${SESSION_SECRET}
Restart=always
RestartSec=5
StandardOutput=journal
StandardError=journal
SyslogIdentifier=zedproxy
EnvironmentFile=${INSTALL_DIR}/.env
NoNewPrivileges=true
PrivateTmp=true

[Install]
WantedBy=multi-user.target
EOF

  systemctl daemon-reload
  systemctl enable "$SERVICE_NAME"
  info "سرویس systemd تنظیم شد"
}

setup_nginx() {
  step "تنظیم Nginx"
  cat > "/etc/nginx/sites-available/zedproxy" <<EOF
server {
    listen 80;
    listen [::]:80;
    server_name ${DOMAIN};

    location / {
        proxy_pass http://127.0.0.1:${APP_PORT};
        proxy_http_version 1.1;
        proxy_set_header Host \$host;
        proxy_set_header X-Real-IP \$remote_addr;
        proxy_set_header X-Forwarded-For \$proxy_add_x_forwarded_for;
        proxy_set_header X-Forwarded-Proto \$scheme;
        proxy_read_timeout 60s;
        proxy_connect_timeout 10s;

        add_header X-Frame-Options "SAMEORIGIN" always;
        add_header X-Content-Type-Options "nosniff" always;
        add_header X-XSS-Protection "1; mode=block" always;
    }

    location /static/ {
        proxy_pass http://127.0.0.1:${APP_PORT}/static/;
        expires 7d;
        add_header Cache-Control "public, max-age=604800";
    }

    location /uploads/ {
        proxy_pass http://127.0.0.1:${APP_PORT}/uploads/;
        expires 30d;
        add_header Cache-Control "public, max-age=2592000";
    }

    client_max_body_size 10M;
}
EOF

  ln -sf /etc/nginx/sites-available/zedproxy /etc/nginx/sites-enabled/zedproxy
  rm -f /etc/nginx/sites-enabled/default 2>/dev/null || true

  nginx -t
  systemctl reload nginx
  info "Nginx تنظیم شد"
}

setup_ssl() {
  step "دریافت گواهی SSL از Let's Encrypt"
  warn "در حال دریافت گواهی SSL برای $DOMAIN ..."

  if certbot --nginx -d "$DOMAIN" \
    --non-interactive \
    --agree-tos \
    --email "ssl@${DOMAIN}" \
    --redirect 2>/dev/null; then
    info "SSL با موفقیت تنظیم شد"
  else
    warn "تنظیم SSL ناموفق بود. می‌توانید بعداً با دستور زیر دوباره تلاش کنید:"
    warn "certbot --nginx -d $DOMAIN"
  fi
}

set_permissions() {
  step "تنظیم دسترسی‌ها"
  chown -R www-data:www-data "$INSTALL_DIR"
  chmod -R 755 "$INSTALL_DIR"
  chmod 600 "$INSTALL_DIR/.env"
  chmod 755 "$INSTALL_DIR/zedproxy"
  chmod -R 775 "$INSTALL_DIR/data"
  chmod -R 775 "$INSTALL_DIR/static/uploads"
  info "دسترسی‌ها تنظیم شدند"
}

setup_firewall() {
  step "تنظیم فایروال"
  ufw allow 22/tcp 2>/dev/null || true
  ufw allow 80/tcp 2>/dev/null || true
  ufw allow 443/tcp 2>/dev/null || true
  ufw --force enable 2>/dev/null || true
  info "فایروال تنظیم شد"
}

start_service() {
  step "راه‌اندازی سرویس"
  systemctl start "$SERVICE_NAME"
  sleep 2

  if systemctl is-active --quiet "$SERVICE_NAME"; then
    info "سرویس ZedProxy با موفقیت راه‌اندازی شد"
  else
    warn "سرویس راه‌اندازی نشد. لاگ‌ها را بررسی کنید:"
    journalctl -u "$SERVICE_NAME" -n 20 --no-pager
    error "راه‌اندازی سرویس ناموفق"
  fi
}

print_result() {
  SITE_URL="https://$DOMAIN"
  ADMIN_URL="$SITE_URL/zed-admin"

  echo ""
  echo -e "${GREEN}╔═══════════════════════════════════════════════════╗${NC}"
  echo -e "${GREEN}║          ✅ نصب با موفقیت انجام شد!               ║${NC}"
  echo -e "${GREEN}╚═══════════════════════════════════════════════════╝${NC}"
  echo ""
  echo -e "${WHITE}اطلاعات دسترسی:${NC}"
  echo -e "  🌐  سایت:         ${CYAN}${SITE_URL}${NC}"
  echo -e "  🔐  پنل مدیریت:   ${CYAN}${ADMIN_URL}${NC}"
  echo -e "  👤  نام کاربری:   ${WHITE}${ADMIN_USERNAME}${NC}"
  echo -e "  🔑  رمز عبور:     ${RED}${ADMIN_PASSWORD}${NC}"
  echo ""
  echo -e "${YELLOW}⚠️  رمز عبور را در جای امنی ذخیره کنید!${NC}"
  echo ""
  echo -e "${WHITE}دستورات مفید:${NC}"
  echo -e "  sudo systemctl status zedproxy         # وضعیت سرویس"
  echo -e "  sudo systemctl restart zedproxy        # ریستارت سرویس"
  echo -e "  sudo journalctl -u zedproxy -f         # مشاهده لاگ"
  echo -e "  sudo bash /opt/zedproxy/update.sh      # بروزرسانی سایت"
  echo ""
}

# =====================================================
# Main execution
# =====================================================
banner
check_root
get_domain
generate_secrets
install_packages
install_go
setup_directory
clone_or_copy
build_app
copy_assets
install_update_script
create_env
seed_database
set_permissions
setup_systemd
setup_nginx
setup_firewall
start_service
setup_ssl
print_result
