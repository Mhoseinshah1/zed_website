package models

import (
	"database/sql"
	"fmt"
	"strings"
	"time"

	"zedproxy/internal/database"
)

// Setting

func GetSetting(key string) string {
	var value string
	err := database.DB.QueryRow("SELECT value FROM settings WHERE key = ?", key).Scan(&value)
	if err != nil {
		return ""
	}
	return value
}

func SetSetting(key, value string) error {
	_, err := database.DB.Exec(
		"INSERT INTO settings (key, value, updated_at) VALUES (?, ?, CURRENT_TIMESTAMP) ON CONFLICT(key) DO UPDATE SET value=excluded.value, updated_at=excluded.updated_at",
		key, value,
	)
	return err
}

func GetAllSettings() map[string]string {
	rows, err := database.DB.Query("SELECT key, value FROM settings")
	if err != nil {
		return map[string]string{}
	}
	defer rows.Close()
	result := map[string]string{}
	for rows.Next() {
		var k, v string
		rows.Scan(&k, &v)
		result[k] = v
	}
	return result
}

// Plan

type Plan struct {
	ID          int
	Name        string
	Traffic     string
	Duration    string
	Price       string
	Badge       string
	Description string
	Features    []string
	IsPopular   bool
	SortOrder   int
	IsActive    bool
	CreatedAt   time.Time
}

func GetActivePlans() ([]Plan, error) {
	rows, err := database.DB.Query("SELECT id, name, traffic, duration, price, badge, description, features, is_popular, sort_order FROM plans WHERE is_active=1 ORDER BY sort_order ASC, id ASC")
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var plans []Plan
	for rows.Next() {
		var p Plan
		var featuresStr string
		rows.Scan(&p.ID, &p.Name, &p.Traffic, &p.Duration, &p.Price, &p.Badge, &p.Description, &featuresStr, &p.IsPopular, &p.SortOrder)
		if featuresStr != "" {
			p.Features = strings.Split(featuresStr, "\n")
		}
		plans = append(plans, p)
	}
	return plans, nil
}

func GetAllPlans() ([]Plan, error) {
	rows, err := database.DB.Query("SELECT id, name, traffic, duration, price, badge, description, features, is_popular, sort_order, is_active FROM plans ORDER BY sort_order ASC, id ASC")
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var plans []Plan
	for rows.Next() {
		var p Plan
		var featuresStr string
		rows.Scan(&p.ID, &p.Name, &p.Traffic, &p.Duration, &p.Price, &p.Badge, &p.Description, &featuresStr, &p.IsPopular, &p.SortOrder, &p.IsActive)
		if featuresStr != "" {
			p.Features = strings.Split(featuresStr, "\n")
		}
		plans = append(plans, p)
	}
	return plans, nil
}

func GetPlanByID(id int) (*Plan, error) {
	var p Plan
	var featuresStr string
	err := database.DB.QueryRow("SELECT id, name, traffic, duration, price, badge, description, features, is_popular, sort_order, is_active FROM plans WHERE id=?", id).
		Scan(&p.ID, &p.Name, &p.Traffic, &p.Duration, &p.Price, &p.Badge, &p.Description, &featuresStr, &p.IsPopular, &p.SortOrder, &p.IsActive)
	if err != nil {
		return nil, err
	}
	if featuresStr != "" {
		p.Features = strings.Split(featuresStr, "\n")
	}
	return &p, nil
}

func CreatePlan(p Plan) error {
	_, err := database.DB.Exec(
		"INSERT INTO plans (name, traffic, duration, price, badge, description, features, is_popular, sort_order, is_active) VALUES (?,?,?,?,?,?,?,?,?,?)",
		p.Name, p.Traffic, p.Duration, p.Price, p.Badge, p.Description, strings.Join(p.Features, "\n"), p.IsPopular, p.SortOrder, p.IsActive,
	)
	return err
}

func UpdatePlan(p Plan) error {
	_, err := database.DB.Exec(
		"UPDATE plans SET name=?, traffic=?, duration=?, price=?, badge=?, description=?, features=?, is_popular=?, sort_order=?, is_active=? WHERE id=?",
		p.Name, p.Traffic, p.Duration, p.Price, p.Badge, p.Description, strings.Join(p.Features, "\n"), p.IsPopular, p.SortOrder, p.IsActive, p.ID,
	)
	return err
}

func DeletePlan(id int) error {
	_, err := database.DB.Exec("DELETE FROM plans WHERE id=?", id)
	return err
}

// Feature

type Feature struct {
	ID          int
	Icon        string
	Title       string
	Description string
	SortOrder   int
	IsActive    bool
}

func GetActiveFeatures() ([]Feature, error) {
	rows, err := database.DB.Query("SELECT id, icon, title, description, sort_order FROM features WHERE is_active=1 ORDER BY sort_order ASC, id ASC")
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var features []Feature
	for rows.Next() {
		var f Feature
		rows.Scan(&f.ID, &f.Icon, &f.Title, &f.Description, &f.SortOrder)
		features = append(features, f)
	}
	return features, nil
}

func GetAllFeatures() ([]Feature, error) {
	rows, err := database.DB.Query("SELECT id, icon, title, description, sort_order, is_active FROM features ORDER BY sort_order ASC, id ASC")
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var features []Feature
	for rows.Next() {
		var f Feature
		rows.Scan(&f.ID, &f.Icon, &f.Title, &f.Description, &f.SortOrder, &f.IsActive)
		features = append(features, f)
	}
	return features, nil
}

func GetFeatureByID(id int) (*Feature, error) {
	var f Feature
	err := database.DB.QueryRow("SELECT id, icon, title, description, sort_order, is_active FROM features WHERE id=?", id).
		Scan(&f.ID, &f.Icon, &f.Title, &f.Description, &f.SortOrder, &f.IsActive)
	if err != nil {
		return nil, err
	}
	return &f, nil
}

func CreateFeature(f Feature) error {
	_, err := database.DB.Exec("INSERT INTO features (icon, title, description, sort_order, is_active) VALUES (?,?,?,?,?)",
		f.Icon, f.Title, f.Description, f.SortOrder, f.IsActive)
	return err
}

func UpdateFeature(f Feature) error {
	_, err := database.DB.Exec("UPDATE features SET icon=?, title=?, description=?, sort_order=?, is_active=? WHERE id=?",
		f.Icon, f.Title, f.Description, f.SortOrder, f.IsActive, f.ID)
	return err
}

func DeleteFeature(id int) error {
	_, err := database.DB.Exec("DELETE FROM features WHERE id=?", id)
	return err
}

// FAQ

type FAQ struct {
	ID        int
	Question  string
	Answer    string
	Category  string
	SortOrder int
	IsActive  bool
}

func GetActiveFAQs() ([]FAQ, error) {
	rows, err := database.DB.Query("SELECT id, question, answer, category, sort_order FROM faqs WHERE is_active=1 ORDER BY sort_order ASC, id ASC")
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var faqs []FAQ
	for rows.Next() {
		var f FAQ
		rows.Scan(&f.ID, &f.Question, &f.Answer, &f.Category, &f.SortOrder)
		faqs = append(faqs, f)
	}
	return faqs, nil
}

func GetAllFAQs() ([]FAQ, error) {
	rows, err := database.DB.Query("SELECT id, question, answer, category, sort_order, is_active FROM faqs ORDER BY sort_order ASC, id ASC")
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var faqs []FAQ
	for rows.Next() {
		var f FAQ
		rows.Scan(&f.ID, &f.Question, &f.Answer, &f.Category, &f.SortOrder, &f.IsActive)
		faqs = append(faqs, f)
	}
	return faqs, nil
}

func GetFAQByID(id int) (*FAQ, error) {
	var f FAQ
	err := database.DB.QueryRow("SELECT id, question, answer, category, sort_order, is_active FROM faqs WHERE id=?", id).
		Scan(&f.ID, &f.Question, &f.Answer, &f.Category, &f.SortOrder, &f.IsActive)
	if err != nil {
		return nil, err
	}
	return &f, nil
}

func CreateFAQ(f FAQ) error {
	_, err := database.DB.Exec("INSERT INTO faqs (question, answer, category, sort_order, is_active) VALUES (?,?,?,?,?)",
		f.Question, f.Answer, f.Category, f.SortOrder, f.IsActive)
	return err
}

func UpdateFAQ(f FAQ) error {
	_, err := database.DB.Exec("UPDATE faqs SET question=?, answer=?, category=?, sort_order=?, is_active=? WHERE id=?",
		f.Question, f.Answer, f.Category, f.SortOrder, f.IsActive, f.ID)
	return err
}

func DeleteFAQ(id int) error {
	_, err := database.DB.Exec("DELETE FROM faqs WHERE id=?", id)
	return err
}

// BlogPost

type BlogPost struct {
	ID              int
	Slug            string
	Title           string
	MetaTitle       string
	MetaDescription string
	Excerpt         string
	Content         string
	Image           string
	Category        string
	IsPublished     bool
	PublishedAt     sql.NullTime
	CreatedAt       time.Time
	UpdatedAt       time.Time
}

func GetPublishedPosts(limit int) ([]BlogPost, error) {
	q := "SELECT id, slug, title, meta_title, meta_description, excerpt, image, category, published_at, created_at FROM blog_posts WHERE is_published=1 ORDER BY published_at DESC"
	if limit > 0 {
		q += fmt.Sprintf(" LIMIT %d", limit)
	}
	rows, err := database.DB.Query(q)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var posts []BlogPost
	for rows.Next() {
		var p BlogPost
		rows.Scan(&p.ID, &p.Slug, &p.Title, &p.MetaTitle, &p.MetaDescription, &p.Excerpt, &p.Image, &p.Category, &p.PublishedAt, &p.CreatedAt)
		posts = append(posts, p)
	}
	return posts, nil
}

func GetAllPosts() ([]BlogPost, error) {
	rows, err := database.DB.Query("SELECT id, slug, title, meta_title, meta_description, excerpt, image, category, is_published, published_at, created_at FROM blog_posts ORDER BY created_at DESC")
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var posts []BlogPost
	for rows.Next() {
		var p BlogPost
		rows.Scan(&p.ID, &p.Slug, &p.Title, &p.MetaTitle, &p.MetaDescription, &p.Excerpt, &p.Image, &p.Category, &p.IsPublished, &p.PublishedAt, &p.CreatedAt)
		posts = append(posts, p)
	}
	return posts, nil
}

func GetPostBySlug(slug string) (*BlogPost, error) {
	var p BlogPost
	err := database.DB.QueryRow("SELECT id, slug, title, meta_title, meta_description, excerpt, content, image, category, is_published, published_at, created_at FROM blog_posts WHERE slug=? AND is_published=1", slug).
		Scan(&p.ID, &p.Slug, &p.Title, &p.MetaTitle, &p.MetaDescription, &p.Excerpt, &p.Content, &p.Image, &p.Category, &p.IsPublished, &p.PublishedAt, &p.CreatedAt)
	if err != nil {
		return nil, err
	}
	return &p, nil
}

func GetPostByID(id int) (*BlogPost, error) {
	var p BlogPost
	err := database.DB.QueryRow("SELECT id, slug, title, meta_title, meta_description, excerpt, content, image, category, is_published, published_at, created_at FROM blog_posts WHERE id=?", id).
		Scan(&p.ID, &p.Slug, &p.Title, &p.MetaTitle, &p.MetaDescription, &p.Excerpt, &p.Content, &p.Image, &p.Category, &p.IsPublished, &p.PublishedAt, &p.CreatedAt)
	if err != nil {
		return nil, err
	}
	return &p, nil
}

func CreatePost(p BlogPost) error {
	_, err := database.DB.Exec(
		"INSERT INTO blog_posts (slug, title, meta_title, meta_description, excerpt, content, image, category, is_published, published_at) VALUES (?,?,?,?,?,?,?,?,?,?)",
		p.Slug, p.Title, p.MetaTitle, p.MetaDescription, p.Excerpt, p.Content, p.Image, p.Category, p.IsPublished, p.PublishedAt,
	)
	return err
}

func UpdatePost(p BlogPost) error {
	_, err := database.DB.Exec(
		"UPDATE blog_posts SET slug=?, title=?, meta_title=?, meta_description=?, excerpt=?, content=?, image=?, category=?, is_published=?, published_at=?, updated_at=CURRENT_TIMESTAMP WHERE id=?",
		p.Slug, p.Title, p.MetaTitle, p.MetaDescription, p.Excerpt, p.Content, p.Image, p.Category, p.IsPublished, p.PublishedAt, p.ID,
	)
	return err
}

func DeletePost(id int) error {
	_, err := database.DB.Exec("DELETE FROM blog_posts WHERE id=?", id)
	return err
}

// Tutorial

type Tutorial struct {
	ID          int
	Slug        string
	Title       string
	Excerpt     string
	Content     string
	Image       string
	Category    string
	Platform    string
	SortOrder   int
	IsPublished bool
	CreatedAt   time.Time
}

func GetPublishedTutorials() ([]Tutorial, error) {
	rows, err := database.DB.Query("SELECT id, slug, title, excerpt, image, category, platform, sort_order, created_at FROM tutorials WHERE is_published=1 ORDER BY sort_order ASC, id ASC")
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var tutorials []Tutorial
	for rows.Next() {
		var t Tutorial
		rows.Scan(&t.ID, &t.Slug, &t.Title, &t.Excerpt, &t.Image, &t.Category, &t.Platform, &t.SortOrder, &t.CreatedAt)
		tutorials = append(tutorials, t)
	}
	return tutorials, nil
}

func GetAllTutorials() ([]Tutorial, error) {
	rows, err := database.DB.Query("SELECT id, slug, title, excerpt, image, category, platform, sort_order, is_published, created_at FROM tutorials ORDER BY sort_order ASC, id ASC")
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var tutorials []Tutorial
	for rows.Next() {
		var t Tutorial
		rows.Scan(&t.ID, &t.Slug, &t.Title, &t.Excerpt, &t.Image, &t.Category, &t.Platform, &t.SortOrder, &t.IsPublished, &t.CreatedAt)
		tutorials = append(tutorials, t)
	}
	return tutorials, nil
}

func GetTutorialBySlug(slug string) (*Tutorial, error) {
	var t Tutorial
	err := database.DB.QueryRow("SELECT id, slug, title, excerpt, content, image, category, platform, sort_order, created_at FROM tutorials WHERE slug=? AND is_published=1", slug).
		Scan(&t.ID, &t.Slug, &t.Title, &t.Excerpt, &t.Content, &t.Image, &t.Category, &t.Platform, &t.SortOrder, &t.CreatedAt)
	if err != nil {
		return nil, err
	}
	return &t, nil
}

func GetTutorialByID(id int) (*Tutorial, error) {
	var t Tutorial
	err := database.DB.QueryRow("SELECT id, slug, title, excerpt, content, image, category, platform, sort_order, is_published, created_at FROM tutorials WHERE id=?", id).
		Scan(&t.ID, &t.Slug, &t.Title, &t.Excerpt, &t.Content, &t.Image, &t.Category, &t.Platform, &t.SortOrder, &t.IsPublished, &t.CreatedAt)
	if err != nil {
		return nil, err
	}
	return &t, nil
}

func CreateTutorial(t Tutorial) error {
	_, err := database.DB.Exec(
		"INSERT INTO tutorials (slug, title, excerpt, content, image, category, platform, sort_order, is_published) VALUES (?,?,?,?,?,?,?,?,?)",
		t.Slug, t.Title, t.Excerpt, t.Content, t.Image, t.Category, t.Platform, t.SortOrder, t.IsPublished,
	)
	return err
}

func UpdateTutorial(t Tutorial) error {
	_, err := database.DB.Exec(
		"UPDATE tutorials SET slug=?, title=?, excerpt=?, content=?, image=?, category=?, platform=?, sort_order=?, is_published=?, updated_at=CURRENT_TIMESTAMP WHERE id=?",
		t.Slug, t.Title, t.Excerpt, t.Content, t.Image, t.Category, t.Platform, t.SortOrder, t.IsPublished, t.ID,
	)
	return err
}

func DeleteTutorial(id int) error {
	_, err := database.DB.Exec("DELETE FROM tutorials WHERE id=?", id)
	return err
}

// Location

type Location struct {
	ID        int
	Name      string
	Flag      string
	Code      string
	Speed     string
	IsActive  bool
	SortOrder int
}

func GetActiveLocations() ([]Location, error) {
	rows, err := database.DB.Query("SELECT id, name, flag, code, speed, sort_order FROM locations WHERE is_active=1 ORDER BY sort_order ASC, id ASC")
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var locs []Location
	for rows.Next() {
		var l Location
		rows.Scan(&l.ID, &l.Name, &l.Flag, &l.Code, &l.Speed, &l.SortOrder)
		locs = append(locs, l)
	}
	return locs, nil
}

func GetAllLocations() ([]Location, error) {
	rows, err := database.DB.Query("SELECT id, name, flag, code, speed, is_active, sort_order FROM locations ORDER BY sort_order ASC, id ASC")
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var locs []Location
	for rows.Next() {
		var l Location
		rows.Scan(&l.ID, &l.Name, &l.Flag, &l.Code, &l.Speed, &l.IsActive, &l.SortOrder)
		locs = append(locs, l)
	}
	return locs, nil
}

func GetLocationByID(id int) (*Location, error) {
	var l Location
	err := database.DB.QueryRow("SELECT id, name, flag, code, speed, is_active, sort_order FROM locations WHERE id=?", id).
		Scan(&l.ID, &l.Name, &l.Flag, &l.Code, &l.Speed, &l.IsActive, &l.SortOrder)
	if err != nil {
		return nil, err
	}
	return &l, nil
}

func CreateLocation(l Location) error {
	_, err := database.DB.Exec("INSERT INTO locations (name, flag, code, speed, is_active, sort_order) VALUES (?,?,?,?,?,?)",
		l.Name, l.Flag, l.Code, l.Speed, l.IsActive, l.SortOrder)
	return err
}

func UpdateLocation(l Location) error {
	_, err := database.DB.Exec("UPDATE locations SET name=?, flag=?, code=?, speed=?, is_active=?, sort_order=? WHERE id=?",
		l.Name, l.Flag, l.Code, l.Speed, l.IsActive, l.SortOrder, l.ID)
	return err
}

func DeleteLocation(id int) error {
	_, err := database.DB.Exec("DELETE FROM locations WHERE id=?", id)
	return err
}

// StatusUpdate

type StatusUpdate struct {
	ID        int
	Title     string
	Content   string
	Status    string
	CreatedAt time.Time
}

func GetStatusUpdates(limit int) ([]StatusUpdate, error) {
	rows, err := database.DB.Query("SELECT id, title, content, status, created_at FROM status_updates ORDER BY created_at DESC LIMIT ?", limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var updates []StatusUpdate
	for rows.Next() {
		var u StatusUpdate
		rows.Scan(&u.ID, &u.Title, &u.Content, &u.Status, &u.CreatedAt)
		updates = append(updates, u)
	}
	return updates, nil
}

func GetAllStatusUpdates() ([]StatusUpdate, error) {
	rows, err := database.DB.Query("SELECT id, title, content, status, created_at FROM status_updates ORDER BY created_at DESC")
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var updates []StatusUpdate
	for rows.Next() {
		var u StatusUpdate
		rows.Scan(&u.ID, &u.Title, &u.Content, &u.Status, &u.CreatedAt)
		updates = append(updates, u)
	}
	return updates, nil
}

func GetStatusUpdateByID(id int) (*StatusUpdate, error) {
	var u StatusUpdate
	err := database.DB.QueryRow("SELECT id, title, content, status, created_at FROM status_updates WHERE id=?", id).
		Scan(&u.ID, &u.Title, &u.Content, &u.Status, &u.CreatedAt)
	if err != nil {
		return nil, err
	}
	return &u, nil
}

func CreateStatusUpdate(u StatusUpdate) error {
	_, err := database.DB.Exec("INSERT INTO status_updates (title, content, status) VALUES (?,?,?)", u.Title, u.Content, u.Status)
	return err
}

func UpdateStatusUpdate(u StatusUpdate) error {
	_, err := database.DB.Exec("UPDATE status_updates SET title=?, content=?, status=? WHERE id=?", u.Title, u.Content, u.Status, u.ID)
	return err
}

func DeleteStatusUpdate(id int) error {
	_, err := database.DB.Exec("DELETE FROM status_updates WHERE id=?", id)
	return err
}

// Page

type Page struct {
	ID              int
	Slug            string
	Title           string
	Content         string
	MetaTitle       string
	MetaDescription string
	UpdatedAt       time.Time
}

func GetPageBySlug(slug string) (*Page, error) {
	var p Page
	err := database.DB.QueryRow("SELECT id, slug, title, content, meta_title, meta_description, updated_at FROM pages WHERE slug=?", slug).
		Scan(&p.ID, &p.Slug, &p.Title, &p.Content, &p.MetaTitle, &p.MetaDescription, &p.UpdatedAt)
	if err != nil {
		return nil, err
	}
	return &p, nil
}

func GetAllPages() ([]Page, error) {
	rows, err := database.DB.Query("SELECT id, slug, title, content, meta_title, meta_description, updated_at FROM pages ORDER BY slug ASC")
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var pages []Page
	for rows.Next() {
		var p Page
		rows.Scan(&p.ID, &p.Slug, &p.Title, &p.Content, &p.MetaTitle, &p.MetaDescription, &p.UpdatedAt)
		pages = append(pages, p)
	}
	return pages, nil
}

func UpsertPage(p Page) error {
	_, err := database.DB.Exec(
		"INSERT INTO pages (slug, title, content, meta_title, meta_description, updated_at) VALUES (?,?,?,?,?,CURRENT_TIMESTAMP) ON CONFLICT(slug) DO UPDATE SET title=excluded.title, content=excluded.content, meta_title=excluded.meta_title, meta_description=excluded.meta_description, updated_at=CURRENT_TIMESTAMP",
		p.Slug, p.Title, p.Content, p.MetaTitle, p.MetaDescription,
	)
	return err
}

// ClickEvent

func RecordClick(page, source, ip, userAgent string) error {
	_, err := database.DB.Exec("INSERT INTO click_events (page, source, ip, user_agent) VALUES (?,?,?,?)", page, source, ip, userAgent)
	return err
}

type ClickStats struct {
	Total       int
	Today       int
	MostClicked string
}

func GetClickStats() ClickStats {
	var stats ClickStats
	database.DB.QueryRow("SELECT COUNT(*) FROM click_events").Scan(&stats.Total)
	database.DB.QueryRow("SELECT COUNT(*) FROM click_events WHERE date(created_at)=date('now')").Scan(&stats.Today)
	database.DB.QueryRow("SELECT page FROM click_events GROUP BY page ORDER BY COUNT(*) DESC LIMIT 1").Scan(&stats.MostClicked)
	return stats
}

type ClickEvent struct {
	ID        int
	Page      string
	Source    string
	IP        string
	UserAgent string
	CreatedAt time.Time
}

func GetRecentClicks(limit int) ([]ClickEvent, error) {
	rows, err := database.DB.Query("SELECT id, page, source, ip, user_agent, created_at FROM click_events ORDER BY created_at DESC LIMIT ?", limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var events []ClickEvent
	for rows.Next() {
		var e ClickEvent
		rows.Scan(&e.ID, &e.Page, &e.Source, &e.IP, &e.UserAgent, &e.CreatedAt)
		events = append(events, e)
	}
	return events, nil
}

// Admin

type Admin struct {
	ID           int
	Username     string
	Email        string
	PasswordHash string
}

func GetAdminByUsername(username string) (*Admin, error) {
	var a Admin
	err := database.DB.QueryRow("SELECT id, username, email, password_hash FROM admins WHERE username=?", username).
		Scan(&a.ID, &a.Username, &a.Email, &a.PasswordHash)
	if err != nil {
		return nil, err
	}
	return &a, nil
}

func CreateAdmin(username, email, passwordHash string) error {
	_, err := database.DB.Exec("INSERT INTO admins (username, email, password_hash) VALUES (?,?,?)", username, email, passwordHash)
	return err
}

func UpdateAdminPassword(id int, hash string) error {
	_, err := database.DB.Exec("UPDATE admins SET password_hash=? WHERE id=?", hash, id)
	return err
}

// UploadedFile

type UploadedFile struct {
	ID           int
	Filename     string
	OriginalName string
	MimeType     string
	Size         int64
	Path         string
	CreatedAt    time.Time
}

func CreateUploadedFile(f UploadedFile) error {
	_, err := database.DB.Exec("INSERT INTO uploaded_files (filename, original_name, mime_type, size, path) VALUES (?,?,?,?,?)",
		f.Filename, f.OriginalName, f.MimeType, f.Size, f.Path)
	return err
}

func GetAllUploadedFiles() ([]UploadedFile, error) {
	rows, err := database.DB.Query("SELECT id, filename, original_name, mime_type, size, path, created_at FROM uploaded_files ORDER BY created_at DESC")
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var files []UploadedFile
	for rows.Next() {
		var f UploadedFile
		rows.Scan(&f.ID, &f.Filename, &f.OriginalName, &f.MimeType, &f.Size, &f.Path, &f.CreatedAt)
		files = append(files, f)
	}
	return files, nil
}

func DeleteUploadedFile(id int) (*UploadedFile, error) {
	var f UploadedFile
	err := database.DB.QueryRow("SELECT id, filename, path FROM uploaded_files WHERE id=?", id).
		Scan(&f.ID, &f.Filename, &f.Path)
	if err != nil {
		return nil, err
	}
	database.DB.Exec("DELETE FROM uploaded_files WHERE id=?", id)
	return &f, nil
}
