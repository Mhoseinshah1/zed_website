package handlers

import (
	"net/http"

	"github.com/gin-contrib/sessions"
	"github.com/gin-gonic/gin"
	"zedproxy/internal/models"
)

var adminAppearanceDefaults = map[string]string{
	"admin_theme_name":           "zed-dark-neon",
	"admin_accent_color":         "#06b6d4",
	"admin_background_color":     "#0d0d16",
	"admin_sidebar_color":        "#0f0f1a",
	"admin_card_color":           "#1a1a2e",
	"admin_text_color":           "#f1f5f9",
	"admin_muted_text_color":     "#94a3b8",
	"admin_border_color":         "rgba(255,255,255,0.1)",
	"admin_button_color":         "#06b6d4",
	"admin_hover_color":          "rgba(255,255,255,0.07)",
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

var siteAppearanceDefaults = map[string]string{
	"site_theme_name":           "default",
	"site_accent_color":         "#6366f1",
	"site_background_color":     "#0a0a0f",
	"site_card_color":           "rgba(255,255,255,0.05)",
	"site_text_color":           "#e2e8f0",
	"site_muted_text_color":     "#94a3b8",
	"site_border_color":         "rgba(255,255,255,0.1)",
	"site_button_color":         "#6366f1",
	"site_hover_color":          "rgba(255,255,255,0.05)",
	"site_hero_style":           "gradient",
	"site_card_radius":          "xl",
	"site_card_shadow":          "medium",
	"site_glass_effect_enabled": "1",
	"site_animations_enabled":   "1",
	"site_custom_logo":          "",
	"site_custom_background":    "",
}

type themePreset struct {
	ID          string
	Name        string
	AccentColor string
	BGColor     string
	SidebarColor string
	CardColor   string
}

var adminThemePresets = []themePreset{
	{"zed-dark-neon", "Zed Dark Neon", "#06b6d4", "#0d0d16", "#0f0f1a", "#1a1a2e"},
	{"cyber-blue", "Cyber Blue", "#3b82f6", "#050d1a", "#071220", "#0c1f35"},
	{"purple-glass", "Purple Glass", "#a855f7", "#0d0514", "#12081a", "#1a0d28"},
	{"gold-premium", "Gold Premium", "#f59e0b", "#120e00", "#1a1400", "#261e00"},
	{"emerald-dark", "Emerald Dark", "#10b981", "#011a10", "#021f13", "#03301d"},
	{"red-alert", "Red Alert", "#ef4444", "#1a0505", "#200808", "#300d0d"},
	{"minimal-dark", "Minimal Dark", "#6b7280", "#111827", "#1f2937", "#374151"},
	{"light-clean", "Light Clean", "#0ea5e9", "#f8fafc", "#f1f5f9", "#ffffff"},
}

var siteThemePresets = []themePreset{
	{"default", "پیش‌فرض (بنفش)", "#6366f1", "#0a0a0f", "", "rgba(255,255,255,0.05)"},
	{"cyber-cyan", "Cyber Cyan", "#06b6d4", "#050d14", "", "rgba(255,255,255,0.05)"},
	{"purple-neon", "Purple Neon", "#a855f7", "#0d0514", "", "rgba(255,255,255,0.05)"},
	{"gold-luxury", "Gold Luxury", "#f59e0b", "#100e00", "", "rgba(255,255,255,0.05)"},
	{"emerald", "Emerald", "#10b981", "#011810", "", "rgba(255,255,255,0.05)"},
	{"rose", "Rose", "#f43f5e", "#180508", "", "rgba(255,255,255,0.05)"},
}

func AdminAppearancePage(c *gin.Context) {
	sess := sessions.Default(c)
	role, _ := sess.Get("role").(string)
	if role != "owner" && role != "" {
		c.Redirect(http.StatusFound, "/zed-admin")
		return
	}

	data := adminData(c, "appearance")
	data["Title"] = "تنظیمات ظاهر"
	data["AdminThemePresets"] = adminThemePresets
	data["SiteThemePresets"] = siteThemePresets

	settings := models.GetAllSettings()
	for k := range adminAppearanceDefaults {
		if v := settings[k]; v != "" {
			data[k] = v
		} else {
			data[k] = adminAppearanceDefaults[k]
		}
	}
	for k := range siteAppearanceDefaults {
		if v := settings[k]; v != "" {
			data[k] = v
		} else {
			data[k] = siteAppearanceDefaults[k]
		}
	}

	if f := sess.Get("flash_ok"); f != nil {
		data["FlashOK"] = f.(string)
		sess.Delete("flash_ok")
		sess.Save()
	}
	if f := sess.Get("flash_err"); f != nil {
		data["FlashErr"] = f.(string)
		sess.Delete("flash_err")
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

	tab := c.PostForm("tab")

	if tab == "site" {
		// Save public site appearance
		siteKeys := []string{
			"site_theme_name", "site_accent_color", "site_background_color",
			"site_card_color", "site_text_color", "site_muted_text_color",
			"site_border_color", "site_button_color", "site_hover_color",
			"site_hero_style", "site_card_radius", "site_card_shadow",
			"site_custom_logo", "site_custom_background",
		}
		for _, k := range siteKeys {
			v := c.PostForm(k)
			// Allow empty string for custom_logo and custom_background
			models.SetSetting(k, v)
		}
		for _, k := range []string{"site_glass_effect_enabled", "site_animations_enabled"} {
			if c.PostForm(k) == "1" {
				models.SetSetting(k, "1")
			} else {
				models.SetSetting(k, "0")
			}
		}
		sess.Set("flash_ok", "تنظیمات ظاهر سایت عمومی ذخیره شد")
	} else {
		// Save admin panel appearance
		adminKeys := []string{
			"admin_theme_name", "admin_accent_color", "admin_background_color",
			"admin_sidebar_color", "admin_card_color", "admin_text_color",
			"admin_muted_text_color", "admin_border_color", "admin_button_color",
			"admin_hover_color", "admin_sidebar_mode", "admin_sidebar_width",
			"admin_icon_size", "admin_menu_text_size", "admin_font_size",
			"admin_card_radius", "admin_card_shadow", "admin_card_border",
			"admin_dashboard_density", "admin_custom_logo", "admin_custom_background",
		}
		for _, k := range adminKeys {
			v := c.PostForm(k)
			models.SetSetting(k, v)
		}
		for _, k := range []string{"admin_glass_effect_enabled", "admin_animations_enabled", "admin_compact_mode_enabled"} {
			if c.PostForm(k) == "1" {
				models.SetSetting(k, "1")
			} else {
				models.SetSetting(k, "0")
			}
		}
		sess.Set("flash_ok", "تنظیمات ظاهر پنل مدیریت ذخیره شد")
	}

	sess.Save()
	c.Redirect(http.StatusFound, "/zed-admin/settings/appearance?tab="+tab)
}

func AdminAppearanceReset(c *gin.Context) {
	sess := sessions.Default(c)
	role, _ := sess.Get("role").(string)
	if role != "owner" && role != "" {
		c.Status(http.StatusForbidden)
		return
	}

	tab := c.PostForm("tab")

	if tab == "site" {
		for k, v := range siteAppearanceDefaults {
			models.SetSetting(k, v)
		}
		sess.Set("flash_ok", "ظاهر سایت عمومی به حالت پیش‌فرض بازگشت")
	} else {
		for k, v := range adminAppearanceDefaults {
			models.SetSetting(k, v)
		}
		sess.Set("flash_ok", "ظاهر پنل مدیریت به حالت پیش‌فرض بازگشت")
	}

	sess.Save()
	c.Redirect(http.StatusFound, "/zed-admin/settings/appearance?tab="+tab)
}
