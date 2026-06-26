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

	log.Println("Database migrations completed")
}
