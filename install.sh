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

  _VER="$(git -C "$BUILD_DIR" describe --tags --always 2>/dev/null || echo 'v1.0')"
  _COMMIT="$(git -C "$BUILD_DIR" rev-parse --short HEAD 2>/dev/null || echo 'unknown')"
  _DATE="$(date -u +%Y%m%dT%H%M%SZ)"
  CGO_ENABLED=1 go build \
    -ldflags="-s -w -X main.Version=${_VER} -X main.GitCommit=${_COMMIT} -X main.BuildDate=${_DATE}" \
    -o "$INSTALL_DIR/zedproxy" .

  if [[ ! -f "$INSTALL_DIR/zedproxy" ]]; then
    error "Build failed: binary not found at $INSTALL_DIR/zedproxy"
  fi
  info "Application built successfully"
  ls -lh "$INSTALL_DIR/zedproxy"
}

copy_assets() {
  step "Copying templates and static files"

  if [[ ! -d "$BUILD_DIR/templates" ]]; then
    error "templates directory not found in source: $BUILD_DIR/templates"
  fi
  local tmpl_src_count
  tmpl_src_count=$(find "$BUILD_DIR/templates" -name "*.html" | wc -l)
  if [[ "$tmpl_src_count" -eq 0 ]]; then
    error "No HTML templates found in $BUILD_DIR/templates -- cannot install"
  fi

  cp -r "$BUILD_DIR/templates/"* "$INSTALL_DIR/templates/"

  local tmpl_dst_count
  tmpl_dst_count=$(find "$INSTALL_DIR/templates" -name "*.html" | wc -l)
  if [[ "$tmpl_dst_count" -eq 0 ]]; then
    error "Template copy failed -- $INSTALL_DIR/templates contains no HTML files"
  fi
  info "Templates copied ($tmpl_dst_count HTML files)"

  rsync -a --exclude 'uploads/' "$BUILD_DIR/static/" "$INSTALL_DIR/static/" 2>/dev/null || \
    cp -r "$BUILD_DIR/static/"* "$INSTALL_DIR/static/" 2>/dev/null || true
  info "Static files copied"
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

    cat > /usr/local/bin/zedproxy-doctor << 'WRAPPER'
#!/bin/bash
exec /opt/zedproxy/zedproxy --doctor "$@"
WRAPPER
    chmod +x /usr/local/bin/zedproxy-doctor

    cat > /usr/local/bin/zedproxy-repair << 'WRAPPER'
#!/bin/bash
exec /opt/zedproxy/zedproxy --repair "$@"
WRAPPER
    chmod +x /usr/local/bin/zedproxy-repair
    info "Installed: zedproxy-doctor, zedproxy-repair"
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

check_port() {
  step "Checking port $APP_PORT availability"
  if ss -tlnp 2>/dev/null | grep -q ":${APP_PORT} "; then
    local pid proc
    pid="$(ss -tlnp 2>/dev/null | grep ":${APP_PORT} " | grep -oP 'pid=\K[0-9]+' | head -1 || true)"
    if [[ -n "$pid" ]]; then
      proc="$(cat /proc/$pid/comm 2>/dev/null || echo unknown)"
      if [[ "$proc" == "$SERVICE_NAME" ]] || [[ "$proc" == "zedproxy" ]]; then
        warn "Stale $SERVICE_NAME process on port $APP_PORT (pid $pid) -- stopping..."
        systemctl stop "$SERVICE_NAME" 2>/dev/null || kill "$pid" 2>/dev/null || true
        sleep 2
        info "Stale process stopped"
      else
        error "Port $APP_PORT is already in use by '$proc' (pid $pid). Free this port before installing ZedProxy."
      fi
    fi
  fi
  info "Port $APP_PORT is available"
}

run_install_self_test() {
  step "Running binary self-test"
  if ! "$INSTALL_DIR/zedproxy" \
      --db="$INSTALL_DIR/data/zedproxy.db" \
      --templates="$INSTALL_DIR/templates" \
      --static="$INSTALL_DIR/static" \
      --uploads="$INSTALL_DIR/static/uploads" \
      --self-test 2>&1; then
    error "Self-test FAILED -- check the output above for missing files, DB issues, or template parse errors"
  fi
  info "Self-test passed"
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
  systemctl daemon-reload
  systemctl enable "$SERVICE_NAME" 2>/dev/null || true
  systemctl restart "$SERVICE_NAME"
  sleep 3

  if ! systemctl is-active --quiet "$SERVICE_NAME"; then
    warn "Service failed to start. Recent logs:"
    journalctl -u "$SERVICE_NAME" -n 30 --no-pager
    warn "Port status:"
    ss -tlnp | grep ":${APP_PORT}" || warn "Nothing on port $APP_PORT"
    error "Failed to start ZedProxy service. See logs above."
  fi
  info "ZedProxy service is active"

  # Verify HTTP responds
  local attempts=6 passed=false
  for i in $(seq 1 $attempts); do
    local code
    code="$(curl -s -o /dev/null -w '%{http_code}' --connect-timeout 5 --max-time 10 \
      "http://127.0.0.1:${APP_PORT}/health" 2>/dev/null || echo '000')"
    if [[ "$code" == "200" ]]; then
      info "Health check passed (HTTP $code)"
      passed=true
      break
    fi
    warn "Attempt $i/$attempts -- /health returned HTTP $code -- waiting 3s..."
    sleep 3
  done

  if [[ "$passed" != "true" ]]; then
    warn "Last 30 service log lines:"
    journalctl -u "$SERVICE_NAME" -n 30 --no-pager
    warn "Port status:"
    ss -tlnp | grep ":${APP_PORT}" || warn "Nothing on port $APP_PORT"
    error "Service started but /health did not respond with HTTP 200. Check logs above."
  fi
}

info_telegram_setup() {
  step "Telegram Admin Reporter"
  echo ""
  echo -e "  Telegram admin reporter can be configured later from:"
  echo -e "  Admin panel: ${CYAN}https://${DOMAIN}/zed-admin/integrations/telegram${NC}"
  echo ""
  info "Telegram configuration is available in the admin panel."
}

final_verify() {
  step "Final installation verification"
  local failed=0

  # Required files
  for f in "$INSTALL_DIR/zedproxy" "$INSTALL_DIR/data/zedproxy.db" \
            "$INSTALL_DIR/.env" "$INSTALL_DIR/update.sh" "$INSTALL_DIR/manage.sh"; do
    if [[ -f "$f" ]]; then
      info "OK: $f"
    else
      warn "MISSING: $f"
      failed=1
    fi
  done

  # Required directories
  for d in "$INSTALL_DIR/templates" "$INSTALL_DIR/static" \
            "$INSTALL_DIR/static/uploads" "$INSTALL_DIR/data" \
            "$INSTALL_DIR/data/backups" "$INSTALL_DIR/logs"; do
    if [[ -d "$d" ]]; then
      info "OK dir: $d"
    else
      warn "MISSING dir: $d"
      failed=1
    fi
  done

  # Symlink
  if [[ -L /usr/local/bin/zedproxy-manager ]]; then
    info "OK: /usr/local/bin/zedproxy-manager"
  else
    warn "Missing symlink: /usr/local/bin/zedproxy-manager"
  fi

  # Template count
  local tmpl_count
  tmpl_count=$(find "$INSTALL_DIR/templates" -name "*.html" 2>/dev/null | wc -l)
  if [[ "$tmpl_count" -gt 0 ]]; then
    info "OK: templates ($tmpl_count HTML files)"
  else
    warn "FAIL: no HTML templates in $INSTALL_DIR/templates"
    failed=1
  fi

  # Full service restart + verify
  systemctl daemon-reload
  systemctl enable "$SERVICE_NAME" 2>/dev/null || true
  systemctl restart "$SERVICE_NAME"
  sleep 3

  if systemctl is-active --quiet "$SERVICE_NAME"; then
    info "OK: $SERVICE_NAME service is active"
  else
    warn "FAIL: $SERVICE_NAME is not active"
    journalctl -u "$SERVICE_NAME" -n 20 --no-pager
    ss -tlnp | grep ":${APP_PORT}" || true
    failed=1
  fi

  # /health
  local code
  code="$(curl -s -o /dev/null -w '%{http_code}' --connect-timeout 5 --max-time 10 \
    "http://127.0.0.1:${APP_PORT}/health" 2>/dev/null || echo '000')"
  if [[ "$code" == "200" ]]; then
    info "OK: /health HTTP $code"
  else
    warn "FAIL: /health returned HTTP $code"
    journalctl -u "$SERVICE_NAME" -n 10 --no-pager
    failed=1
  fi

  # Homepage
  local home_code
  home_code="$(curl -sI -o /dev/null -w '%{http_code}' --connect-timeout 5 --max-time 10 \
    "http://127.0.0.1:${APP_PORT}/" 2>/dev/null || echo '000')"
  if [[ "$home_code" =~ ^(200|301|302)$ ]]; then
    info "OK: / HTTP $home_code"
  else
    warn "FAIL: / returned HTTP $home_code"
    failed=1
  fi

  # Nginx
  if nginx -t 2>/dev/null; then
    info "OK: nginx config valid"
  else
    warn "FAIL: nginx config has errors:"
    nginx -t 2>&1
    failed=1
  fi
  systemctl reload nginx 2>/dev/null || true

  # Version metadata
  local ver_out
  ver_out="$("$INSTALL_DIR/zedproxy" --db="$INSTALL_DIR/data/zedproxy.db" --version 2>&1 || true)"
  info "Binary: $ver_out"
  if echo "$ver_out" | grep -qw "dev"; then
    warn "Version still shows 'dev' -- ldflags may not have applied correctly"
  fi

  if [[ $failed -eq 1 ]]; then
    warn "Some verification checks failed. Review warnings above."
  else
    info "All verification checks passed."
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
  echo -e "${WHITE}Telegram reporter:${NC} Configure at ${CYAN}${ADMIN_URL}/integrations/telegram${NC}"
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
run_install_self_test
setup_systemd
check_port
setup_nginx
setup_firewall
start_service
setup_ssl
info_telegram_setup
final_verify
print_result
