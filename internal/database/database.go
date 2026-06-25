package database

import (
	"database/sql"
	"log"
	"os"
	"path/filepath"

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

func Migrate() {
	queries := []string{
		`CREATE TABLE IF NOT EXISTS admins (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			username TEXT NOT NULL UNIQUE,
			email TEXT NOT NULL UNIQUE,
			password_hash TEXT NOT NULL,
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
			is_active INTEGER DEFAULT 1
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
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP
		)`,
	}

	for _, q := range queries {
		if _, err := DB.Exec(q); err != nil {
			log.Fatalf("migration failed: %v\nQuery: %s", err, q)
		}
	}
	log.Println("Database migrations completed")
}
