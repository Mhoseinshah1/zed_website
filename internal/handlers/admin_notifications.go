package handlers

import (
	"net/http"
	"strconv"
	"time"

	"github.com/gin-contrib/sessions"
	"github.com/gin-gonic/gin"
	"zedproxy/internal/database"
)

type AdminNotification struct {
	ID        int64
	UserID    int64
	UserEmail string
	UserPhone string
	Title     string
	Message   string
	Type      string
	ReadAt    *time.Time
	CreatedAt time.Time
}

func AdminUserNotificationsPage(c *gin.Context) {
	rows, _ := database.DB.Query(`
		SELECT n.id, n.user_id, COALESCE(u.email,''), COALESCE(u.phone,''),
		n.title, n.message, n.type, n.read_at, n.created_at
		FROM user_notifications n
		LEFT JOIN users u ON u.id = n.user_id
		ORDER BY n.created_at DESC LIMIT 200`)

	var notifs []AdminNotification
	if rows != nil {
		defer rows.Close()
		for rows.Next() {
			var n AdminNotification
			var readAt *time.Time
			rows.Scan(&n.ID, &n.UserID, &n.UserEmail, &n.UserPhone,
				&n.Title, &n.Message, &n.Type, &readAt, &n.CreatedAt)
			n.ReadAt = readAt
			notifs = append(notifs, n)
		}
	}

	data := adminData(c, "user-notifications")
	data["Title"] = "اعلان‌های کاربران"
	data["Notifications"] = notifs
	renderAdmin(c, "user-notifications", data)
}

func AdminUserNotificationNewPage(c *gin.Context) {
	rows, _ := database.DB.Query(`SELECT id, COALESCE(email,''), COALESCE(phone,'') FROM users WHERE deleted_at IS NULL ORDER BY created_at DESC`)
	type SimpleUser struct {
		ID    int64
		Email string
		Phone string
	}
	var users []SimpleUser
	if rows != nil {
		defer rows.Close()
		for rows.Next() {
			var u SimpleUser
			rows.Scan(&u.ID, &u.Email, &u.Phone)
			users = append(users, u)
		}
	}

	data := adminData(c, "user-notifications")
	data["Title"] = "ارسال اعلان جدید"
	data["Users"] = users
	renderAdmin(c, "notification-new", data)
}

func AdminUserNotificationSend(c *gin.Context) {
	targetType := c.PostForm("target_type")
	userIDStr := c.PostForm("user_id")
	title := c.PostForm("title")
	message := c.PostForm("message")
	notifType := c.PostForm("type")
	if notifType == "" {
		notifType = "info"
	}

	sess := sessions.Default(c)

	if targetType == "all" {
		rows, err := database.DB.Query(`SELECT id FROM users WHERE deleted_at IS NULL AND status='active'`)
		if err != nil {
			sess.AddFlash("خطا در ارسال اعلان", "ok")
			sess.Save()
			c.Redirect(http.StatusFound, "/zed-admin/user-notifications")
			return
		}
		defer rows.Close()
		count := 0
		for rows.Next() {
			var uid int64
			rows.Scan(&uid)
			database.DB.Exec(`INSERT INTO user_notifications (user_id, title, message, type) VALUES (?,?,?,?)`,
				uid, title, message, notifType)
			count++
		}
		LogAdminActivity(c, "send_notification_all", "ارسال اعلان به همه کاربران: "+title)
		sess.AddFlash("اعلان با موفقیت به "+strconv.Itoa(count)+" کاربر ارسال شد", "ok")
	} else {
		userID, _ := strconv.ParseInt(userIDStr, 10, 64)
		if userID == 0 {
			sess.AddFlash("کاربر انتخاب نشده", "ok")
			sess.Save()
			c.Redirect(http.StatusFound, "/zed-admin/user-notifications/new")
			return
		}
		database.DB.Exec(`INSERT INTO user_notifications (user_id, title, message, type) VALUES (?,?,?,?)`,
			userID, title, message, notifType)
		LogAdminActivity(c, "send_notification_user", "ارسال اعلان به کاربر "+userIDStr+": "+title)
		sess.AddFlash("اعلان با موفقیت ارسال شد", "ok")
	}
	sess.Save()
	c.Redirect(http.StatusFound, "/zed-admin/user-notifications")
}
