package handlers

import (
	"fmt"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/gin-contrib/sessions"
	"github.com/gin-gonic/gin"
	"golang.org/x/crypto/bcrypt"
	"zedproxy/internal/models"
	tg "zedproxy/internal/telegram"
)

var adminAppearanceKeys = []string{
	"admin_theme_name", "admin_accent_color", "admin_background_color",
	"admin_sidebar_color", "admin_card_color", "admin_text_color",
	"admin_muted_text_color", "admin_border_color", "admin_button_color",
	"admin_hover_color", "admin_sidebar_mode", "admin_sidebar_width",
	"admin_icon_size", "admin_menu_text_size", "admin_font_size",
	"admin_card_radius", "admin_card_shadow", "admin_card_border",
	"admin_glass_effect_enabled", "admin_animations_enabled",
	"admin_compact_mode_enabled", "admin_dashboard_density",
	"admin_custom_logo", "admin_custom_background",
}

// buildAdminCSSVars converts saved appearance settings into a CSS :root block
// injected into every admin page so themes apply immediately.
func buildAdminCSSVars(s map[string]string) string {
	get := func(key, def string) string {
		if v := s[key]; v != "" {
			return v
		}
		return def
	}
	radius := map[string]string{
		"none": "0", "md": "8px", "xl": "12px", "2xl": "18px",
	}[get("admin_card_radius", "xl")]
	if radius == "" {
		radius = "12px"
	}
	shadow := map[string]string{
		"none":   "none",
		"soft":   "0 2px 8px rgba(0,0,0,0.3)",
		"medium": "0 4px 16px rgba(0,0,0,0.5)",
		"strong": "0 8px 32px rgba(0,0,0,0.7)",
		"neon":   "0 0 24px " + get("admin_accent_color", "#06b6d4") + "40",
	}[get("admin_card_shadow", "soft")]
	if shadow == "" {
		shadow = "0 2px 8px rgba(0,0,0,0.3)"
	}
	fontSize := map[string]string{
		"small": "13px", "normal": "15px", "large": "17px",
	}[get("admin_font_size", "normal")]
	if fontSize == "" {
		fontSize = "15px"
	}
	iconSize := map[string]string{
		"small": "14px", "medium": "16px", "large": "20px", "xl": "24px",
	}[get("admin_icon_size", "medium")]
	if iconSize == "" {
		iconSize = "16px"
	}
	menuTextSize := map[string]string{
		"small": "12px", "medium": "13px", "large": "15px",
	}[get("admin_menu_text_size", "medium")]
	if menuTextSize == "" {
		menuTextSize = "13px"
	}
	sidebarWidth := map[string]string{
		"narrow": "200px", "normal": "240px", "wide": "280px",
	}[get("admin_sidebar_width", "normal")]
	if sidebarWidth == "" {
		sidebarWidth = "240px"
	}

	accent := get("admin_accent_color", "#06b6d4")
	bg := get("admin_background_color", "#0d0d16")
	sidebar := get("admin_sidebar_color", "#0f0f1a")
	card := get("admin_card_color", "#1a1a2e")
	text := get("admin_text_color", "#f1f5f9")
	muted := get("admin_muted_text_color", "#94a3b8")
	border := get("admin_border_color", "rgba(255,255,255,0.1)")
	button := get("admin_button_color", accent)
	hover := get("admin_hover_color", "rgba(255,255,255,0.08)")

	customBG := get("admin_custom_background", "")
	bgStyle := ""
	if customBG != "" {
		if strings.HasPrefix(customBG, "/") || strings.HasPrefix(customBG, "http") {
			bgStyle = fmt.Sprintf("url('%s') center/cover no-repeat fixed", customBG)
		} else {
			bgStyle = customBG
		}
	}

	css := fmt.Sprintf(`
:root {
  --admin-accent: %s;
  --admin-bg: %s;
  --admin-sidebar: %s;
  --admin-card: %s;
  --admin-text: %s;
  --admin-muted: %s;
  --admin-border: %s;
  --admin-button: %s;
  --admin-hover: %s;
  --admin-radius: %s;
  --admin-shadow: %s;
  --admin-font-size: %s;
  --admin-icon-size: %s;
  --admin-menu-text: %s;
  --admin-sidebar-width: %s;
}`, accent, bg, sidebar, card, text, muted, border, button, hover,
		radius, shadow, fontSize, iconSize, menuTextSize, sidebarWidth)

	if bgStyle != "" {
		css += fmt.Sprintf("\nbody.admin-body { background: %s !important; }", bgStyle)
	}
	return css
}

// buildSiteCSSVars converts saved public site appearance settings into a CSS :root block.
func buildSiteCSSVars(s map[string]string) string {
	get := func(key, def string) string {
		if v := s[key]; v != "" {
			return v
		}
		return def
	}
	radius := map[string]string{
		"none": "0", "md": "8px", "xl": "12px", "2xl": "18px",
	}[get("site_card_radius", "xl")]
	if radius == "" {
		radius = "12px"
	}
	shadow := map[string]string{
		"none":   "none",
		"soft":   "0 2px 8px rgba(0,0,0,0.3)",
		"medium": "0 4px 16px rgba(0,0,0,0.5)",
		"strong": "0 8px 32px rgba(0,0,0,0.7)",
	}[get("site_card_shadow", "medium")]
	if shadow == "" {
		shadow = "0 4px 16px rgba(0,0,0,0.5)"
	}

	accent := get("site_accent_color", "#6366f1")
	bg := get("site_background_color", "#0a0a0f")
	card := get("site_card_color", "rgba(255,255,255,0.05)")
	text := get("site_text_color", "#e2e8f0")
	muted := get("site_muted_text_color", "#94a3b8")
	border := get("site_border_color", "rgba(255,255,255,0.1)")
	button := get("site_button_color", accent)
	hover := get("site_hover_color", "rgba(255,255,255,0.05)")

	customBG := get("site_custom_background", "")
	bgStyle := ""
	if customBG != "" {
		if strings.HasPrefix(customBG, "/") || strings.HasPrefix(customBG, "http") {
			bgStyle = fmt.Sprintf("url('%s') center/cover no-repeat fixed", customBG)
		} else {
			bgStyle = customBG
		}
	}

	css := fmt.Sprintf(`:root {
  --site-accent: %s;
  --site-bg: %s;
  --site-card: %s;
  --site-text: %s;
  --site-muted: %s;
  --site-border: %s;
  --site-button: %s;
  --site-hover: %s;
  --site-radius: %s;
  --site-shadow: %s;
}`, accent, bg, card, text, muted, border, button, hover, radius, shadow)

	if bgStyle != "" {
		css += fmt.Sprintf("\nbody { background: %s !important; }", bgStyle)
	}
	return css
}

var uploadDir string

func SetUploadDir(dir string) {
	uploadDir = dir
}

func renderAdmin(c *gin.Context, name string, data map[string]interface{}) {
	// Inject appearance settings as CSS vars for every admin page
	if data != nil {
		settings := models.GetAllSettings()
		data["AdminCSS"] = buildAdminCSSVars(settings)
		// Also pass appearance keys so templates can reference them
		for _, k := range adminAppearanceKeys {
			if _, exists := data[k]; !exists {
				data[k] = settings[k]
			}
		}
	}

	t, err := getAdminTemplate(name)
	if err != nil {
		renderAdminError(c, fmt.Sprintf("خطای قالب (%s): %v", name, err))
		return
	}
	c.Header("Content-Type", "text/html; charset=utf-8")
	if err := t.ExecuteTemplate(c.Writer, "admin", data); err != nil {
		log.Printf("admin template execute error (%s): %v", name, err)
	}
}

func adminData(c *gin.Context, section string) map[string]interface{} {
	session := sessions.Default(c)
	data := map[string]interface{}{
		"Section":       section,
		"Settings":      models.GetAllSettings(),
		"AdminUsername": session.Get("admin_username"),
	}
	// Consume flashes
	flashes := session.Flashes("ok")
	session.Save()
	data["FlashOK"] = flashes
	return data
}

// Auth

func AdminLoginPage(c *gin.Context) {
	session := sessions.Default(c)
	if session.Get("admin_id") != nil {
		c.Redirect(http.StatusFound, "/zed-admin")
		return
	}
	// Login uses its own standalone template
	t, err := getAdminTemplate("login")
	if err != nil {
		c.String(500, "Template error: %v", err)
		return
	}
	c.Header("Content-Type", "text/html; charset=utf-8")
	data := map[string]interface{}{
		"Error": c.Query("error"),
	}
	t.ExecuteTemplate(c.Writer, "login", data)
}

func AdminLoginPost(c *gin.Context) {
	username := c.PostForm("username")
	password := c.PostForm("password")

	admin, err := models.GetAdminByUsername(username)
	if err != nil || bcrypt.CompareHashAndPassword([]byte(admin.PasswordHash), []byte(password)) != nil {
		tg.Send(tg.LevelWarn, tg.CatSecurity, "🔐 تلاش ورود ناموفق", fmt.Sprintf("نام کاربری: %s", username))
		c.Redirect(http.StatusFound, "/zed-admin/login?error=invalid")
		return
	}

	session := sessions.Default(c)
	session.Set("admin_id", admin.ID)
	session.Set("admin_username", admin.Username)
	session.Set("role", admin.Role)
	session.Save()
	c.Redirect(http.StatusFound, "/zed-admin")
}

func AdminLogout(c *gin.Context) {
	session := sessions.Default(c)
	session.Clear()
	session.Save()
	c.Redirect(http.StatusFound, "/zed-admin/login")
}

// Dashboard

func AdminDashboard(c *gin.Context) {
	data := adminData(c, "dashboard")
	stats := models.GetClickStats()
	recentClicks, _ := models.GetRecentClicks(10)
	plans, _ := models.GetActivePlans()
	posts, _ := models.GetAllPosts()
	tutorials, _ := models.GetAllTutorials()

	data["ClickStats"] = stats
	data["RecentClicks"] = recentClicks
	data["PlanCount"] = len(plans)
	data["PostCount"] = len(posts)
	data["TutorialCount"] = len(tutorials)
	data["Title"] = "داشبورد مدیریت"

	// System status cards
	settings := models.GetAllSettings()
	data["MaintenanceEnabled"] = settings["maintenance_enabled"]
	data["TGBotEnabled"] = settings["telegram_admin_bot_enabled"]
	data["TGBotUsername"] = settings["telegram_admin_bot_username"]
	data["AppVersionDash"] = AppVersion

	// Latest backup
	backups, _ := models.GetAllBackups()
	if len(backups) > 0 {
		data["LastBackup"] = backups[len(backups)-1]
	}

	renderAdmin(c, "dashboard", data)
}

// Settings

func AdminSettingsPage(c *gin.Context) {
	data := adminData(c, "settings")
	data["Title"] = "تنظیمات سایت"
	renderAdmin(c, "settings", data)
}

func AdminSettingsPost(c *gin.Context) {
	AdminSettingsPostV2(c)
}

// Plans

func AdminPlansPage(c *gin.Context) {
	data := adminData(c, "plans")
	plans, _ := models.GetAllPlans()
	data["Plans"] = plans
	data["Title"] = "مدیریت پلن‌ها"
	renderAdmin(c, "plans", data)
}

func AdminPlanNew(c *gin.Context) {
	data := adminData(c, "plans")
	data["Plan"] = nil
	data["FeaturesText"] = ""
	data["Title"] = "افزودن پلن جدید"
	renderAdmin(c, "plan-form", data)
}

func AdminPlanEdit(c *gin.Context) {
	id, _ := strconv.Atoi(c.Param("id"))
	plan, err := models.GetPlanByID(id)
	if err != nil {
		c.Redirect(http.StatusFound, "/zed-admin/plans")
		return
	}
	data := adminData(c, "plans")
	data["Plan"] = plan
	data["FeaturesText"] = strings.Join(plan.Features, "\n")
	data["Title"] = "ویرایش پلن"
	renderAdmin(c, "plan-form", data)
}

func AdminPlanSave(c *gin.Context) {
	idStr := c.PostForm("id")
	featuresRaw := c.PostForm("features")
	var features []string
	for _, line := range strings.Split(featuresRaw, "\n") {
		line = strings.TrimSpace(line)
		if line != "" {
			features = append(features, line)
		}
	}
	sortOrder, _ := strconv.Atoi(c.PostForm("sort_order"))
	plan := models.Plan{
		Name:        c.PostForm("name"),
		Traffic:     c.PostForm("traffic"),
		Duration:    c.PostForm("duration"),
		Price:       c.PostForm("price"),
		Badge:       c.PostForm("badge"),
		Description: c.PostForm("description"),
		Features:    features,
		IsPopular:   c.PostForm("is_popular") == "1",
		SortOrder:   sortOrder,
		IsActive:    c.PostForm("is_active") == "1",
	}

	var err error
	if idStr != "" && idStr != "0" {
		plan.ID, _ = strconv.Atoi(idStr)
		err = models.UpdatePlan(plan)
	} else {
		err = models.CreatePlan(plan)
	}

	if err != nil {
		data := adminData(c, "plans")
		data["Error"] = err.Error()
		data["Plan"] = plan
		data["FeaturesText"] = featuresRaw
		renderAdmin(c, "plan-form", data)
		return
	}
	c.Redirect(http.StatusFound, "/zed-admin/plans")
}

func AdminPlanDelete(c *gin.Context) {
	id, _ := strconv.Atoi(c.Param("id"))
	models.DeletePlan(id)
	c.Redirect(http.StatusFound, "/zed-admin/plans")
}

// Features

func AdminFeaturesPage(c *gin.Context) {
	data := adminData(c, "features")
	features, _ := models.GetAllFeatures()
	data["Features"] = features
	data["Title"] = "مدیریت ویژگی‌ها"
	renderAdmin(c, "features", data)
}

func AdminFeatureNew(c *gin.Context) {
	data := adminData(c, "features")
	data["Feature"] = nil
	data["Title"] = "افزودن ویژگی جدید"
	renderAdmin(c, "feature-form", data)
}

func AdminFeatureEdit(c *gin.Context) {
	id, _ := strconv.Atoi(c.Param("id"))
	f, err := models.GetFeatureByID(id)
	if err != nil {
		c.Redirect(http.StatusFound, "/zed-admin/features")
		return
	}
	data := adminData(c, "features")
	data["Feature"] = f
	data["Title"] = "ویرایش ویژگی"
	renderAdmin(c, "feature-form", data)
}

func AdminFeatureSave(c *gin.Context) {
	idStr := c.PostForm("id")
	sortOrder, _ := strconv.Atoi(c.PostForm("sort_order"))
	f := models.Feature{
		Icon:        c.PostForm("icon"),
		Title:       c.PostForm("title"),
		Description: c.PostForm("description"),
		SortOrder:   sortOrder,
		IsActive:    c.PostForm("is_active") == "1",
	}
	var err error
	if idStr != "" && idStr != "0" {
		f.ID, _ = strconv.Atoi(idStr)
		err = models.UpdateFeature(f)
	} else {
		err = models.CreateFeature(f)
	}
	if err != nil {
		data := adminData(c, "features")
		data["Error"] = err.Error()
		renderAdmin(c, "feature-form", data)
		return
	}
	c.Redirect(http.StatusFound, "/zed-admin/features")
}

func AdminFeatureDelete(c *gin.Context) {
	id, _ := strconv.Atoi(c.Param("id"))
	models.DeleteFeature(id)
	c.Redirect(http.StatusFound, "/zed-admin/features")
}

// FAQs

func AdminFAQsPage(c *gin.Context) {
	data := adminData(c, "faqs")
	faqs, _ := models.GetAllFAQs()
	data["FAQs"] = faqs
	data["Title"] = "مدیریت سوالات متداول"
	renderAdmin(c, "faqs", data)
}

func AdminFAQNew(c *gin.Context) {
	data := adminData(c, "faqs")
	data["FAQ"] = nil
	data["Title"] = "افزودن سوال جدید"
	renderAdmin(c, "faq-form", data)
}

func AdminFAQEdit(c *gin.Context) {
	AdminFAQEditV2(c)
}

func AdminFAQSave(c *gin.Context) {
	AdminFAQSaveV2(c)
}

func AdminFAQDelete(c *gin.Context) {
	id, _ := strconv.Atoi(c.Param("id"))
	models.DeleteFAQ(id)
	c.Redirect(http.StatusFound, "/zed-admin/faqs")
}

// Blog Posts

func AdminPostsPage(c *gin.Context) {
	data := adminData(c, "posts")
	posts, _ := models.GetAllPosts()
	data["Posts"] = posts
	data["Title"] = "مدیریت مقالات"
	renderAdmin(c, "posts", data)
}

func AdminPostNew(c *gin.Context) {
	data := adminData(c, "posts")
	data["Post"] = nil
	data["Title"] = "نوشتن مقاله جدید"
	renderAdmin(c, "post-form", data)
}

func AdminPostEdit(c *gin.Context) {
	id, _ := strconv.Atoi(c.Param("id"))
	post, err := models.GetPostByID(id)
	if err != nil {
		c.Redirect(http.StatusFound, "/zed-admin/posts")
		return
	}
	data := adminData(c, "posts")
	data["Post"] = post
	data["Title"] = "ویرایش مقاله"
	renderAdmin(c, "post-form", data)
}

func AdminPostSave(c *gin.Context) {
	idStr := c.PostForm("id")
	isPublished := c.PostForm("is_published") == "1"
	post := models.BlogPost{
		Slug:            c.PostForm("slug"),
		Title:           c.PostForm("title"),
		MetaTitle:       c.PostForm("meta_title"),
		MetaDescription: c.PostForm("meta_description"),
		Excerpt:         c.PostForm("excerpt"),
		Content:         c.PostForm("content"),
		Image:           c.PostForm("image"),
		Category:        c.PostForm("category"),
		IsPublished:     isPublished,
	}
	if isPublished {
		post.PublishedAt.Valid = true
		post.PublishedAt.Time = time.Now()
	}
	var err error
	if idStr != "" && idStr != "0" {
		post.ID, _ = strconv.Atoi(idStr)
		err = models.UpdatePost(post)
	} else {
		err = models.CreatePost(post)
	}
	if err != nil {
		data := adminData(c, "posts")
		data["Error"] = err.Error()
		data["Post"] = post
		renderAdmin(c, "post-form", data)
		return
	}
	c.Redirect(http.StatusFound, "/zed-admin/posts")
}

func AdminPostDelete(c *gin.Context) {
	id, _ := strconv.Atoi(c.Param("id"))
	models.DeletePost(id)
	c.Redirect(http.StatusFound, "/zed-admin/posts")
}

// Tutorials

func AdminTutorialsPage(c *gin.Context) {
	data := adminData(c, "tutorials")
	tutorials, _ := models.GetAllTutorials()
	data["Tutorials"] = tutorials
	data["Title"] = "مدیریت آموزش‌ها"
	renderAdmin(c, "tutorials", data)
}

func AdminTutorialNew(c *gin.Context) {
	data := adminData(c, "tutorials")
	data["Tutorial"] = nil
	data["Title"] = "افزودن آموزش جدید"
	renderAdmin(c, "tutorial-form", data)
}

func AdminTutorialEdit(c *gin.Context) {
	id, _ := strconv.Atoi(c.Param("id"))
	t, err := models.GetTutorialByID(id)
	if err != nil {
		c.Redirect(http.StatusFound, "/zed-admin/tutorials")
		return
	}
	data := adminData(c, "tutorials")
	data["Tutorial"] = t
	data["Title"] = "ویرایش آموزش"
	renderAdmin(c, "tutorial-form", data)
}

func AdminTutorialSave(c *gin.Context) {
	AdminTutorialSaveV2(c)
}

func AdminTutorialDelete(c *gin.Context) {
	id, _ := strconv.Atoi(c.Param("id"))
	models.DeleteTutorial(id)
	c.Redirect(http.StatusFound, "/zed-admin/tutorials")
}

// Locations

func AdminLocationsPage(c *gin.Context) {
	data := adminData(c, "locations")
	locs, _ := models.GetAllLocations()
	data["Locations"] = locs
	data["Title"] = "مدیریت لوکیشن‌ها"
	renderAdmin(c, "locations", data)
}

func AdminLocationNew(c *gin.Context) {
	data := adminData(c, "locations")
	data["Location"] = nil
	data["Title"] = "افزودن لوکیشن جدید"
	renderAdmin(c, "location-form", data)
}

func AdminLocationEdit(c *gin.Context) {
	id, _ := strconv.Atoi(c.Param("id"))
	l, err := models.GetLocationByID(id)
	if err != nil {
		c.Redirect(http.StatusFound, "/zed-admin/locations")
		return
	}
	data := adminData(c, "locations")
	data["Location"] = l
	data["Title"] = "ویرایش لوکیشن"
	renderAdmin(c, "location-form", data)
}

func AdminLocationSave(c *gin.Context) {
	idStr := c.PostForm("id")
	sortOrder, _ := strconv.Atoi(c.PostForm("sort_order"))
	l := models.Location{
		Name:      c.PostForm("name"),
		Flag:      c.PostForm("flag"),
		Code:      c.PostForm("code"),
		Speed:     c.PostForm("speed"),
		IsActive:  c.PostForm("is_active") == "1",
		SortOrder: sortOrder,
	}
	var err error
	if idStr != "" && idStr != "0" {
		l.ID, _ = strconv.Atoi(idStr)
		err = models.UpdateLocation(l)
	} else {
		err = models.CreateLocation(l)
	}
	if err != nil {
		data := adminData(c, "locations")
		data["Error"] = err.Error()
		renderAdmin(c, "location-form", data)
		return
	}
	c.Redirect(http.StatusFound, "/zed-admin/locations")
}

func AdminLocationDelete(c *gin.Context) {
	id, _ := strconv.Atoi(c.Param("id"))
	models.DeleteLocation(id)
	c.Redirect(http.StatusFound, "/zed-admin/locations")
}

// Status Updates

func AdminStatusPage(c *gin.Context) {
	data := adminData(c, "status")
	updates, _ := models.GetAllStatusUpdates()
	data["Updates"] = updates
	data["Title"] = "مدیریت اطلاعیه‌ها"
	renderAdmin(c, "status", data)
}

func AdminStatusNew(c *gin.Context) {
	data := adminData(c, "status")
	data["Update"] = nil
	data["Title"] = "اطلاعیه جدید"
	renderAdmin(c, "status-form", data)
}

func AdminStatusEdit(c *gin.Context) {
	id, _ := strconv.Atoi(c.Param("id"))
	u, err := models.GetStatusUpdateByID(id)
	if err != nil {
		c.Redirect(http.StatusFound, "/zed-admin/status")
		return
	}
	data := adminData(c, "status")
	data["Update"] = u
	data["Title"] = "ویرایش اطلاعیه"
	renderAdmin(c, "status-form", data)
}

func AdminStatusSave(c *gin.Context) {
	idStr := c.PostForm("id")
	u := models.StatusUpdate{
		Title:   c.PostForm("title"),
		Content: c.PostForm("content"),
		Status:  c.PostForm("status"),
	}
	var err error
	if idStr != "" && idStr != "0" {
		u.ID, _ = strconv.Atoi(idStr)
		err = models.UpdateStatusUpdate(u)
	} else {
		err = models.CreateStatusUpdate(u)
	}
	if err != nil {
		data := adminData(c, "status")
		data["Error"] = err.Error()
		renderAdmin(c, "status-form", data)
		return
	}
	c.Redirect(http.StatusFound, "/zed-admin/status")
}

func AdminStatusDelete(c *gin.Context) {
	id, _ := strconv.Atoi(c.Param("id"))
	models.DeleteStatusUpdate(id)
	c.Redirect(http.StatusFound, "/zed-admin/status")
}

// Pages (Legal)

func AdminPagesPage(c *gin.Context) {
	data := adminData(c, "pages")
	pages, _ := models.GetAllPages()
	data["Pages"] = pages
	data["Title"] = "مدیریت صفحات"
	renderAdmin(c, "pages", data)
}

func AdminPageEdit(c *gin.Context) {
	slug := c.Param("slug")
	page, _ := models.GetPageBySlug(slug)
	if page == nil {
		page = &models.Page{Slug: slug}
	}
	data := adminData(c, "pages")
	data["Page"] = page
	data["Title"] = "ویرایش صفحه: " + slug
	renderAdmin(c, "page-form", data)
}

func AdminPageSave(c *gin.Context) {
	p := models.Page{
		Slug:            c.PostForm("slug"),
		Title:           c.PostForm("title"),
		Content:         c.PostForm("content"),
		MetaTitle:       c.PostForm("meta_title"),
		MetaDescription: c.PostForm("meta_description"),
	}
	if err := models.UpsertPage(p); err != nil {
		data := adminData(c, "pages")
		data["Error"] = err.Error()
		data["Page"] = p
		renderAdmin(c, "page-form", data)
		return
	}
	c.Redirect(http.StatusFound, "/zed-admin/pages")
}

// Media / File Upload

func AdminMediaPage(c *gin.Context) {
	data := adminData(c, "media")
	files, _ := models.GetAllUploadedFiles()
	data["Files"] = files
	data["Title"] = "مدیریت فایل‌ها"
	renderAdmin(c, "media", data)
}

func AdminMediaUpload(c *gin.Context) {
	file, header, err := c.Request.FormFile("file")
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "فایل انتخاب نشده"})
		return
	}
	defer file.Close()

	allowedMimes := map[string]bool{
		"image/jpeg": true, "image/png": true, "image/gif": true,
		"image/webp": true, "image/svg+xml": true,
	}
	contentType := header.Header.Get("Content-Type")
	if !allowedMimes[contentType] {
		c.JSON(http.StatusBadRequest, gin.H{"error": "نوع فایل مجاز نیست"})
		return
	}
	if header.Size > 5*1024*1024 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "حجم فایل بیش از ۵ مگابایت"})
		return
	}

	ext := filepath.Ext(header.Filename)
	filename := fmt.Sprintf("%d%s", time.Now().UnixNano(), ext)
	savePath := filepath.Join(uploadDir, filename)

	if err := os.MkdirAll(uploadDir, 0755); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "خطای سیستمی"})
		return
	}

	buf := make([]byte, header.Size)
	file.Read(buf)
	if err := os.WriteFile(savePath, buf, 0644); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "خطا در ذخیره فایل"})
		return
	}

	uf := models.UploadedFile{
		Filename:     filename,
		OriginalName: header.Filename,
		MimeType:     contentType,
		Size:         header.Size,
		Path:         "/uploads/" + filename,
	}
	models.CreateUploadedFile(uf)
	c.JSON(http.StatusOK, gin.H{"url": "/uploads/" + filename, "filename": filename})
}

func AdminMediaDelete(c *gin.Context) {
	id, _ := strconv.Atoi(c.Param("id"))
	f, err := models.DeleteUploadedFile(id)
	if err == nil && f != nil {
		os.Remove(filepath.Join(uploadDir, f.Filename))
	}
	c.Redirect(http.StatusFound, "/zed-admin/media")
}

// Change Password

func AdminPasswordPage(c *gin.Context) {
	data := adminData(c, "password")
	data["Title"] = "تغییر رمز عبور"
	renderAdmin(c, "password", data)
}

func AdminPasswordPost(c *gin.Context) {
	session := sessions.Default(c)
	adminID := session.Get("admin_id")
	usernameVal := session.Get("admin_username")
	if adminID == nil {
		c.Redirect(http.StatusFound, "/zed-admin/login")
		return
	}

	username, _ := usernameVal.(string)
	current := c.PostForm("current_password")
	newPass := c.PostForm("new_password")
	confirm := c.PostForm("confirm_password")

	if newPass != confirm {
		data := adminData(c, "password")
		data["Error"] = "رمز عبور جدید با تکرار آن مطابقت ندارد"
		data["Title"] = "تغییر رمز عبور"
		renderAdmin(c, "password", data)
		return
	}

	admin, err := models.GetAdminByUsername(username)
	if err != nil || bcrypt.CompareHashAndPassword([]byte(admin.PasswordHash), []byte(current)) != nil {
		data := adminData(c, "password")
		data["Error"] = "رمز عبور فعلی اشتباه است"
		data["Title"] = "تغییر رمز عبور"
		renderAdmin(c, "password", data)
		return
	}

	hash, _ := bcrypt.GenerateFromPassword([]byte(newPass), bcrypt.DefaultCost)
	models.UpdateAdminPassword(admin.ID, string(hash))
	tg.Send(tg.LevelWarn, tg.CatSecurity, "🔑 رمز عبور ادمین تغییر کرد", fmt.Sprintf("کاربر: %s", username))

	data := adminData(c, "password")
	data["Success"] = "رمز عبور با موفقیت تغییر یافت"
	data["Title"] = "تغییر رمز عبور"
	renderAdmin(c, "password", data)
}
