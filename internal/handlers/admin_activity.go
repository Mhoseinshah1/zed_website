package handlers

import (
	"time"

	"github.com/gin-contrib/sessions"
	"github.com/gin-gonic/gin"
	"zedproxy/internal/database"
)

type ActivityLog struct {
	ID            int64
	AdminUsername string
	Action        string
	Details       string
	IP            string
	CreatedAt     time.Time
}

func LogAdminActivity(c *gin.Context, action, details string) {
	sess := sessions.Default(c)
	username := ""
	if u := sess.Get("admin_username"); u != nil {
		username = u.(string)
	}
	ip := c.ClientIP()
	database.DB.Exec(`INSERT INTO admin_activity_logs (admin_username, action, details, ip) VALUES (?,?,?,?)`,
		username, action, details, ip)
}

func AdminActivityPage(c *gin.Context) {
	rows, _ := database.DB.Query(`SELECT id, admin_username, action, details, ip, created_at FROM admin_activity_logs ORDER BY created_at DESC LIMIT 200`)
	var logs []ActivityLog
	if rows != nil {
		defer rows.Close()
		for rows.Next() {
			var l ActivityLog
			rows.Scan(&l.ID, &l.AdminUsername, &l.Action, &l.Details, &l.IP, &l.CreatedAt)
			logs = append(logs, l)
		}
	}
	data := adminData(c, "system-logs")
	data["Title"] = "لاگ فعالیت‌ها"
	data["Logs"] = logs
	renderAdmin(c, "activity", data)
}
