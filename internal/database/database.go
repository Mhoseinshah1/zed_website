package database

import (
	"database/sql"
	"log"
	"os"
	"path/filepath"
	"strings"

	_ "github.com/mattn/go-sqlite3"
)

var DB *sql.DB

func Init(dbPath string) {
	dir := filepath.Dir(dbPath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		log.Fatalf("failed to create db directory: %v", err)
	}

	var err error
	DB, err = sql.Open("sqlite3", dbPath+"?_journal_mode=WAL&_foreign_keys=on")
	if err != nil {
		log.Fatalf("failed to open database: %v", err)
	}
	DB.SetMaxOpenConns(1)
	if err = DB.Ping(); err != nil {
		log.Fatalf("failed to ping database: %v", err)
	}
	Migrate()
}

func safeExec(query string) {
	if _, err := DB.Exec(query); err != nil {
		// Ignore "duplicate column name" and "already exists" errors
		msg := err.Error()
		if strings.Contains(msg, "duplicate column name") ||
			strings.Contains(msg, "already exists") {
			return
		}
		log.Fatalf("migration failed: %v\nQuery: %s", err, query)
	}
}

func Migrate() {
	// Core tables (idempotent)
	queries := []string{
		`CREATE TABLE IF NOT EXISTS admins (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			username TEXT NOT NULL UNIQUE,
			email TEXT NOT NULL UNIQUE,
			password_hash TEXT NOT NULL,
			role TEXT NOT NULL DEFAULT 'owner',
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP
		)`,
		`CREATE TABLE IF NOT EXISTS settings (
			key TEXT PRIMARY KEY,
			value TEXT NOT NULL DEFAULT '',
			updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
		)`,
		`CREATE TABLE IF NOT EXISTS plans (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			name TEXT NOT NULL,
			traffic TEXT NOT NULL,
			duration TEXT NOT NULL,
			price TEXT NOT NULL,
			badge TEXT DEFAULT '',
			description TEXT DEFAULT '',
			features TEXT DEFAULT '',
			is_popular INTEGER DEFAULT 0,
			sort_order INTEGER DEFAULT 0,
			is_active INTEGER DEFAULT 1,
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP
		)`,
		`CREATE TABLE IF NOT EXISTS features (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			icon TEXT NOT NULL,
			title TEXT NOT NULL,
			description TEXT NOT NULL,
			sort_order INTEGER DEFAULT 0,
			is_active INTEGER DEFAULT 1
		)`,
		`CREATE TABLE IF NOT EXISTS faqs (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			question TEXT NOT NULL,
			answer TEXT NOT NULL,
			category TEXT DEFAULT 'general',
			sort_order INTEGER DEFAULT 0,
			is_active INTEGER DEFAULT 1,
			show_on_homepage INTEGER DEFAULT 0,
			show_on_faq INTEGER DEFAULT 1
		)`,
		`CREATE TABLE IF NOT EXISTS blog_posts (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			slug TEXT NOT NULL UNIQUE,
			title TEXT NOT NULL,
			meta_title TEXT DEFAULT '',
			meta_description TEXT DEFAULT '',
			excerpt TEXT DEFAULT '',
			content TEXT NOT NULL,
			image TEXT DEFAULT '',
			category TEXT DEFAULT '',
			is_published INTEGER DEFAULT 0,
			published_at DATETIME,
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
		)`,
		`CREATE TABLE IF NOT EXISTS tutorials (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			slug TEXT NOT NULL UNIQUE,
			title TEXT NOT NULL,
			excerpt TEXT DEFAULT '',
			content TEXT NOT NULL,
			image TEXT DEFAULT '',
			category TEXT DEFAULT '',
			platform TEXT DEFAULT '',
			video_url TEXT DEFAULT '',
			meta_title TEXT DEFAULT '',
			meta_description TEXT DEFAULT '',
			sort_order INTEGER DEFAULT 0,
			is_published INTEGER DEFAULT 1,
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
		)`,
		`CREATE TABLE IF NOT EXISTS locations (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			name TEXT NOT NULL,
			flag TEXT DEFAULT '',
			code TEXT DEFAULT '',
			speed TEXT DEFAULT '',
			is_active INTEGER DEFAULT 1,
			sort_order INTEGER DEFAULT 0
		)`,
		`CREATE TABLE IF NOT EXISTS status_updates (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			title TEXT NOT NULL,
			content TEXT NOT NULL,
			status TEXT DEFAULT 'info',
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP
		)`,
		`CREATE TABLE IF NOT EXISTS pages (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			slug TEXT NOT NULL UNIQUE,
			title TEXT NOT NULL,
			content TEXT NOT NULL,
			meta_title TEXT DEFAULT '',
			meta_description TEXT DEFAULT '',
			updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
		)`,
		`CREATE TABLE IF NOT EXISTS click_events (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			page TEXT NOT NULL,
			source TEXT DEFAULT '',
			ip TEXT DEFAULT '',
			user_agent TEXT DEFAULT '',
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP
		)`,
		`CREATE TABLE IF NOT EXISTS uploaded_files (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			filename TEXT NOT NULL,
			original_name TEXT NOT NULL,
			mime_type TEXT DEFAULT '',
			size INTEGER DEFAULT 0,
			path TEXT NOT NULL,
			alt_text TEXT DEFAULT '',
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP
		)`,

		// New tables for 20 advanced features
		`CREATE TABLE IF NOT EXISTS announcements (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			message TEXT NOT NULL,
			color TEXT DEFAULT 'blue',
			is_closable INTEGER DEFAULT 1,
			is_active INTEGER DEFAULT 1,
			target_pages TEXT DEFAULT 'all',
			start_at DATETIME,
			end_at DATETIME,
			sort_order INTEGER DEFAULT 0,
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP
		)`,
		`CREATE TABLE IF NOT EXISTS discount_codes (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			code TEXT NOT NULL,
			description TEXT DEFAULT '',
			discount_percent INTEGER DEFAULT 0,
			is_active INTEGER DEFAULT 1,
			expires_at DATETIME,
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP
		)`,
		`CREATE TABLE IF NOT EXISTS telegram_clicks (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			page TEXT NOT NULL,
			source TEXT DEFAULT '',
			plan_id INTEGER DEFAULT 0,
			campaign TEXT DEFAULT '',
			device_type TEXT DEFAULT '',
			referrer TEXT DEFAULT '',
			utm_source TEXT DEFAULT '',
			utm_medium TEXT DEFAULT '',
			utm_campaign TEXT DEFAULT '',
			ip_hash TEXT DEFAULT '',
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP
		)`,
		`CREATE TABLE IF NOT EXISTS status_items (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			name TEXT NOT NULL,
			service_type TEXT DEFAULT 'general',
			status TEXT DEFAULT 'operational',
			description TEXT DEFAULT '',
			sort_order INTEGER DEFAULT 0,
			is_active INTEGER DEFAULT 1
		)`,
		`CREATE TABLE IF NOT EXISTS trust_cards (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			icon TEXT DEFAULT '',
			title TEXT NOT NULL,
			description TEXT NOT NULL,
			sort_order INTEGER DEFAULT 0,
			is_active INTEGER DEFAULT 1
		)`,
		`CREATE TABLE IF NOT EXISTS campaigns (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			slug TEXT NOT NULL UNIQUE,
			title TEXT NOT NULL,
			subtitle TEXT DEFAULT '',
			description TEXT DEFAULT '',
			discount_code TEXT DEFAULT '',
			discount_percent INTEGER DEFAULT 0,
			countdown_at DATETIME,
			cta_text TEXT DEFAULT 'خرید از ربات تلگرام',
			image TEXT DEFAULT '',
			meta_title TEXT DEFAULT '',
			meta_description TEXT DEFAULT '',
			is_active INTEGER DEFAULT 1,
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP
		)`,
		`CREATE TABLE IF NOT EXISTS homepage_sections (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			section_key TEXT NOT NULL UNIQUE,
			title TEXT DEFAULT '',
			subtitle TEXT DEFAULT '',
			is_active INTEGER DEFAULT 1,
			sort_order INTEGER DEFAULT 0,
			bg_style TEXT DEFAULT 'default'
		)`,
		`CREATE TABLE IF NOT EXISTS landing_pages (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			slug TEXT NOT NULL UNIQUE,
			title TEXT NOT NULL,
			hero_title TEXT DEFAULT '',
			hero_subtitle TEXT DEFAULT '',
			content TEXT DEFAULT '',
			cta_text TEXT DEFAULT 'خرید از ربات تلگرام',
			featured_image TEXT DEFAULT '',
			meta_title TEXT DEFAULT '',
			meta_description TEXT DEFAULT '',
			og_image TEXT DEFAULT '',
			noindex INTEGER DEFAULT 0,
			is_active INTEGER DEFAULT 1,
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
		)`,
		`CREATE TABLE IF NOT EXISTS popups (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			title TEXT NOT NULL,
			message TEXT DEFAULT '',
			cta_text TEXT DEFAULT 'خرید از ربات',
			show_after_seconds INTEGER DEFAULT 5,
			exit_intent INTEGER DEFAULT 0,
			once_per_session INTEGER DEFAULT 1,
			target_pages TEXT DEFAULT 'all',
			is_active INTEGER DEFAULT 0,
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP
		)`,
		`CREATE TABLE IF NOT EXISTS plan_comparisons (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			feature_name TEXT NOT NULL,
			bronze_value TEXT DEFAULT '',
			silver_value TEXT DEFAULT '',
			gold_value TEXT DEFAULT '',
			platinum_value TEXT DEFAULT '',
			sort_order INTEGER DEFAULT 0
		)`,
		`CREATE TABLE IF NOT EXISTS db_backups (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			filename TEXT NOT NULL,
			size INTEGER DEFAULT 0,
			backup_type TEXT DEFAULT 'database',
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP
		)`,
		`CREATE TABLE IF NOT EXISTS system_logs (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			admin_username TEXT DEFAULT '',
			action TEXT NOT NULL,
			details TEXT DEFAULT '',
			ip_hash TEXT DEFAULT '',
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP
		)`,
		`CREATE TABLE IF NOT EXISTS telegram_topics (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			key TEXT NOT NULL UNIQUE,
			title TEXT NOT NULL,
			message_thread_id INTEGER DEFAULT 0,
			enabled INTEGER DEFAULT 1,
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
		)`,
		`CREATE TABLE IF NOT EXISTS telegram_notifications (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			level TEXT DEFAULT '',
			category TEXT DEFAULT '',
			topic_key TEXT DEFAULT '',
			title TEXT DEFAULT '',
			message TEXT DEFAULT '',
			status TEXT DEFAULT 'sent',
			error TEXT DEFAULT '',
			sent_at DATETIME,
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP
		)`,
		`CREATE TABLE IF NOT EXISTS telegram_queue (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			level TEXT DEFAULT '',
			category TEXT DEFAULT '',
			topic_key TEXT DEFAULT '',
			payload TEXT NOT NULL,
			attempts INTEGER DEFAULT 0,
			last_error TEXT DEFAULT '',
			status TEXT DEFAULT 'pending',
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
		)`,
	}

	for _, q := range queries {
		safeExec(q)
	}

	// Safe ALTER TABLE migrations for existing tables
	alterQueries := []string{
		`ALTER TABLE admins ADD COLUMN role TEXT NOT NULL DEFAULT 'owner'`,
		`ALTER TABLE db_backups ADD COLUMN backup_type TEXT DEFAULT 'database'`,
		`ALTER TABLE faqs ADD COLUMN show_on_homepage INTEGER DEFAULT 0`,
		`ALTER TABLE faqs ADD COLUMN show_on_faq INTEGER DEFAULT 1`,
		`ALTER TABLE tutorials ADD COLUMN video_url TEXT DEFAULT ''`,
		`ALTER TABLE tutorials ADD COLUMN meta_title TEXT DEFAULT ''`,
		`ALTER TABLE tutorials ADD COLUMN meta_description TEXT DEFAULT ''`,
		`ALTER TABLE uploaded_files ADD COLUMN alt_text TEXT DEFAULT ''`,
	}
	for _, q := range alterQueries {
		safeExec(q)
	}

	// Seed homepage sections if empty
	var count int
	DB.QueryRow("SELECT COUNT(*) FROM homepage_sections").Scan(&count)
	if count == 0 {
		sections := []struct {
			key       string
			title     string
			sortOrder int
		}{
			{"hero", "بخش هیرو", 1},
			{"announcement", "اطلاعیه", 2},
			{"discount", "کد تخفیف", 3},
			{"features", "ویژگی‌ها", 4},
			{"plans", "پلن‌ها", 5},
			{"comparison", "مقایسه پلن‌ها", 6},
			{"tutorials", "آموزش‌ها", 7},
			{"faq", "سوالات متداول", 8},
			{"trust", "اعتمادسازی", 9},
			{"status", "وضعیت سرویس", 10},
			{"blog", "وبلاگ", 11},
			{"final_cta", "دعوت به اقدام نهایی", 12},
		}
		for _, s := range sections {
			active := 1
			if s.key == "comparison" || s.key == "discount" || s.key == "announcement" {
				active = 0
			}
			DB.Exec("INSERT INTO homepage_sections (section_key, title, sort_order, is_active) VALUES (?,?,?,?)",
				s.key, s.title, s.sortOrder, active)
		}
	}

	// Seed maintenance_enabled default (idempotent)
	DB.Exec("INSERT INTO settings (key, value) VALUES ('maintenance_enabled', '0') ON CONFLICT(key) DO NOTHING")

	// Seed Telegram admin bot settings defaults
	tgDefaults := []struct{ key, val string }{
		{"telegram_admin_bot_enabled", "0"},
		{"telegram_admin_bot_token", ""},
		{"telegram_admin_chat_id", ""},
		{"telegram_admin_group_title", ""},
		{"telegram_admin_bot_username", ""},
		{"telegram_admin_daily_report_enabled", "0"},
		{"telegram_admin_daily_report_time", "09:00"},
		{"telegram_admin_daily_report_timezone", "Asia/Tehran"},
		{"telegram_admin_last_daily_report_date", ""},
		{"telegram_admin_alerts_enabled", "1"},
		{"telegram_admin_security_alerts_enabled", "1"},
		{"telegram_admin_update_alerts_enabled", "1"},
		{"telegram_admin_backup_alerts_enabled", "1"},
		{"telegram_admin_analytics_enabled", "0"},
		{"telegram_admin_error_alerts_enabled", "1"},
		{"telegram_admin_maintenance_alerts_enabled", "1"},
		{"telegram_admin_admin_activity_enabled", "1"},
		// Backup-to-Telegram settings
		{"telegram_admin_send_db_zip_enabled", "0"},
		{"telegram_admin_daily_db_backup_enabled", "0"},
		{"telegram_admin_daily_db_backup_time", "02:00"},
		{"telegram_admin_backup_before_update", "1"},
		{"telegram_admin_backup_before_rollback", "1"},
	}
	for _, s := range tgDefaults {
		DB.Exec("INSERT INTO settings (key, value) VALUES (?,?) ON CONFLICT(key) DO NOTHING", s.key, s.val)
	}

	// Seed admin appearance settings defaults
	appearanceDefaults := []struct{ key, val string }{
		{"admin_theme_name", "zed-dark-neon"},
		{"admin_accent_color", "#06b6d4"},
		{"admin_background_color", "#0d0d16"},
		{"admin_sidebar_color", "#0f0f1a"},
		{"admin_card_color", "#1a1a2e"},
		{"admin_text_color", "#f1f5f9"},
		{"admin_muted_text_color", "#94a3b8"},
		{"admin_border_color", "rgba(255,255,255,0.1)"},
		{"admin_button_color", "#06b6d4"},
		{"admin_hover_color", "rgba(255,255,255,0.07)"},
		{"admin_sidebar_mode", "full"},
		{"admin_sidebar_width", "normal"},
		{"admin_icon_size", "medium"},
		{"admin_menu_text_size", "medium"},
		{"admin_font_size", "normal"},
		{"admin_card_radius", "xl"},
		{"admin_card_shadow", "soft"},
		{"admin_card_border", "subtle"},
		{"admin_glass_effect_enabled", "1"},
		{"admin_animations_enabled", "1"},
		{"admin_compact_mode_enabled", "0"},
		{"admin_dashboard_density", "comfortable"},
		{"admin_custom_logo", ""},
		{"admin_custom_background", ""},
	}
	for _, s := range appearanceDefaults {
		DB.Exec("INSERT INTO settings (key, value) VALUES (?,?) ON CONFLICT(key) DO NOTHING", s.key, s.val)
	}

	// Seed public site appearance settings defaults
	siteAppearanceDefaults := []struct{ key, val string }{
		{"site_theme_name", "default"},
		{"site_accent_color", "#6366f1"},
		{"site_background_color", "#0a0a0f"},
		{"site_card_color", "rgba(255,255,255,0.05)"},
		{"site_text_color", "#e2e8f0"},
		{"site_muted_text_color", "#94a3b8"},
		{"site_border_color", "rgba(255,255,255,0.1)"},
		{"site_button_color", "#6366f1"},
		{"site_hover_color", "rgba(255,255,255,0.05)"},
		{"site_hero_style", "gradient"},
		{"site_card_radius", "xl"},
		{"site_card_shadow", "medium"},
		{"site_glass_effect_enabled", "1"},
		{"site_animations_enabled", "1"},
		{"site_custom_logo", ""},
		{"site_custom_background", ""},
	}
	for _, s := range siteAppearanceDefaults {
		DB.Exec("INSERT INTO settings (key, value) VALUES (?,?) ON CONFLICT(key) DO NOTHING", s.key, s.val)
	}

	// ── Customer User System ─────────────────────────
	userTables := []string{
		`CREATE TABLE IF NOT EXISTS users (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			public_id TEXT NOT NULL UNIQUE,
			email TEXT UNIQUE,
			phone TEXT UNIQUE,
			password_hash TEXT NOT NULL,
			status TEXT NOT NULL DEFAULT 'active',
			role TEXT NOT NULL DEFAULT 'user',
			telegram_id TEXT,
			telegram_username TEXT,
			telegram_connected_at DATETIME,
			email_verified_at DATETIME,
			phone_verified_at DATETIME,
			last_login_at DATETIME,
			last_login_ip_hash TEXT,
			created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
			updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
			deleted_at DATETIME
		)`,
		`CREATE TABLE IF NOT EXISTS user_profiles (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			user_id INTEGER NOT NULL UNIQUE,
			first_name TEXT,
			last_name TEXT,
			display_name TEXT,
			timezone TEXT DEFAULT 'Asia/Tehran',
			country TEXT,
			primary_device TEXT,
			usage_type TEXT,
			avatar_path TEXT,
			created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
			updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
			FOREIGN KEY(user_id) REFERENCES users(id)
		)`,
		`CREATE TABLE IF NOT EXISTS user_sessions (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			user_id INTEGER NOT NULL,
			session_token_hash TEXT NOT NULL,
			ip_hash TEXT,
			user_agent TEXT,
			created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
			expires_at DATETIME NOT NULL,
			revoked_at DATETIME
		)`,
		`CREATE TABLE IF NOT EXISTS password_reset_tokens (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			user_id INTEGER NOT NULL,
			token_hash TEXT NOT NULL UNIQUE,
			created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
			expires_at DATETIME NOT NULL,
			used_at DATETIME
		)`,
		`CREATE TABLE IF NOT EXISTS telegram_connect_tokens (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			user_id INTEGER NOT NULL,
			token_hash TEXT NOT NULL UNIQUE,
			token_public TEXT NOT NULL,
			created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
			expires_at DATETIME NOT NULL,
			used_at DATETIME
		)`,
		`CREATE TABLE IF NOT EXISTS user_services (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			user_id INTEGER NOT NULL,
			title TEXT NOT NULL,
			plan_name TEXT,
			status TEXT NOT NULL DEFAULT 'pending',
			traffic_total_bytes INTEGER NOT NULL DEFAULT 0,
			traffic_used_bytes INTEGER NOT NULL DEFAULT 0,
			traffic_remaining_bytes INTEGER NOT NULL DEFAULT 0,
			started_at DATETIME,
			expires_at DATETIME,
			location TEXT,
			subscription_url TEXT,
			qr_code_path TEXT,
			source TEXT NOT NULL DEFAULT 'manual',
			external_service_id TEXT,
			created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
			updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
		)`,
		`CREATE TABLE IF NOT EXISTS user_orders (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			user_id INTEGER NOT NULL,
			order_number TEXT NOT NULL UNIQUE,
			plan_name TEXT,
			amount INTEGER NOT NULL DEFAULT 0,
			currency TEXT NOT NULL DEFAULT 'IRT',
			payment_method TEXT,
			payment_status TEXT NOT NULL DEFAULT 'pending',
			order_status TEXT NOT NULL DEFAULT 'pending',
			discount_code TEXT,
			source TEXT,
			telegram_start_param TEXT,
			created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
			updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
		)`,
		`CREATE TABLE IF NOT EXISTS wallet_transactions (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			user_id INTEGER NOT NULL,
			type TEXT NOT NULL,
			amount INTEGER NOT NULL,
			currency TEXT NOT NULL DEFAULT 'IRT',
			balance_after INTEGER NOT NULL DEFAULT 0,
			description TEXT,
			reference_type TEXT,
			reference_id TEXT,
			created_by_admin_id INTEGER,
			created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
		)`,
		`CREATE TABLE IF NOT EXISTS support_tickets (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			user_id INTEGER NOT NULL,
			ticket_number TEXT NOT NULL UNIQUE,
			subject TEXT NOT NULL,
			category TEXT NOT NULL,
			priority TEXT NOT NULL DEFAULT 'normal',
			status TEXT NOT NULL DEFAULT 'open',
			last_message_at DATETIME,
			assigned_admin_id INTEGER,
			created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
			updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
			closed_at DATETIME
		)`,
		`CREATE TABLE IF NOT EXISTS support_ticket_messages (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			ticket_id INTEGER NOT NULL,
			sender_type TEXT NOT NULL,
			sender_user_id INTEGER,
			sender_admin_id INTEGER,
			message TEXT NOT NULL,
			attachment_path TEXT,
			created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
		)`,
		`CREATE TABLE IF NOT EXISTS user_notifications (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			user_id INTEGER NOT NULL,
			title TEXT NOT NULL,
			message TEXT NOT NULL,
			type TEXT NOT NULL DEFAULT 'info',
			link_url TEXT,
			read_at DATETIME,
			created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
		)`,
		`CREATE TABLE IF NOT EXISTS user_activity_logs (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			user_id INTEGER NOT NULL,
			action TEXT NOT NULL,
			description TEXT,
			ip_hash TEXT,
			user_agent TEXT,
			metadata_json TEXT,
			created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
		)`,
		`CREATE TABLE IF NOT EXISTS admin_user_notes (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			user_id INTEGER NOT NULL,
			admin_id INTEGER NOT NULL,
			note TEXT NOT NULL,
			created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
		)`,
		// Indexes
		`CREATE INDEX IF NOT EXISTS idx_users_email ON users(email)`,
		`CREATE INDEX IF NOT EXISTS idx_users_phone ON users(phone)`,
		`CREATE INDEX IF NOT EXISTS idx_users_telegram_id ON users(telegram_id)`,
		`CREATE INDEX IF NOT EXISTS idx_users_created_at ON users(created_at)`,
		`CREATE INDEX IF NOT EXISTS idx_user_sessions_user_id ON user_sessions(user_id)`,
		`CREATE INDEX IF NOT EXISTS idx_user_services_user_id ON user_services(user_id)`,
		`CREATE INDEX IF NOT EXISTS idx_user_orders_user_id ON user_orders(user_id)`,
		`CREATE INDEX IF NOT EXISTS idx_user_orders_number ON user_orders(order_number)`,
		`CREATE INDEX IF NOT EXISTS idx_wallet_user_id ON wallet_transactions(user_id)`,
		`CREATE INDEX IF NOT EXISTS idx_tickets_user_id ON support_tickets(user_id)`,
		`CREATE INDEX IF NOT EXISTS idx_tickets_number ON support_tickets(ticket_number)`,
		`CREATE INDEX IF NOT EXISTS idx_ticket_msgs_ticket_id ON support_ticket_messages(ticket_id)`,
		`CREATE INDEX IF NOT EXISTS idx_notifications_user_id ON user_notifications(user_id)`,
		`CREATE INDEX IF NOT EXISTS idx_activity_logs_user_id ON user_activity_logs(user_id)`,
	}
	for _, q := range userTables {
		safeExec(q)
	}

	// Seed internal API key for bot callbacks
	DB.Exec("INSERT INTO settings (key, value) VALUES ('internal_api_key', '') ON CONFLICT(key) DO NOTHING")
	// Seed customer bot username
	DB.Exec("INSERT INTO settings (key, value) VALUES ('customer_telegram_bot_username', '') ON CONFLICT(key) DO NOTHING")

	log.Println("Database migrations completed")
}
