package handlers

import (
	"fmt"
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/gin-contrib/sessions"
	"github.com/gin-gonic/gin"

	bkp "zedproxy/internal/backup"
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

	// Backup-to-Telegram settings
	data["TGSendDBZip"] = settings["telegram_admin_send_db_zip_enabled"]
	data["TGDailyDBBackup"] = settings["telegram_admin_daily_db_backup_enabled"]
	data["TGDailyDBBackupTime"] = settings["telegram_admin_daily_db_backup_time"]
	data["TGBackupBeforeUpdate"] = settings["telegram_admin_backup_before_update"]
	data["TGBackupBeforeRollback"] = settings["telegram_admin_backup_before_rollback"]

	// Masked token
	rawToken := settings["telegram_admin_bot_token"]
	data["TGTokenMasked"] = maskToken(rawToken)
	data["TGTokenSet"] = rawToken != ""

	// Owner check
	role, _ := sess.Get("role").(string)
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

	t, err := getAdminTemplate("telegram")
	if err != nil {
		renderAdminError(c, fmt.Sprintf("خطای قالب تلگرام: %v", err))
		return
	}
	c.Header("Content-Type", "text/html; charset=utf-8")
	if err := t.ExecuteTemplate(c.Writer, "admin", data); err != nil {
		log.Printf("telegram template execute error: %v", err)
	}
}

// AdminTelegramSave handles the settings form.
func AdminTelegramSave(c *gin.Context) {
	sess := sessions.Default(c)
	role, _ := sess.Get("role").(string)
	isOwner := role == "owner" || role == ""

	// Only owner can change token / chat ID
	if isOwner {
		// Handle token removal
		if c.PostForm("remove_telegram_token") == "1" {
			models.SetSetting("telegram_admin_bot_token", "")
			models.SetSetting("telegram_admin_bot_username", "")
		} else {
			rawToken := strings.TrimSpace(c.PostForm("telegram_admin_bot_token"))
			// Accept new token only if it's not empty and not a masked placeholder
			if rawToken != "" && !strings.Contains(rawToken, "...") && rawToken != "***" {
				models.SetSetting("telegram_admin_bot_token", rawToken)
				// Try to resolve bot username in background (non-blocking)
				if me, err := tg.GetMe(rawToken); err == nil && me != nil {
					models.SetSetting("telegram_admin_bot_username", me.Username)
				}
			}
		}

		chatID := strings.TrimSpace(c.PostForm("telegram_admin_chat_id"))
		existingChatID := models.GetSetting("telegram_admin_chat_id")
		if chatID != "" {
			models.SetSetting("telegram_admin_chat_id", chatID)
			// Only attempt chat title lookup if chat ID changed and token is present
			if chatID != existingChatID {
				currentToken := models.GetSetting("telegram_admin_bot_token")
				if currentToken != "" {
					if chat, err := tg.GetChat(currentToken, chatID); err == nil {
						models.SetSetting("telegram_admin_group_title", chat.Title)
					}
				}
			}
		}
	}

	checkbox := func(name string) string {
		if c.PostForm(name) == "1" {
			return "1"
		}
		return "0"
	}

	models.SetSetting("telegram_admin_bot_enabled", checkbox("telegram_admin_bot_enabled"))
	models.SetSetting("telegram_admin_daily_report_enabled", checkbox("telegram_admin_daily_report_enabled"))
	models.SetSetting("telegram_admin_alerts_enabled", checkbox("telegram_admin_alerts_enabled"))
	models.SetSetting("telegram_admin_security_alerts_enabled", checkbox("security_enabled"))
	models.SetSetting("telegram_admin_update_alerts_enabled", checkbox("updates_enabled"))
	models.SetSetting("telegram_admin_backup_alerts_enabled", checkbox("backups_enabled"))
	models.SetSetting("telegram_admin_analytics_enabled", checkbox("analytics_enabled"))
	models.SetSetting("telegram_admin_error_alerts_enabled", checkbox("errors_enabled"))
	models.SetSetting("telegram_admin_maintenance_alerts_enabled", checkbox("maintenance_enabled"))
	models.SetSetting("telegram_admin_admin_activity_enabled", checkbox("admin_activity_enabled"))

	// Backup-to-Telegram toggles
	models.SetSetting("telegram_admin_send_db_zip_enabled", checkbox("telegram_backup_db_zip_enabled"))
	models.SetSetting("telegram_admin_daily_db_backup_enabled", checkbox("telegram_backup_daily_enabled"))
	models.SetSetting("telegram_admin_backup_before_update", checkbox("telegram_backup_before_update_enabled"))
	models.SetSetting("telegram_admin_backup_before_rollback", checkbox("telegram_backup_before_rollback_enabled"))

	if t := c.PostForm("telegram_admin_daily_report_time"); t != "" {
		models.SetSetting("telegram_admin_daily_report_time", t)
	}
	if tz := c.PostForm("telegram_admin_daily_report_timezone"); tz != "" {
		models.SetSetting("telegram_admin_daily_report_timezone", tz)
	}
	if t := c.PostForm("telegram_backup_daily_time"); t != "" {
		models.SetSetting("telegram_admin_daily_db_backup_time", t)
	}

	LogAdminActivity(c, "telegram_settings_saved", "تنظیمات تلگرام ذخیره شد")

	sess.Set("flash_ok", "تنظیمات تلگرام با موفقیت ذخیره شد.")
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
	// Update cached bot username after successful test
	token := models.GetSetting("telegram_admin_bot_token")
	if token != "" {
		if me, err := tg.GetMe(token); err == nil && me != nil {
			models.SetSetting("telegram_admin_bot_username", me.Username)
		}
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
	c.JSON(http.StatusOK, gin.H{"ok": true, "description": "پیام تست ارسال شد"})
}

// AdminTelegramCreateTopics creates forum topics in the group.
func AdminTelegramCreateTopics(c *gin.Context) {
	err := tg.CreateTopicsInGroup()
	if err != nil {
		c.JSON(http.StatusOK, gin.H{"ok": false, "error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"ok": true, "description": "تاپیک‌های گروه ایجاد شدند"})
}

// AdminTelegramSendDailyReport sends today's daily report immediately.
func AdminTelegramSendDailyReport(c *gin.Context) {
	err := tg.SendDailyReport()
	if err != nil {
		c.JSON(http.StatusOK, gin.H{"ok": false, "error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"ok": true, "description": "گزارش روزانه ارسال شد"})
}

// AdminTelegramSendDBBackup creates a ZIP backup and sends it to Telegram.
func AdminTelegramSendDBBackup(c *gin.Context) {
	dbp := resolveDBPath()
	zipData, filename, err := bkp.CreateDBZip(dbp)
	if err != nil {
		c.JSON(http.StatusOK, gin.H{"ok": false, "error": "خطا در ایجاد بکاپ: " + err.Error()})
		return
	}

	// Always save locally first
	if _, err := bkp.SaveZipToDir(zipData, backupDir, filename); err != nil {
		c.JSON(http.StatusOK, gin.H{"ok": false, "error": "خطا در ذخیره بکاپ: " + err.Error()})
		return
	}
	models.RecordBackup(filename, int64(len(zipData)))

	caption := fmt.Sprintf("💾 بکاپ دیتابیس ZedProxy\nتاریخ: %s", time.Now().Format("2006/01/02 15:04"))
	if err := tg.SendBackupToTelegram(zipData, filename, caption); err != nil {
		c.JSON(http.StatusOK, gin.H{
			"ok":          false,
			"error":       "بکاپ ذخیره شد اما ارسال به تلگرام ناموفق بود: " + err.Error(),
			"description": fmt.Sprintf("فایل محلی ذخیره شد: %s", filename),
		})
		return
	}
	c.JSON(http.StatusOK, gin.H{"ok": true, "description": fmt.Sprintf("بکاپ %s ارسال شد", filename)})
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
