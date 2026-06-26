#!/usr/bin/env bash
set -euo pipefail

# =====================================================
#  ZedProxy Website Installer
#  Tested on Ubuntu 20.04 / 22.04 / 24.04
#  Usage: sudo bash install.sh
#    or:  bash <(curl -fsSL https://raw.githubusercontent.com/mhoseinshah1/zed_website/main/install.sh)
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

# Will be set by prompts
DOMAIN=""
ADMIN_USERNAME=""
ADMIN_EMAIL=""
ADMIN_PASSWORD=""
SESSION_SECRET=""
TG_CONFIGURED="no"

banner() {
  echo -e "${CYAN}"
  echo "╔═══════════════════════════════════════╗"
  echo "║        ZedProxy Website Installer     ║"
  echo "║     Premium Persian RTL VPN Website   ║"
  echo "╚═══════════════════════════════════════╝"
  echo -e "${NC}"
}

info()    { echo -e "${GREEN}[+]${NC} $1"; }
warn()    { echo -e "${YELLOW}[!]${NC} $1"; }
error()   { echo -e "${RED}[x]${NC} $1"; exit 1; }
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
  DOMAIN=$(echo "$DOMAIN" | tr '[:upper:]' '[:lower:]' | sed 's|https\?://||; s|/.*||')
  info "Domain: $DOMAIN"
}

validate_username() {
  local u="$1"
  if [[ ${#u} -lt 3 || ${#u} -gt 32 ]]; then return 1; fi
  if [[ ! "$u" =~ ^[a-zA-Z0-9._-]+$ ]]; then return 1; fi
  return 0
}

validate_password() {
  local p="$1"
  if [[ ${#p} -lt 8 || ${#p} -gt 128 ]]; then return 1; fi
  return 0
}

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

get_admin_credentials() {
  step "Admin Account Setup"

  # Username
  echo ""
  echo -e "${CYAN}Enter admin username, or press Enter to auto-generate:${NC}"
  read -rp "Admin username: " INPUT_USER
  if [[ -z "$INPUT_USER" ]]; then
    ADMIN_USERNAME="$(gen_username)"
    info "Auto-generated username: $ADMIN_USERNAME"
  else
    if ! validate_username "$INPUT_USER"; then
      warn "Invalid username. Use 3-32 characters: letters, numbers, underscore, hyphen, or dot."
      warn "Auto-generating username instead."
      ADMIN_USERNAME="$(gen_username)"
      info "Auto-generated username: $ADMIN_USERNAME"
    else
      ADMIN_USERNAME="$INPUT_USER"
      info "Username: $ADMIN_USERNAME"
    fi
  fi

  # Email
  echo ""
  echo -e "${CYAN}Enter admin email, or press Enter to use admin@zedproxy.com:${NC}"
  read -rp "Admin email: " INPUT_EMAIL
  if [[ -z "$INPUT_EMAIL" ]]; then
    ADMIN_EMAIL="admin@zedproxy.com"
  else
    if [[ "$INPUT_EMAIL" =~ ^[^@]+@[^@]+\.[^@]+$ ]]; then
      ADMIN_EMAIL="$INPUT_EMAIL"
    else
      warn "Invalid email format. Using admin@zedproxy.com instead."
      ADMIN_EMAIL="admin@zedproxy.com"
    fi
  fi
  info "Admin email: $ADMIN_EMAIL"

  # Password
  echo ""
  echo -e "${CYAN}Enter admin password, or press Enter to auto-generate:${NC}"
  read -rsp "Admin password: " INPUT_PASS
  echo ""
  if [[ -z "$INPUT_PASS" ]]; then
    ADMIN_PASSWORD="$(gen_password)"
    info "Admin password auto-generated (will be shown at the end)"
  else
    if ! validate_password "$INPUT_PASS"; then
      warn "Invalid password. Password must be at least 8 characters."
      warn "Auto-generating password instead."
      ADMIN_PASSWORD="$(gen_password)"
      info "Admin password auto-generated (will be shown at the end)"
    else
      ADMIN_PASSWORD="$INPUT_PASS"
      info "Admin password accepted"
    fi
  fi
}

generate_session_secret() {
  SESSION_SECRET=$(openssl rand -base64 48 | tr -dc 'a-zA-Z0-9' | head -c 64)
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
  export PATH="/usr/local/go/bin:/usr/local/bin:$PATH"
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
  export PATH="/usr/local/go/bin:/usr/local/bin:$PATH"
  cd "$BUILD_DIR"

  go mod download
  CGO_ENABLED=1 go build -ldflags="-s -w" -o "$INSTALL_DIR/zedproxy" .

  info "Application built successfully"
  ls -lh "$INSTALL_DIR/zedproxy"
}

copy_assets() {
  step "Copying templates and static files"
  cp -r "$BUILD_DIR/templates/"* "$INSTALL_DIR/templates/"
  rsync -a --exclude 'uploads/' "$BUILD_DIR/static/" "$INSTALL_DIR/static/" 2>/dev/null || \
    cp -r "$BUILD_DIR/static/"* "$INSTALL_DIR/static/" 2>/dev/null || true
  info "Templates and static files copied"
}

install_scripts() {
  step "Installing deployment scripts"

  # update.sh
  if [[ -f "$BUILD_DIR/update.sh" ]]; then
    cp "$BUILD_DIR/update.sh" "$INSTALL_DIR/update.sh"
    chmod +x "$INSTALL_DIR/update.sh"
    chown root:root "$INSTALL_DIR/update.sh"
    info "update.sh installed"
  else
    warn "update.sh not found in project — downloading from GitHub..."
    curl -fsSL "https://raw.githubusercontent.com/mhoseinshah1/zed_website/main/update.sh" \
      -o "$INSTALL_DIR/update.sh" 2>/dev/null || true
    chmod +x "$INSTALL_DIR/update.sh" 2>/dev/null || true
    chown root:root "$INSTALL_DIR/update.sh" 2>/dev/null || true
    [[ -f "$INSTALL_DIR/update.sh" ]] && info "update.sh downloaded" || warn "update.sh could not be installed"
  fi

  # manage.sh
  if [[ -f "$BUILD_DIR/manage.sh" ]]; then
    cp "$BUILD_DIR/manage.sh" "$INSTALL_DIR/manage.sh"
    chmod +x "$INSTALL_DIR/manage.sh"
    chown root:root "$INSTALL_DIR/manage.sh"
    ln -sf "$INSTALL_DIR/manage.sh" /usr/local/bin/zedproxy-manager
    info "manage.sh installed"
    info "Server manager shortcut: sudo zedproxy-manager"
  else
    warn "manage.sh not found in project — downloading from GitHub..."
    curl -fsSL "https://raw.githubusercontent.com/mhoseinshah1/zed_website/main/manage.sh" \
      -o "$INSTALL_DIR/manage.sh" 2>/dev/null || true
    if [[ -f "$INSTALL_DIR/manage.sh" ]]; then
      chmod +x "$INSTALL_DIR/manage.sh"
      chown root:root "$INSTALL_DIR/manage.sh"
      ln -sf "$INSTALL_DIR/manage.sh" /usr/local/bin/zedproxy-manager
      info "manage.sh downloaded"
    else
      warn "manage.sh could not be installed"
    fi
  fi

  # rollback.sh
  if [[ -f "$BUILD_DIR/rollback.sh" ]]; then
    cp "$BUILD_DIR/rollback.sh" "$INSTALL_DIR/rollback.sh"
    chmod +x "$INSTALL_DIR/rollback.sh"
    chown root:root "$INSTALL_DIR/rollback.sh"
    info "rollback.sh installed"
  fi
}

seed_database() {
  step "Initializing database"
  if [[ -f "$INSTALL_DIR/data/zedproxy.db" ]]; then
    warn "Existing database found — skipping seed (data preserved)"
    return
  fi
  export PATH="/usr/local/go/bin:/usr/local/bin:$PATH"
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

  sqlite3 "$INSTALL_DIR/data/zedproxy.db" \
    "INSERT INTO settings (key, value) VALUES ('site_url', 'https://$DOMAIN') ON CONFLICT(key) DO UPDATE SET value='https://$DOMAIN';"

  info "Database initialized"
}

create_env() {
  step "Creating environment config"
  if [[ -f "$INSTALL_DIR/.env" ]]; then
    warn "Existing .env found — not overwriting (SESSION_SECRET preserved)"
    return
  fi
  cat > "$INSTALL_DIR/.env" <<ENVEOF
SESSION_SECRET=${SESSION_SECRET}
GIN_MODE=release
ENVEOF
  chmod 600 "$INSTALL_DIR/.env"
  info ".env file created"
}

setup_systemd() {
  step "Configuring systemd service"
  cat > "/etc/systemd/system/${SERVICE_NAME}.service" <<SVCEOF
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
SVCEOF

  systemctl daemon-reload
  systemctl enable "$SERVICE_NAME"
  info "systemd service configured"
}

setup_nginx() {
  step "Configuring Nginx"
  mkdir -p /var/www/letsencrypt/.well-known/acme-challenge
  cat > "/etc/nginx/sites-available/zedproxy" <<NGXEOF
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
NGXEOF

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
    warn "SSL setup failed. Retry later with:"
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
  # Scripts stay root-owned
  for f in update.sh manage.sh rollback.sh; do
    if [[ -f "$INSTALL_DIR/$f" ]]; then
      chown root:root "$INSTALL_DIR/$f"
      chmod 755 "$INSTALL_DIR/$f"
    fi
  done
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
  step "Optional: Telegram Admin Reporter Bot"
  echo ""
  echo -e "${CYAN}Configure Telegram admin reporter bot now? [y/N]:${NC}"
  read -rp "Choice: " TG_CHOICE
  if [[ "${TG_CHOICE,,}" != "y" ]]; then
    info "Telegram bot skipped."
    echo "  Configure later at: /zed-admin/integrations/telegram"
    echo "  Or via:             sudo zedproxy-manager"
    return
  fi

  echo ""
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

  "$INSTALL_DIR/zedproxy" --db="$INSTALL_DIR/data/zedproxy.db" --telegram-set-token="$TG_TOKEN" && \
    info "Bot token saved" || warn "Failed to save bot token"
  "$INSTALL_DIR/zedproxy" --db="$INSTALL_DIR/data/zedproxy.db" --telegram-set-chat-id="$TG_CHAT" && \
    info "Chat ID saved" || warn "Failed to save chat ID"
  "$INSTALL_DIR/zedproxy" --db="$INSTALL_DIR/data/zedproxy.db" --telegram-enable && \
    info "Telegram bot enabled" || warn "Failed to enable bot"

  echo ""
  echo -e "${CYAN}Create forum topics in your Telegram group now? [y/N]:${NC}"
  read -rp "Choice: " TOPICS_CHOICE
  if [[ "${TOPICS_CHOICE,,}" == "y" ]]; then
    "$INSTALL_DIR/zedproxy" --db="$INSTALL_DIR/data/zedproxy.db" --telegram-create-topics && \
      info "Forum topics created" || warn "Topic creation had errors — try later from admin panel"
  fi

  "$INSTALL_DIR/zedproxy" --db="$INSTALL_DIR/data/zedproxy.db" \
    --telegram-notify-title="ZedProxy Installed" \
    --telegram-notify-msg="Installation completed on domain: $DOMAIN" \
    --telegram-notify-cat="system_status" 2>/dev/null || true

  TG_CONFIGURED="yes"
  info "Telegram admin reporter configured"
}

validate_installation() {
  step "Validating installation"
  local failed=0

  for f in "$INSTALL_DIR/zedproxy" "$INSTALL_DIR/data/zedproxy.db" \
            "$INSTALL_DIR/update.sh" "$INSTALL_DIR/manage.sh"; do
    if [[ -f "$f" ]]; then
      info "OK: $f"
    else
      warn "Missing: $f"
      failed=1
    fi
  done

  if [[ -L /usr/local/bin/zedproxy-manager ]]; then
    info "OK: /usr/local/bin/zedproxy-manager"
  else
    warn "Missing symlink: /usr/local/bin/zedproxy-manager"
  fi

  if [[ $failed -eq 1 ]]; then
    warn "Some files are missing but installation may still work."
  fi
}

print_result() {
  local SITE_URL="https://$DOMAIN"
  local ADMIN_URL="$SITE_URL/zed-admin"

  echo ""
  echo -e "${GREEN}╔════════════════════════════════════════════════════╗${NC}"
  echo -e "${GREEN}║      Installation completed successfully!           ║${NC}"
  echo -e "${GREEN}╚════════════════════════════════════════════════════╝${NC}"
  echo ""
  echo -e "${WHITE}Website:${NC}       ${CYAN}${SITE_URL}${NC}"
  echo -e "${WHITE}Admin panel:${NC}   ${CYAN}${ADMIN_URL}${NC}"
  echo -e "${WHITE}Admin username:${NC} ${WHITE}${ADMIN_USERNAME}${NC}"
  echo -e "${WHITE}Admin email:${NC}   ${WHITE}${ADMIN_EMAIL}${NC}"
  echo -e "${WHITE}Admin password:${NC} ${RED}${ADMIN_PASSWORD}${NC}"
  echo -e "${WHITE}Telegram reporter:${NC} ${TG_CONFIGURED}"
  echo ""
  echo -e "${YELLOW}WARNING: Save these credentials in a secure place!${NC}"
  echo -e "${YELLOW}This is the only time the password is shown.${NC}"
  echo ""
  echo -e "${WHITE}Useful commands:${NC}"
  echo "  sudo systemctl status zedproxy         # Service status"
  echo "  sudo systemctl restart zedproxy        # Restart service"
  echo "  sudo journalctl -u zedproxy -f         # Live logs"
  echo "  sudo bash /opt/zedproxy/update.sh      # Update website"
  echo "  sudo zedproxy-manager                  # Server management menu"
  echo ""
  echo -e "${WHITE}Recovery — if update.sh is missing:${NC}"
  echo "  sudo curl -fsSL https://raw.githubusercontent.com/mhoseinshah1/zed_website/main/update.sh \\"
  echo "    -o /opt/zedproxy/update.sh && sudo chmod +x /opt/zedproxy/update.sh"
  echo ""
}

# =====================================================
# Main
# =====================================================
banner
check_root
get_domain
get_admin_credentials
generate_session_secret
install_packages
install_go
setup_directory
clone_or_copy
build_app
copy_assets
install_scripts
create_env
seed_database
set_permissions
setup_systemd
setup_nginx
setup_firewall
start_service
setup_ssl
setup_telegram_optional
validate_installation
print_result
