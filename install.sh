#!/usr/bin/env bash
set -e

# =====================================================
#  ZedProxy Website Installer
#  Tested on Ubuntu 20.04 / 22.04 / 24.04
# =====================================================

export LC_ALL=C.UTF-8
export LANG=C.UTF-8

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
    error "This script must be run as root. Please use sudo."
  fi
}

get_domain() {
  echo ""
  echo -e "${CYAN}Please enter your domain name (example: zedproxy.com):${NC}"
  read -rp "Domain: " DOMAIN
  if [[ -z "$DOMAIN" ]]; then
    error "Domain cannot be empty."
  fi
  DOMAIN=$(echo "$DOMAIN" | tr '[:upper:]' '[:lower:]' | sed 's|https\?://||')
  info "Domain: $DOMAIN"
}

generate_secrets() {
  SESSION_SECRET=$(openssl rand -base64 48 | tr -dc 'a-zA-Z0-9' | head -c 64)
  ADMIN_PASSWORD=$(openssl rand -base64 18 | tr -dc 'a-zA-Z0-9' | head -c 16)
  info "Admin password generated"
}

install_packages() {
  step "Installing required packages"
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
  info "Required packages installed"
}

install_go() {
  step "Installing Go $GO_VERSION"
  if command -v go &>/dev/null; then
    CURRENT_GO=$(go version | awk '{print $3}' | sed 's/go//')
    if [[ "$CURRENT_GO" == "$GO_VERSION" ]]; then
      info "Go $GO_VERSION is already installed"
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

  info "Go $(go version | awk '{print $3}') installed"
}

setup_directory() {
  step "Creating directory structure"
  mkdir -p "$INSTALL_DIR"/{data/backups,templates,static/uploads,logs}
  info "Directories created: $INSTALL_DIR"
}

clone_or_copy() {
  step "Fetching project files"
  SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"

  if [[ -f "$SCRIPT_DIR/main.go" ]]; then
    info "Using local files from $SCRIPT_DIR"
    BUILD_DIR="$SCRIPT_DIR"
  else
    info "Cloning from GitHub..."
    CLONE_DIR="/tmp/zedproxy-build-$$"
    git clone "$REPO_URL" "$CLONE_DIR"
    BUILD_DIR="$CLONE_DIR"
  fi
}

build_app() {
  step "Building Go application"
  export PATH="/usr/local/go/bin:$PATH"
  cd "$BUILD_DIR"

  go mod download
  CGO_ENABLED=1 go build -ldflags="-s -w" -o "$INSTALL_DIR/zedproxy" .

  info "Application built successfully"
  ls -lh "$INSTALL_DIR/zedproxy"
}

copy_assets() {
  step "Copying templates and static files"
  cp -r "$BUILD_DIR/templates/"* "$INSTALL_DIR/templates/"
  # Preserve uploaded files; only copy non-uploads static content
  rsync -a --exclude 'uploads/' "$BUILD_DIR/static/" "$INSTALL_DIR/static/" 2>/dev/null || \
    cp -r "$BUILD_DIR/static/"* "$INSTALL_DIR/static/" 2>/dev/null || true
  info "Templates and static files copied"
}

install_update_script() {
  step "Installing update script"
  if [[ -f "$BUILD_DIR/update.sh" ]]; then
    cp "$BUILD_DIR/update.sh" "$INSTALL_DIR/update.sh"
    chmod +x "$INSTALL_DIR/update.sh"
    chown root:root "$INSTALL_DIR/update.sh"
  else
    warn "update.sh not found in project — attempting direct download..."
    curl -fsSL "https://raw.githubusercontent.com/mhoseinshah1/zed_website/main/update.sh" \
      -o "$INSTALL_DIR/update.sh" 2>/dev/null || true
    chmod +x "$INSTALL_DIR/update.sh" 2>/dev/null || true
    chown root:root "$INSTALL_DIR/update.sh" 2>/dev/null || true
  fi
  if [[ ! -f "$INSTALL_DIR/update.sh" ]]; then
    warn "update.sh could not be installed. To update manually, run:"
    warn "  curl -fsSL https://raw.githubusercontent.com/mhoseinshah1/zed_website/main/update.sh -o /opt/zedproxy/update.sh"
    warn "  chmod +x /opt/zedproxy/update.sh"
  else
    info "update.sh installed: $INSTALL_DIR/update.sh"
  fi
}

install_manage_script() {
  step "Installing manager script"
  if [[ -f "$BUILD_DIR/manage.sh" ]]; then
    cp "$BUILD_DIR/manage.sh" "$INSTALL_DIR/manage.sh"
    chmod +x "$INSTALL_DIR/manage.sh"
    chown root:root "$INSTALL_DIR/manage.sh"
    ln -sf "$INSTALL_DIR/manage.sh" /usr/local/bin/zedproxy-manager
    info "Manager installed: $INSTALL_DIR/manage.sh"
    info "Shortcut: zedproxy-manager (run as root)"
  else
    warn "manage.sh not found — skipping"
  fi
}

seed_database() {
  step "Initializing database"
  # Skip seeding if DB already exists (reinstall protection)
  if [[ -f "$INSTALL_DIR/data/zedproxy.db" ]]; then
    warn "Existing database found — skipping seed (data preserved)"
    return
  fi
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

  info "Database initialized successfully"
}

create_env() {
  step "Creating environment config"
  # Skip if .env already exists (reinstall protection)
  if [[ -f "$INSTALL_DIR/.env" ]]; then
    warn "Existing .env found — not overwriting (SESSION_SECRET preserved)"
    return
  fi
  cat > "$INSTALL_DIR/.env" <<EOF
SESSION_SECRET=${SESSION_SECRET}
GIN_MODE=release
EOF
  chmod 600 "$INSTALL_DIR/.env"
  info ".env file created"
}

setup_systemd() {
  step "Configuring systemd service"
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
  info "systemd service configured"
}

setup_nginx() {
  step "Configuring Nginx"
  mkdir -p /var/www/letsencrypt/.well-known/acme-challenge
  cat > "/etc/nginx/sites-available/zedproxy" <<EOF
server {
    listen 80;
    listen [::]:80;
    server_name ${DOMAIN};

    location ^~ /.well-known/acme-challenge/ {
        root /var/www/letsencrypt;
        default_type "text/plain";
        try_files \$uri =404;
    }

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
  info "Nginx configured"
}

setup_ssl() {
  step "Obtaining SSL certificate from Let's Encrypt"
  warn "Requesting SSL certificate for $DOMAIN ..."

  if certbot --nginx -d "$DOMAIN" \
    --non-interactive \
    --agree-tos \
    --email "ssl@${DOMAIN}" \
    --redirect 2>/dev/null; then
    info "SSL configured successfully"
  else
    warn "SSL setup failed. You can retry later with:"
    warn "  certbot --nginx -d $DOMAIN"
  fi
}

set_permissions() {
  step "Setting file permissions"
  chown -R www-data:www-data "$INSTALL_DIR"
  chmod -R 755 "$INSTALL_DIR"
  chmod 600 "$INSTALL_DIR/.env"
  chmod 755 "$INSTALL_DIR/zedproxy"
  chmod -R 775 "$INSTALL_DIR/data"
  chmod -R 775 "$INSTALL_DIR/static/uploads"
  info "Permissions set"
}

setup_firewall() {
  step "Configuring firewall"
  ufw allow 22/tcp 2>/dev/null || true
  ufw allow 80/tcp 2>/dev/null || true
  ufw allow 443/tcp 2>/dev/null || true
  ufw --force enable 2>/dev/null || true
  info "Firewall configured"
}

start_service() {
  step "Starting service"
  systemctl start "$SERVICE_NAME"
  sleep 2

  if systemctl is-active --quiet "$SERVICE_NAME"; then
    info "ZedProxy service started successfully"
  else
    warn "Service failed to start. Recent logs:"
    journalctl -u "$SERVICE_NAME" -n 20 --no-pager
    error "Failed to start ZedProxy service."
  fi
}

setup_telegram_optional() {
  step "Optional: Telegram Admin Bot"
  echo ""
  echo -e "${CYAN}Do you want to configure the Telegram admin bot now? (y/N):${NC}"
  read -rp "Choice: " TG_CHOICE
  if [[ "${TG_CHOICE,,}" != "y" ]]; then
    info "Telegram bot skipped. Configure later at /zed-admin/integrations/telegram"
    return
  fi

  read -rsp "Enter Telegram bot token (from @BotFather): " TG_TOKEN
  echo ""
  if [[ -z "$TG_TOKEN" ]]; then
    warn "No token entered — skipping Telegram setup"
    return
  fi

  read -rp "Enter Telegram group Chat ID (e.g. -1001234567890): " TG_CHAT
  if [[ -z "$TG_CHAT" ]]; then
    warn "No Chat ID entered — skipping Telegram setup"
    return
  fi

  "$INSTALL_DIR/zedproxy" --db="$INSTALL_DIR/data/zedproxy.db" --telegram-set-token="$TG_TOKEN"
  "$INSTALL_DIR/zedproxy" --db="$INSTALL_DIR/data/zedproxy.db" --telegram-set-chat-id="$TG_CHAT"
  "$INSTALL_DIR/zedproxy" --db="$INSTALL_DIR/data/zedproxy.db" --telegram-enable

  echo ""
  echo -e "${CYAN}Create forum topics in your Telegram group now? (y/N):${NC}"
  read -rp "Choice: " TOPICS_CHOICE
  if [[ "${TOPICS_CHOICE,,}" == "y" ]]; then
    "$INSTALL_DIR/zedproxy" --db="$INSTALL_DIR/data/zedproxy.db" --telegram-create-topics || \
      warn "Topic creation failed — try later from admin panel or manage.sh option 37"
  fi

  info "Telegram bot configured"
  info "Test with: sudo zedproxy-manager (option 35)"
}

print_result() {
  SITE_URL="https://$DOMAIN"
  ADMIN_URL="$SITE_URL/zed-admin"

  echo ""
  echo -e "${GREEN}╔═══════════════════════════════════════════════════╗${NC}"
  echo -e "${GREEN}║       Installation completed successfully!         ║${NC}"
  echo -e "${GREEN}╚═══════════════════════════════════════════════════╝${NC}"
  echo ""
  echo -e "${WHITE}Access information:${NC}"
  echo -e "  Website:       ${CYAN}${SITE_URL}${NC}"
  echo -e "  Admin panel:   ${CYAN}${ADMIN_URL}${NC}"
  echo -e "  Username:      ${WHITE}${ADMIN_USERNAME}${NC}"
  echo -e "  Password:      ${RED}${ADMIN_PASSWORD}${NC}"
  echo ""
  echo -e "${YELLOW}WARNING: Save this password in a secure place!${NC}"
  echo ""
  echo -e "${WHITE}Useful commands:${NC}"
  echo -e "  sudo systemctl status zedproxy         # Service status"
  echo -e "  sudo systemctl restart zedproxy        # Restart service"
  echo -e "  sudo journalctl -u zedproxy -f         # Live logs"
  echo -e "  sudo bash /opt/zedproxy/update.sh      # Update website"
  echo ""
  echo -e "${WHITE}Recovery — if update.sh is missing:${NC}"
  echo -e "  sudo curl -fsSL https://raw.githubusercontent.com/mhoseinshah1/zed_website/main/update.sh \\"
  echo -e "    -o /opt/zedproxy/update.sh && sudo chmod +x /opt/zedproxy/update.sh"
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
install_manage_script
create_env
seed_database
set_permissions
setup_systemd
setup_nginx
setup_firewall
start_service
setup_ssl
setup_telegram_optional
print_result
