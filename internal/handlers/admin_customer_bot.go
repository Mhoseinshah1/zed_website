package handlers

import (
	"github.com/gin-contrib/sessions"
	"github.com/gin-gonic/gin"
	"zedproxy/internal/models"
)

var customerBotKeys = []string{
	"customer_telegram_bot_enabled",
	"customer_telegram_bot_username",
	"customer_telegram_bot_token",
	"customer_bot_internal_api_key",
}

func AdminCustomerBotPage(c *gin.Context) {
	settings := models.GetAllSettings()
	data := adminData(c, "customer-bot")
	data["Title"] = "تنظیمات ربات مشتری"
	data["Settings"] = settings
	renderAdmin(c, "customer-bot", data)
}

func AdminCustomerBotSave(c *gin.Context) {
	for _, k := range customerBotKeys {
		models.SetSetting(k, c.PostForm(k))
	}
	LogAdminActivity(c, "customer_bot_save", "تنظیمات ربات مشتری بروزرسانی شد")
	sess := sessions.Default(c)
	sess.AddFlash("تنظیمات ربات مشتری ذخیره شد", "ok")
	sess.Save()
	c.Redirect(302, "/zed-admin/integrations/customer-bot")
}
