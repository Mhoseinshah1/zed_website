package handlers

import (
	"database/sql"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/gin-contrib/sessions"
	"github.com/gin-gonic/gin"
	"golang.org/x/crypto/bcrypt"
	bkp "zedproxy/internal/backup"
	"zedproxy/internal/models"
	tg "zedproxy/internal/telegram"
)

// -- Announcements --

func AdminAnnouncementsPage(c *gin.Context) {
	data := adminData(c, "announcements")
	items, _ := models.GetAllAnnouncements()
	data["Announcements"] = items
	data["Title"] = "مدیریت اطلاعیه‌های بنر"
	renderAdmin(c, "announcements", data)
}

func AdminAnnouncementNew(c *gin.Context) {
	data := adminData(c, "announcements")
	data["Announcement"] = nil
	data["Title"] = "اطلاعیه جدید"
	renderAdmin(c, "announcement-form", data)
}

func AdminAnnouncementEdit(c *gin.Context) {
	id, _ := strconv.Atoi(c.Param("id"))
	a, err := models.GetAnnouncementByID(id)
	if err != nil {
		c.Redirect(http.StatusFound, "/zed-admin/announcements")
		return
	}
	data := adminData(c, "announcements")
	data["Announcement"] = a
	data["Title"] = "ویرایش اطلاعیه"
	renderAdmin(c, "announcement-form", data)
}

func AdminAnnouncementSave(c *gin.Context) {
	idStr := c.PostForm("id")
	sortOrder, _ := strconv.Atoi(c.PostForm("sort_order"))
	a := models.Announcement{
		Message:     c.PostForm("message"),
		Color:       c.PostForm("color"),
		IsClosable:  c.PostForm("is_closable") == "1",
		IsActive:    c.PostForm("is_active") == "1",
		TargetPages: c.PostForm("target_pages"),
		SortOrder:   sortOrder,
	}
	if v := c.PostForm("start_at"); v != "" {
		if t, err := time.Parse("2006-01-02T15:04", v); err == nil {
			a.StartAt = sql.NullTime{Time: t, Valid: true}
		}
	}
	if v := c.PostForm("end_at"); v != "" {
		if t, err := time.Parse("2006-01-02T15:04", v); err == nil {
			a.EndAt = sql.NullTime{Time: t, Valid: true}
		}
	}
	var err error
	if idStr != "" && idStr != "0" {
		a.ID, _ = strconv.Atoi(idStr)
		err = models.UpdateAnnouncement(a)
	} else {
		err = models.CreateAnnouncement(a)
	}
	if err != nil {
		data := adminData(c, "announcements")
		data["Error"] = err.Error()
		data["Announcement"] = a
		renderAdmin(c, "announcement-form", data)
		return
	}
	c.Redirect(http.StatusFound, "/zed-admin/announcements")
}

func AdminAnnouncementDelete(c *gin.Context) {
	id, _ := strconv.Atoi(c.Param("id"))
	models.DeleteAnnouncement(id)
	c.Redirect(http.StatusFound, "/zed-admin/announcements")
}

// -- Discount Codes --

func AdminDiscountCodesPage(c *gin.Context) {
	data := adminData(c, "discount-codes")
	codes, _ := models.GetAllDiscountCodes()
	data["Codes"] = codes
	data["Title"] = "مدیریت کدهای تخفیف"
	renderAdmin(c, "discount-codes", data)
}

func AdminDiscountCodeNew(c *gin.Context) {
	data := adminData(c, "discount-codes")
	data["Code"] = nil
	data["Title"] = "کد تخفیف جدید"
	renderAdmin(c, "discount-code-form", data)
}

func AdminDiscountCodeEdit(c *gin.Context) {
	id, _ := strconv.Atoi(c.Param("id"))
	d, err := models.GetDiscountCodeByID(id)
	if err != nil {
		c.Redirect(http.StatusFound, "/zed-admin/discount-codes")
		return
	}
	data := adminData(c, "discount-codes")
	data["Code"] = d
	data["Title"] = "ویرایش کد تخفیف"
	renderAdmin(c, "discount-code-form", data)
}

func AdminDiscountCodeSave(c *gin.Context) {
	idStr := c.PostForm("id")
	percent, _ := strconv.Atoi(c.PostForm("discount_percent"))
	d := models.DiscountCode{
		Code:            strings.ToUpper(strings.TrimSpace(c.PostForm("code"))),
		Description:     c.PostForm("description"),
		DiscountPercent: percent,
		IsActive:        c.PostForm("is_active") == "1",
	}
	if v := c.PostForm("expires_at"); v != "" {
		if t, err := time.Parse("2006-01-02T15:04", v); err == nil {
			d.ExpiresAt = sql.NullTime{Time: t, Valid: true}
		}
	}
	var err error
	if idStr != "" && idStr != "0" {
		d.ID, _ = strconv.Atoi(idStr)
		err = models.UpdateDiscountCode(d)
	} else {
		err = models.CreateDiscountCode(d)
	}
	if err != nil {
		data := adminData(c, "discount-codes")
		data["Error"] = err.Error()
		data["Code"] = d
		renderAdmin(c, "discount-code-form", data)
		return
	}
	c.Redirect(http.StatusFound, "/zed-admin/discount-codes")
}

func AdminDiscountCodeDelete(c *gin.Context) {
	id, _ := strconv.Atoi(c.Param("id"))
	models.DeleteDiscountCode(id)
	c.Redirect(http.StatusFound, "/zed-admin/discount-codes")
}

// -- Analytics --

func AdminAnalyticsPage(c *gin.Context) {
	data := adminData(c, "analytics")
	analytics := models.GetClickAnalytics()
	data["Analytics"] = analytics
	data["Title"] = "آمار و تحلیل کلیک‌ها"
	renderAdmin(c, "analytics", data)
}

// -- Status Items --

func AdminStatusItemsPage(c *gin.Context) {
	data := adminData(c, "status-items")
	items, _ := models.GetAllStatusItems()
	data["Items"] = items
	data["Title"] = "وضعیت سرویس‌ها"
	renderAdmin(c, "status-items", data)
}

func AdminStatusItemNew(c *gin.Context) {
	data := adminData(c, "status-items")
	data["Item"] = nil
	data["Title"] = "سرویس جدید"
	renderAdmin(c, "status-item-form", data)
}

func AdminStatusItemEdit(c *gin.Context) {
	id, _ := strconv.Atoi(c.Param("id"))
	s, err := models.GetStatusItemByID(id)
	if err != nil {
		c.Redirect(http.StatusFound, "/zed-admin/status-items")
		return
	}
	data := adminData(c, "status-items")
	data["Item"] = s
	data["Title"] = "ویرایش وضعیت سرویس"
	renderAdmin(c, "status-item-form", data)
}

func AdminStatusItemSave(c *gin.Context) {
	idStr := c.PostForm("id")
	sortOrder, _ := strconv.Atoi(c.PostForm("sort_order"))
	s := models.StatusItem{
		Name:        c.PostForm("name"),
		ServiceType: c.PostForm("service_type"),
		Status:      c.PostForm("status"),
		Description: c.PostForm("description"),
		SortOrder:   sortOrder,
		IsActive:    c.PostForm("is_active") == "1",
	}
	var err error
	if idStr != "" && idStr != "0" {
		s.ID, _ = strconv.Atoi(idStr)
		err = models.UpdateStatusItem(s)
	} else {
		err = models.CreateStatusItem(s)
	}
	if err != nil {
		data := adminData(c, "status-items")
		data["Error"] = err.Error()
		renderAdmin(c, "status-item-form", data)
		return
	}
	c.Redirect(http.StatusFound, "/zed-admin/status-items")
}

func AdminStatusItemDelete(c *gin.Context) {
	id, _ := strconv.Atoi(c.Param("id"))
	models.DeleteStatusItem(id)
	c.Redirect(http.StatusFound, "/zed-admin/status-items")
}

// -- Trust Cards --

func AdminTrustCardsPage(c *gin.Context) {
	data := adminData(c, "trust-cards")
	cards, _ := models.GetAllTrustCards()
	data["Cards"] = cards
	data["Title"] = "مدیریت کارت‌های اعتماد"
	renderAdmin(c, "trust-cards", data)
}

func AdminTrustCardNew(c *gin.Context) {
	data := adminData(c, "trust-cards")
	data["Card"] = nil
	data["Title"] = "کارت اعتماد جدید"
	renderAdmin(c, "trust-card-form", data)
}

func AdminTrustCardEdit(c *gin.Context) {
	id, _ := strconv.Atoi(c.Param("id"))
	t, err := models.GetTrustCardByID(id)
	if err != nil {
		c.Redirect(http.StatusFound, "/zed-admin/trust-cards")
		return
	}
	data := adminData(c, "trust-cards")
	data["Card"] = t
	data["Title"] = "ویرایش کارت اعتماد"
	renderAdmin(c, "trust-card-form", data)
}

func AdminTrustCardSave(c *gin.Context) {
	idStr := c.PostForm("id")
	sortOrder, _ := strconv.Atoi(c.PostForm("sort_order"))
	t := models.TrustCard{
		Icon:        c.PostForm("icon"),
		Title:       c.PostForm("title"),
		Description: c.PostForm("description"),
		SortOrder:   sortOrder,
		IsActive:    c.PostForm("is_active") == "1",
	}
	var err error
	if idStr != "" && idStr != "0" {
		t.ID, _ = strconv.Atoi(idStr)
		err = models.UpdateTrustCard(t)
	} else {
		err = models.CreateTrustCard(t)
	}
	if err != nil {
		data := adminData(c, "trust-cards")
		data["Error"] = err.Error()
		renderAdmin(c, "trust-card-form", data)
		return
	}
	c.Redirect(http.StatusFound, "/zed-admin/trust-cards")
}

func AdminTrustCardDelete(c *gin.Context) {
	id, _ := strconv.Atoi(c.Param("id"))
	models.DeleteTrustCard(id)
	c.Redirect(http.StatusFound, "/zed-admin/trust-cards")
}

// -- Plan Comparison --

func AdminPlanComparisonPage(c *gin.Context) {
	data := adminData(c, "plan-comparison")
	items, _ := models.GetAllPlanComparisons()
	data["Comparisons"] = items
	data["Title"] = "جدول مقایسه پلن‌ها"
	renderAdmin(c, "plan-comparison", data)
}

func AdminPlanComparisonNew(c *gin.Context) {
	data := adminData(c, "plan-comparison")
	data["Comparison"] = nil
	data["Title"] = "ردیف جدید مقایسه"
	renderAdmin(c, "plan-comparison-form", data)
}

func AdminPlanComparisonEdit(c *gin.Context) {
	id, _ := strconv.Atoi(c.Param("id"))
	p, err := models.GetPlanComparisonByID(id)
	if err != nil {
		c.Redirect(http.StatusFound, "/zed-admin/plan-comparison")
		return
	}
	data := adminData(c, "plan-comparison")
	data["Comparison"] = p
	data["Title"] = "ویرایش ردیف مقایسه"
	renderAdmin(c, "plan-comparison-form", data)
}

func AdminPlanComparisonSave(c *gin.Context) {
	idStr := c.PostForm("id")
	sortOrder, _ := strconv.Atoi(c.PostForm("sort_order"))
	p := models.PlanComparison{
		FeatureName:   c.PostForm("feature_name"),
		BronzeValue:   c.PostForm("bronze_value"),
		SilverValue:   c.PostForm("silver_value"),
		GoldValue:     c.PostForm("gold_value"),
		PlatinumValue: c.PostForm("platinum_value"),
		SortOrder:     sortOrder,
	}
	var err error
	if idStr != "" && idStr != "0" {
		p.ID, _ = strconv.Atoi(idStr)
		err = models.UpdatePlanComparison(p)
	} else {
		err = models.CreatePlanComparison(p)
	}
	if err != nil {
		data := adminData(c, "plan-comparison")
		data["Error"] = err.Error()
		renderAdmin(c, "plan-comparison-form", data)
		return
	}
	c.Redirect(http.StatusFound, "/zed-admin/plan-comparison")
}

func AdminPlanComparisonDelete(c *gin.Context) {
	id, _ := strconv.Atoi(c.Param("id"))
	models.DeletePlanComparison(id)
	c.Redirect(http.StatusFound, "/zed-admin/plan-comparison")
}

// -- Homepage Sections --

func AdminHomepageSectionsPage(c *gin.Context) {
	data := adminData(c, "homepage-sections")
	sections, _ := models.GetHomepageSections()
	data["Sections"] = sections
	data["Title"] = "مدیریت بخش‌های صفحه اصلی"
	renderAdmin(c, "homepage-sections", data)
}

func AdminHomepageSectionsSave(c *gin.Context) {
	sections, _ := models.GetHomepageSections()
	for _, s := range sections {
		s.IsActive = c.PostForm("active_"+s.SectionKey) == "1"
		sortStr := c.PostForm("sort_" + s.SectionKey)
		if n, err := strconv.Atoi(sortStr); err == nil {
			s.SortOrder = n
		}
		s.Title = c.PostForm("title_" + s.SectionKey)
		models.UpdateHomepageSection(s)
	}
	c.Redirect(http.StatusFound, "/zed-admin/homepage-sections")
}

// -- Campaigns --

func AdminCampaignsPage(c *gin.Context) {
	data := adminData(c, "campaigns")
	campaigns, _ := models.GetAllCampaigns()
	data["Campaigns"] = campaigns
	data["Title"] = "مدیریت کمپین‌ها"
	renderAdmin(c, "campaigns", data)
}

func AdminCampaignNew(c *gin.Context) {
	data := adminData(c, "campaigns")
	data["Campaign"] = nil
	data["Title"] = "کمپین جدید"
	renderAdmin(c, "campaign-form", data)
}

func AdminCampaignEdit(c *gin.Context) {
	id, _ := strconv.Atoi(c.Param("id"))
	camp, err := models.GetCampaignByID(id)
	if err != nil {
		c.Redirect(http.StatusFound, "/zed-admin/campaigns")
		return
	}
	data := adminData(c, "campaigns")
	data["Campaign"] = camp
	data["Title"] = "ویرایش کمپین"
	renderAdmin(c, "campaign-form", data)
}

func AdminCampaignSave(c *gin.Context) {
	idStr := c.PostForm("id")
	percent, _ := strconv.Atoi(c.PostForm("discount_percent"))
	camp := models.Campaign{
		Slug:            c.PostForm("slug"),
		Title:           c.PostForm("title"),
		Subtitle:        c.PostForm("subtitle"),
		Description:     c.PostForm("description"),
		DiscountCode:    strings.ToUpper(strings.TrimSpace(c.PostForm("discount_code"))),
		DiscountPercent: percent,
		CTAText:         c.PostForm("cta_text"),
		Image:           c.PostForm("image"),
		MetaTitle:       c.PostForm("meta_title"),
		MetaDescription: c.PostForm("meta_description"),
		IsActive:        c.PostForm("is_active") == "1",
	}
	if v := c.PostForm("countdown_at"); v != "" {
		if t, err := time.Parse("2006-01-02T15:04", v); err == nil {
			camp.CountdownAt = sql.NullTime{Time: t, Valid: true}
		}
	}
	var err error
	if idStr != "" && idStr != "0" {
		camp.ID, _ = strconv.Atoi(idStr)
		err = models.UpdateCampaign(camp)
	} else {
		err = models.CreateCampaign(camp)
	}
	if err != nil {
		data := adminData(c, "campaigns")
		data["Error"] = err.Error()
		data["Campaign"] = camp
		renderAdmin(c, "campaign-form", data)
		return
	}
	c.Redirect(http.StatusFound, "/zed-admin/campaigns")
}

func AdminCampaignDelete(c *gin.Context) {
	id, _ := strconv.Atoi(c.Param("id"))
	models.DeleteCampaign(id)
	c.Redirect(http.StatusFound, "/zed-admin/campaigns")
}

// -- Landing Pages --

func AdminLandingPagesPage(c *gin.Context) {
	data := adminData(c, "landing-pages")
	pages, _ := models.GetAllLandingPages()
	data["Pages"] = pages
	data["Title"] = "صفحات فرود SEO"
	renderAdmin(c, "landing-pages", data)
}

func AdminLandingPageNew(c *gin.Context) {
	data := adminData(c, "landing-pages")
	data["Page"] = nil
	data["Title"] = "صفحه فرود جدید"
	renderAdmin(c, "landing-page-form", data)
}

func AdminLandingPageEdit(c *gin.Context) {
	id, _ := strconv.Atoi(c.Param("id"))
	page, err := models.GetLandingPageByID(id)
	if err != nil {
		c.Redirect(http.StatusFound, "/zed-admin/landing-pages")
		return
	}
	data := adminData(c, "landing-pages")
	data["Page"] = page
	data["Title"] = "ویرایش صفحه فرود"
	renderAdmin(c, "landing-page-form", data)
}

func AdminLandingPageSave(c *gin.Context) {
	idStr := c.PostForm("id")
	p := models.LandingPage{
		Slug:            c.PostForm("slug"),
		Title:           c.PostForm("title"),
		HeroTitle:       c.PostForm("hero_title"),
		HeroSubtitle:    c.PostForm("hero_subtitle"),
		Content:         c.PostForm("content"),
		CTAText:         c.PostForm("cta_text"),
		FeaturedImage:   c.PostForm("featured_image"),
		MetaTitle:       c.PostForm("meta_title"),
		MetaDescription: c.PostForm("meta_description"),
		OGImage:         c.PostForm("og_image"),
		NoIndex:         c.PostForm("noindex") == "1",
		IsActive:        c.PostForm("is_active") == "1",
	}
	var err error
	if idStr != "" && idStr != "0" {
		p.ID, _ = strconv.Atoi(idStr)
		err = models.UpdateLandingPage(p)
	} else {
		err = models.CreateLandingPage(p)
	}
	if err != nil {
		data := adminData(c, "landing-pages")
		data["Error"] = err.Error()
		data["Page"] = p
		renderAdmin(c, "landing-page-form", data)
		return
	}
	c.Redirect(http.StatusFound, "/zed-admin/landing-pages")
}

func AdminLandingPageDelete(c *gin.Context) {
	id, _ := strconv.Atoi(c.Param("id"))
	models.DeleteLandingPage(id)
	c.Redirect(http.StatusFound, "/zed-admin/landing-pages")
}

// -- Popups --

func AdminPopupsPage(c *gin.Context) {
	data := adminData(c, "popups")
	popups, _ := models.GetAllPopups()
	data["Popups"] = popups
	data["Title"] = "مدیریت پاپ‌آپ‌ها"
	renderAdmin(c, "popups", data)
}

func AdminPopupNew(c *gin.Context) {
	data := adminData(c, "popups")
	data["Popup"] = nil
	data["Title"] = "پاپ‌آپ جدید"
	renderAdmin(c, "popup-form", data)
}

func AdminPopupEdit(c *gin.Context) {
	id, _ := strconv.Atoi(c.Param("id"))
	p, err := models.GetPopupByID(id)
	if err != nil {
		c.Redirect(http.StatusFound, "/zed-admin/popups")
		return
	}
	data := adminData(c, "popups")
	data["Popup"] = p
	data["Title"] = "ویرایش پاپ‌آپ"
	renderAdmin(c, "popup-form", data)
}

func AdminPopupSave(c *gin.Context) {
	idStr := c.PostForm("id")
	delay, _ := strconv.Atoi(c.PostForm("show_after_seconds"))
	p := models.Popup{
		Title:            c.PostForm("title"),
		Message:          c.PostForm("message"),
		CTAText:          c.PostForm("cta_text"),
		ShowAfterSeconds: delay,
		ExitIntent:       c.PostForm("exit_intent") == "1",
		OncePerSession:   c.PostForm("once_per_session") == "1",
		TargetPages:      c.PostForm("target_pages"),
		IsActive:         c.PostForm("is_active") == "1",
	}
	var err error
	if idStr != "" && idStr != "0" {
		p.ID, _ = strconv.Atoi(idStr)
		err = models.UpdatePopup(p)
	} else {
		err = models.CreatePopup(p)
	}
	if err != nil {
		data := adminData(c, "popups")
		data["Error"] = err.Error()
		renderAdmin(c, "popup-form", data)
		return
	}
	c.Redirect(http.StatusFound, "/zed-admin/popups")
}

func AdminPopupDelete(c *gin.Context) {
	id, _ := strconv.Atoi(c.Param("id"))
	models.DeletePopup(id)
	c.Redirect(http.StatusFound, "/zed-admin/popups")
}

// -- Admin Users --

func AdminUsersPage(c *gin.Context) {
	session := sessions.Default(c)
	adminID := session.Get("admin_id")
	currentAdmin, _ := models.GetAdminByID(adminID.(int))
	if currentAdmin == nil || currentAdmin.Role != "owner" {
		c.Redirect(http.StatusFound, "/zed-admin")
		return
	}
	data := adminData(c, "users")
	admins, _ := models.GetAllAdmins()
	data["Admins"] = admins
	data["CurrentAdmin"] = currentAdmin
	data["Title"] = "مدیریت کاربران"
	renderAdmin(c, "users", data)
}

func AdminUserNew(c *gin.Context) {
	session := sessions.Default(c)
	adminID := session.Get("admin_id")
	currentAdmin, _ := models.GetAdminByID(adminID.(int))
	if currentAdmin == nil || currentAdmin.Role != "owner" {
		c.Redirect(http.StatusFound, "/zed-admin")
		return
	}
	data := adminData(c, "users")
	data["AdminUser"] = nil
	data["Title"] = "کاربر جدید"
	renderAdmin(c, "user-form", data)
}

func AdminUserSave(c *gin.Context) {
	session := sessions.Default(c)
	adminID := session.Get("admin_id")
	currentAdmin, _ := models.GetAdminByID(adminID.(int))
	if currentAdmin == nil || currentAdmin.Role != "owner" {
		c.Redirect(http.StatusFound, "/zed-admin")
		return
	}

	idStr := c.PostForm("id")
	username := c.PostForm("username")
	email := c.PostForm("email")
	role := c.PostForm("role")
	password := c.PostForm("password")

	if idStr != "" && idStr != "0" {
		id, _ := strconv.Atoi(idStr)
		models.UpdateAdminRole(id, role)
		if password != "" {
			hash, _ := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
			models.UpdateAdminPassword(id, string(hash))
		}
	} else {
		if password == "" {
			data := adminData(c, "users")
			data["Error"] = "رمز عبور الزامی است"
			renderAdmin(c, "user-form", data)
			return
		}
		hash, _ := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
		if err := models.CreateAdminWithRole(username, email, string(hash), role); err != nil {
			data := adminData(c, "users")
			data["Error"] = err.Error()
			renderAdmin(c, "user-form", data)
			return
		}
		tg.Send(tg.LevelWarn, tg.CatAdminActivity, "🧾 ادمین جدید ایجاد شد", fmt.Sprintf("نام کاربری: %s\nنقش: %s", username, role))
	}
	c.Redirect(http.StatusFound, "/zed-admin/users")
}

func AdminUserDelete(c *gin.Context) {
	session := sessions.Default(c)
	adminID := session.Get("admin_id")
	currentAdmin, _ := models.GetAdminByID(adminID.(int))
	if currentAdmin == nil || currentAdmin.Role != "owner" {
		c.Redirect(http.StatusFound, "/zed-admin")
		return
	}
	id, _ := strconv.Atoi(c.Param("id"))
	if id != currentAdmin.ID {
		models.DeleteAdmin(id)
	}
	c.Redirect(http.StatusFound, "/zed-admin/users")
}

// -- DB Backup --

var backupDir = "/opt/zedproxy/backups"

func SetBackupDir(dir string) {
	backupDir = dir
}

func AdminBackupsPage(c *gin.Context) {
	data := adminData(c, "backups")
	backups, _ := models.GetAllBackups()
	data["Backups"] = backups
	data["Title"] = "پشتیبان‌گیری از پایگاه داده"
	renderAdmin(c, "backups", data)
}

func AdminBackupCreate(c *gin.Context) {
	dbp := resolveDBPath()
	zipData, filename, err := bkp.CreateDBZip(dbp)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "خطا در ایجاد بکاپ: " + err.Error()})
		return
	}

	savedPath, err := bkp.SaveZipToDir(zipData, backupDir, filename)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "خطا در ذخیره بکاپ: " + err.Error()})
		return
	}
	_ = savedPath

	models.RecordBackup(filename, int64(len(zipData)))
	tg.Send(tg.LevelInfo, tg.CatBackups, "💾 بکاپ ZIP ایجاد شد",
		fmt.Sprintf("فایل: %s\nحجم: %d بایت", filename, len(zipData)))

	// Optionally send to Telegram
	if models.GetSetting("telegram_admin_send_db_zip_enabled") == "1" {
		go func() {
			caption := fmt.Sprintf("💾 بکاپ دیتابیس ZedProxy\nتاریخ: %s", time.Now().Format("2006/01/02 15:04"))
			if err := tg.SendBackupToTelegram(zipData, filename, caption); err != nil {
				tg.Send(tg.LevelWarn, tg.CatBackups, "⚠️ ارسال بکاپ به تلگرام ناموفق", err.Error())
			}
		}()
	}

	c.Redirect(http.StatusFound, "/zed-admin/backups")
}

func AdminBackupDownload(c *gin.Context) {
	// Only owner can download backups
	sess := sessions.Default(c)
	role, _ := sess.Get("role").(string)
	if role != "owner" && role != "" {
		c.Status(http.StatusForbidden)
		return
	}

	id, _ := strconv.Atoi(c.Param("id"))
	// Validate id is positive to prevent manipulation
	if id <= 0 {
		c.Status(http.StatusBadRequest)
		return
	}
	backups, _ := models.GetAllBackups()
	for _, b := range backups {
		if b.ID == id {
			// Sanitize filename to prevent path traversal
			cleanName := filepath.Base(b.Filename)
			if cleanName == "." || cleanName == "/" {
				c.Status(http.StatusBadRequest)
				return
			}
			path := filepath.Join(backupDir, cleanName)
			// Ensure the resolved path is inside backupDir
			absBackup, _ := filepath.Abs(backupDir)
			absPath, _ := filepath.Abs(path)
			if len(absPath) < len(absBackup) || absPath[:len(absBackup)] != absBackup {
				c.Status(http.StatusForbidden)
				return
			}
			c.FileAttachment(path, cleanName)
			return
		}
	}
	c.Status(http.StatusNotFound)
}

func AdminBackupDelete(c *gin.Context) {
	id, _ := strconv.Atoi(c.Param("id"))
	backups, _ := models.GetAllBackups()
	for _, b := range backups {
		if b.ID == id {
			os.Remove(filepath.Join(backupDir, b.Filename))
			models.DeleteBackupRecord(id)
			break
		}
	}
	c.Redirect(http.StatusFound, "/zed-admin/backups")
}

// -- Maintenance Mode --

func AdminMaintenancePage(c *gin.Context) {
	data := adminData(c, "maintenance")
	data["Title"] = "حالت تعمیر و نگهداری"
	data["MaintenanceMode"] = models.GetSetting("maintenance_enabled")
	data["MaintenanceMsg"] = models.GetSetting("maintenance_msg")
	renderAdmin(c, "maintenance", data)
}

func AdminMaintenanceSave(c *gin.Context) {
	mode := "0"
	if c.PostForm("maintenance_mode") == "1" {
		mode = "1"
	}
	prev := models.GetSetting("maintenance_enabled")
	models.SetSetting("maintenance_enabled", mode)
	models.SetSetting("maintenance_msg", c.PostForm("maintenance_msg"))
	if mode != prev {
		if mode == "1" {
			tg.Send(tg.LevelWarn, tg.CatMaintenance, "🧰 حالت تعمیرات فعال شد", "سایت در حالت تعمیر قرار گرفت.")
		} else {
			tg.Send(tg.LevelInfo, tg.CatMaintenance, "✅ حالت تعمیرات غیرفعال شد", "سایت به حالت عادی بازگشت.")
		}
	}
	c.Redirect(http.StatusFound, "/zed-admin/maintenance")
}

// -- System: Logs --

func AdminSystemLogsPage(c *gin.Context) {
	data := adminData(c, "system-logs")
	data["Title"] = "لاگ‌های سیستم"
	logs, _ := models.GetSystemLogs(200)
	data["Logs"] = logs
	renderAdmin(c, "system-logs", data)
}

// -- System: Health --

func AdminSystemHealthPage(c *gin.Context) {
	data := adminData(c, "system-health")
	data["Title"] = "سلامت سیستم"
	data["MaintenanceEnabled"] = models.GetSetting("maintenance_enabled") == "1"
	data["AppVersion"] = AppVersion
	renderAdmin(c, "system-health", data)
}

// -- Media Alt Text Update --

func AdminMediaUpdateAlt(c *gin.Context) {
	id, _ := strconv.Atoi(c.Param("id"))
	alt := c.PostForm("alt_text")
	models.UpdateUploadedFileAlt(id, alt)
	c.Redirect(http.StatusFound, "/zed-admin/media")
}

// -- Tutorial Save (updated with new fields) --

func AdminTutorialSaveV2(c *gin.Context) {
	idStr := c.PostForm("id")
	sortOrder, _ := strconv.Atoi(c.PostForm("sort_order"))
	t := models.Tutorial{
		Slug:            c.PostForm("slug"),
		Title:           c.PostForm("title"),
		Excerpt:         c.PostForm("excerpt"),
		Content:         c.PostForm("content"),
		Image:           c.PostForm("image"),
		Category:        c.PostForm("category"),
		Platform:        c.PostForm("platform"),
		VideoURL:        c.PostForm("video_url"),
		MetaTitle:       c.PostForm("meta_title"),
		MetaDescription: c.PostForm("meta_description"),
		SortOrder:       sortOrder,
		IsPublished:     c.PostForm("is_published") == "1",
	}
	var err error
	if idStr != "" && idStr != "0" {
		t.ID, _ = strconv.Atoi(idStr)
		err = models.UpdateTutorial(t)
	} else {
		err = models.CreateTutorial(t)
	}
	if err != nil {
		data := adminData(c, "tutorials")
		data["Error"] = err.Error()
		data["Tutorial"] = t
		renderAdmin(c, "tutorial-form", data)
		return
	}
	c.Redirect(http.StatusFound, "/zed-admin/tutorials")
}

// -- FAQ Save (updated with new flags) --

func AdminFAQSaveV2(c *gin.Context) {
	idStr := c.PostForm("id")
	sortOrder, _ := strconv.Atoi(c.PostForm("sort_order"))
	f := models.FAQ{
		Question:       c.PostForm("question"),
		Answer:         c.PostForm("answer"),
		Category:       c.PostForm("category"),
		SortOrder:      sortOrder,
		IsActive:       c.PostForm("is_active") == "1",
		ShowOnHomepage: c.PostForm("show_on_homepage") == "1",
		ShowOnFAQ:      c.PostForm("show_on_faq") == "1",
	}
	var err error
	if idStr != "" && idStr != "0" {
		f.ID, _ = strconv.Atoi(idStr)
		err = models.UpdateFAQ(f)
	} else {
		err = models.CreateFAQ(f)
	}
	if err != nil {
		data := adminData(c, "faqs")
		data["Error"] = err.Error()
		renderAdmin(c, "faq-form", data)
		return
	}
	c.Redirect(http.StatusFound, "/zed-admin/faqs")
}

// AdminSettingsPostV2 saves extended settings fields
func AdminSettingsPostV2(c *gin.Context) {
	keys := []string{
		"site_name", "site_tagline", "site_url", "logo_text", "logo_image", "favicon", "og_image", "hero_image",
		"hero_title", "hero_subtitle", "hero_cta_text", "hero_secondary",
		"telegram_bot", "telegram_channel", "telegram_support",
		"btn_buy_text", "btn_support_text", "btn_channel_text",
		"seo_title", "seo_description", "gsc_verification",
		"trust_count_users", "trust_uptime", "trust_speed", "trust_support", "footer_text",
		"google_analytics", "custom_js", "custom_css",
		"primary_color", "secondary_color", "accent_color", "bg_style",
		"maintenance_mode", "maintenance_msg", "robots_txt_extra",
	}
	for _, k := range keys {
		v := c.PostForm(k)
		models.SetSetting(k, v)
	}
	session := sessions.Default(c)
	session.AddFlash("saved", "ok")
	session.Save()
	c.Redirect(http.StatusFound, "/zed-admin/settings")
}

// AdminFAQEdit with new fields
func AdminFAQEditV2(c *gin.Context) {
	id, _ := strconv.Atoi(c.Param("id"))
	f, err := models.GetFAQByID(id)
	if err != nil {
		c.Redirect(http.StatusFound, "/zed-admin/faqs")
		return
	}
	data := adminData(c, "faqs")
	data["FAQ"] = f
	data["Title"] = "ویرایش سوال"
	renderAdmin(c, "faq-form", data)
}

// Helper: get db path from settings (set by main.go)
var currentDBPath string

func SetDBPath(path string) {
	currentDBPath = path
	models.SetSetting("_db_path", path)
}

func resolveDBPath() string {
	if currentDBPath != "" {
		return currentDBPath
	}
	if p := models.GetSetting("_db_path"); p != "" {
		return p
	}
	for _, p := range []string{"./data/zedproxy.db", "/opt/zedproxy/data/zedproxy.db"} {
		if _, err := os.Stat(p); err == nil {
			return p
		}
	}
	return "./data/zedproxy.db"
}
