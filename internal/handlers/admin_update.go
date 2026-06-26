package handlers

import (
	"fmt"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"time"

	"github.com/gin-contrib/sessions"
	"github.com/gin-gonic/gin"
	"zedproxy/internal/database"
	"zedproxy/internal/models"
)

type UpdateJob struct {
	ID          int64
	JobType     string
	Status      string
	TriggeredBy string
	LogPath     string
	StartedAt   time.Time
	FinishedAt  *time.Time
}

func AdminUpdatePage(c *gin.Context) {
	sess := sessions.Default(c)
	role := sess.Get("role")
	if role == nil || role.(string) != "owner" {
		c.Redirect(http.StatusFound, "/zed-admin")
		return
	}

	rows, _ := database.DB.Query(`SELECT id, job_type, status, triggered_by, log_path, started_at, finished_at FROM update_jobs ORDER BY started_at DESC LIMIT 20`)
	var jobs []UpdateJob
	if rows != nil {
		defer rows.Close()
		for rows.Next() {
			var j UpdateJob
			rows.Scan(&j.ID, &j.JobType, &j.Status, &j.TriggeredBy, &j.LogPath, &j.StartedAt, &j.FinishedAt)
			jobs = append(jobs, j)
		}
	}

	data := adminData(c, "update")
	data["Title"] = "آپدیت و نسخه"
	data["Version"] = AppVersion
	data["BuildDate"] = ""
	data["GitCommit"] = ""
	data["Jobs"] = jobs
	data["UpdatesLocked"] = models.GetSetting("updates_locked") == "1"
	data["UpdatesLockedReason"] = models.GetSetting("updates_locked_reason")
	renderAdmin(c, "update", data)
}

func AdminUpdateRun(c *gin.Context) {
	if !isOwner(c) {
		c.Redirect(http.StatusFound, "/zed-admin")
		return
	}
	confirm := c.PostForm("confirm")
	if confirm != "UPDATE" {
		sess := sessions.Default(c)
		sess.AddFlash("برای تایید، کلمه UPDATE را وارد کنید", "ok")
		sess.Save()
		c.Redirect(http.StatusFound, "/zed-admin/system/update")
		return
	}
	if models.GetSetting("updates_locked") == "1" {
		sess := sessions.Default(c)
		sess.AddFlash("آپدیت قفل است: "+models.GetSetting("updates_locked_reason"), "ok")
		sess.Save()
		c.Redirect(http.StatusFound, "/zed-admin/system/update")
		return
	}
	runUpdateJob(c, "update", "/opt/zedproxy/update.sh")
}

func AdminUpdateRollback(c *gin.Context) {
	if !isOwner(c) {
		c.Redirect(http.StatusFound, "/zed-admin")
		return
	}
	confirm := c.PostForm("confirm")
	if confirm != "ROLLBACK" {
		sess := sessions.Default(c)
		sess.AddFlash("برای تایید، کلمه ROLLBACK را وارد کنید", "ok")
		sess.Save()
		c.Redirect(http.StatusFound, "/zed-admin/system/update")
		return
	}
	runUpdateJob(c, "rollback", "/opt/zedproxy/rollback.sh")
}

func AdminUpdateLock(c *gin.Context) {
	if !isOwner(c) {
		c.Redirect(http.StatusFound, "/zed-admin")
		return
	}
	reason := c.PostForm("reason")
	models.SetSetting("updates_locked", "1")
	models.SetSetting("updates_locked_reason", reason)
	LogAdminActivity(c, "update_lock", "آپدیت قفل شد: "+reason)
	sess := sessions.Default(c)
	sess.AddFlash("آپدیت قفل شد", "ok")
	sess.Save()
	c.Redirect(http.StatusFound, "/zed-admin/system/update")
}

func AdminUpdateUnlock(c *gin.Context) {
	if !isOwner(c) {
		c.Redirect(http.StatusFound, "/zed-admin")
		return
	}
	models.SetSetting("updates_locked", "0")
	models.SetSetting("updates_locked_reason", "")
	LogAdminActivity(c, "update_unlock", "قفل آپدیت برداشته شد")
	sess := sessions.Default(c)
	sess.AddFlash("قفل آپدیت برداشته شد", "ok")
	sess.Save()
	c.Redirect(http.StatusFound, "/zed-admin/system/update")
}

func AdminUpdateCheck(c *gin.Context) {
	if !isOwner(c) {
		c.Redirect(http.StatusFound, "/zed-admin")
		return
	}
	sess := sessions.Default(c)
	sess.AddFlash("بررسی نسخه جدید در حال انجام است...", "ok")
	sess.Save()
	c.Redirect(http.StatusFound, "/zed-admin/system/update")
}

func runUpdateJob(c *gin.Context, jobType, script string) {
	sess := sessions.Default(c)
	username := ""
	if u := sess.Get("admin_username"); u != nil {
		username = u.(string)
	}

	logDir := "/opt/zedproxy/data/logs"
	os.MkdirAll(logDir, 0755)
	logPath := filepath.Join(logDir, fmt.Sprintf("%s_%d.log", jobType, time.Now().Unix()))

	var jobID int64
	database.DB.QueryRow(`INSERT INTO update_jobs (job_type, status, triggered_by, log_path) VALUES (?,?,?,?) RETURNING id`,
		jobType, "running", username, logPath).Scan(&jobID)

	go func() {
		f, err := os.Create(logPath)
		if err == nil {
			defer f.Close()
			cmd := exec.Command("/bin/bash", script)
			cmd.Stdout = f
			cmd.Stderr = f
			err = cmd.Run()
		}
		status := "done"
		if err != nil {
			status = "failed"
		}
		database.DB.Exec(`UPDATE update_jobs SET status=?, finished_at=CURRENT_TIMESTAMP WHERE id=?`, status, jobID)
	}()

	LogAdminActivity(c, jobType+"_triggered", fmt.Sprintf("%s اجرا شد", script))
	sess.AddFlash(fmt.Sprintf("عملیات %s آغاز شد (job #%d)", jobType, jobID), "ok")
	sess.Save()
	c.Redirect(http.StatusFound, "/zed-admin/system/update")
}

func isOwner(c *gin.Context) bool {
	sess := sessions.Default(c)
	role := sess.Get("role")
	return role != nil && role.(string) == "owner"
}
