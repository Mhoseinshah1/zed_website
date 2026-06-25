package handlers

import (
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"zedproxy/internal/models"
)

func HomePage(c *gin.Context) {
	data := basePageData("home")
	plans, _ := models.GetActivePlans()
	features, _ := models.GetActiveFeatures()
	faqs, _ := models.GetActiveFAQs()
	locations, _ := models.GetActiveLocations()
	posts, _ := models.GetPublishedPosts(3)
	settings := data["Settings"].(map[string]string)

	data["Plans"] = plans
	data["Features"] = features
	data["FAQs"] = faqs
	data["Locations"] = locations
	data["Posts"] = posts
	data["Title"] = settings["seo_title"]
	data["Description"] = settings["seo_description"]
	data["CanonicalURL"] = settings["site_url"]

	renderPage(c, "home", data)
}

func PlansPage(c *gin.Context) {
	data := basePageData("plans")
	plans, _ := models.GetActivePlans()
	settings := data["Settings"].(map[string]string)
	data["Plans"] = plans
	data["Title"] = "پلن‌های ZedProxy - خرید اشتراک پروکسی"
	data["Description"] = "مشاهده و خرید انواع پلن‌های ZedProxy. از پلن برنز تا پلاتینیوم برای همه نیازها."
	data["CanonicalURL"] = settings["site_url"] + "/plans"
	renderPage(c, "plans", data)
}

func TutorialsPage(c *gin.Context) {
	data := basePageData("tutorials")
	tutorials, _ := models.GetPublishedTutorials()
	settings := data["Settings"].(map[string]string)
	data["Tutorials"] = tutorials
	data["Title"] = "آموزش نصب و راه‌اندازی ZedProxy"
	data["Description"] = "آموزش کامل نصب و راه‌اندازی ZedProxy روی تمام دستگاه‌ها: اندروید، iOS، ویندوز، مک."
	data["CanonicalURL"] = settings["site_url"] + "/tutorials"
	renderPage(c, "tutorials", data)
}

func TutorialDetailPage(c *gin.Context) {
	slug := c.Param("slug")
	tutorial, err := models.GetTutorialBySlug(slug)
	if err != nil {
		c.Status(http.StatusNotFound)
		data := basePageData("tutorials")
		data["Title"] = "صفحه یافت نشد"
		renderPage(c, "404", data)
		return
	}
	data := basePageData("tutorials")
	settings := data["Settings"].(map[string]string)
	data["Tutorial"] = tutorial
	data["Title"] = tutorial.Title + " - ZedProxy"
	data["Description"] = tutorial.Excerpt
	data["CanonicalURL"] = settings["site_url"] + "/tutorials/" + slug
	renderPage(c, "tutorial-detail", data)
}

func BlogPage(c *gin.Context) {
	data := basePageData("blog")
	posts, _ := models.GetPublishedPosts(0)
	settings := data["Settings"].(map[string]string)
	data["Posts"] = posts
	data["Title"] = "وبلاگ ZedProxy - مقالات و آموزش‌های اینترنت آزاد"
	data["Description"] = "جدیدترین مقالات و اخبار درباره پروکسی، VPN و اینترنت آزاد در وبلاگ ZedProxy."
	data["CanonicalURL"] = settings["site_url"] + "/blog"
	renderPage(c, "blog", data)
}

func BlogPostPage(c *gin.Context) {
	slug := c.Param("slug")
	post, err := models.GetPostBySlug(slug)
	if err != nil {
		c.Status(http.StatusNotFound)
		data := basePageData("blog")
		data["Title"] = "صفحه یافت نشد"
		renderPage(c, "404", data)
		return
	}
	data := basePageData("blog")
	settings := data["Settings"].(map[string]string)
	data["Post"] = post
	metaTitle := post.MetaTitle
	if metaTitle == "" {
		metaTitle = post.Title + " - ZedProxy"
	}
	data["Title"] = metaTitle
	data["Description"] = post.MetaDescription
	data["CanonicalURL"] = settings["site_url"] + "/blog/" + slug
	renderPage(c, "blog-post", data)
}

func FAQPage(c *gin.Context) {
	data := basePageData("faq")
	faqs, _ := models.GetActiveFAQs()
	settings := data["Settings"].(map[string]string)
	data["FAQs"] = faqs
	data["Title"] = "سوالات متداول - ZedProxy"
	data["Description"] = "پاسخ به سوالات متداول درباره خرید، نصب و استفاده از ZedProxy."
	data["CanonicalURL"] = settings["site_url"] + "/faq"
	data["FAQSchema"] = buildFAQSchema(faqs)
	renderPage(c, "faq", data)
}

func buildFAQSchema(faqs []models.FAQ) string {
	if len(faqs) == 0 {
		return ""
	}
	var sb strings.Builder
	sb.WriteString(`{"@context":"https://schema.org","@type":"FAQPage","mainEntity":[`)
	for i, f := range faqs {
		if i > 0 {
			sb.WriteString(",")
		}
		q := strings.ReplaceAll(f.Question, `"`, `\"`)
		a := strings.ReplaceAll(f.Answer, `"`, `\"`)
		a = strings.ReplaceAll(a, "\n", " ")
		sb.WriteString(fmt.Sprintf(`{"@type":"Question","name":"%s","acceptedAnswer":{"@type":"Answer","text":"%s"}}`, q, a))
	}
	sb.WriteString("]}")
	return sb.String()
}

func ContactPage(c *gin.Context) {
	data := basePageData("contact")
	settings := data["Settings"].(map[string]string)
	data["Title"] = "تماس و پشتیبانی - ZedProxy"
	data["Description"] = "ارتباط با تیم پشتیبانی ZedProxy از طریق تلگرام."
	data["CanonicalURL"] = settings["site_url"] + "/contact"
	renderPage(c, "contact", data)
}

func StatusPage(c *gin.Context) {
	data := basePageData("status")
	updates, _ := models.GetStatusUpdates(20)
	settings := data["Settings"].(map[string]string)
	data["Updates"] = updates
	data["Title"] = "وضعیت سرویس - ZedProxy"
	data["Description"] = "وضعیت فعلی سرویس‌های ZedProxy و آخرین اطلاعیه‌ها."
	data["CanonicalURL"] = settings["site_url"] + "/status"
	renderPage(c, "status", data)
}

func TermsPage(c *gin.Context) {
	data := basePageData("terms")
	settings := data["Settings"].(map[string]string)
	page, _ := models.GetPageBySlug("terms")
	if page != nil {
		data["Page"] = page
		data["Title"] = page.MetaTitle
		data["Description"] = page.MetaDescription
	}
	data["CanonicalURL"] = settings["site_url"] + "/terms"
	renderPage(c, "legal", data)
}

func PrivacyPage(c *gin.Context) {
	data := basePageData("privacy")
	settings := data["Settings"].(map[string]string)
	page, _ := models.GetPageBySlug("privacy")
	if page != nil {
		data["Page"] = page
		data["Title"] = page.MetaTitle
		data["Description"] = page.MetaDescription
	}
	data["CanonicalURL"] = settings["site_url"] + "/privacy"
	renderPage(c, "legal", data)
}

func TrackClick(c *gin.Context) {
	page := c.PostForm("page")
	source := c.PostForm("source")
	ip := c.ClientIP()
	ua := c.Request.UserAgent()
	if page != "" {
		models.RecordClick(page, source, ip, ua)
	}
	c.JSON(http.StatusOK, gin.H{"ok": true})
}

func SitemapXML(c *gin.Context) {
	settings := models.GetAllSettings()
	siteURL := settings["site_url"]

	var sb strings.Builder
	sb.WriteString(`<?xml version="1.0" encoding="UTF-8"?>`)
	sb.WriteString(`<urlset xmlns="http://www.sitemaps.org/schemas/sitemap/0.9">`)

	staticPages := []struct {
		loc      string
		freq     string
		priority string
	}{
		{siteURL + "/", "daily", "1.0"},
		{siteURL + "/plans", "weekly", "0.9"},
		{siteURL + "/tutorials", "weekly", "0.8"},
		{siteURL + "/blog", "daily", "0.8"},
		{siteURL + "/faq", "monthly", "0.7"},
		{siteURL + "/contact", "monthly", "0.6"},
		{siteURL + "/status", "daily", "0.5"},
		{siteURL + "/terms", "monthly", "0.3"},
		{siteURL + "/privacy", "monthly", "0.3"},
	}

	now := time.Now().Format("2006-01-02")
	for _, p := range staticPages {
		sb.WriteString(fmt.Sprintf(`<url><loc>%s</loc><lastmod>%s</lastmod><changefreq>%s</changefreq><priority>%s</priority></url>`, p.loc, now, p.freq, p.priority))
	}

	posts, _ := models.GetPublishedPosts(0)
	for _, post := range posts {
		sb.WriteString(fmt.Sprintf(`<url><loc>%s/blog/%s</loc><lastmod>%s</lastmod><changefreq>weekly</changefreq><priority>0.7</priority></url>`, siteURL, post.Slug, now))
	}

	tutorials, _ := models.GetPublishedTutorials()
	for _, t := range tutorials {
		sb.WriteString(fmt.Sprintf(`<url><loc>%s/tutorials/%s</loc><lastmod>%s</lastmod><changefreq>monthly</changefreq><priority>0.7</priority></url>`, siteURL, t.Slug, now))
	}

	sb.WriteString(`</urlset>`)
	c.Header("Content-Type", "application/xml")
	c.String(http.StatusOK, sb.String())
}

func RobotsTXT(c *gin.Context) {
	settings := models.GetAllSettings()
	siteURL := settings["site_url"]
	content := fmt.Sprintf("User-agent: *\nAllow: /\nDisallow: /zed-admin/\nSitemap: %s/sitemap.xml\n", siteURL)
	c.Header("Content-Type", "text/plain")
	c.String(http.StatusOK, content)
}

func renderPage(c *gin.Context, name string, data map[string]interface{}) {
	t, err := getTemplate(name)
	if err != nil {
		c.String(http.StatusInternalServerError, "Template error: %v", err)
		return
	}
	c.Header("Content-Type", "text/html; charset=utf-8")
	if err := t.ExecuteTemplate(c.Writer, "base", data); err != nil {
		// Don't write again if headers already sent
		_ = err
	}
}
