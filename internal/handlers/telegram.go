package handlers

import (
	"net/http"
	"strings"

	"github.com/gin-contrib/sessions"
	"github.com/gin-gonic/gin"

	"zedproxy/internal/database"
	"zedproxy/internal/models"
	tg "zedproxy/internal/telegram"
)

// AdminTelegramPage renders the Telegram integration page.
func AdminTelegramPage(c *gin.Context) {
	sess := sessions.Default(c)
	data := adminData(c, "telegram")

	data["Title"] = "یکپارچه‌سازی تلگرام"
	data["Section"] = "telegram"

	// Collect settings
	settings := models.GetAllSettings()
	data["TGEnabled"] = settings["telegram_admin_bot_enabled"]
	data["TGChatID"] = settings["telegram_admin_chat_id"]
	data["TGGroupTitle"] = settings["telegram_admin_group_title"]
	data["TGBotUsername"] = settings["telegram_admin_bot_username"]
	data["TGDailyEnabled"] = settings["telegram_admin_daily_report_enabled"]
	data["TGDailyTime"] = settings["telegram_admin_daily_report_time"]
	data["TGDailyTZ"] = settings["telegram_admin_daily_report_timezone"]
	data["TGAlerts"] = settings["telegram_admin_alerts_enabled"]
	data["TGSecurity"] = settings["telegram_admin_security_alerts_enabled"]
	data["TGUpdates"] = settings["telegram_admin_update_alerts_enabled"]
	data["TGBackups"] = settings["telegram_admin_backup_alerts_enabled"]
	data["TGAnalytics"] = settings["telegram_admin_analytics_enabled"]
	data["TGErrors"] = settings["telegram_admin_error_alerts_enabled"]
	data["TGMaintenance"] = settings["telegram_admin_maintenance_alerts_enabled"]
	data["TGAdminActivity"] = settings["telegram_admin_admin_activity_enabled"]

	// Masked token
	rawToken := settings["telegram_admin_bot_token"]
	data["TGTokenMasked"] = maskToken(rawToken)
	data["TGTokenSet"] = rawToken != ""

	// Owner check for sensitive fields
	role := ""
	if v, ok := sess.Get("role").(string); ok {
		role = v
	}
	data["IsOwner"] = role == "owner" || role == ""

	// Topics
	rows, _ := database.DB.Query(
		`SELECT id, key, title, message_thread_id, enabled FROM telegram_topics ORDER BY id`,
	)
	type Topic struct {
		ID       int
		Key      string
		Title    string
		ThreadID int
		Enabled  bool
	}
	var topics []Topic
	if rows != nil {
		defer rows.Close()
		for rows.Next() {
			var t Topic
			var enabled int
			rows.Scan(&t.ID, &t.Key, &t.Title, &t.ThreadID, &enabled)
			t.Enabled = enabled == 1
			topics = append(topics, t)
		}
	}
	data["Topics"] = topics

	// Recent notifications (last 20)
	nrows, _ := database.DB.Query(
		`SELECT id, level, category, topic_key, status, error, created_at FROM telegram_notifications ORDER BY id DESC LIMIT 20`,
	)
	type Notif struct {
		ID        int
		Level     string
		Category  string
		TopicKey  string
		Status    string
		Error     string
		CreatedAt string
	}
	var notifs []Notif
	if nrows != nil {
		defer nrows.Close()
		for nrows.Next() {
			var n Notif
			nrows.Scan(&n.ID, &n.Level, &n.Category, &n.TopicKey, &n.Status, &n.Error, &n.CreatedAt)
			notifs = append(notifs, n)
		}
	}
	data["Notifications"] = notifs

	// Flash
	if f := sess.Get("flash_ok"); f != nil {
		data["FlashOK"] = f.(string)
		sess.Delete("flash_ok")
		sess.Save()
	}

	t, err := getAdminTemplate("telegram")
	if err != nil {
		c.String(http.StatusInternalServerError, "template error: %v", err)
		return
	}
	t.ExecuteTemplate(c.Writer, "admin", data)
}

// AdminTelegramSave handles the settings form.
func AdminTelegramSave(c *gin.Context) {
	sess := sessions.Default(c)
	role := ""
	if v, ok := sess.Get("role").(string); ok {
		role = v
	}
	isOwner := role == "owner" || role == ""

	// Only owner can change token / chat ID
	if isOwner {
		rawToken := strings.TrimSpace(c.PostForm("bot_token"))
		if rawToken != "" && rawToken != "***" && !strings.HasSuffix(rawToken, "...") {
			models.SetSetting("telegram_admin_bot_token", rawToken)
		}
		chatID := strings.TrimSpace(c.PostForm("chat_id"))
		if chatID != "" {
			models.SetSetting("telegram_admin_chat_id", chatID)
		}
	}

	checkbox := func(name string) string {
		if c.PostForm(name) == "1" {
			return "1"
		}
		return "0"
	}

	models.SetSetting("telegram_admin_bot_enabled", checkbox("bot_enabled"))
	models.SetSetting("telegram_admin_daily_report_enabled", checkbox("daily_enabled"))
	models.SetSetting("telegram_admin_alerts_enabled", checkbox("alerts_enabled"))
	models.SetSetting("telegram_admin_security_alerts_enabled", checkbox("security_enabled"))
	models.SetSetting("telegram_admin_update_alerts_enabled", checkbox("updates_enabled"))
	models.SetSetting("telegram_admin_backup_alerts_enabled", checkbox("backups_enabled"))
	models.SetSetting("telegram_admin_analytics_enabled", checkbox("analytics_enabled"))
	models.SetSetting("telegram_admin_error_alerts_enabled", checkbox("errors_enabled"))
	models.SetSetting("telegram_admin_maintenance_alerts_enabled", checkbox("maintenance_enabled"))
	models.SetSetting("telegram_admin_admin_activity_enabled", checkbox("admin_activity_enabled"))

	if t := c.PostForm("daily_time"); t != "" {
		models.SetSetting("telegram_admin_daily_report_time", t)
	}
	if tz := c.PostForm("daily_timezone"); tz != "" {
		models.SetSetting("telegram_admin_daily_report_timezone", tz)
	}

	sess.Set("flash_ok", "تنظیمات تلگرام ذخیره شد")
	sess.Save()
	c.Redirect(http.StatusFound, "/zed-admin/integrations/telegram")
}

// AdminTelegramTest validates bot token and chat ID.
func AdminTelegramTest(c *gin.Context) {
	desc, err := tg.TestConnection()
	if err != nil {
		c.JSON(http.StatusOK, gin.H{"ok": false, "error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"ok": true, "description": desc})
}

// AdminTelegramSendTest sends a test message.
func AdminTelegramSendTest(c *gin.Context) {
	err := tg.SendTestMessage()
	if err != nil {
		c.JSON(http.StatusOK, gin.H{"ok": false, "error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"ok": true})
}

// AdminTelegramCreateTopics creates forum topics in the group.
func AdminTelegramCreateTopics(c *gin.Context) {
	err := tg.CreateTopicsInGroup()
	if err != nil {
		c.JSON(http.StatusOK, gin.H{"ok": false, "error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"ok": true})
}

// AdminTelegramSendDailyReport sends today's daily report immediately.
func AdminTelegramSendDailyReport(c *gin.Context) {
	err := tg.SendDailyReport()
	if err != nil {
		c.JSON(http.StatusOK, gin.H{"ok": false, "error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"ok": true})
}

// AdminTelegramDisable disables the bot.
func AdminTelegramDisable(c *gin.Context) {
	models.SetSetting("telegram_admin_bot_enabled", "0")
	sess := sessions.Default(c)
	sess.Set("flash_ok", "بات تلگرام غیرفعال شد")
	sess.Save()
	c.Redirect(http.StatusFound, "/zed-admin/integrations/telegram")
}

func maskToken(token string) string {
	if token == "" {
		return ""
	}
	parts := strings.SplitN(token, ":", 2)
	if len(parts) != 2 || len(parts[1]) < 6 {
		return "***"
	}
	s := parts[1]
	return parts[0] + ":" + s[:3] + "..." + s[len(s)-3:]
}
