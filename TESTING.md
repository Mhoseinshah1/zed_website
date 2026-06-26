# ZedProxy — Testing Guide

## Script Syntax Validation

```bash
bash -n install.sh    # Must pass with no output
bash -n update.sh     # Must pass with no output
bash -n manage.sh     # Must pass with no output
bash -n rollback.sh   # Must pass with no output
```

## No Persian in Shell Scripts

```bash
grep -RInP '[\x{0600}-\x{06FF}]' --include='*.sh' .
# Must return NO output
```

## Go Build

```bash
go mod tidy
go build ./...
# Must complete with no errors
```

---

## Installation Tests

### Test 1: Manual admin credentials

```bash
sudo bash install.sh
# At prompts:
#   Domain: example.com
#   Admin username: myadmin
#   Admin email: me@example.com
#   Admin password: MySecure123!
#   Telegram: N
```

Expected: Final output shows `myadmin` / `me@example.com` / `MySecure123!`

### Test 2: Auto-generated credentials

```bash
sudo bash install.sh
# At prompts:
#   Domain: example.com
#   Admin username: [press Enter]
#   Admin email: [press Enter]
#   Admin password: [press Enter]
#   Telegram: N
```

Expected: Final output shows auto-generated username like `admin_a3f2b1`, default email `admin@zedproxy.com`, and a generated password.

### Test 3: Invalid then fallback

```bash
sudo bash install.sh
# Admin username: ab     <- too short, triggers auto-generation
# Admin password: 123    <- too short, triggers auto-generation
```

Expected: Warning shown, auto-generation used, credentials printed at end.

### Test 4: Verify credentials work

After install, log in at `https://yourdomain.com/zed-admin` with the printed credentials.

---

## Server Manager Tests

### Test: Script exists after install

```bash
ls -la /opt/zedproxy/manage.sh       # Must exist
ls -la /usr/local/bin/zedproxy-manager  # Must be a symlink
which zedproxy-manager               # Must return a path
```

### Test: Syntax

```bash
bash -n /opt/zedproxy/manage.sh
```

### Test: Run the manager

```bash
sudo zedproxy-manager
# Menu must appear with 44 options
```

### Test: Maintenance off (option 10)

```bash
sudo zedproxy-manager
# Select 10
# Confirm
# Expected: Maintenance disabled + service restarted
```

### Test: Reset admin (option 6)

```bash
sudo zedproxy-manager
# Select 6
# Enter username to reset
# Press Enter for auto-generated password
# Expected: New password shown once
```

### Test: Create admin (option 7)

```bash
sudo zedproxy-manager
# Select 7
# Enter new username
# Enter email
# Enter password
# Expected: New owner admin created, credentials shown
```

### Test: Update from manager (option 11)

```bash
sudo zedproxy-manager
# Select 11
# Confirm
# Expected: update.sh runs
```

---

## CLI Tests

```bash
BIN=/opt/zedproxy/zedproxy
DB="--db=/opt/zedproxy/data/zedproxy.db"

# Version
$BIN --version

# Maintenance
$BIN $DB --maintenance-status
$BIN $DB --maintenance-on
$BIN $DB --maintenance-status   # Should show ENABLED
$BIN $DB --maintenance-off
$BIN $DB --maintenance-status   # Should show disabled

# Self-test
$BIN $DB --self-test \
  --templates=/opt/zedproxy/templates \
  --static=/opt/zedproxy/static \
  --uploads=/opt/zedproxy/static/uploads
# Expected: === Self-test PASSED ===

# Reset admin password
$BIN $DB --reset-admin --admin-user="admin" --admin-pass="NewPass123!"
# Expected: [✓] Admin credentials reset for user: admin

# Create admin
$BIN $DB --create-admin \
  --admin-user="testadmin" \
  --admin-email="test@example.com" \
  --admin-pass="SecurePass123!" \
  --role="owner"
# Expected: [✓] Admin created: testadmin (role: owner)
```

---

## Health Check Test

```bash
curl -s http://localhost:8080/health | python3 -m json.tool
# Expected: {"status":"ok", "maintenance":false, ...}
```

---

## Telegram Tests

> **Note:** Live API tests require a real bot token and group chat ID. Skip if unavailable.

### Test: Telegram CLI (no token needed)

```bash
BIN=/opt/zedproxy/zedproxy
DB="--db=/opt/zedproxy/data/zedproxy.db"

$BIN $DB --telegram-status
# Shows: enabled/disabled state, chat ID, bot username

$BIN $DB --telegram-enable
$BIN $DB --telegram-status   # Should show enabled: 1

$BIN $DB --telegram-disable
$BIN $DB --telegram-status   # Should show enabled: 0
```

### Test: Configure Telegram (requires real token)

```bash
$BIN $DB --telegram-set-token="YOUR_REAL_TOKEN"
$BIN $DB --telegram-set-chat-id="-1001234567890"
$BIN $DB --telegram-enable

$BIN $DB --telegram-test
# Expected: [✓] Bot: @yourbotname | Chat: Your Group (supergroup)

$BIN $DB --telegram-send-test
# Expected: [✓] Test message sent
# Check: Test message appears in Telegram group
```

### Test: Create Persian forum topics

```bash
$BIN $DB --telegram-create-topics
# Expected: [✓] Forum topics created
# Check: 11 topics appear in Telegram group with Persian names
```

### Test: Daily report

```bash
$BIN $DB --send-daily-report
# Expected: [✓] Daily report sent
# Check: Report appears in daily_reports topic
```

### Test: Configure from admin panel

1. Log in to `/zed-admin/integrations/telegram`
2. Enter bot token (displayed masked after save)
3. Enter Chat ID
4. Enable bot
5. Click "Test Connection" button
6. Click "Send Test Message" button
7. Click "Create Group Topics" button
8. Verify topic list shows with thread IDs

### Test: Configure from manager

```bash
sudo zedproxy-manager
# Option 34: Telegram Bot Status
# Option 35: Set bot token (hidden input)
# Option 36: Set Chat ID
# Option 37: Test connection
# Option 38: Send test message
# Option 39: Create topics
# Option 40: Enable
# Option 42: Send daily report
```

---

## Security Verification

```bash
# No secrets in logs
journalctl -u zedproxy | grep -i token   # Should be empty
journalctl -u zedproxy | grep -i password  # Should be empty

# No Persian in shell scripts
grep -RInP '[\x{0600}-\x{06FF}]' --include='*.sh' /opt/zedproxy/
# Must return no output

# .env permissions
ls -la /opt/zedproxy/.env
# Must be: -rw------- (600), owned by root

# Token not in DB as plaintext (it IS stored, but only shown masked in UI)
# Verify admin panel shows: 123456:ABC...xyz (masked)
```

---

## File Existence After Install

```bash
ls -la /opt/zedproxy/zedproxy          # binary
ls -la /opt/zedproxy/data/zedproxy.db  # database
ls -la /opt/zedproxy/update.sh         # update script
ls -la /opt/zedproxy/manage.sh         # manager
ls -la /opt/zedproxy/rollback.sh       # rollback
ls -la /opt/zedproxy/.env              # env (600 permissions)
ls -la /usr/local/bin/zedproxy-manager # symlink
```

---

## Admin Panel Smoke Tests

| URL | Expected |
|---|---|
| `/zed-admin/login` | Login page loads |
| `/zed-admin` | Dashboard (after login) |
| `/zed-admin/integrations/telegram` | Telegram config page |
| `/zed-admin/system/health` | Health info |
| `/zed-admin/system/logs` | System log table |
| `/zed-admin/maintenance` | Maintenance toggle |
| `/zed-admin/backups` | Backup list |
| `/zed-admin/users` | Admin user list |
| `/health` | JSON health response |
| `/sitemap.xml` | Valid XML sitemap |
| `/robots.txt` | Robots file |
