package seed

import (
	"log"

	"zedproxy/internal/database"
	"zedproxy/internal/models"

	"golang.org/x/crypto/bcrypt"
)

func Run(adminUsername, adminEmail, adminPassword string) {
	seedAdmin(adminUsername, adminEmail, adminPassword)
	seedSettings()
	seedPlans()
	seedFeatures()
	seedFAQs()
	seedLocations()
	seedPages()
	seedStatusUpdates()
	seedStatusItems()
	seedTrustCards()
	seedCampaigns()
	seedPlanComparisons()
	log.Println("Seed completed successfully")
}

func seedAdmin(username, email, password string) {
	var count int
	database.DB.QueryRow("SELECT COUNT(*) FROM admins").Scan(&count)
	if count > 0 {
		return
	}
	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		log.Fatalf("failed to hash password: %v", err)
	}
	if err := models.CreateAdmin(username, email, string(hash)); err != nil {
		log.Fatalf("failed to create admin: %v", err)
	}
	log.Printf("Admin created: %s / %s", username, email)
}

func seedSettings() {
	defaults := map[string]string{
		"site_name":         "ZedProxy",
		"site_tagline":      "سریع‌ترین و امن‌ترین سرویس پروکسی ایران",
		"site_url":          "https://zedproxy.com",
		"logo_text":         "ZedProxy",
		"logo_image":        "",
		"favicon":           "",
		"og_image":          "",
		"hero_image":        "",
		"hero_title":        "اینترنت آزاد، سریع و امن با ZedProxy",
		"hero_subtitle":     "با بهترین سرویس پروکسی ایران، بدون محدودیت به اینترنت دسترسی داشته باشید. سرعت بالا، پایداری عالی، پشتیبانی ۲۴ ساعته.",
		"hero_cta_text":     "خرید از ربات تلگرام",
		"hero_secondary":    "مشاهده آموزش خرید",
		"telegram_bot":      "https://t.me/zedproxy_bot",
		"telegram_channel":  "https://t.me/Zed_Proxy1",
		"telegram_support":  "https://t.me/zedproxy_support",
		"btn_buy_text":      "خرید اشتراک",
		"btn_support_text":  "پشتیبانی",
		"btn_channel_text":  "کانال تلگرام",
		"seo_title":         "ZedProxy - سریع‌ترین سرویس پروکسی و VPN ایران",
		"seo_description":   "ZedProxy بهترین سرویس پروکسی و فیلترشکن ایران با سرعت بالا، پایداری ۹۹٪ و پشتیبانی شبانه‌روزی. خرید آسان از طریق ربات تلگرام.",
		"gsc_verification":  "",
		"google_analytics":  "",
		"custom_js":         "",
		"primary_color":     "#6366f1",
		"secondary_color":   "#8b5cf6",
		"accent_color":      "#06b6d4",
		"bg_style":          "dark",
		"trust_count_users": "۵۰۰۰+",
		"trust_uptime":      "۹۹٪",
		"trust_speed":       "۱ گیگابیت",
		"trust_support":     "۲۴/۷",
		"footer_text":       "ZedProxy - ارائه‌دهنده خدمات پروکسی و اینترنت آزاد",
		"custom_css":        "",
		"maintenance_mode":  "0",
		"maintenance_msg":   "سرویس در حال بروزرسانی است. به زودی بازمی‌گردیم.",
		"robots_txt_extra":  "",
	}
	for k, v := range defaults {
		var count int
		database.DB.QueryRow("SELECT COUNT(*) FROM settings WHERE key=?", k).Scan(&count)
		if count == 0 {
			models.SetSetting(k, v)
		}
	}
}

func seedPlans() {
	var count int
	database.DB.QueryRow("SELECT COUNT(*) FROM plans").Scan(&count)
	if count > 0 {
		return
	}

	plans := []models.Plan{
		{
			Name:        "برنز",
			Traffic:     "۳۰ گیگابایت",
			Duration:    "۱ ماه",
			Price:       "۴۵,۰۰۰ تومان",
			Badge:       "",
			Description: "مناسب برای استفاده روزانه",
			Features:    []string{"سرعت تا ۵۰۰ مگابیت", "پشتیبانی از همه دستگاه‌ها", "بدون محدودیت سایت", "پشتیبانی آنلاین"},
			IsPopular:   false,
			SortOrder:   1,
			IsActive:    true,
		},
		{
			Name:        "نقره",
			Traffic:     "۷۵ گیگابایت",
			Duration:    "۱ ماه",
			Price:       "۸۵,۰۰۰ تومان",
			Badge:       "پرفروش",
			Description: "بهترین انتخاب برای کاربران روزانه",
			Features:    []string{"سرعت تا ۱ گیگابیت", "پشتیبانی از همه دستگاه‌ها", "بدون محدودیت سایت", "پشتیبانی آنلاین ۲۴/۷", "چند دستگاه همزمان"},
			IsPopular:   true,
			SortOrder:   2,
			IsActive:    true,
		},
		{
			Name:        "طلا",
			Traffic:     "۱۵۰ گیگابایت",
			Duration:    "۱ ماه",
			Price:       "۱۴۵,۰۰۰ تومان",
			Badge:       "پیشنهادی",
			Description: "برای کاربران پرمصرف",
			Features:    []string{"سرعت تا ۱ گیگابیت", "پشتیبانی از همه دستگاه‌ها", "بدون محدودیت سایت", "پشتیبانی اولویت‌دار", "تا ۵ دستگاه همزمان", "لوکیشن‌های پریمیوم"},
			IsPopular:   false,
			SortOrder:   3,
			IsActive:    true,
		},
		{
			Name:        "پلاتینیوم",
			Traffic:     "نامحدود",
			Duration:    "۱ ماه",
			Price:       "۲۲۰,۰۰۰ تومان",
			Badge:       "VIP",
			Description: "بدون هیچ محدودیتی",
			Features:    []string{"ترافیک نامحدود", "سرعت تا ۱ گیگابیت", "پشتیبانی از همه دستگاه‌ها", "پشتیبانی VIP اختصاصی", "تا ۱۰ دستگاه همزمان", "تمام لوکیشن‌های پریمیوم", "آدرس IP اختصاصی"},
			IsPopular:   false,
			SortOrder:   4,
			IsActive:    true,
		},
	}

	for _, p := range plans {
		if err := models.CreatePlan(p); err != nil {
			log.Printf("failed to seed plan: %v", err)
		}
	}
}

func seedFeatures() {
	var count int
	database.DB.QueryRow("SELECT COUNT(*) FROM features").Scan(&count)
	if count > 0 {
		return
	}

	features := []models.Feature{
		{Icon: "⚡", Title: "سرعت فوق‌العاده", Description: "با پهنای باند ۱ گیگابیت، از استریم، گیم و دانلود بدون مشکل لذت ببرید", SortOrder: 1, IsActive: true},
		{Icon: "🔒", Title: "امنیت بالا", Description: "رمزگذاری پیشرفته برای حفاظت کامل از اطلاعات شما در برابر هرگونه نظارت", SortOrder: 2, IsActive: true},
		{Icon: "🌍", Title: "لوکیشن‌های متعدد", Description: "دسترسی به سرورهای متعدد در کشورهای مختلف جهان برای بهترین تجربه", SortOrder: 3, IsActive: true},
		{Icon: "📱", Title: "همه دستگاه‌ها", Description: "iOS، اندروید، ویندوز، مک، لینوکس - روی تمام دستگاه‌های شما کار می‌کند", SortOrder: 4, IsActive: true},
		{Icon: "🕐", Title: "پشتیبانی ۲۴/۷", Description: "تیم پشتیبانی ما همیشه آماده پاسخگویی به سوالات و حل مشکلات شماست", SortOrder: 5, IsActive: true},
		{Icon: "💰", Title: "قیمت مناسب", Description: "بهترین کیفیت با مناسب‌ترین قیمت. طرح‌های مختلف برای همه نیازها", SortOrder: 6, IsActive: true},
		{Icon: "🚀", Title: "اتصال فوری", Description: "راه‌اندازی سریع و آسان. در کمتر از ۵ دقیقه متصل شوید", SortOrder: 7, IsActive: true},
		{Icon: "🛡️", Title: "پایداری ۹۹٪", Description: "زیرساخت قدرتمند ما تضمین می‌کند که همیشه متصل باشید", SortOrder: 8, IsActive: true},
	}

	for _, f := range features {
		if err := models.CreateFeature(f); err != nil {
			log.Printf("failed to seed feature: %v", err)
		}
	}
}

func seedFAQs() {
	var count int
	database.DB.QueryRow("SELECT COUNT(*) FROM faqs").Scan(&count)
	if count > 0 {
		return
	}

	faqs := []models.FAQ{
		{Question: "چطور می‌توانم ZedProxy را خریداری کنم؟", Answer: "خرید بسیار ساده است! کافیست به ربات تلگرام ما به آدرس @zedproxy_bot مراجعه کنید، طرح مورد نظر خود را انتخاب کرده و پرداخت را انجام دهید. تنظیمات به صورت خودکار برای شما ارسال می‌شود.", Category: "خرید", SortOrder: 1, IsActive: true, ShowOnHomepage: true, ShowOnFAQ: true},
		{Question: "آیا ZedProxy روی موبایل هم کار می‌کند؟", Answer: "بله! ZedProxy کاملاً با iOS، اندروید، ویندوز، مک و لینوکس سازگار است. می‌توانید روی تمام دستگاه‌های خود از آن استفاده کنید.", Category: "فنی", SortOrder: 2, IsActive: true, ShowOnHomepage: true, ShowOnFAQ: true},
		{Question: "آیا سرعت اینترنت کاهش می‌یابد؟", Answer: "خیر! با زیرساخت ۱ گیگابیتی ما، تفاوت سرعت بسیار ناچیز است. اکثر کاربران ما حتی سرعت بهتری نسبت به قبل تجربه می‌کنند.", Category: "فنی", SortOrder: 3, IsActive: true, ShowOnHomepage: true, ShowOnFAQ: true},
		{Question: "آیا اطلاعاتم امن است؟", Answer: "کاملاً! ما از رمزگذاری سطح نظامی استفاده می‌کنیم. هیچ‌گونه لاگ از فعالیت‌های شما نگه نمی‌داریم. حریم خصوصی شما برای ما اولویت است.", Category: "امنیت", SortOrder: 4, IsActive: true, ShowOnHomepage: false, ShowOnFAQ: true},
		{Question: "اگر مشکلی داشتم، پشتیبانی چطور انجام می‌شود؟", Answer: "تیم پشتیبانی ما ۲۴ ساعته و ۷ روز هفته از طریق تلگرام در دسترس است. معمولاً در کمتر از ۳۰ دقیقه پاسخ می‌دهیم.", Category: "پشتیبانی", SortOrder: 5, IsActive: true, ShowOnHomepage: true, ShowOnFAQ: true},
		{Question: "آیا می‌توانم چند دستگاه را همزمان متصل کنم؟", Answer: "بله! بسته به طرح انتخابی، می‌توانید از ۱ تا ۱۰ دستگاه را به صورت همزمان متصل کنید.", Category: "فنی", SortOrder: 6, IsActive: true, ShowOnHomepage: false, ShowOnFAQ: true},
		{Question: "روش‌های پرداخت کدامند؟", Answer: "ما از کارت‌های بانکی ایرانی، کیف‌پول‌های اینترنتی و ارزهای دیجیتال پشتیبانی می‌کنیم. پرداخت از طریق ربات تلگرام ما به سادگی انجام می‌شود.", Category: "خرید", SortOrder: 7, IsActive: true, ShowOnHomepage: false, ShowOnFAQ: true},
		{Question: "آیا امکان بازگشت وجه وجود دارد؟", Answer: "بله! اگر در ۲۴ ساعت اول از سرویس راضی نبودید، کاملاً وجه شما را بازگشت می‌دهیم. رضایت شما برای ما مهم است.", Category: "خرید", SortOrder: 8, IsActive: true, ShowOnHomepage: true, ShowOnFAQ: true},
		{Question: "کانفیگ از ربات تلگرام چه پروتکل‌هایی دارد؟", Answer: "ZedProxy از پروتکل‌های V2Ray، Vmess، Vless، Trojan و Shadowsocks پشتیبانی می‌کند. پس از خرید از ربات @zedproxy_bot، کانفیگ مناسب دستگاه شما ارسال می‌شود.", Category: "فنی", SortOrder: 9, IsActive: true, ShowOnHomepage: false, ShowOnFAQ: true},
		{Question: "آیا سرعت در ساعات پیک هم حفظ می‌شود؟", Answer: "بله! زیرساخت ما طوری طراحی شده که حتی در ساعات اوج مصرف نیز سرعت مطلوب را تجربه کنید.", Category: "فنی", SortOrder: 10, IsActive: true, ShowOnHomepage: false, ShowOnFAQ: true},
	}

	for _, f := range faqs {
		if err := models.CreateFAQ(f); err != nil {
			log.Printf("failed to seed faq: %v", err)
		}
	}
}

func seedLocations() {
	var count int
	database.DB.QueryRow("SELECT COUNT(*) FROM locations").Scan(&count)
	if count > 0 {
		return
	}

	locations := []models.Location{
		{Name: "آلمان", Flag: "🇩🇪", Code: "DE", Speed: "۱ گیگابیت", IsActive: true, SortOrder: 1},
		{Name: "هلند", Flag: "🇳🇱", Code: "NL", Speed: "۱ گیگابیت", IsActive: true, SortOrder: 2},
		{Name: "فرانسه", Flag: "🇫🇷", Code: "FR", Speed: "۱ گیگابیت", IsActive: true, SortOrder: 3},
		{Name: "انگلستان", Flag: "🇬🇧", Code: "GB", Speed: "۱ گیگابیت", IsActive: true, SortOrder: 4},
		{Name: "آمریکا", Flag: "🇺🇸", Code: "US", Speed: "۱ گیگابیت", IsActive: true, SortOrder: 5},
		{Name: "کانادا", Flag: "🇨🇦", Code: "CA", Speed: "۱ گیگابیت", IsActive: true, SortOrder: 6},
		{Name: "ترکیه", Flag: "🇹🇷", Code: "TR", Speed: "۱ گیگابیت", IsActive: true, SortOrder: 7},
		{Name: "لهستان", Flag: "🇵🇱", Code: "PL", Speed: "۱ گیگابیت", IsActive: true, SortOrder: 8},
		{Name: "سوئد", Flag: "🇸🇪", Code: "SE", Speed: "۱ گیگابیت", IsActive: true, SortOrder: 9},
		{Name: "سنگاپور", Flag: "🇸🇬", Code: "SG", Speed: "۱ گیگابیت", IsActive: true, SortOrder: 10},
		{Name: "ژاپن", Flag: "🇯🇵", Code: "JP", Speed: "۱ گیگابیت", IsActive: true, SortOrder: 11},
		{Name: "امارات", Flag: "🇦🇪", Code: "AE", Speed: "۱ گیگابیت", IsActive: true, SortOrder: 12},
	}

	for _, l := range locations {
		if err := models.CreateLocation(l); err != nil {
			log.Printf("failed to seed location: %v", err)
		}
	}
}

func seedPages() {
	pages := []models.Page{
		{
			Slug:            "terms",
			Title:           "شرایط و ضوابط استفاده",
			MetaTitle:       "شرایط و ضوابط - ZedProxy",
			MetaDescription: "شرایط و ضوابط استفاده از خدمات ZedProxy را مطالعه کنید.",
			Content: `<h2>شرایط و ضوابط استفاده از خدمات ZedProxy</h2>
<p>با استفاده از خدمات ZedProxy، شما با شرایط و ضوابط زیر موافقت می‌کنید.</p>
<h3>۱. پذیرش شرایط</h3>
<p>استفاده از خدمات ZedProxy به منزله پذیرش کامل این شرایط است.</p>
<h3>۲. استفاده مجاز</h3>
<p>خدمات ZedProxy تنها برای استفاده قانونی و شخصی ارائه می‌شود. استفاده برای فعالیت‌های غیرقانونی ممنوع است.</p>
<h3>۳. حریم خصوصی</h3>
<p>ما هیچ‌گونه اطلاعاتی درباره فعالیت‌های اینترنتی شما ذخیره نمی‌کنیم.</p>
<h3>۴. پرداخت و بازگشت وجه</h3>
<p>تمام پرداخت‌ها قطعی هستند مگر اینکه در ۲۴ ساعت اول رضایت نداشته باشید.</p>
<h3>۵. محدودیت‌ها</h3>
<p>اشتراک‌گذاری حساب با دیگران ممنوع است.</p>`,
		},
		{
			Slug:            "privacy",
			Title:           "سیاست حریم خصوصی",
			MetaTitle:       "سیاست حریم خصوصی - ZedProxy",
			MetaDescription: "سیاست حریم خصوصی ZedProxy - چگونه از اطلاعات شما محافظت می‌کنیم.",
			Content: `<h2>سیاست حریم خصوصی ZedProxy</h2>
<p>ZedProxy متعهد به حفاظت از حریم خصوصی کاربران خود است.</p>
<h3>اطلاعاتی که جمع‌آوری می‌کنیم</h3>
<ul>
<li>نام کاربری تلگرام برای ارائه خدمات</li>
<li>اطلاعات پرداخت برای پردازش تراکنش‌ها</li>
</ul>
<h3>اطلاعاتی که جمع‌آوری نمی‌کنیم</h3>
<ul>
<li>تاریخچه مرور اینترنت</li>
<li>محتوای ارتباطات شما</li>
<li>هیچ‌گونه لاگ از فعالیت‌های اینترنتی</li>
</ul>
<h3>امنیت داده‌ها</h3>
<p>تمام اطلاعات شما با رمزگذاری پیشرفته محافظت می‌شود.</p>`,
		},
	}

	for _, p := range pages {
		if err := models.UpsertPage(p); err != nil {
			log.Printf("failed to seed page: %v", err)
		}
	}
}

func seedStatusUpdates() {
	var count int
	database.DB.QueryRow("SELECT COUNT(*) FROM status_updates").Scan(&count)
	if count > 0 {
		return
	}

	updates := []models.StatusUpdate{
		{
			Title:   "تمام سرویس‌ها عملیاتی هستند",
			Content: "سرویس‌های ZedProxy در تمام لوکیشن‌ها به خوبی کار می‌کنند.",
			Status:  "success",
		},
		{
			Title:   "لوکیشن جدید: ژاپن اضافه شد",
			Content: "لوکیشن جدید ژاپن با سرعت ۱ گیگابیت به مجموعه ZedProxy اضافه شد.",
			Status:  "info",
		},
	}

	for _, u := range updates {
		if err := models.CreateStatusUpdate(u); err != nil {
			log.Printf("failed to seed status update: %v", err)
		}
	}
}

func seedStatusItems() {
	var count int
	database.DB.QueryRow("SELECT COUNT(*) FROM status_items").Scan(&count)
	if count > 0 {
		return
	}

	items := []models.StatusItem{
		{Name: "سرورهای آلمان", ServiceType: "server", Status: "operational", Description: "سرعت ۱ گیگابیت", SortOrder: 1, IsActive: true},
		{Name: "سرورهای آمریکا", ServiceType: "server", Status: "operational", Description: "سرعت ۱ گیگابیت", SortOrder: 2, IsActive: true},
		{Name: "لوکیشن یوتیوب", ServiceType: "service", Status: "operational", Description: "پخش ۴K بدون بافر", SortOrder: 3, IsActive: true},
		{Name: "ربات تلگرام", ServiceType: "bot", Status: "operational", Description: "خرید و تمدید سرویس", SortOrder: 4, IsActive: true},
		{Name: "پشتیبانی آنلاین", ServiceType: "support", Status: "operational", Description: "پاسخگو ۲۴/۷", SortOrder: 5, IsActive: true},
		{Name: "درگاه پرداخت", ServiceType: "payment", Status: "operational", Description: "پرداخت امن", SortOrder: 6, IsActive: true},
	}

	for _, s := range items {
		if err := models.CreateStatusItem(s); err != nil {
			log.Printf("failed to seed status item: %v", err)
		}
	}
}

func seedTrustCards() {
	var count int
	database.DB.QueryRow("SELECT COUNT(*) FROM trust_cards").Scan(&count)
	if count > 0 {
		return
	}

	cards := []models.TrustCard{
		{Icon: "👥", Title: "۵۰۰۰+ کاربر فعال", Description: "هزاران کاربر روزانه به ZedProxy اعتماد می‌کنند و از خدمات ما راضی هستند", SortOrder: 1, IsActive: true},
		{Icon: "⏱️", Title: "پشتیبانی زیر ۳۰ دقیقه", Description: "تیم پشتیبانی ما به‌طور میانگین در کمتر از ۳۰ دقیقه به پیام‌ها پاسخ می‌دهد", SortOrder: 2, IsActive: true},
		{Icon: "🔄", Title: "۳ سال سابقه فعالیت", Description: "از سال ۱۴۰۱ تاکنون خدمات پروکسی باکیفیت ارائه می‌دهیم", SortOrder: 3, IsActive: true},
		{Icon: "🛡️", Title: "بدون لاگ و کاملاً خصوصی", Description: "هیچ‌گونه اطلاعاتی از فعالیت اینترنتی شما ذخیره نمی‌شود", SortOrder: 4, IsActive: true},
	}

	for _, c := range cards {
		if err := models.CreateTrustCard(c); err != nil {
			log.Printf("failed to seed trust card: %v", err)
		}
	}
}

func seedCampaigns() {
	var count int
	database.DB.QueryRow("SELECT COUNT(*) FROM campaigns").Scan(&count)
	if count > 0 {
		return
	}

	campaigns := []models.Campaign{
		{
			Slug:            "off393",
			Title:           "جشنواره تخفیف ۳۰٪",
			Subtitle:        "فقط تا آخر ماه",
			Description:     "<p>به مناسبت سالگرد ZedProxy، تمام پلن‌ها با ۳۰٪ تخفیف در دسترس هستند. همین الان از ربات تلگرام خرید کنید.</p>",
			DiscountCode:    "ZED30",
			DiscountPercent: 30,
			CTAText:         "خرید با تخفیف از ربات",
			MetaTitle:       "تخفیف ۳۰٪ ZedProxy - جشنواره خرید",
			MetaDescription: "با کد ZED30 سی درصد تخفیف روی تمام پلن‌های ZedProxy دریافت کنید.",
			IsActive:        true,
		},
		{
			Slug:            "youtube",
			Title:           "یوتیوب بدون بافر",
			Subtitle:        "پخش ۴K با ZedProxy",
			Description:     "<p>با ZedProxy یوتیوب را با کیفیت ۴K و بدون بافر تماشا کنید. لوکیشن اختصاصی یوتیوب با سرعت بالا.</p>",
			DiscountCode:    "YTUBE",
			DiscountPercent: 15,
			CTAText:         "خرید پلن یوتیوب",
			MetaTitle:       "فیلترشکن یوتیوب - ZedProxy",
			MetaDescription: "با ZedProxy یوتیوب را در بهترین کیفیت تماشا کنید.",
			IsActive:        true,
		},
	}

	for _, c := range campaigns {
		if err := models.CreateCampaign(c); err != nil {
			log.Printf("failed to seed campaign: %v", err)
		}
	}
}

func seedPlanComparisons() {
	var count int
	database.DB.QueryRow("SELECT COUNT(*) FROM plan_comparisons").Scan(&count)
	if count > 0 {
		return
	}

	comparisons := []models.PlanComparison{
		{FeatureName: "حجم ترافیک", BronzeValue: "۳۰ گیگ", SilverValue: "۷۵ گیگ", GoldValue: "۱۵۰ گیگ", PlatinumValue: "نامحدود", SortOrder: 1},
		{FeatureName: "سرعت", BronzeValue: "۵۰۰ مگابیت", SilverValue: "۱ گیگابیت", GoldValue: "۱ گیگابیت", PlatinumValue: "۱ گیگابیت", SortOrder: 2},
		{FeatureName: "تعداد دستگاه", BronzeValue: "۲", SilverValue: "۳", GoldValue: "۵", PlatinumValue: "۱۰", SortOrder: 3},
		{FeatureName: "پشتیبانی", BronzeValue: "عادی", SilverValue: "۲۴/۷", GoldValue: "اولویت‌دار", PlatinumValue: "VIP", SortOrder: 4},
		{FeatureName: "لوکیشن‌های پریمیوم", BronzeValue: "❌", SilverValue: "❌", GoldValue: "✅", PlatinumValue: "✅", SortOrder: 5},
		{FeatureName: "IP اختصاصی", BronzeValue: "❌", SilverValue: "❌", GoldValue: "❌", PlatinumValue: "✅", SortOrder: 6},
	}

	for _, c := range comparisons {
		if err := models.CreatePlanComparison(c); err != nil {
			log.Printf("failed to seed plan comparison: %v", err)
		}
	}
}
