package handlers

import (
	"net/http"

	"github.com/gin-contrib/sessions"
	"github.com/gin-gonic/gin"
	"zedproxy/internal/models"
)

var appearanceDefaults = map[string]string{
	"admin_theme_name":           "zed-dark-neon",
	"admin_accent_color":         "#06b6d4",
	"admin_sidebar_mode":         "full",
	"admin_sidebar_width":        "normal",
	"admin_icon_size":            "medium",
	"admin_menu_text_size":       "medium",
	"admin_font_size":            "normal",
	"admin_card_radius":          "xl",
	"admin_card_shadow":          "soft",
	"admin_card_border":          "subtle",
	"admin_glass_effect_enabled": "1",
	"admin_animations_enabled":   "1",
	"admin_compact_mode_enabled": "0",
	"admin_dashboard_density":    "comfortable",
	"admin_custom_logo":          "",
	"admin_custom_background":    "",
}

type themePreset struct {
	ID          string
	Name        string
	AccentColor string
	Preview     string
}

var themePresets = []themePreset{
	{"zed-dark-neon", "Zed Dark Neon", "#06b6d4", "bg-slate-900 border-cyan-500"},
	{"cyber-blue", "Cyber Blue", "#3b82f6", "bg-blue-950 border-blue-400"},
	{"purple-glass", "Purple Glass", "#a855f7", "bg-purple-950 border-purple-400"},
	{"gold-premium", "Gold Premium", "#f59e0b", "bg-yellow-950 border-yellow-400"},
	{"emerald-dark", "Emerald Dark", "#10b981", "bg-emerald-950 border-emerald-400"},
	{"red-alert", "Red Alert", "#ef4444", "bg-red-950 border-red-400"},
	{"minimal-dark", "Minimal Dark", "#6b7280", "bg-gray-900 border-gray-500"},
	{"light-clean", "Light Clean", "#0ea5e9", "bg-white border-sky-400"},
}

func AdminAppearancePage(c *gin.Context) {
	sess := sessions.Default(c)
	role, _ := sess.Get("role").(string)
	if role != "owner" && role != "" {
		c.Redirect(http.StatusFound, "/zed-admin")
		return
	}

	data := adminData(c, "appearance")
	data["Title"] = "تنظیمات ظاهر پنل مدیریت"
	data["ThemePresets"] = themePresets

	settings := models.GetAllSettings()
	for k := range appearanceDefaults {
		data[k] = settings[k]
	}

	if f := sess.Get("flash_ok"); f != nil {
		data["FlashOK"] = f.(string)
		sess.Delete("flash_ok")
		sess.Save()
	}

	renderAdmin(c, "appearance", data)
}

func AdminAppearanceSave(c *gin.Context) {
	sess := sessions.Default(c)
	role, _ := sess.Get("role").(string)
	if role != "owner" && role != "" {
		c.Status(http.StatusForbidden)
		return
	}

	keys := []string{
		"admin_theme_name", "admin_accent_color", "admin_sidebar_mode",
		"admin_sidebar_width", "admin_icon_size", "admin_menu_text_size",
		"admin_font_size", "admin_card_radius", "admin_card_shadow",
		"admin_card_border", "admin_dashboard_density",
		"admin_custom_logo", "admin_custom_background",
	}
	for _, k := range keys {
		if v := c.PostForm(k); v != "" {
			models.SetSetting(k, v)
		}
	}

	checkboxes := []string{
		"admin_glass_effect_enabled", "admin_animations_enabled", "admin_compact_mode_enabled",
	}
	for _, k := range checkboxes {
		if c.PostForm(k) == "1" {
			models.SetSetting(k, "1")
		} else {
			models.SetSetting(k, "0")
		}
	}

	sess.Set("flash_ok", "تنظیمات ظاهر ذخیره شد")
	sess.Save()
	c.Redirect(http.StatusFound, "/zed-admin/settings/appearance")
}

func AdminAppearanceReset(c *gin.Context) {
	sess := sessions.Default(c)
	role, _ := sess.Get("role").(string)
	if role != "owner" && role != "" {
		c.Status(http.StatusForbidden)
		return
	}

	for k, v := range appearanceDefaults {
		models.SetSetting(k, v)
	}

	sess.Set("flash_ok", "تنظیمات ظاهر به حالت پیش‌فرض بازگشت")
	sess.Save()
	c.Redirect(http.StatusFound, "/zed-admin/settings/appearance")
}
