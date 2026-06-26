# ZedProxy — Premium Persian RTL VPN Website

A complete Persian RTL VPN/proxy service website built with Go, SQLite, Tailwind CSS, and Alpine.js. Includes a full admin panel, Telegram Admin Reporter Bot, and production Ubuntu deployment scripts.

---

## Quick Install

```bash
bash <(curl -fsSL https://raw.githubusercontent.com/mhoseinshah1/zed_website/main/install.sh)
```

Or on the server:

```bash
sudo bash install.sh
```

### Installer Prompts

The installer asks the following in order:

1. **Domain name** (required)
2. **Admin username** — Enter a username or press Enter to auto-generate (`admin_xxxxxx` format)
3. **Admin email** — Enter email or press Enter to use `admin@zedproxy.com`
4. **Admin password** — Enter a password (min 8 chars) or press Enter to auto-generate securely
5. **Telegram Admin Reporter Bot** — Optional setup (can be done later)

All credentials are printed at the end. The password is shown only once — save it securely.

---

## Server Manager

After installation:

```bash
sudo zedproxy-manager
```

Or:

```bash
sudo bash /opt/zedproxy/manage.sh
```

### Manager Menu (44 options)

| # | Action |
|---|--------|
| 1 | Show System Status |
| 2 | Emergency Recovery |
| 3 | Restart Service |
| 4 | Start Service |
| 5 | Stop Service |
| 6 | Reset Admin Username / Password |
| 7 | Create New Owner Admin |
| 8 | Maintenance Mode Status |
| 9 | Maintenance Mode ON |
| 10 | Maintenance Mode OFF / Emergency Disable |
| 11 | Run Website Update |
| 12 | Repair Missing update.sh |
| 13 | Rollback to Previous Release |
| 14 | Create Backup |
| 15 | List Backups |
| 16 | Restore Backup |
| 17 | Run Self-Test |
| 18 | Show Health Check |
| 19 | Check SQLite Database |
| 20 | Repair SQLite Database |
| 21 | Vacuum SQLite Database |
| 22 | View Recent Logs |
| 23 | Follow Live Logs |
| 24 | Export Diagnostic Report |
| 25 | Fix File Permissions |
| 26 | Show App Version |
| 27 | Show Domain and URLs |
| 28 | Change Domain |
| 29 | Check Nginx Config |
| 30 | Reload Nginx |
| 31 | Renew SSL Certificate |
| 32 | Show Disk Usage |
| 33 | Clean Temporary Files |
| 34 | Telegram Bot Status |
| 35 | Configure Bot Token |
| 36 | Configure Group Chat ID |
| 37 | Test Bot Connection |
| 38 | Send Test Message |
| 39 | Create Group Topics |
| 40 | Enable Telegram Alerts |
| 41 | Disable Telegram Alerts |
| 42 | Send Daily Report Now |
| 43 | Uninstall ZedProxy |
| 44 | Exit |

---

## Update

```bash
sudo bash /opt/zedproxy/update.sh
```

### If update.sh is missing

```bash
sudo curl -fsSL https://raw.githubusercontent.com/mhoseinshah1/zed_website/main/update.sh \
  -o /opt/zedproxy/update.sh && sudo chmod +x /opt/zedproxy/update.sh
sudo bash /opt/zedproxy/update.sh
```

---

## Rollback

```bash
sudo bash /opt/zedproxy/rollback.sh
# or from manager: option 13
```

---

## CLI Reference

```bash
BIN=/opt/zedproxy/zedproxy
DB="--db=/opt/zedproxy/data/zedproxy.db"

# Version
$BIN --version

# Maintenance
$BIN $DB --maintenance-on
$BIN $DB --maintenance-off
$BIN $DB --maintenance-status

# Self-test
$BIN $DB --self-test --templates=/opt/zedproxy/templates \
  --static=/opt/zedproxy/static --uploads=/opt/zedproxy/static/uploads

# Admin management
$BIN $DB --reset-admin --admin-user="admin" --admin-pass="newpassword"
$BIN $DB --create-admin --admin-user="newadmin" --admin-email="new@example.com" \
  --admin-pass="securepass" --role="owner"

# Telegram
$BIN $DB --telegram-status
$BIN $DB --telegram-set-token="YOUR_BOT_TOKEN"
$BIN $DB --telegram-set-chat-id="-1001234567890"
$BIN $DB --telegram-enable
$BIN $DB --telegram-disable
$BIN $DB --telegram-test
$BIN $DB --telegram-send-test
$BIN $DB --telegram-create-topics
$BIN $DB --send-daily-report
```

---

## Telegram Admin Reporter Bot

This is an **internal admin reporting bot** — separate from the customer purchase bot. It sends alerts to a private Telegram supergroup.

### Setup Steps

1. Create a bot with [@BotFather](https://t.me/BotFather) — save the token
2. Create a Telegram supergroup
3. Enable **Topics** in group settings
4. Add the bot to the group as admin with: Manage Topics + Send Messages permissions
5. Get the group Chat ID (use @userinfobot or check bot API)

### Configuration

**From admin panel:**
```
https://yourdomain.com/zed-admin/integrations/telegram
```

**From server manager** (options 34-42):
```bash
sudo zedproxy-manager
```

### Telegram Forum Topics (Persian titles, English internal keys)

| Internal Key | Persian Title |
|---|---|
| system_status | 📌 وضعیت سیستم |
| critical_alerts | 🚨 هشدارهای مهم |
| updates | 🔄 بروزرسانی‌ها |
| maintenance | 🧰 حالت تعمیرات |
| backups | 💾 بکاپ‌ها |
| security | 🔐 امنیت |
| daily_reports | 📊 گزارش روزانه |
| click_analytics | 📈 آمار کلیک‌ها |
| seo_pages | 🌐 سئو و صفحات |
| admin_activity | 🧾 فعالیت ادمین |
| errors | ❌ خطاها |

### Events That Trigger Notifications

- Maintenance mode on/off
- Backup created
- Admin login failed
- Admin password changed
- New admin created
- Update started/completed/failed
- Rollback performed

### Daily Report

Auto-sent at 09:00 Asia/Tehran (configurable). Includes maintenance status, CTA clicks, plan/post counts.

```bash
# Send manually
/opt/zedproxy/zedproxy --db=/opt/zedproxy/data/zedproxy.db --send-daily-report
# or: sudo zedproxy-manager (option 42)
```

---

## Admin Panel Routes

| Route | Description |
|---|---|
| `/zed-admin` | Dashboard |
| `/zed-admin/settings` | Site settings |
| `/zed-admin/integrations/telegram` | Telegram Admin Reporter |
| `/zed-admin/system/health` | System health |
| `/zed-admin/system/logs` | Audit logs |
| `/zed-admin/maintenance` | Maintenance mode |
| `/zed-admin/backups` | Database backups |
| `/zed-admin/users` | Admin user management |
| `/zed-admin/plans` | VPN plans |
| `/zed-admin/analytics` | Click analytics |

---

## Tech Stack

- **Go 1.22** + Gin web framework
- **SQLite** (CGO_ENABLED=1, WAL mode)
- **Tailwind CSS** + **Alpine.js** via CDN
- **Vazirmatn** Persian font, RTL layout throughout
- **systemd** service (www-data user, PrivateTmp, NoNewPrivileges)
- **Nginx** reverse proxy with ACME challenge support
- **Let's Encrypt** SSL via Certbot
- **Telegram Bot API** for admin reporting (queue/retry, token masking)

---

## Build with Version Info

```bash
CGO_ENABLED=1 go build \
  -ldflags="-s -w -X main.Version=1.0.0 -X main.BuildDate=$(date -u +%Y-%m-%d) -X main.GitCommit=$(git rev-parse --short HEAD)" \
  -o zedproxy .
```

---

## Service Commands

```bash
sudo systemctl status zedproxy
sudo systemctl restart zedproxy
sudo journalctl -u zedproxy -f
```
