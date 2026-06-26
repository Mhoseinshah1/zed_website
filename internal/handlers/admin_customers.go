package handlers

import (
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/gin-contrib/sessions"
	"github.com/gin-gonic/gin"
	"zedproxy/internal/models"
)

// ─── Customer Users List ──────────────────────────────

func AdminCustomersPage(c *gin.Context) {
	search := c.Query("search")
	status := c.Query("status")
	hasTG := c.Query("has_tg")
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))

	result, err := models.ListUsers(models.UserListFilter{
		Search:   search,
		Status:   status,
		HasTG:    hasTG,
		Page:     page,
		PageSize: 25,
	})
	if err != nil {
		result = &models.UserListResult{}
	}

	data := adminData(c, "customers")
	data["Title"] = "مدیریت کاربران"
	data["Users"] = result.Users
	data["Total"] = result.Total
	data["Page"] = result.Page
	data["Pages"] = result.Pages
	data["Search"] = search
	data["Status"] = status
	data["HasTG"] = hasTG
	renderAdmin(c, "customers", data)
}

// ─── Customer Detail ──────────────────────────────────

func AdminCustomerDetailPage(c *gin.Context) {
	publicID := c.Param("public_id")
	user, err := models.GetUserByPublicID(publicID)
	if err != nil {
		c.Redirect(http.StatusFound, "/zed-admin/customers")
		return
	}

	services, _ := models.GetUserServices(user.ID)
	orders, _ := models.GetUserOrders(user.ID)
	txns, _ := models.GetWalletTransactions(user.ID)
	tickets, _ := models.GetUserTickets(user.ID)
	notifications, _ := models.GetUserNotifications(user.ID)
	notes, _ := models.GetUserNotes(user.ID)
	logs, _ := models.GetUserActivityLogs(user.ID, 30)
	balance := models.GetWalletBalance(user.ID)

	sess := sessions.Default(c)
	data := adminData(c, "customers")
	data["Title"] = "پروفایل کاربر"
	data["User"] = user
	data["Services"] = services
	data["Orders"] = orders
	data["Transactions"] = txns
	data["Tickets"] = tickets
	data["Notifications"] = notifications
	data["Notes"] = notes
	data["Logs"] = logs
	data["Balance"] = balance

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
	renderAdmin(c, "customer-detail", data)
}

// ─── Block/Unblock ────────────────────────────────────

func AdminCustomerSetStatus(c *gin.Context) {
	publicID := c.Param("public_id")
	status := c.PostForm("status")

	if status != "active" && status != "blocked" {
		c.Redirect(http.StatusFound, "/zed-admin/customers")
		return
	}

	user, err := models.GetUserByPublicID(publicID)
	if err != nil {
		c.Redirect(http.StatusFound, "/zed-admin/customers")
		return
	}

	models.UpdateUserStatus(user.ID, status)

	sess := sessions.Default(c)
	if status == "blocked" {
		sess.Set("flash_ok", "کاربر مسدود شد")
		models.CreateNotification(user.ID, "حساب مسدود شد", "حساب شما توسط مدیر مسدود شده است.", "error", "")
	} else {
		sess.Set("flash_ok", "مسدودیت کاربر برداشته شد")
		models.CreateNotification(user.ID, "حساب فعال شد", "حساب شما توسط مدیر فعال شد.", "success", "")
	}
	sess.Save()
	c.Redirect(http.StatusFound, "/zed-admin/customers/"+publicID)
}

// ─── Wallet Adjust ────────────────────────────────────

func AdminCustomerWalletAdjust(c *gin.Context) {
	publicID := c.Param("public_id")
	txType := c.PostForm("type")
	amountStr := c.PostForm("amount")
	description := strings.TrimSpace(c.PostForm("description"))

	user, err := models.GetUserByPublicID(publicID)
	sess := sessions.Default(c)
	if err != nil {
		c.Redirect(http.StatusFound, "/zed-admin/customers")
		return
	}

	amount, err := strconv.ParseInt(amountStr, 10, 64)
	if err != nil || amount <= 0 {
		sess.Set("flash_err", "مبلغ نامعتبر")
		sess.Save()
		c.Redirect(http.StatusFound, "/zed-admin/customers/"+publicID)
		return
	}

	validTypes := map[string]bool{"credit": true, "debit": true, "gift": true, "refund": true, "adjustment": true}
	if !validTypes[txType] {
		sess.Set("flash_err", "نوع تراکنش نامعتبر")
		sess.Save()
		c.Redirect(http.StatusFound, "/zed-admin/customers/"+publicID)
		return
	}

	adminID := int64(0)
	if v := sess.Get("admin_id"); v != nil {
		if id, ok := v.(int); ok {
			adminID = int64(id)
		}
	}

	if err := models.AdjustWallet(user.ID, adminID, txType, amount, description); err != nil {
		sess.Set("flash_err", "خطا در تراکنش کیف پول")
	} else {
		sess.Set("flash_ok", "کیف پول به‌روزرسانی شد")
		models.CreateNotification(user.ID, "کیف پول به‌روزرسانی شد", description, "info", "/user/wallet")
	}
	sess.Save()
	c.Redirect(http.StatusFound, "/zed-admin/customers/"+publicID)
}

// ─── Add Note ─────────────────────────────────────────

func AdminCustomerNote(c *gin.Context) {
	publicID := c.Param("public_id")
	note := strings.TrimSpace(c.PostForm("note"))

	user, err := models.GetUserByPublicID(publicID)
	sess := sessions.Default(c)
	if err != nil {
		c.Redirect(http.StatusFound, "/zed-admin/customers")
		return
	}

	if note == "" {
		sess.Set("flash_err", "یادداشت نمی‌تواند خالی باشد")
		sess.Save()
		c.Redirect(http.StatusFound, "/zed-admin/customers/"+publicID)
		return
	}

	adminID := int64(0)
	if v := sess.Get("admin_id"); v != nil {
		if id, ok := v.(int); ok {
			adminID = int64(id)
		}
	}

	models.AddUserNote(user.ID, adminID, note)
	sess.Set("flash_ok", "یادداشت ذخیره شد")
	sess.Save()
	c.Redirect(http.StatusFound, "/zed-admin/customers/"+publicID)
}

// ─── Add Service ──────────────────────────────────────

func AdminCustomerAddService(c *gin.Context) {
	publicID := c.Param("public_id")
	user, err := models.GetUserByPublicID(publicID)
	sess := sessions.Default(c)
	if err != nil {
		c.Redirect(http.StatusFound, "/zed-admin/customers")
		return
	}

	title := strings.TrimSpace(c.PostForm("title"))
	plan := strings.TrimSpace(c.PostForm("plan_name"))
	status := c.PostForm("status")
	location := strings.TrimSpace(c.PostForm("location"))
	subURL := strings.TrimSpace(c.PostForm("subscription_url"))
	trafficGB, _ := strconv.ParseInt(c.PostForm("traffic_gb"), 10, 64)
	startStr := c.PostForm("started_at")
	expStr := c.PostForm("expires_at")

	if title == "" {
		sess.Set("flash_err", "عنوان سرویس الزامی است")
		sess.Save()
		c.Redirect(http.StatusFound, "/zed-admin/customers/"+publicID)
		return
	}
	if status == "" {
		status = "active"
	}

	var startAt, expiresAt *time.Time
	if startStr != "" {
		if t, err := time.Parse("2006-01-02", startStr); err == nil {
			startAt = &t
		}
	}
	if expStr != "" {
		if t, err := time.Parse("2006-01-02", expStr); err == nil {
			expiresAt = &t
		}
	}

	totalBytes := trafficGB * 1073741824

	if err := models.CreateUserService(user.ID, title, plan, status, location, subURL, totalBytes, startAt, expiresAt); err != nil {
		sess.Set("flash_err", "خطا در ایجاد سرویس")
	} else {
		sess.Set("flash_ok", "سرویس با موفقیت اضافه شد")
		models.CreateNotification(user.ID, "سرویس جدید فعال شد", "سرویس «"+title+"» به حساب شما اضافه شد.", "success", "/user/services")
	}
	sess.Save()
	c.Redirect(http.StatusFound, "/zed-admin/customers/"+publicID)
}

// ─── Add Notification ─────────────────────────────────

func AdminCustomerAddNotification(c *gin.Context) {
	publicID := c.Param("public_id")
	user, err := models.GetUserByPublicID(publicID)
	sess := sessions.Default(c)
	if err != nil {
		c.Redirect(http.StatusFound, "/zed-admin/customers")
		return
	}

	title := strings.TrimSpace(c.PostForm("title"))
	message := strings.TrimSpace(c.PostForm("message"))
	nType := c.PostForm("type")
	if nType == "" {
		nType = "info"
	}

	if title == "" || message == "" {
		sess.Set("flash_err", "عنوان و پیام الزامی است")
		sess.Save()
		c.Redirect(http.StatusFound, "/zed-admin/customers/"+publicID)
		return
	}

	models.CreateNotification(user.ID, title, message, nType, "")
	sess.Set("flash_ok", "اعلان ارسال شد")
	sess.Save()
	c.Redirect(http.StatusFound, "/zed-admin/customers/"+publicID)
}
