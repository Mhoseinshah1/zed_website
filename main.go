package main

import (
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"time"

	"github.com/gin-contrib/sessions"
	"github.com/gin-contrib/sessions/cookie"
	"github.com/gin-gonic/gin"

	"zedproxy/internal/database"
	"zedproxy/internal/handlers"
	"zedproxy/internal/middleware"
	"zedproxy/internal/models"
	"zedproxy/internal/seed"
	tg "zedproxy/internal/telegram"
)

// Set via ldflags: -X main.Version=... -X main.BuildDate=... -X main.GitCommit=...
var (
	Version   = "dev"
	BuildDate = "unknown"
	GitCommit = "unknown"
)

func main() {
	var (
		addr          = flag.String("addr", ":8080", "listen address")
		dbPath        = flag.String("db", "./data/zedproxy.db", "SQLite database path")
		templateDir   = flag.String("templates", "./templates", "templates directory")
		staticDir     = flag.String("static", "./static", "static files directory")
		uploadDirFlag = flag.String("uploads", "./static/uploads", "upload directory")
		backupDirFlag = flag.String("backups", "./data/backups", "backup directory")
		sessionSecret = flag.String("secret", getEnvOrDefault("SESSION_SECRET", "change-me-in-production"), "session secret")
		dev           = flag.Bool("dev", false, "development mode (disable template cache)")
		seedFlag      = flag.Bool("seed", false, "seed the database with default content")
		adminUser     = flag.String("admin-user", "admin", "admin username for seeding")
		adminEmail    = flag.String("admin-email", "admin@zedproxy.com", "admin email for seeding")
		adminPass         = flag.String("admin-pass", "", "admin password for seeding")
		maintenanceOn     = flag.Bool("maintenance-on", false, "enable maintenance mode and exit")
		maintenanceOff    = flag.Bool("maintenance-off", false, "disable maintenance mode and exit")
		maintenanceStatus = flag.Bool("maintenance-status", false, "show maintenance mode status and exit")
		selfTest          = flag.Bool("self-test", false, "run self-test and exit")

		// Telegram CLI flags
		telegramStatus       = flag.Bool("telegram-status", false, "show Telegram bot status and exit")
		telegramTest         = flag.Bool("telegram-test", false, "test Telegram bot connection and exit")
		telegramCreateTopics = flag.Bool("telegram-create-topics", false, "create forum topics in group and exit")
		telegramSendTest     = flag.Bool("telegram-send-test", false, "send test message via Telegram and exit")
		sendDailyReport      = flag.Bool("send-daily-report", false, "send daily report now and exit")
		telegramEnable       = flag.Bool("telegram-enable", false, "enable Telegram bot and exit")
		telegramDisable      = flag.Bool("telegram-disable", false, "disable Telegram bot and exit")
		telegramSetToken     = flag.String("telegram-set-token", "", "set Telegram bot token and exit")
		telegramSetChatID    = flag.String("telegram-set-chat-id", "", "set Telegram chat ID and exit")
		telegramNotifyTitle  = flag.String("telegram-notify-title", "", "notification title (use with --telegram-notify-msg)")
		telegramNotifyMsg    = flag.String("telegram-notify-msg", "", "notification message body")
		telegramNotifyCat    = flag.String("telegram-notify-cat", "system_status", "notification category/topic key")
	)
	flag.Parse()

	// Init DB
	database.Init(*dbPath)
	tg.SeedDefaultTopics()

	// CLI maintenance controls (exit after action)
	if *maintenanceOn {
		models.SetSetting("maintenance_enabled", "1")
		fmt.Println("[✓] حالت تعمیر فعال شد")
		return
	}
	if *maintenanceOff {
		models.SetSetting("maintenance_enabled", "0")
		fmt.Println("[✓] حالت تعمیر غیرفعال شد")
		return
	}
	if *maintenanceStatus {
		v := models.GetSetting("maintenance_enabled")
		if v == "1" {
			fmt.Println("[!] حالت تعمیر: فعال")
		} else {
			fmt.Println("[✓] حالت تعمیر: غیرفعال")
		}
		return
	}
	if *selfTest {
		runSelfTest(*dbPath, *templateDir, *staticDir, *uploadDirFlag)
		return
	}

	// Telegram CLI flags
	if *telegramSetToken != "" {
		models.SetSetting("telegram_admin_bot_token", *telegramSetToken)
		fmt.Println("[✓] Telegram bot token updated")
		return
	}
	if *telegramSetChatID != "" {
		models.SetSetting("telegram_admin_chat_id", *telegramSetChatID)
		fmt.Println("[✓] Telegram chat ID updated")
		return
	}
	if *telegramEnable {
		models.SetSetting("telegram_admin_bot_enabled", "1")
		fmt.Println("[✓] Telegram bot enabled")
		return
	}
	if *telegramDisable {
		models.SetSetting("telegram_admin_bot_enabled", "0")
		fmt.Println("[✓] Telegram bot disabled")
		return
	}
	if *telegramStatus {
		enabled := models.GetSetting("telegram_admin_bot_enabled")
		chatID := models.GetSetting("telegram_admin_chat_id")
		botUser := models.GetSetting("telegram_admin_bot_username")
		fmt.Printf("[i] Telegram bot enabled: %s\n", enabled)
		fmt.Printf("[i] Chat ID: %s\n", chatID)
		fmt.Printf("[i] Bot username: %s\n", botUser)
		return
	}
	if *telegramTest {
		desc, err := tg.TestConnection()
		if err != nil {
			fmt.Printf("[✗] Connection test failed: %v\n", err)
			os.Exit(1)
		}
		fmt.Printf("[✓] %s\n", desc)
		return
	}
	if *telegramSendTest {
		if err := tg.SendTestMessage(); err != nil {
			fmt.Printf("[✗] Test message failed: %v\n", err)
			os.Exit(1)
		}
		fmt.Println("[✓] Test message sent")
		return
	}
	if *telegramCreateTopics {
		if err := tg.CreateTopicsInGroup(); err != nil {
			fmt.Printf("[!] Some topics failed: %v\n", err)
		} else {
			fmt.Println("[✓] Forum topics created")
		}
		return
	}
	if *sendDailyReport {
		if err := tg.SendDailyReport(); err != nil {
			fmt.Printf("[✗] Daily report failed: %v\n", err)
			os.Exit(1)
		}
		fmt.Println("[✓] Daily report sent")
		return
	}
	if *telegramNotifyTitle != "" {
		tg.Send(tg.LevelInfo, tg.Category(*telegramNotifyCat), *telegramNotifyTitle, *telegramNotifyMsg)
		tg.ProcessQueue() // flush synchronously since we're exiting
		fmt.Println("[✓] Notification sent")
		return
	}

	// Seed if requested
	if *seedFlag {
		if *adminPass == "" {
			log.Fatal("--admin-pass is required when using --seed")
		}
		seed.Run(*adminUser, *adminEmail, *adminPass)
		if flag.NArg() == 0 {
			return
		}
	}

	handlers.Init(*templateDir, *dev)
	handlers.AppVersion = Version
	handlers.SetUploadDir(*uploadDirFlag)
	handlers.SetBackupDir(*backupDirFlag)
	handlers.SetDBPath(*dbPath)

	if !*dev {
		gin.SetMode(gin.ReleaseMode)
	}

	r := gin.New()
	r.Use(gin.Recovery())
	if *dev {
		r.Use(gin.Logger())
	} else {
		r.Use(requestLogger())
	}

	// Sessions
	store := cookie.NewStore([]byte(*sessionSecret))
	store.Options(sessions.Options{
		Path:     "/",
		MaxAge:   86400 * 7,
		HttpOnly: true,
		Secure:   false,
		SameSite: http.SameSiteLaxMode,
	})
	r.Use(sessions.Sessions("zedproxy_session", store))

	// Static files
	r.Static("/static", *staticDir)
	r.Static("/uploads", *uploadDirFlag)

	// Ensure directories exist
	os.MkdirAll(*uploadDirFlag, 0755)
	os.MkdirAll(*backupDirFlag, 0755)

	// Favicon
	faviconPath := filepath.Join(*staticDir, "favicon.ico")
	r.GET("/favicon.ico", func(c *gin.Context) {
		if _, err := os.Stat(faviconPath); err == nil {
			c.File(faviconPath)
		} else {
			c.Status(http.StatusNoContent)
		}
	})

	// Health check (always available, bypasses maintenance)
	r.GET("/health", handlers.HealthCheck)
	r.HEAD("/health", handlers.HealthCheck)

	// SEO (bypass maintenance)
	r.GET("/sitemap.xml", handlers.SitemapXML)
	r.GET("/robots.txt", handlers.RobotsTXT)

	// Maintenance middleware (applied to public routes below)
	r.Use(handlers.MaintenanceMiddleware())

	// Public routes
	r.GET("/", handlers.HomePage)
	r.HEAD("/", handlers.HomePage)
	r.GET("/plans", handlers.PlansPage)
	r.HEAD("/plans", handlers.PlansPage)
	r.GET("/tutorials", handlers.TutorialsPage)
	r.HEAD("/tutorials", handlers.TutorialsPage)
	r.GET("/tutorials/:slug", handlers.TutorialDetailPage)
	r.HEAD("/tutorials/:slug", handlers.TutorialDetailPage)
	r.GET("/blog", handlers.BlogPage)
	r.HEAD("/blog", handlers.BlogPage)
	r.GET("/blog/:slug", handlers.BlogPostPage)
	r.HEAD("/blog/:slug", handlers.BlogPostPage)
	r.GET("/faq", handlers.FAQPage)
	r.HEAD("/faq", handlers.FAQPage)
	r.GET("/contact", handlers.ContactPage)
	r.HEAD("/contact", handlers.ContactPage)
	r.GET("/status", handlers.StatusPage)
	r.HEAD("/status", handlers.StatusPage)
	r.GET("/terms", handlers.TermsPage)
	r.HEAD("/terms", handlers.TermsPage)
	r.GET("/privacy", handlers.PrivacyPage)
	r.HEAD("/privacy", handlers.PrivacyPage)

	// Campaign pages
	r.GET("/campaign/:slug", handlers.CampaignPage)
	r.HEAD("/campaign/:slug", handlers.CampaignPage)

	// Landing pages (SEO)
	r.GET("/l/:slug", handlers.LandingPage)
	r.HEAD("/l/:slug", handlers.LandingPage)

	// Track clicks API
	r.POST("/api/track", handlers.TrackClick)

	// 404 handler
	r.NoRoute(handlers.NotFoundPage)

	// Admin routes
	admin := r.Group("/zed-admin")
	{
		admin.GET("/login", handlers.AdminLoginPage)
		admin.POST("/login", middleware.RateLimit(5, 15*time.Minute), handlers.AdminLoginPost)
		admin.GET("/logout", handlers.AdminLogout)

		protected := admin.Group("")
		protected.Use(middleware.AdminRequired())
		{
			protected.GET("", handlers.AdminDashboard)
			protected.GET("/", handlers.AdminDashboard)

			// Settings
			protected.GET("/settings", handlers.AdminSettingsPage)
			protected.POST("/settings", handlers.AdminSettingsPost)

			// Plans
			protected.GET("/plans", handlers.AdminPlansPage)
			protected.GET("/plans/new", handlers.AdminPlanNew)
			protected.GET("/plans/:id/edit", handlers.AdminPlanEdit)
			protected.POST("/plans/save", handlers.AdminPlanSave)
			protected.POST("/plans/:id/delete", handlers.AdminPlanDelete)

			// Features
			protected.GET("/features", handlers.AdminFeaturesPage)
			protected.GET("/features/new", handlers.AdminFeatureNew)
			protected.GET("/features/:id/edit", handlers.AdminFeatureEdit)
			protected.POST("/features/save", handlers.AdminFeatureSave)
			protected.POST("/features/:id/delete", handlers.AdminFeatureDelete)

			// FAQs
			protected.GET("/faqs", handlers.AdminFAQsPage)
			protected.GET("/faqs/new", handlers.AdminFAQNew)
			protected.GET("/faqs/:id/edit", handlers.AdminFAQEdit)
			protected.POST("/faqs/save", handlers.AdminFAQSave)
			protected.POST("/faqs/:id/delete", handlers.AdminFAQDelete)

			// Blog Posts
			protected.GET("/posts", handlers.AdminPostsPage)
			protected.GET("/posts/new", handlers.AdminPostNew)
			protected.GET("/posts/:id/edit", handlers.AdminPostEdit)
			protected.POST("/posts/save", handlers.AdminPostSave)
			protected.POST("/posts/:id/delete", handlers.AdminPostDelete)

			// Tutorials
			protected.GET("/tutorials", handlers.AdminTutorialsPage)
			protected.GET("/tutorials/new", handlers.AdminTutorialNew)
			protected.GET("/tutorials/:id/edit", handlers.AdminTutorialEdit)
			protected.POST("/tutorials/save", handlers.AdminTutorialSave)
			protected.POST("/tutorials/:id/delete", handlers.AdminTutorialDelete)

			// Locations
			protected.GET("/locations", handlers.AdminLocationsPage)
			protected.GET("/locations/new", handlers.AdminLocationNew)
			protected.GET("/locations/:id/edit", handlers.AdminLocationEdit)
			protected.POST("/locations/save", handlers.AdminLocationSave)
			protected.POST("/locations/:id/delete", handlers.AdminLocationDelete)

			// Status Updates
			protected.GET("/status", handlers.AdminStatusPage)
			protected.GET("/status/new", handlers.AdminStatusNew)
			protected.GET("/status/:id/edit", handlers.AdminStatusEdit)
			protected.POST("/status/save", handlers.AdminStatusSave)
			protected.POST("/status/:id/delete", handlers.AdminStatusDelete)

			// Pages
			protected.GET("/pages", handlers.AdminPagesPage)
			protected.GET("/pages/:slug/edit", handlers.AdminPageEdit)
			protected.POST("/pages/save", handlers.AdminPageSave)

			// Media
			protected.GET("/media", handlers.AdminMediaPage)
			protected.POST("/media/upload", handlers.AdminMediaUpload)
			protected.POST("/media/:id/delete", handlers.AdminMediaDelete)
			protected.POST("/media/:id/alt", handlers.AdminMediaUpdateAlt)

			// Password
			protected.GET("/password", handlers.AdminPasswordPage)
			protected.POST("/password", handlers.AdminPasswordPost)

			// --- New features ---

			// Announcements
			protected.GET("/announcements", handlers.AdminAnnouncementsPage)
			protected.GET("/announcements/new", handlers.AdminAnnouncementNew)
			protected.GET("/announcements/:id/edit", handlers.AdminAnnouncementEdit)
			protected.POST("/announcements/save", handlers.AdminAnnouncementSave)
			protected.POST("/announcements/:id/delete", handlers.AdminAnnouncementDelete)

			// Discount Codes
			protected.GET("/discount-codes", handlers.AdminDiscountCodesPage)
			protected.GET("/discount-codes/new", handlers.AdminDiscountCodeNew)
			protected.GET("/discount-codes/:id/edit", handlers.AdminDiscountCodeEdit)
			protected.POST("/discount-codes/save", handlers.AdminDiscountCodeSave)
			protected.POST("/discount-codes/:id/delete", handlers.AdminDiscountCodeDelete)

			// Analytics
			protected.GET("/analytics", handlers.AdminAnalyticsPage)

			// Status Items
			protected.GET("/status-items", handlers.AdminStatusItemsPage)
			protected.GET("/status-items/new", handlers.AdminStatusItemNew)
			protected.GET("/status-items/:id/edit", handlers.AdminStatusItemEdit)
			protected.POST("/status-items/save", handlers.AdminStatusItemSave)
			protected.POST("/status-items/:id/delete", handlers.AdminStatusItemDelete)

			// Trust Cards
			protected.GET("/trust-cards", handlers.AdminTrustCardsPage)
			protected.GET("/trust-cards/new", handlers.AdminTrustCardNew)
			protected.GET("/trust-cards/:id/edit", handlers.AdminTrustCardEdit)
			protected.POST("/trust-cards/save", handlers.AdminTrustCardSave)
			protected.POST("/trust-cards/:id/delete", handlers.AdminTrustCardDelete)

			// Plan Comparison
			protected.GET("/plan-comparison", handlers.AdminPlanComparisonPage)
			protected.GET("/plan-comparison/new", handlers.AdminPlanComparisonNew)
			protected.GET("/plan-comparison/:id/edit", handlers.AdminPlanComparisonEdit)
			protected.POST("/plan-comparison/save", handlers.AdminPlanComparisonSave)
			protected.POST("/plan-comparison/:id/delete", handlers.AdminPlanComparisonDelete)

			// Homepage Sections
			protected.GET("/homepage-sections", handlers.AdminHomepageSectionsPage)
			protected.POST("/homepage-sections/save", handlers.AdminHomepageSectionsSave)

			// Campaigns
			protected.GET("/campaigns", handlers.AdminCampaignsPage)
			protected.GET("/campaigns/new", handlers.AdminCampaignNew)
			protected.GET("/campaigns/:id/edit", handlers.AdminCampaignEdit)
			protected.POST("/campaigns/save", handlers.AdminCampaignSave)
			protected.POST("/campaigns/:id/delete", handlers.AdminCampaignDelete)

			// Landing Pages
			protected.GET("/landing-pages", handlers.AdminLandingPagesPage)
			protected.GET("/landing-pages/new", handlers.AdminLandingPageNew)
			protected.GET("/landing-pages/:id/edit", handlers.AdminLandingPageEdit)
			protected.POST("/landing-pages/save", handlers.AdminLandingPageSave)
			protected.POST("/landing-pages/:id/delete", handlers.AdminLandingPageDelete)

			// Popups
			protected.GET("/popups", handlers.AdminPopupsPage)
			protected.GET("/popups/new", handlers.AdminPopupNew)
			protected.GET("/popups/:id/edit", handlers.AdminPopupEdit)
			protected.POST("/popups/save", handlers.AdminPopupSave)
			protected.POST("/popups/:id/delete", handlers.AdminPopupDelete)

			// Admin Users
			protected.GET("/users", handlers.AdminUsersPage)
			protected.GET("/users/new", handlers.AdminUserNew)
			protected.POST("/users/save", handlers.AdminUserSave)
			protected.POST("/users/:id/delete", handlers.AdminUserDelete)

			// DB Backups
			protected.GET("/backups", handlers.AdminBackupsPage)
			protected.POST("/backups/create", handlers.AdminBackupCreate)
			protected.GET("/backups/:id/download", handlers.AdminBackupDownload)
			protected.POST("/backups/:id/delete", handlers.AdminBackupDelete)

			// Maintenance
			protected.GET("/maintenance", handlers.AdminMaintenancePage)
			protected.POST("/maintenance/save", handlers.AdminMaintenanceSave)

			// System
			protected.GET("/system/logs", handlers.AdminSystemLogsPage)
			protected.GET("/system/health", handlers.AdminSystemHealthPage)

			// Telegram Integration
			integrations := protected.Group("/integrations")
			{
				integrations.GET("/telegram", handlers.AdminTelegramPage)
				integrations.POST("/telegram/save", handlers.AdminTelegramSave)
				integrations.POST("/telegram/disable", handlers.AdminTelegramDisable)
				integrations.POST("/telegram/test", handlers.AdminTelegramTest)
				integrations.POST("/telegram/send-test", handlers.AdminTelegramSendTest)
				integrations.POST("/telegram/create-topics", handlers.AdminTelegramCreateTopics)
				integrations.POST("/telegram/daily-report", handlers.AdminTelegramSendDailyReport)
			}
		}
	}

	log.Printf("ZedProxy server starting on %s", *addr)
	if err := r.Run(*addr); err != nil {
		log.Fatalf("server error: %v", err)
	}
}

func getEnvOrDefault(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}

func requestLogger() gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()
		c.Next()
		log.Printf("[%d] %s %s (%v)", c.Writer.Status(), c.Request.Method, c.Request.URL.Path, time.Since(start))
	}
}

func runSelfTest(dbPath, templateDir, staticDir, uploadDir string) {
	ok := true
	check := func(label, val string, pass bool) {
		if pass {
			fmt.Printf("[✓] %s: %s\n", label, val)
		} else {
			fmt.Printf("[✗] %s: %s\n", label, val)
			ok = false
		}
	}
	fmt.Printf("=== ZedProxy Self-Test (version %s) ===\n", Version)

	_, errDB := os.Stat(dbPath)
	check("DB", dbPath, errDB == nil)

	for _, d := range []struct{ label, path string }{
		{"Templates", templateDir},
		{"Static", staticDir},
		{"Uploads", uploadDir},
	} {
		_, err := os.Stat(d.path)
		check(d.label, d.path, err == nil)
	}

	v := models.GetSetting("maintenance_enabled")
	if v == "1" {
		fmt.Println("[!] maintenance_enabled: فعال")
	} else {
		fmt.Printf("[✓] maintenance_enabled: غیرفعال (%q)\n", v)
	}

	var count int
	database.DB.QueryRow("SELECT COUNT(*) FROM admins").Scan(&count)
	check("Admin users", fmt.Sprintf("%d", count), count > 0)

	_, errUpd := os.Stat("/opt/zedproxy/update.sh")
	if errUpd == nil {
		fmt.Println("[✓] update.sh: /opt/zedproxy/update.sh")
	} else {
		fmt.Println("[!] update.sh: /opt/zedproxy/update.sh یافت نشد")
	}

	fmt.Printf("[✓] Build: version=%s date=%s commit=%s\n", Version, BuildDate, GitCommit)

	if ok {
		fmt.Println("=== Self-test PASSED ===")
	} else {
		fmt.Println("=== Self-test FAILED ===")
		os.Exit(1)
	}
}
