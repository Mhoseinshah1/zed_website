package handlers

import (
	"fmt"
	"net/http"
	"strconv"
	"strings"

	"github.com/gin-contrib/sessions"
	"github.com/gin-gonic/gin"
	"golang.org/x/crypto/bcrypt"
	"zedproxy/internal/models"
)

// ─── User panel template helper ───────────────────────

func renderUser(c *gin.Context, name string, data map[string]interface{}) {
	if data == nil {
		data = map[string]interface{}{}
	}
	// Merge base site data
	for k, v := range basePageData("user") {
		if _, exists := data[k]; !exists {
			data[k] = v
		}
	}
	// Inject current user into data
	uid := currentUserID(c)
	if uid > 0 {
		if u, err := models.GetUserByID(uid); err == nil {
			data["CurrentUser"] = u
			data["WalletBalance"] = models.GetWalletBalance(uid)
			data["UnreadCount"] = models.UnreadNotificationCount(uid)
			data["OpenTickets"] = models.OpenTicketCount(uid)
		}
	}
	t, err := getUserTemplate(name)
	if err != nil {
		c.String(http.StatusInternalServerError, "Template error: %v", err)
		return
	}
	c.Header("Content-Type", "text/html; charset=utf-8")
	if err := t.ExecuteTemplate(c.Writer, "user", data); err != nil {
		_ = err
	}
}

// ─── Dashboard ────────────────────────────────────────

func UserDashboard(c *gin.Context) {
	uid := currentUserID(c)
	user, _ := models.GetUserByID(uid)
	activeSvc, _ := models.GetActiveUserService(uid)
	orders, _ := models.GetUserOrders(uid)
	notifications, _ := models.GetUserNotifications(uid)

	var lastOrder *models.UserOrder
	if len(orders) > 0 {
		lastOrder = orders[0]
	}

	unread := 0
	for _, n := range notifications {
		if !n.IsRead() {
			unread++
		}
	}

	renderUser(c, "dashboard", map[string]interface{}{
		"Title":     "داشبورد",
		"User":      user,
		"ActiveSvc": activeSvc,
		"LastOrder": lastOrder,
		"Wallet":    models.GetWalletBalance(uid),
		"Unread":    unread,
		"OpenTkts":  models.OpenTicketCount(uid),
	})
}

// ─── Profile ──────────────────────────────────────────

func UserProfilePage(c *gin.Context) {
	uid := currentUserID(c)
	user, _ := models.GetUserByID(uid)
	sess := sessions.Default(c)

	data := map[string]interface{}{"Title": "پروفایل", "User": user}
	if f := sess.Get("flash_ok"); f != nil {
		data["FlashOK"] = f.(string)
		sess.Delete("flash_ok")
		sess.Save()
	}
	renderUser(c, "profile", data)
}

func UserProfilePost(c *gin.Context) {
	uid := currentUserID(c)

	firstName := strings.TrimSpace(c.PostForm("first_name"))
	lastName := strings.TrimSpace(c.PostForm("last_name"))
	displayName := strings.TrimSpace(c.PostForm("display_name"))
	timezone := strings.TrimSpace(c.PostForm("timezone"))
	country := strings.TrimSpace(c.PostForm("country"))
	device := c.PostForm("primary_device")
	usage := c.PostForm("usage_type")
	email := strings.ToLower(strings.TrimSpace(c.PostForm("email")))
	phone := strings.TrimSpace(c.PostForm("phone"))

	user, _ := models.GetUserByID(uid)
	data := map[string]interface{}{"Title": "پروفایل", "User": user}

	// Validate contact change (check duplicates, skip if unchanged)
	if email != "" && email != user.DisplayEmail() {
		if !strings.Contains(email, "@") {
			data["Error"] = "فرمت ایمیل صحیح نیست"
			renderUser(c, "profile", data)
			return
		}
		if models.EmailExists(email) {
			data["Error"] = "این ایمیل قبلاً ثبت شده است"
			renderUser(c, "profile", data)
			return
		}
		models.UpdateUserContact(uid, email, "")
	}
	if phone != "" && phone != user.DisplayPhone() {
		if models.PhoneExists(phone) {
			data["Error"] = "این شماره موبایل قبلاً ثبت شده است"
			renderUser(c, "profile", data)
			return
		}
		models.UpdateUserContact(uid, "", phone)
	}

	if timezone == "" {
		timezone = "Asia/Tehran"
	}
	models.UpsertUserProfile(uid, firstName, lastName, displayName, timezone, country, device, usage)
	models.LogUserActivity(uid, "profile_updated", "پروفایل به‌روزرسانی شد", models.HashString(c.ClientIP()), c.Request.UserAgent())

	sess := sessions.Default(c)
	sess.Set("flash_ok", "پروفایل با موفقیت به‌روزرسانی شد")
	sess.Save()
	c.Redirect(http.StatusFound, "/user/profile")
}

// ─── Services ────────────────────────────────────────

func UserServicesPage(c *gin.Context) {
	uid := currentUserID(c)
	services, _ := models.GetUserServices(uid)
	renderUser(c, "services", map[string]interface{}{
		"Title":    "سرویس‌های من",
		"Services": services,
	})
}

func UserServiceDetailPage(c *gin.Context) {
	uid := currentUserID(c)
	idStr := c.Param("id")
	id, _ := strconv.ParseInt(idStr, 10, 64)
	svc, err := models.GetUserServiceByID(id, uid)
	if err != nil {
		c.Redirect(http.StatusFound, "/user/services")
		return
	}
	renderUser(c, "service-detail", map[string]interface{}{
		"Title":   "جزئیات سرویس",
		"Service": svc,
	})
}

// ─── Orders ──────────────────────────────────────────

func UserOrdersPage(c *gin.Context) {
	uid := currentUserID(c)
	orders, _ := models.GetUserOrders(uid)
	renderUser(c, "orders", map[string]interface{}{
		"Title":  "سفارش‌های من",
		"Orders": orders,
	})
}

func UserOrderDetailPage(c *gin.Context) {
	uid := currentUserID(c)
	number := c.Param("order_number")
	order, err := models.GetUserOrderByNumber(number, uid)
	if err != nil {
		c.Redirect(http.StatusFound, "/user/orders")
		return
	}
	renderUser(c, "order-detail", map[string]interface{}{
		"Title": "جزئیات سفارش",
		"Order": order,
	})
}

// ─── Wallet ──────────────────────────────────────────

func UserWalletPage(c *gin.Context) {
	uid := currentUserID(c)
	txns, _ := models.GetWalletTransactions(uid)
	balance := models.GetWalletBalance(uid)
	settings := models.GetAllSettings()
	renderUser(c, "wallet", map[string]interface{}{
		"Title":        "کیف پول",
		"Balance":      balance,
		"Transactions": txns,
		"TelegramBot":  settings["telegram_bot"],
	})
}

// ─── Notifications ────────────────────────────────────

func UserNotificationsPage(c *gin.Context) {
	uid := currentUserID(c)
	notifications, _ := models.GetUserNotifications(uid)
	renderUser(c, "notifications", map[string]interface{}{
		"Title":         "اعلان‌ها",
		"Notifications": notifications,
	})
}

func UserMarkNotificationRead(c *gin.Context) {
	uid := currentUserID(c)
	idStr := c.Param("id")
	id, _ := strconv.ParseInt(idStr, 10, 64)
	models.MarkNotificationRead(id, uid)
	c.JSON(http.StatusOK, gin.H{"ok": true})
}

// ─── Tutorials ────────────────────────────────────────

func UserTutorialsPage(c *gin.Context) {
	uid := currentUserID(c)
	user, _ := models.GetUserByID(uid)

	tutorials, _ := models.GetPublishedTutorials()
	settings := models.GetAllSettings()

	var recommended []models.Tutorial
	var rest []models.Tutorial

	preferredDevice := ""
	if user != nil && user.Profile != nil {
		preferredDevice = user.Profile.PrimaryDevice
	}

	platformMap := map[string]string{
		"Android":      "android",
		"iPhone / iOS": "ios",
		"Windows":      "windows",
		"Mac":          "mac",
	}
	preferredPlatform := platformMap[preferredDevice]

	for _, t := range tutorials {
		if preferredPlatform != "" && strings.EqualFold(t.Platform, preferredPlatform) {
			recommended = append(recommended, t)
		} else {
			rest = append(rest, t)
		}
	}

	renderUser(c, "tutorials", map[string]interface{}{
		"Title":           "آموزش نصب",
		"Recommended":     recommended,
		"OtherTutorials":  rest,
		"PreferredDevice": preferredDevice,
		"TelegramBot":     settings["telegram_bot"],
	})
}

// ─── Security ────────────────────────────────────────

func UserSecurityPage(c *gin.Context) {
	uid := currentUserID(c)
	user, _ := models.GetUserByID(uid)
	logs, _ := models.GetUserActivityLogs(uid, 10)
	sess := sessions.Default(c)

	data := map[string]interface{}{
		"Title": "امنیت حساب",
		"User":  user,
		"Logs":  logs,
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
	renderUser(c, "security", data)
}

func UserChangePassword(c *gin.Context) {
	uid := currentUserID(c)
	user, err := models.GetUserByID(uid)
	if err != nil {
		c.Redirect(http.StatusFound, "/user/security")
		return
	}

	current := c.PostForm("current_password")
	newPass := c.PostForm("new_password")
	confirm := c.PostForm("confirm_password")

	sess := sessions.Default(c)

	if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(current)); err != nil {
		sess.Set("flash_err", "رمز عبور فعلی صحیح نیست")
		sess.Save()
		c.Redirect(http.StatusFound, "/user/security")
		return
	}

	if len(newPass) < 8 {
		sess.Set("flash_err", "رمز عبور جدید باید حداقل ۸ کاراکتر باشد")
		sess.Save()
		c.Redirect(http.StatusFound, "/user/security")
		return
	}

	if newPass != confirm {
		sess.Set("flash_err", "رمز عبور جدید و تکرار آن یکسان نیستند")
		sess.Save()
		c.Redirect(http.StatusFound, "/user/security")
		return
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(newPass), 12)
	if err != nil {
		sess.Set("flash_err", "خطای سیستمی")
		sess.Save()
		c.Redirect(http.StatusFound, "/user/security")
		return
	}

	models.UpdateUserPassword(uid, string(hash))
	models.LogUserActivity(uid, "password_changed", "رمز عبور تغییر یافت", models.HashString(c.ClientIP()), c.Request.UserAgent())

	sess.Set("flash_ok", "رمز عبور با موفقیت تغییر یافت")
	sess.Save()
	c.Redirect(http.StatusFound, "/user/security")
}

func UserLogoutAll(c *gin.Context) {
	uid := currentUserID(c)
	models.LogUserActivity(uid, "logout_all", "خروج از همه دستگاه‌ها", models.HashString(c.ClientIP()), c.Request.UserAgent())
	// Revoke all tracked sessions
	models.RevokeAllUserSessions(uid)

	sess := sessions.Default(c)
	sess.Delete("user_id")
	sess.Delete("user_role")
	sess.Save()
	c.Redirect(http.StatusFound, "/auth/login")
}

// ─── Connect Telegram ────────────────────────────────

func UserConnectTelegramPage(c *gin.Context) {
	uid := currentUserID(c)
	user, _ := models.GetUserByID(uid)
	settings := models.GetAllSettings()
	botUsername := settings["customer_telegram_bot_username"]

	data := map[string]interface{}{
		"Title":       "اتصال حساب به تلگرام",
		"User":        user,
		"BotUsername": botUsername,
	}

	sess := sessions.Default(c)
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
	if u := sess.Get("connect_url"); u != nil {
		data["ConnectURL"] = u.(string)
		sess.Delete("connect_url")
		sess.Save()
	}
	renderUser(c, "connect-telegram", data)
}

func UserConnectTelegramCreateToken(c *gin.Context) {
	uid := currentUserID(c)
	settings := models.GetAllSettings()
	botUsername := settings["customer_telegram_bot_username"]

	sess := sessions.Default(c)
	if botUsername == "" {
		sess.Set("flash_err", "ربات مشتری هنوز تنظیم نشده است.")
		sess.Save()
		c.Redirect(http.StatusFound, "/user/connect-telegram")
		return
	}

	token, err := models.CreateTelegramConnectToken(uid)
	if err != nil {
		sess.Set("flash_err", "خطا در ایجاد توکن اتصال")
		sess.Save()
		c.Redirect(http.StatusFound, "/user/connect-telegram")
		return
	}

	connectURL := fmt.Sprintf("https://t.me/%s?start=connect_%s", botUsername, token)
	sess.Set("connect_url", connectURL)
	sess.Save()
	c.Redirect(http.StatusFound, "/user/connect-telegram")
}

func UserDisconnectTelegram(c *gin.Context) {
	uid := currentUserID(c)
	models.DisconnectUserTelegram(uid)
	models.LogUserActivity(uid, "telegram_disconnected", "ارتباط تلگرام قطع شد", models.HashString(c.ClientIP()), c.Request.UserAgent())

	sess := sessions.Default(c)
	sess.Set("flash_ok", "ارتباط تلگرام قطع شد")
	sess.Save()
	c.Redirect(http.StatusFound, "/user/connect-telegram")
}

