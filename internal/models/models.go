package models

import (
	"crypto/sha256"
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
	ID              int
	Question        string
	Answer          string
	Category        string
	SortOrder       int
	IsActive        bool
	ShowOnHomepage  bool
	ShowOnFAQ       bool
}

func GetActiveFAQs() ([]FAQ, error) {
	rows, err := database.DB.Query("SELECT id, question, answer, category, sort_order, COALESCE(show_on_homepage,0), COALESCE(show_on_faq,1) FROM faqs WHERE is_active=1 ORDER BY sort_order ASC, id ASC")
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var faqs []FAQ
	for rows.Next() {
		var f FAQ
		rows.Scan(&f.ID, &f.Question, &f.Answer, &f.Category, &f.SortOrder, &f.ShowOnHomepage, &f.ShowOnFAQ)
		faqs = append(faqs, f)
	}
	return faqs, nil
}

func GetHomepageFAQs() ([]FAQ, error) {
	rows, err := database.DB.Query("SELECT id, question, answer, category, sort_order FROM faqs WHERE is_active=1 AND show_on_homepage=1 ORDER BY sort_order ASC, id ASC LIMIT 8")
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
	rows, err := database.DB.Query("SELECT id, question, answer, category, sort_order, is_active, COALESCE(show_on_homepage,0), COALESCE(show_on_faq,1) FROM faqs ORDER BY sort_order ASC, id ASC")
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var faqs []FAQ
	for rows.Next() {
		var f FAQ
		rows.Scan(&f.ID, &f.Question, &f.Answer, &f.Category, &f.SortOrder, &f.IsActive, &f.ShowOnHomepage, &f.ShowOnFAQ)
		faqs = append(faqs, f)
	}
	return faqs, nil
}

func GetFAQByID(id int) (*FAQ, error) {
	var f FAQ
	err := database.DB.QueryRow("SELECT id, question, answer, category, sort_order, is_active, COALESCE(show_on_homepage,0), COALESCE(show_on_faq,1) FROM faqs WHERE id=?", id).
		Scan(&f.ID, &f.Question, &f.Answer, &f.Category, &f.SortOrder, &f.IsActive, &f.ShowOnHomepage, &f.ShowOnFAQ)
	if err != nil {
		return nil, err
	}
	return &f, nil
}

func CreateFAQ(f FAQ) error {
	_, err := database.DB.Exec("INSERT INTO faqs (question, answer, category, sort_order, is_active, show_on_homepage, show_on_faq) VALUES (?,?,?,?,?,?,?)",
		f.Question, f.Answer, f.Category, f.SortOrder, f.IsActive, f.ShowOnHomepage, f.ShowOnFAQ)
	return err
}

func UpdateFAQ(f FAQ) error {
	_, err := database.DB.Exec("UPDATE faqs SET question=?, answer=?, category=?, sort_order=?, is_active=?, show_on_homepage=?, show_on_faq=? WHERE id=?",
		f.Question, f.Answer, f.Category, f.SortOrder, f.IsActive, f.ShowOnHomepage, f.ShowOnFAQ, f.ID)
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
	ID              int
	Slug            string
	Title           string
	Excerpt         string
	Content         string
	Image           string
	Category        string
	Platform        string
	VideoURL        string
	MetaTitle       string
	MetaDescription string
	SortOrder       int
	IsPublished     bool
	CreatedAt       time.Time
}

func GetPublishedTutorials() ([]Tutorial, error) {
	rows, err := database.DB.Query("SELECT id, slug, title, excerpt, image, category, platform, COALESCE(video_url,''), sort_order, created_at FROM tutorials WHERE is_published=1 ORDER BY sort_order ASC, id ASC")
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var tutorials []Tutorial
	for rows.Next() {
		var t Tutorial
		rows.Scan(&t.ID, &t.Slug, &t.Title, &t.Excerpt, &t.Image, &t.Category, &t.Platform, &t.VideoURL, &t.SortOrder, &t.CreatedAt)
		tutorials = append(tutorials, t)
	}
	return tutorials, nil
}

func GetAllTutorials() ([]Tutorial, error) {
	rows, err := database.DB.Query("SELECT id, slug, title, excerpt, image, category, platform, COALESCE(video_url,''), sort_order, is_published, created_at FROM tutorials ORDER BY sort_order ASC, id ASC")
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var tutorials []Tutorial
	for rows.Next() {
		var t Tutorial
		rows.Scan(&t.ID, &t.Slug, &t.Title, &t.Excerpt, &t.Image, &t.Category, &t.Platform, &t.VideoURL, &t.SortOrder, &t.IsPublished, &t.CreatedAt)
		tutorials = append(tutorials, t)
	}
	return tutorials, nil
}

func GetTutorialBySlug(slug string) (*Tutorial, error) {
	var t Tutorial
	err := database.DB.QueryRow("SELECT id, slug, title, excerpt, content, image, category, platform, COALESCE(video_url,''), COALESCE(meta_title,''), COALESCE(meta_description,''), sort_order, created_at FROM tutorials WHERE slug=? AND is_published=1", slug).
		Scan(&t.ID, &t.Slug, &t.Title, &t.Excerpt, &t.Content, &t.Image, &t.Category, &t.Platform, &t.VideoURL, &t.MetaTitle, &t.MetaDescription, &t.SortOrder, &t.CreatedAt)
	if err != nil {
		return nil, err
	}
	return &t, nil
}

func GetTutorialByID(id int) (*Tutorial, error) {
	var t Tutorial
	err := database.DB.QueryRow("SELECT id, slug, title, excerpt, content, image, category, platform, COALESCE(video_url,''), COALESCE(meta_title,''), COALESCE(meta_description,''), sort_order, is_published, created_at FROM tutorials WHERE id=?", id).
		Scan(&t.ID, &t.Slug, &t.Title, &t.Excerpt, &t.Content, &t.Image, &t.Category, &t.Platform, &t.VideoURL, &t.MetaTitle, &t.MetaDescription, &t.SortOrder, &t.IsPublished, &t.CreatedAt)
	if err != nil {
		return nil, err
	}
	return &t, nil
}

func CreateTutorial(t Tutorial) error {
	_, err := database.DB.Exec(
		"INSERT INTO tutorials (slug, title, excerpt, content, image, category, platform, video_url, meta_title, meta_description, sort_order, is_published) VALUES (?,?,?,?,?,?,?,?,?,?,?,?)",
		t.Slug, t.Title, t.Excerpt, t.Content, t.Image, t.Category, t.Platform, t.VideoURL, t.MetaTitle, t.MetaDescription, t.SortOrder, t.IsPublished,
	)
	return err
}

func UpdateTutorial(t Tutorial) error {
	_, err := database.DB.Exec(
		"UPDATE tutorials SET slug=?, title=?, excerpt=?, content=?, image=?, category=?, platform=?, video_url=?, meta_title=?, meta_description=?, sort_order=?, is_published=?, updated_at=CURRENT_TIMESTAMP WHERE id=?",
		t.Slug, t.Title, t.Excerpt, t.Content, t.Image, t.Category, t.Platform, t.VideoURL, t.MetaTitle, t.MetaDescription, t.SortOrder, t.IsPublished, t.ID,
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

// StatusItem (professional service status)

type StatusItem struct {
	ID          int
	Name        string
	ServiceType string
	Status      string
	Description string
	SortOrder   int
	IsActive    bool
}

func GetActiveStatusItems() ([]StatusItem, error) {
	rows, err := database.DB.Query("SELECT id, name, service_type, status, description, sort_order FROM status_items WHERE is_active=1 ORDER BY sort_order ASC, id ASC")
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var items []StatusItem
	for rows.Next() {
		var s StatusItem
		rows.Scan(&s.ID, &s.Name, &s.ServiceType, &s.Status, &s.Description, &s.SortOrder)
		items = append(items, s)
	}
	return items, nil
}

func GetAllStatusItems() ([]StatusItem, error) {
	rows, err := database.DB.Query("SELECT id, name, service_type, status, description, sort_order, is_active FROM status_items ORDER BY sort_order ASC, id ASC")
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var items []StatusItem
	for rows.Next() {
		var s StatusItem
		rows.Scan(&s.ID, &s.Name, &s.ServiceType, &s.Status, &s.Description, &s.SortOrder, &s.IsActive)
		items = append(items, s)
	}
	return items, nil
}

func GetStatusItemByID(id int) (*StatusItem, error) {
	var s StatusItem
	err := database.DB.QueryRow("SELECT id, name, service_type, status, description, sort_order, is_active FROM status_items WHERE id=?", id).
		Scan(&s.ID, &s.Name, &s.ServiceType, &s.Status, &s.Description, &s.SortOrder, &s.IsActive)
	if err != nil {
		return nil, err
	}
	return &s, nil
}

func CreateStatusItem(s StatusItem) error {
	_, err := database.DB.Exec("INSERT INTO status_items (name, service_type, status, description, sort_order, is_active) VALUES (?,?,?,?,?,?)",
		s.Name, s.ServiceType, s.Status, s.Description, s.SortOrder, s.IsActive)
	return err
}

func UpdateStatusItem(s StatusItem) error {
	_, err := database.DB.Exec("UPDATE status_items SET name=?, service_type=?, status=?, description=?, sort_order=?, is_active=? WHERE id=?",
		s.Name, s.ServiceType, s.Status, s.Description, s.SortOrder, s.IsActive, s.ID)
	return err
}

func DeleteStatusItem(id int) error {
	_, err := database.DB.Exec("DELETE FROM status_items WHERE id=?", id)
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

// ClickEvent (legacy)

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
	database.DB.QueryRow("SELECT COUNT(*) FROM telegram_clicks").Scan(&stats.Total)
	database.DB.QueryRow("SELECT COUNT(*) FROM telegram_clicks WHERE date(created_at)=date('now')").Scan(&stats.Today)
	database.DB.QueryRow("SELECT page FROM telegram_clicks GROUP BY page ORDER BY COUNT(*) DESC LIMIT 1").Scan(&stats.MostClicked)
	return stats
}

type ClickEvent struct {
	ID        int
	Page      string
	Source    string
	IPHash    string
	UserAgent string
	CreatedAt time.Time
}

func GetRecentClicks(limit int) ([]ClickEvent, error) {
	rows, err := database.DB.Query("SELECT id, page, source, ip_hash, created_at FROM telegram_clicks ORDER BY created_at DESC LIMIT ?", limit)
	if err != nil {
		// fallback to old table
		rows, err = database.DB.Query("SELECT id, page, source, ip, user_agent, created_at FROM click_events ORDER BY created_at DESC LIMIT ?", limit)
		if err != nil {
			return nil, err
		}
		defer rows.Close()
		var events []ClickEvent
		for rows.Next() {
			var e ClickEvent
			rows.Scan(&e.ID, &e.Page, &e.Source, &e.IPHash, &e.UserAgent, &e.CreatedAt)
			events = append(events, e)
		}
		return events, nil
	}
	defer rows.Close()
	var events []ClickEvent
	for rows.Next() {
		var e ClickEvent
		var ua string
		rows.Scan(&e.ID, &e.Page, &e.Source, &e.IPHash, &e.CreatedAt)
		_ = ua
		events = append(events, e)
	}
	return events, nil
}

// TelegramClick (enhanced)

type TelegramClick struct {
	ID          int
	Page        string
	Source      string
	PlanID      int
	Campaign    string
	DeviceType  string
	Referrer    string
	UTMSource   string
	UTMMedium   string
	UTMCampaign string
	IPHash      string
	CreatedAt   time.Time
}

func hashIP(ip string) string {
	h := sha256.Sum256([]byte(ip))
	return fmt.Sprintf("%x", h[:8])
}

func RecordTelegramClick(page, source, planID, campaign, deviceType, referrer, utmSrc, utmMed, utmCamp, ip string) error {
	ipHash := hashIP(ip)
	_, err := database.DB.Exec(
		"INSERT INTO telegram_clicks (page, source, plan_id, campaign, device_type, referrer, utm_source, utm_medium, utm_campaign, ip_hash) VALUES (?,?,?,?,?,?,?,?,?,?)",
		page, source, planID, campaign, deviceType, referrer, utmSrc, utmMed, utmCamp, ipHash,
	)
	return err
}

type ClickAnalytics struct {
	TotalClicks   int
	TodayClicks   int
	WeekClicks    int
	TopPages      []PageStat
	TopSources    []PageStat
	TopCampaigns  []PageStat
	DeviceBreakdown []PageStat
}

type PageStat struct {
	Label string
	Count int
}

func GetClickAnalytics() ClickAnalytics {
	var a ClickAnalytics
	database.DB.QueryRow("SELECT COUNT(*) FROM telegram_clicks").Scan(&a.TotalClicks)
	database.DB.QueryRow("SELECT COUNT(*) FROM telegram_clicks WHERE date(created_at)=date('now')").Scan(&a.TodayClicks)
	database.DB.QueryRow("SELECT COUNT(*) FROM telegram_clicks WHERE created_at >= datetime('now', '-7 days')").Scan(&a.WeekClicks)

	rows, _ := database.DB.Query("SELECT page, COUNT(*) as c FROM telegram_clicks GROUP BY page ORDER BY c DESC LIMIT 10")
	if rows != nil {
		defer rows.Close()
		for rows.Next() {
			var s PageStat
			rows.Scan(&s.Label, &s.Count)
			a.TopPages = append(a.TopPages, s)
		}
	}

	rows2, _ := database.DB.Query("SELECT COALESCE(NULLIF(source,''),'مستقیم'), COUNT(*) as c FROM telegram_clicks GROUP BY source ORDER BY c DESC LIMIT 10")
	if rows2 != nil {
		defer rows2.Close()
		for rows2.Next() {
			var s PageStat
			rows2.Scan(&s.Label, &s.Count)
			a.TopSources = append(a.TopSources, s)
		}
	}

	rows3, _ := database.DB.Query("SELECT COALESCE(NULLIF(device_type,''),'نامشخص'), COUNT(*) as c FROM telegram_clicks GROUP BY device_type ORDER BY c DESC LIMIT 5")
	if rows3 != nil {
		defer rows3.Close()
		for rows3.Next() {
			var s PageStat
			rows3.Scan(&s.Label, &s.Count)
			a.DeviceBreakdown = append(a.DeviceBreakdown, s)
		}
	}

	rows4, _ := database.DB.Query("SELECT COALESCE(NULLIF(campaign,''),'بدون کمپین'), COUNT(*) as c FROM telegram_clicks WHERE campaign!='' GROUP BY campaign ORDER BY c DESC LIMIT 5")
	if rows4 != nil {
		defer rows4.Close()
		for rows4.Next() {
			var s PageStat
			rows4.Scan(&s.Label, &s.Count)
			a.TopCampaigns = append(a.TopCampaigns, s)
		}
	}

	return a
}

// Admin

type Admin struct {
	ID           int
	Username     string
	Email        string
	PasswordHash string
	Role         string
}

func GetAdminByUsername(username string) (*Admin, error) {
	var a Admin
	err := database.DB.QueryRow("SELECT id, username, email, password_hash, COALESCE(role,'owner') FROM admins WHERE username=?", username).
		Scan(&a.ID, &a.Username, &a.Email, &a.PasswordHash, &a.Role)
	if err != nil {
		return nil, err
	}
	return &a, nil
}

func GetAdminByID(id int) (*Admin, error) {
	var a Admin
	err := database.DB.QueryRow("SELECT id, username, email, password_hash, COALESCE(role,'owner') FROM admins WHERE id=?", id).
		Scan(&a.ID, &a.Username, &a.Email, &a.PasswordHash, &a.Role)
	if err != nil {
		return nil, err
	}
	return &a, nil
}

func GetAllAdmins() ([]Admin, error) {
	rows, err := database.DB.Query("SELECT id, username, email, COALESCE(role,'owner') FROM admins ORDER BY id ASC")
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var admins []Admin
	for rows.Next() {
		var a Admin
		rows.Scan(&a.ID, &a.Username, &a.Email, &a.Role)
		admins = append(admins, a)
	}
	return admins, nil
}

func CreateAdmin(username, email, passwordHash string) error {
	_, err := database.DB.Exec("INSERT INTO admins (username, email, password_hash, role) VALUES (?,?,?,'owner')", username, email, passwordHash)
	return err
}

func CreateAdminWithRole(username, email, passwordHash, role string) error {
	_, err := database.DB.Exec("INSERT INTO admins (username, email, password_hash, role) VALUES (?,?,?,?)", username, email, passwordHash, role)
	return err
}

func UpdateAdminPassword(id int, hash string) error {
	_, err := database.DB.Exec("UPDATE admins SET password_hash=? WHERE id=?", hash, id)
	return err
}

func UpdateAdminRole(id int, role string) error {
	_, err := database.DB.Exec("UPDATE admins SET role=? WHERE id=?", role, id)
	return err
}

func DeleteAdmin(id int) error {
	_, err := database.DB.Exec("DELETE FROM admins WHERE id=?", id)
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
	AltText      string
	CreatedAt    time.Time
}

func CreateUploadedFile(f UploadedFile) error {
	_, err := database.DB.Exec("INSERT INTO uploaded_files (filename, original_name, mime_type, size, path, alt_text) VALUES (?,?,?,?,?,?)",
		f.Filename, f.OriginalName, f.MimeType, f.Size, f.Path, f.AltText)
	return err
}

func GetAllUploadedFiles() ([]UploadedFile, error) {
	rows, err := database.DB.Query("SELECT id, filename, original_name, mime_type, size, path, COALESCE(alt_text,''), created_at FROM uploaded_files ORDER BY created_at DESC")
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var files []UploadedFile
	for rows.Next() {
		var f UploadedFile
		rows.Scan(&f.ID, &f.Filename, &f.OriginalName, &f.MimeType, &f.Size, &f.Path, &f.AltText, &f.CreatedAt)
		files = append(files, f)
	}
	return files, nil
}

func GetUploadedFileByID(id int) (*UploadedFile, error) {
	var f UploadedFile
	err := database.DB.QueryRow("SELECT id, filename, original_name, mime_type, size, path, COALESCE(alt_text,''), created_at FROM uploaded_files WHERE id=?", id).
		Scan(&f.ID, &f.Filename, &f.OriginalName, &f.MimeType, &f.Size, &f.Path, &f.AltText, &f.CreatedAt)
	if err != nil {
		return nil, err
	}
	return &f, nil
}

func UpdateUploadedFileAlt(id int, altText string) error {
	_, err := database.DB.Exec("UPDATE uploaded_files SET alt_text=? WHERE id=?", altText, id)
	return err
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

// Announcement

type Announcement struct {
	ID          int
	Message     string
	Color       string
	IsClosable  bool
	IsActive    bool
	TargetPages string
	StartAt     sql.NullTime
	EndAt       sql.NullTime
	SortOrder   int
	CreatedAt   time.Time
}

func GetActiveAnnouncements(page string) ([]Announcement, error) {
	rows, err := database.DB.Query(`SELECT id, message, color, is_closable, target_pages FROM announcements
		WHERE is_active=1 AND (start_at IS NULL OR start_at <= datetime('now')) AND (end_at IS NULL OR end_at >= datetime('now'))
		AND (target_pages='all' OR target_pages LIKE ? OR target_pages LIKE ? OR target_pages LIKE ?)
		ORDER BY sort_order ASC, id ASC`,
		page, "%,"+page+",%", page+",%")
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var items []Announcement
	for rows.Next() {
		var a Announcement
		rows.Scan(&a.ID, &a.Message, &a.Color, &a.IsClosable, &a.TargetPages)
		items = append(items, a)
	}
	return items, nil
}

func GetAllAnnouncements() ([]Announcement, error) {
	rows, err := database.DB.Query("SELECT id, message, color, is_closable, is_active, target_pages, start_at, end_at, sort_order, created_at FROM announcements ORDER BY sort_order ASC, id ASC")
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var items []Announcement
	for rows.Next() {
		var a Announcement
		rows.Scan(&a.ID, &a.Message, &a.Color, &a.IsClosable, &a.IsActive, &a.TargetPages, &a.StartAt, &a.EndAt, &a.SortOrder, &a.CreatedAt)
		items = append(items, a)
	}
	return items, nil
}

func GetAnnouncementByID(id int) (*Announcement, error) {
	var a Announcement
	err := database.DB.QueryRow("SELECT id, message, color, is_closable, is_active, target_pages, start_at, end_at, sort_order FROM announcements WHERE id=?", id).
		Scan(&a.ID, &a.Message, &a.Color, &a.IsClosable, &a.IsActive, &a.TargetPages, &a.StartAt, &a.EndAt, &a.SortOrder)
	if err != nil {
		return nil, err
	}
	return &a, nil
}

func CreateAnnouncement(a Announcement) error {
	_, err := database.DB.Exec("INSERT INTO announcements (message, color, is_closable, is_active, target_pages, start_at, end_at, sort_order) VALUES (?,?,?,?,?,?,?,?)",
		a.Message, a.Color, a.IsClosable, a.IsActive, a.TargetPages, nullableTime(a.StartAt), nullableTime(a.EndAt), a.SortOrder)
	return err
}

func UpdateAnnouncement(a Announcement) error {
	_, err := database.DB.Exec("UPDATE announcements SET message=?, color=?, is_closable=?, is_active=?, target_pages=?, start_at=?, end_at=?, sort_order=? WHERE id=?",
		a.Message, a.Color, a.IsClosable, a.IsActive, a.TargetPages, nullableTime(a.StartAt), nullableTime(a.EndAt), a.SortOrder, a.ID)
	return err
}

func DeleteAnnouncement(id int) error {
	_, err := database.DB.Exec("DELETE FROM announcements WHERE id=?", id)
	return err
}

func nullableTime(t sql.NullTime) interface{} {
	if t.Valid {
		return t.Time
	}
	return nil
}

// DiscountCode

type DiscountCode struct {
	ID              int
	Code            string
	Description     string
	DiscountPercent int
	IsActive        bool
	ExpiresAt       sql.NullTime
	CreatedAt       time.Time
}

func GetActiveDiscountCodes() ([]DiscountCode, error) {
	rows, err := database.DB.Query(`SELECT id, code, description, discount_percent, expires_at FROM discount_codes
		WHERE is_active=1 AND (expires_at IS NULL OR expires_at > datetime('now')) ORDER BY id DESC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var codes []DiscountCode
	for rows.Next() {
		var d DiscountCode
		rows.Scan(&d.ID, &d.Code, &d.Description, &d.DiscountPercent, &d.ExpiresAt)
		codes = append(codes, d)
	}
	return codes, nil
}

func GetAllDiscountCodes() ([]DiscountCode, error) {
	rows, err := database.DB.Query("SELECT id, code, description, discount_percent, is_active, expires_at, created_at FROM discount_codes ORDER BY id DESC")
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var codes []DiscountCode
	for rows.Next() {
		var d DiscountCode
		rows.Scan(&d.ID, &d.Code, &d.Description, &d.DiscountPercent, &d.IsActive, &d.ExpiresAt, &d.CreatedAt)
		codes = append(codes, d)
	}
	return codes, nil
}

func GetDiscountCodeByID(id int) (*DiscountCode, error) {
	var d DiscountCode
	err := database.DB.QueryRow("SELECT id, code, description, discount_percent, is_active, expires_at FROM discount_codes WHERE id=?", id).
		Scan(&d.ID, &d.Code, &d.Description, &d.DiscountPercent, &d.IsActive, &d.ExpiresAt)
	if err != nil {
		return nil, err
	}
	return &d, nil
}

func CreateDiscountCode(d DiscountCode) error {
	_, err := database.DB.Exec("INSERT INTO discount_codes (code, description, discount_percent, is_active, expires_at) VALUES (?,?,?,?,?)",
		d.Code, d.Description, d.DiscountPercent, d.IsActive, nullableTime(d.ExpiresAt))
	return err
}

func UpdateDiscountCode(d DiscountCode) error {
	_, err := database.DB.Exec("UPDATE discount_codes SET code=?, description=?, discount_percent=?, is_active=?, expires_at=? WHERE id=?",
		d.Code, d.Description, d.DiscountPercent, d.IsActive, nullableTime(d.ExpiresAt), d.ID)
	return err
}

func DeleteDiscountCode(id int) error {
	_, err := database.DB.Exec("DELETE FROM discount_codes WHERE id=?", id)
	return err
}

// TrustCard

type TrustCard struct {
	ID          int
	Icon        string
	Title       string
	Description string
	SortOrder   int
	IsActive    bool
}

func GetActiveTrustCards() ([]TrustCard, error) {
	rows, err := database.DB.Query("SELECT id, icon, title, description, sort_order FROM trust_cards WHERE is_active=1 ORDER BY sort_order ASC, id ASC")
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var cards []TrustCard
	for rows.Next() {
		var t TrustCard
		rows.Scan(&t.ID, &t.Icon, &t.Title, &t.Description, &t.SortOrder)
		cards = append(cards, t)
	}
	return cards, nil
}

func GetAllTrustCards() ([]TrustCard, error) {
	rows, err := database.DB.Query("SELECT id, icon, title, description, sort_order, is_active FROM trust_cards ORDER BY sort_order ASC, id ASC")
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var cards []TrustCard
	for rows.Next() {
		var t TrustCard
		rows.Scan(&t.ID, &t.Icon, &t.Title, &t.Description, &t.SortOrder, &t.IsActive)
		cards = append(cards, t)
	}
	return cards, nil
}

func GetTrustCardByID(id int) (*TrustCard, error) {
	var t TrustCard
	err := database.DB.QueryRow("SELECT id, icon, title, description, sort_order, is_active FROM trust_cards WHERE id=?", id).
		Scan(&t.ID, &t.Icon, &t.Title, &t.Description, &t.SortOrder, &t.IsActive)
	if err != nil {
		return nil, err
	}
	return &t, nil
}

func CreateTrustCard(t TrustCard) error {
	_, err := database.DB.Exec("INSERT INTO trust_cards (icon, title, description, sort_order, is_active) VALUES (?,?,?,?,?)",
		t.Icon, t.Title, t.Description, t.SortOrder, t.IsActive)
	return err
}

func UpdateTrustCard(t TrustCard) error {
	_, err := database.DB.Exec("UPDATE trust_cards SET icon=?, title=?, description=?, sort_order=?, is_active=? WHERE id=?",
		t.Icon, t.Title, t.Description, t.SortOrder, t.IsActive, t.ID)
	return err
}

func DeleteTrustCard(id int) error {
	_, err := database.DB.Exec("DELETE FROM trust_cards WHERE id=?", id)
	return err
}

// Campaign

type Campaign struct {
	ID              int
	Slug            string
	Title           string
	Subtitle        string
	Description     string
	DiscountCode    string
	DiscountPercent int
	CountdownAt     sql.NullTime
	CTAText         string
	Image           string
	MetaTitle       string
	MetaDescription string
	IsActive        bool
	CreatedAt       time.Time
}

func GetCampaignBySlug(slug string) (*Campaign, error) {
	var c Campaign
	err := database.DB.QueryRow("SELECT id, slug, title, subtitle, description, discount_code, discount_percent, countdown_at, cta_text, image, meta_title, meta_description FROM campaigns WHERE slug=? AND is_active=1", slug).
		Scan(&c.ID, &c.Slug, &c.Title, &c.Subtitle, &c.Description, &c.DiscountCode, &c.DiscountPercent, &c.CountdownAt, &c.CTAText, &c.Image, &c.MetaTitle, &c.MetaDescription)
	if err != nil {
		return nil, err
	}
	return &c, nil
}

func GetAllCampaigns() ([]Campaign, error) {
	rows, err := database.DB.Query("SELECT id, slug, title, subtitle, discount_code, discount_percent, countdown_at, is_active, created_at FROM campaigns ORDER BY created_at DESC")
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var campaigns []Campaign
	for rows.Next() {
		var c Campaign
		rows.Scan(&c.ID, &c.Slug, &c.Title, &c.Subtitle, &c.DiscountCode, &c.DiscountPercent, &c.CountdownAt, &c.IsActive, &c.CreatedAt)
		campaigns = append(campaigns, c)
	}
	return campaigns, nil
}

func GetCampaignByID(id int) (*Campaign, error) {
	var c Campaign
	err := database.DB.QueryRow("SELECT id, slug, title, subtitle, description, discount_code, discount_percent, countdown_at, cta_text, image, meta_title, meta_description, is_active FROM campaigns WHERE id=?", id).
		Scan(&c.ID, &c.Slug, &c.Title, &c.Subtitle, &c.Description, &c.DiscountCode, &c.DiscountPercent, &c.CountdownAt, &c.CTAText, &c.Image, &c.MetaTitle, &c.MetaDescription, &c.IsActive)
	if err != nil {
		return nil, err
	}
	return &c, nil
}

func CreateCampaign(c Campaign) error {
	_, err := database.DB.Exec(
		"INSERT INTO campaigns (slug, title, subtitle, description, discount_code, discount_percent, countdown_at, cta_text, image, meta_title, meta_description, is_active) VALUES (?,?,?,?,?,?,?,?,?,?,?,?)",
		c.Slug, c.Title, c.Subtitle, c.Description, c.DiscountCode, c.DiscountPercent, nullableTime(c.CountdownAt), c.CTAText, c.Image, c.MetaTitle, c.MetaDescription, c.IsActive,
	)
	return err
}

func UpdateCampaign(c Campaign) error {
	_, err := database.DB.Exec(
		"UPDATE campaigns SET slug=?, title=?, subtitle=?, description=?, discount_code=?, discount_percent=?, countdown_at=?, cta_text=?, image=?, meta_title=?, meta_description=?, is_active=? WHERE id=?",
		c.Slug, c.Title, c.Subtitle, c.Description, c.DiscountCode, c.DiscountPercent, nullableTime(c.CountdownAt), c.CTAText, c.Image, c.MetaTitle, c.MetaDescription, c.IsActive, c.ID,
	)
	return err
}

func DeleteCampaign(id int) error {
	_, err := database.DB.Exec("DELETE FROM campaigns WHERE id=?", id)
	return err
}

// LandingPage

type LandingPage struct {
	ID              int
	Slug            string
	Title           string
	HeroTitle       string
	HeroSubtitle    string
	Content         string
	CTAText         string
	FeaturedImage   string
	MetaTitle       string
	MetaDescription string
	OGImage         string
	NoIndex         bool
	IsActive        bool
	CreatedAt       time.Time
}

func GetLandingPageBySlug(slug string) (*LandingPage, error) {
	var p LandingPage
	err := database.DB.QueryRow("SELECT id, slug, title, hero_title, hero_subtitle, content, cta_text, featured_image, meta_title, meta_description, og_image, noindex FROM landing_pages WHERE slug=? AND is_active=1", slug).
		Scan(&p.ID, &p.Slug, &p.Title, &p.HeroTitle, &p.HeroSubtitle, &p.Content, &p.CTAText, &p.FeaturedImage, &p.MetaTitle, &p.MetaDescription, &p.OGImage, &p.NoIndex)
	if err != nil {
		return nil, err
	}
	return &p, nil
}

func GetAllLandingPages() ([]LandingPage, error) {
	rows, err := database.DB.Query("SELECT id, slug, title, hero_title, is_active, created_at FROM landing_pages ORDER BY created_at DESC")
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var pages []LandingPage
	for rows.Next() {
		var p LandingPage
		rows.Scan(&p.ID, &p.Slug, &p.Title, &p.HeroTitle, &p.IsActive, &p.CreatedAt)
		pages = append(pages, p)
	}
	return pages, nil
}

func GetLandingPageByID(id int) (*LandingPage, error) {
	var p LandingPage
	err := database.DB.QueryRow("SELECT id, slug, title, hero_title, hero_subtitle, content, cta_text, featured_image, meta_title, meta_description, og_image, noindex, is_active FROM landing_pages WHERE id=?", id).
		Scan(&p.ID, &p.Slug, &p.Title, &p.HeroTitle, &p.HeroSubtitle, &p.Content, &p.CTAText, &p.FeaturedImage, &p.MetaTitle, &p.MetaDescription, &p.OGImage, &p.NoIndex, &p.IsActive)
	if err != nil {
		return nil, err
	}
	return &p, nil
}

func CreateLandingPage(p LandingPage) error {
	_, err := database.DB.Exec(
		"INSERT INTO landing_pages (slug, title, hero_title, hero_subtitle, content, cta_text, featured_image, meta_title, meta_description, og_image, noindex, is_active) VALUES (?,?,?,?,?,?,?,?,?,?,?,?)",
		p.Slug, p.Title, p.HeroTitle, p.HeroSubtitle, p.Content, p.CTAText, p.FeaturedImage, p.MetaTitle, p.MetaDescription, p.OGImage, p.NoIndex, p.IsActive,
	)
	return err
}

func UpdateLandingPage(p LandingPage) error {
	_, err := database.DB.Exec(
		"UPDATE landing_pages SET slug=?, title=?, hero_title=?, hero_subtitle=?, content=?, cta_text=?, featured_image=?, meta_title=?, meta_description=?, og_image=?, noindex=?, is_active=?, updated_at=CURRENT_TIMESTAMP WHERE id=?",
		p.Slug, p.Title, p.HeroTitle, p.HeroSubtitle, p.Content, p.CTAText, p.FeaturedImage, p.MetaTitle, p.MetaDescription, p.OGImage, p.NoIndex, p.IsActive, p.ID,
	)
	return err
}

func DeleteLandingPage(id int) error {
	_, err := database.DB.Exec("DELETE FROM landing_pages WHERE id=?", id)
	return err
}

// HomepageSection

type HomepageSection struct {
	ID         int
	SectionKey string
	Title      string
	Subtitle   string
	IsActive   bool
	SortOrder  int
	BGStyle    string
}

func GetHomepageSections() ([]HomepageSection, error) {
	rows, err := database.DB.Query("SELECT id, section_key, title, subtitle, is_active, sort_order, bg_style FROM homepage_sections ORDER BY sort_order ASC, id ASC")
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var sections []HomepageSection
	for rows.Next() {
		var s HomepageSection
		rows.Scan(&s.ID, &s.SectionKey, &s.Title, &s.Subtitle, &s.IsActive, &s.SortOrder, &s.BGStyle)
		sections = append(sections, s)
	}
	return sections, nil
}

func GetActiveSectionsMap() map[string]HomepageSection {
	sections, _ := GetHomepageSections()
	m := map[string]HomepageSection{}
	for _, s := range sections {
		m[s.SectionKey] = s
	}
	return m
}

func UpdateHomepageSection(s HomepageSection) error {
	_, err := database.DB.Exec("UPDATE homepage_sections SET title=?, subtitle=?, is_active=?, sort_order=?, bg_style=? WHERE section_key=?",
		s.Title, s.Subtitle, s.IsActive, s.SortOrder, s.BGStyle, s.SectionKey)
	return err
}

// Popup

type Popup struct {
	ID               int
	Title            string
	Message          string
	CTAText          string
	ShowAfterSeconds int
	ExitIntent       bool
	OncePerSession   bool
	TargetPages      string
	IsActive         bool
}

func GetActivePopup() (*Popup, error) {
	var p Popup
	err := database.DB.QueryRow("SELECT id, title, message, cta_text, show_after_seconds, exit_intent, once_per_session, target_pages FROM popups WHERE is_active=1 ORDER BY id DESC LIMIT 1").
		Scan(&p.ID, &p.Title, &p.Message, &p.CTAText, &p.ShowAfterSeconds, &p.ExitIntent, &p.OncePerSession, &p.TargetPages)
	if err != nil {
		return nil, err
	}
	return &p, nil
}

func GetAllPopups() ([]Popup, error) {
	rows, err := database.DB.Query("SELECT id, title, message, cta_text, show_after_seconds, exit_intent, once_per_session, target_pages, is_active FROM popups ORDER BY id DESC")
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var popups []Popup
	for rows.Next() {
		var p Popup
		rows.Scan(&p.ID, &p.Title, &p.Message, &p.CTAText, &p.ShowAfterSeconds, &p.ExitIntent, &p.OncePerSession, &p.TargetPages, &p.IsActive)
		popups = append(popups, p)
	}
	return popups, nil
}

func GetPopupByID(id int) (*Popup, error) {
	var p Popup
	err := database.DB.QueryRow("SELECT id, title, message, cta_text, show_after_seconds, exit_intent, once_per_session, target_pages, is_active FROM popups WHERE id=?", id).
		Scan(&p.ID, &p.Title, &p.Message, &p.CTAText, &p.ShowAfterSeconds, &p.ExitIntent, &p.OncePerSession, &p.TargetPages, &p.IsActive)
	if err != nil {
		return nil, err
	}
	return &p, nil
}

func CreatePopup(p Popup) error {
	_, err := database.DB.Exec("INSERT INTO popups (title, message, cta_text, show_after_seconds, exit_intent, once_per_session, target_pages, is_active) VALUES (?,?,?,?,?,?,?,?)",
		p.Title, p.Message, p.CTAText, p.ShowAfterSeconds, p.ExitIntent, p.OncePerSession, p.TargetPages, p.IsActive)
	return err
}

func UpdatePopup(p Popup) error {
	_, err := database.DB.Exec("UPDATE popups SET title=?, message=?, cta_text=?, show_after_seconds=?, exit_intent=?, once_per_session=?, target_pages=?, is_active=? WHERE id=?",
		p.Title, p.Message, p.CTAText, p.ShowAfterSeconds, p.ExitIntent, p.OncePerSession, p.TargetPages, p.IsActive, p.ID)
	return err
}

func DeletePopup(id int) error {
	_, err := database.DB.Exec("DELETE FROM popups WHERE id=?", id)
	return err
}

// PlanComparison

type PlanComparison struct {
	ID            int
	FeatureName   string
	BronzeValue   string
	SilverValue   string
	GoldValue     string
	PlatinumValue string
	SortOrder     int
}

func GetAllPlanComparisons() ([]PlanComparison, error) {
	rows, err := database.DB.Query("SELECT id, feature_name, bronze_value, silver_value, gold_value, platinum_value, sort_order FROM plan_comparisons ORDER BY sort_order ASC, id ASC")
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var items []PlanComparison
	for rows.Next() {
		var p PlanComparison
		rows.Scan(&p.ID, &p.FeatureName, &p.BronzeValue, &p.SilverValue, &p.GoldValue, &p.PlatinumValue, &p.SortOrder)
		items = append(items, p)
	}
	return items, nil
}

func GetPlanComparisonByID(id int) (*PlanComparison, error) {
	var p PlanComparison
	err := database.DB.QueryRow("SELECT id, feature_name, bronze_value, silver_value, gold_value, platinum_value, sort_order FROM plan_comparisons WHERE id=?", id).
		Scan(&p.ID, &p.FeatureName, &p.BronzeValue, &p.SilverValue, &p.GoldValue, &p.PlatinumValue, &p.SortOrder)
	if err != nil {
		return nil, err
	}
	return &p, nil
}

func CreatePlanComparison(p PlanComparison) error {
	_, err := database.DB.Exec("INSERT INTO plan_comparisons (feature_name, bronze_value, silver_value, gold_value, platinum_value, sort_order) VALUES (?,?,?,?,?,?)",
		p.FeatureName, p.BronzeValue, p.SilverValue, p.GoldValue, p.PlatinumValue, p.SortOrder)
	return err
}

func UpdatePlanComparison(p PlanComparison) error {
	_, err := database.DB.Exec("UPDATE plan_comparisons SET feature_name=?, bronze_value=?, silver_value=?, gold_value=?, platinum_value=?, sort_order=? WHERE id=?",
		p.FeatureName, p.BronzeValue, p.SilverValue, p.GoldValue, p.PlatinumValue, p.SortOrder, p.ID)
	return err
}

func DeletePlanComparison(id int) error {
	_, err := database.DB.Exec("DELETE FROM plan_comparisons WHERE id=?", id)
	return err
}

// DBBackup

type DBBackup struct {
	ID        int
	Filename  string
	Size      int64
	CreatedAt time.Time
}

func RecordBackup(filename string, size int64) error {
	_, err := database.DB.Exec("INSERT INTO db_backups (filename, size) VALUES (?,?)", filename, size)
	return err
}

func GetAllBackups() ([]DBBackup, error) {
	rows, err := database.DB.Query("SELECT id, filename, size, created_at FROM db_backups ORDER BY created_at DESC")
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var backups []DBBackup
	for rows.Next() {
		var b DBBackup
		rows.Scan(&b.ID, &b.Filename, &b.Size, &b.CreatedAt)
		backups = append(backups, b)
	}
	return backups, nil
}

func DeleteBackupRecord(id int) error {
	_, err := database.DB.Exec("DELETE FROM db_backups WHERE id=?", id)
	return err
}
