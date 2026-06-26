package handlers

import (
	"github.com/gin-contrib/sessions"
	"github.com/gin-gonic/gin"
	"zedproxy/internal/models"
)

var siteAppearanceKeys = []string{
	"site_theme_name", "site_accent_color", "site_background_color",
	"site_card_color", "site_text_color", "site_muted_text_color",
	"site_border_color", "site_button_color", "site_hover_color",
	"site_hero_style", "site_card_radius", "site_card_shadow",
	"site_glass_effect_enabled", "site_animations_enabled",
	"site_custom_logo", "site_custom_background",
}

func AdminSiteAppearancePage(c *gin.Context) {
	settings := models.GetAllSettings()
	data := adminData(c, "site-appearance")
	data["Title"] = "ظاهر سایت عمومی"
	data["Settings"] = settings
	renderAdmin(c, "site-appearance", data)
}

func AdminSiteAppearanceSave(c *gin.Context) {
	for _, k := range siteAppearanceKeys {
		models.SetSetting(k, c.PostForm(k))
	}
	LogAdminActivity(c, "site_appearance_save", "تنظیمات ظاهر سایت بروزرسانی شد")
	sess := sessions.Default(c)
	sess.AddFlash("تنظیمات ظاهر سایت ذخیره شد", "ok")
	sess.Save()
	c.Redirect(302, "/zed-admin/settings/site")
}
