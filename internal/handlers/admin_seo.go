package handlers

import (
	"github.com/gin-contrib/sessions"
	"github.com/gin-gonic/gin"
	"zedproxy/internal/models"
)

var seoKeys = []string{
	"seo_default_title", "seo_default_description", "seo_default_keywords",
	"seo_og_image", "seo_robots_txt", "seo_enable_sitemap",
}

func AdminSEOPage(c *gin.Context) {
	settings := models.GetAllSettings()
	data := adminData(c, "seo")
	data["Title"] = "تنظیمات سئو"
	data["Settings"] = settings
	renderAdmin(c, "seo", data)
}

func AdminSEOSave(c *gin.Context) {
	for _, k := range seoKeys {
		models.SetSetting(k, c.PostForm(k))
	}
	LogAdminActivity(c, "seo_save", "تنظیمات سئو بروزرسانی شد")
	sess := sessions.Default(c)
	sess.AddFlash("تنظیمات سئو ذخیره شد", "ok")
	sess.Save()
	c.Redirect(302, "/zed-admin/seo")
}
