package main

import (
	"flag"
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
	"zedproxy/internal/seed"
)

func main() {
	var (
		addr          = flag.String("addr", ":8080", "listen address")
		dbPath        = flag.String("db", "./data/zedproxy.db", "SQLite database path")
		templateDir   = flag.String("templates", "./templates", "templates directory")
		staticDir     = flag.String("static", "./static", "static files directory")
		uploadDirFlag = flag.String("uploads", "./static/uploads", "upload directory")
		sessionSecret = flag.String("secret", getEnvOrDefault("SESSION_SECRET", "change-me-in-production"), "session secret")
		dev           = flag.Bool("dev", false, "development mode (disable template cache)")
		seedFlag      = flag.Bool("seed", false, "seed the database with default content")
		adminUser     = flag.String("admin-user", "admin", "admin username for seeding")
		adminEmail    = flag.String("admin-email", "admin@zedproxy.com", "admin email for seeding")
		adminPass     = flag.String("admin-pass", "", "admin password for seeding")
	)
	flag.Parse()

	// Init DB
	database.Init(*dbPath)

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
	handlers.SetUploadDir(*uploadDirFlag)

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
		Secure:   false, // set to true behind HTTPS
		SameSite: http.SameSiteLaxMode,
	})
	r.Use(sessions.Sessions("zedproxy_session", store))

	// Static files
	r.Static("/static", *staticDir)
	r.Static("/uploads", *uploadDirFlag)

	// Ensure uploads directory exists
	os.MkdirAll(*uploadDirFlag, 0755)

	// Favicon
	faviconPath := filepath.Join(*staticDir, "favicon.ico")
	r.GET("/favicon.ico", func(c *gin.Context) {
		if _, err := os.Stat(faviconPath); err == nil {
			c.File(faviconPath)
		} else {
			c.Status(http.StatusNoContent)
		}
	})

	// SEO
	r.GET("/sitemap.xml", handlers.SitemapXML)
	r.GET("/robots.txt", handlers.RobotsTXT)

	// Public routes
	r.GET("/", handlers.HomePage)
	r.GET("/plans", handlers.PlansPage)
	r.GET("/tutorials", handlers.TutorialsPage)
	r.GET("/tutorials/:slug", handlers.TutorialDetailPage)
	r.GET("/blog", handlers.BlogPage)
	r.GET("/blog/:slug", handlers.BlogPostPage)
	r.GET("/faq", handlers.FAQPage)
	r.GET("/contact", handlers.ContactPage)
	r.GET("/status", handlers.StatusPage)
	r.GET("/terms", handlers.TermsPage)
	r.GET("/privacy", handlers.PrivacyPage)

	// Track clicks API
	r.POST("/api/track", handlers.TrackClick)

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

			// Password
			protected.GET("/password", handlers.AdminPasswordPage)
			protected.POST("/password", handlers.AdminPasswordPost)
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
