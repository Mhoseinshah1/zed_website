package telegram

import (
	"fmt"
	"log"
	"sync"
	"time"

	"zedproxy/internal/database"
	"zedproxy/internal/models"
)

// Level represents notification severity.
type Level string

const (
	LevelInfo     Level = "info"
	LevelWarn     Level = "warn"
	LevelError    Level = "error"
	LevelCritical Level = "critical"
)

// Category maps to topic keys.
type Category string

const (
	CatSystem        Category = "system_status"
	CatCritical      Category = "critical_alerts"
	CatUpdates       Category = "updates"
	CatMaintenance   Category = "maintenance"
	CatBackups       Category = "backups"
	CatSecurity      Category = "security"
	CatDailyReport   Category = "daily_reports"
	CatClickAnalytics Category = "click_analytics"
	CatSEO           Category = "seo_pages"
	CatAdminActivity Category = "admin_activity"
	CatErrors        Category = "errors"
)

// rateLimiter prevents sending the same message too frequently.
var (
	rateMap  = map[string]time.Time{}
	rateMu   sync.Mutex
	rateWin  = 5 * time.Minute
)

func rateAllowed(key string) bool {
	rateMu.Lock()
	defer rateMu.Unlock()
	if last, ok := rateMap[key]; ok && time.Since(last) < rateWin {
		return false
	}
	rateMap[key] = time.Now()
	return true
}

// Send queues a Telegram notification. Never panics; errors are logged only.
func Send(level Level, cat Category, title, message string) {
	if models.GetSetting("telegram_admin_bot_enabled") != "1" {
		return
	}
	if !alertCategoryEnabled(cat) {
		return
	}

	rateKey := fmt.Sprintf("%s:%s:%s", cat, level, title)
	if !rateAllowed(rateKey) {
		return
	}

	go func() {
		if err := enqueue(level, cat, title, message); err != nil {
			log.Printf("telegram: enqueue error: %v", err)
		}
	}()
}

// alertCategoryEnabled checks the per-category toggle.
func alertCategoryEnabled(cat Category) bool {
	switch cat {
	case CatCritical, CatSystem:
		return models.GetSetting("telegram_admin_alerts_enabled") == "1"
	case CatSecurity:
		return models.GetSetting("telegram_admin_security_alerts_enabled") == "1"
	case CatUpdates:
		return models.GetSetting("telegram_admin_update_alerts_enabled") == "1"
	case CatBackups:
		return models.GetSetting("telegram_admin_backup_alerts_enabled") == "1"
	case CatClickAnalytics:
		return models.GetSetting("telegram_admin_analytics_enabled") == "1"
	case CatErrors:
		return models.GetSetting("telegram_admin_error_alerts_enabled") == "1"
	case CatMaintenance:
		return models.GetSetting("telegram_admin_maintenance_alerts_enabled") == "1"
	case CatAdminActivity:
		return models.GetSetting("telegram_admin_admin_activity_enabled") == "1"
	}
	return true
}

// enqueue inserts into telegram_queue and tries to dispatch.
func enqueue(level Level, cat Category, title, message string) error {
	payload := fmt.Sprintf("%s\n\n%s", title, message)
	_, err := database.DB.Exec(
		`INSERT INTO telegram_queue (level, category, topic_key, payload, status) VALUES (?,?,?,?,'pending')`,
		string(level), string(cat), string(cat), payload,
	)
	if err != nil {
		return err
	}
	ProcessQueue()
	return nil
}

// ProcessQueue attempts to send pending queue items.
func ProcessQueue() {
	token := models.GetSetting("telegram_admin_bot_token")
	chatID := models.GetSetting("telegram_admin_chat_id")
	if token == "" || chatID == "" {
		return
	}

	rows, err := database.DB.Query(
		`SELECT id, topic_key, payload, attempts FROM telegram_queue WHERE status='pending' AND attempts < 3 ORDER BY id LIMIT 10`,
	)
	if err != nil {
		return
	}
	defer rows.Close()

	type qItem struct {
		id       int
		topicKey string
		payload  string
		attempts int
	}
	var items []qItem
	for rows.Next() {
		var it qItem
		rows.Scan(&it.id, &it.topicKey, &it.payload, &it.attempts)
		items = append(items, it)
	}
	rows.Close()

	for _, it := range items {
		threadID := getTopicThreadID(it.topicKey)
		_, sendErr := SendMessage(token, chatID, it.payload, threadID, "")
		if sendErr != nil {
			database.DB.Exec(
				`UPDATE telegram_queue SET attempts=attempts+1, last_error=?, updated_at=CURRENT_TIMESTAMP WHERE id=?`,
				sendErr.Error(), it.id,
			)
			// Record failed notification
			database.DB.Exec(
				`INSERT INTO telegram_notifications (level, category, topic_key, title, message, status, error)
				 VALUES ('','','','',?,'failed',?)`, it.payload, sendErr.Error(),
			)
		} else {
			database.DB.Exec(
				`UPDATE telegram_queue SET status='sent', updated_at=CURRENT_TIMESTAMP WHERE id=?`,
				it.id,
			)
			database.DB.Exec(
				`INSERT INTO telegram_notifications (level, category, topic_key, title, message, status, sent_at)
				 VALUES ('','',?,'',' ','sent',CURRENT_TIMESTAMP)`, it.topicKey,
			)
		}
	}
}

func getTopicThreadID(topicKey string) int {
	var threadID int
	database.DB.QueryRow(
		`SELECT message_thread_id FROM telegram_topics WHERE key=? AND enabled=1`,
		topicKey,
	).Scan(&threadID)
	return threadID
}

// SendDailyReport generates and sends the daily report.
func SendDailyReport() error {
	if models.GetSetting("telegram_admin_bot_enabled") != "1" {
		return fmt.Errorf("telegram bot not enabled")
	}
	if models.GetSetting("telegram_admin_daily_report_enabled") != "1" {
		return fmt.Errorf("daily report not enabled")
	}

	today := time.Now().Format("2006-01-02")
	last := models.GetSetting("telegram_admin_last_daily_report_date")
	if last == today {
		return fmt.Errorf("daily report already sent today")
	}

	report := buildDailyReport()
	token := models.GetSetting("telegram_admin_bot_token")
	chatID := models.GetSetting("telegram_admin_chat_id")
	threadID := getTopicThreadID(string(CatDailyReport))

	_, err := SendMessage(token, chatID, report, threadID, "")
	if err != nil {
		return fmt.Errorf("send daily report: %w", err)
	}

	models.SetSetting("telegram_admin_last_daily_report_date", today)
	log.Println("telegram: daily report sent")
	return nil
}

func buildDailyReport() string {
	var adminCount, planCount, postCount int
	database.DB.QueryRow("SELECT COUNT(*) FROM admins").Scan(&adminCount)
	database.DB.QueryRow("SELECT COUNT(*) FROM plans WHERE is_active=1").Scan(&planCount)
	database.DB.QueryRow("SELECT COUNT(*) FROM blog_posts WHERE is_published=1").Scan(&postCount)

	var clickCount int
	database.DB.QueryRow(
		"SELECT COUNT(*) FROM telegram_clicks WHERE created_at >= date('now','-1 day')",
	).Scan(&clickCount)

	maintenance := models.GetSetting("maintenance_enabled")
	maintenanceStr := "غیرفعال"
	if maintenance == "1" {
		maintenanceStr = "فعال"
	}

	return fmt.Sprintf(`📊 گزارش روزانه ZedProxy
تاریخ: %s

👥 ادمین‌ها: %d
📦 پلن‌های فعال: %d
📝 مقالات منتشرشده: %d
📈 کلیک‌های تلگرام (24 ساعت): %d
🧰 حالت تعمیرات: %s`,
		time.Now().Format("2006/01/02"),
		adminCount, planCount, postCount, clickCount, maintenanceStr,
	)
}

// SendBackupToTelegram sends a ZIP backup file to the backups topic.
// If the Telegram upload fails, the error is returned but the local file is NOT deleted.
func SendBackupToTelegram(zipData []byte, filename, caption string) error {
	token := models.GetSetting("telegram_admin_bot_token")
	chatID := models.GetSetting("telegram_admin_chat_id")
	if token == "" || chatID == "" {
		return fmt.Errorf("bot token or chat ID not configured")
	}
	threadID := getTopicThreadID(string(CatBackups))
	return SendDocument(token, chatID, zipData, filename, caption, threadID)
}

// SeedDefaultTopics inserts default forum topics if not already present.
func SeedDefaultTopics() {
	topics := []struct {
		key   string
		title string
	}{
		{"system_status", "📌 وضعیت سیستم"},
		{"critical_alerts", "🚨 هشدارهای مهم"},
		{"updates", "🔄 بروزرسانی‌ها"},
		{"maintenance", "🧰 حالت تعمیرات"},
		{"backups", "💾 بکاپ‌ها"},
		{"security", "🔐 امنیت"},
		{"daily_reports", "📊 گزارش روزانه"},
		{"click_analytics", "📈 آمار کلیک‌ها"},
		{"seo_pages", "🌐 سئو و صفحات"},
		{"admin_activity", "🧾 فعالیت ادمین"},
		{"errors", "❌ خطاها"},
	}
	for _, t := range topics {
		database.DB.Exec(
			`INSERT INTO telegram_topics (key, title, enabled) VALUES (?,?,1) ON CONFLICT(key) DO NOTHING`,
			t.key, t.title,
		)
	}
}

// CreateTopicsInGroup creates forum topics in the group and stores their thread IDs.
func CreateTopicsInGroup() error {
	token := models.GetSetting("telegram_admin_bot_token")
	chatID := models.GetSetting("telegram_admin_chat_id")
	if token == "" || chatID == "" {
		return fmt.Errorf("bot token or chat ID not configured")
	}

	rows, err := database.DB.Query(`SELECT key, title FROM telegram_topics WHERE enabled=1`)
	if err != nil {
		return err
	}
	defer rows.Close()

	type topic struct{ key, title string }
	var topics []topic
	for rows.Next() {
		var t topic
		rows.Scan(&t.key, &t.title)
		topics = append(topics, t)
	}
	rows.Close()

	var lastErr error
	for _, t := range topics {
		threadID, err := CreateForumTopic(token, chatID, t.title, "")
		if err != nil {
			log.Printf("telegram: create topic %s: %v", t.key, err)
			lastErr = err
			continue
		}
		database.DB.Exec(
			`UPDATE telegram_topics SET message_thread_id=?, updated_at=CURRENT_TIMESTAMP WHERE key=?`,
			threadID, t.key,
		)
		log.Printf("telegram: created topic %s (thread %d)", t.key, threadID)
		time.Sleep(500 * time.Millisecond) // avoid flood
	}
	return lastErr
}

// TestConnection validates token and chat ID, returns a description or error.
func TestConnection() (string, error) {
	token := models.GetSetting("telegram_admin_bot_token")
	chatID := models.GetSetting("telegram_admin_chat_id")
	if token == "" {
		return "", fmt.Errorf("bot token not configured")
	}
	me, err := GetMe(token)
	if err != nil {
		return "", fmt.Errorf("getMe failed: %w", err)
	}
	if chatID == "" {
		return fmt.Sprintf("Bot: @%s (chat ID not set)", me.Username), nil
	}
	chat, err := GetChat(token, chatID)
	if err != nil {
		return fmt.Sprintf("Bot: @%s | Chat error: %v", me.Username, err), nil
	}
	return fmt.Sprintf("Bot: @%s | Chat: %s (%s)", me.Username, chat.Title, chat.Type), nil
}

// SendTestMessage sends a test notification to the configured group.
func SendTestMessage() error {
	token := models.GetSetting("telegram_admin_bot_token")
	chatID := models.GetSetting("telegram_admin_chat_id")
	if token == "" || chatID == "" {
		return fmt.Errorf("bot token or chat ID not configured")
	}
	msg := fmt.Sprintf("✅ تست اتصال ZedProxy\nزمان: %s\nبات به درستی تنظیم شده است.", time.Now().Format("2006/01/02 15:04:05"))
	_, err := SendMessage(token, chatID, msg, 0, "")
	return err
}
