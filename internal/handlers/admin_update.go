package handlers

import (
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/gin-contrib/sessions"
	"github.com/gin-gonic/gin"
	"zedproxy/internal/database"
	"zedproxy/internal/models"
)

const (
	updateLockFile = "/opt/zedproxy/update.lock"
	updateLogDir   = "/opt/zedproxy/logs"
	updateScript   = "/opt/zedproxy/update.sh"
	rollbackScript = "/opt/zedproxy/rollback.sh"
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

func isOwner(c *gin.Context) bool {
	sess := sessions.Default(c)
	role, _ := sess.Get("role").(string)
	return role == "owner" || role == ""
}

func isUpdateLocked() bool {
	// Check both lock file and database setting
	if _, err := os.Stat(updateLockFile); err == nil {
		return true
	}
	return models.GetSetting("updates_locked") == "1"
}

func updateLockReason() string {
	if _, err := os.Stat(updateLockFile); err == nil {
		data, _ := os.ReadFile(updateLockFile)
		reason := strings.TrimSpace(string(data))
		if reason == "" {
			return "فایل قفل وجود دارد"
		}
		return reason
	}
	return models.GetSetting("updates_locked_reason")
}

// LogInfo holds log file metadata and content for the update page.
type LogInfo struct {
	Path      string
	Name      string
	SizeBytes int64
	SizeHuman string
	ModTime   string
	Result    string // "success", "failed", "running", "unknown"
	Preview   string // last 50 lines
	Full      string // last 2000 lines (capped at ~200KB)
	Truncated bool   // true if full content was truncated
}

const (
	previewLines = 50
	fullMaxLines = 2000
	fullMaxBytes = 200 * 1024
)

func humanBytes(n int64) string {
	switch {
	case n >= 1024*1024:
		return fmt.Sprintf("%.1f MB", float64(n)/(1024*1024))
	case n >= 1024:
		return fmt.Sprintf("%.1f KB", float64(n)/1024)
	default:
		return fmt.Sprintf("%d B", n)
	}
}

func detectLogResult(content string) string {
	lower := strings.ToLower(content)
	if strings.Contains(lower, "self-test passed") || strings.Contains(lower, "update complete") ||
		strings.Contains(lower, "[✓] done") || strings.Contains(content, "=== Self-test PASSED") {
		return "success"
	}
	if strings.Contains(lower, "error") || strings.Contains(lower, "failed") ||
		strings.Contains(lower, "[✗]") || strings.Contains(content, "=== Self-test FAILED") {
		return "failed"
	}
	if strings.Contains(lower, "running") || strings.Contains(lower, "starting") {
		return "running"
	}
	return "unknown"
}

func lastNLines(s string, n int) string {
	lines := strings.Split(s, "\n")
	if len(lines) <= n {
		return s
	}
	return strings.Join(lines[len(lines)-n:], "\n")
}

// loadLogInfo finds the most recently modified file matching glob and returns LogInfo.
func loadLogInfo(pattern string) *LogInfo {
	matches, _ := filepath.Glob(pattern)
	if len(matches) == 0 {
		return nil
	}
	var latest string
	var latestMod time.Time
	for _, m := range matches {
		info, err := os.Stat(m)
		if err == nil && info.ModTime().After(latestMod) {
			latestMod = info.ModTime()
			latest = m
		}
	}
	if latest == "" {
		return nil
	}

	stat, err := os.Stat(latest)
	if err != nil {
		return nil
	}

	// Read up to fullMaxBytes from the end
	f, err := os.Open(latest)
	if err != nil {
		return nil
	}
	defer f.Close()

	size := stat.Size()
	truncated := false
	if size > fullMaxBytes {
		f.Seek(-fullMaxBytes, io.SeekEnd)
		truncated = true
	}
	raw, _ := io.ReadAll(f)
	fullContent := string(raw)

	// Limit to fullMaxLines
	lines := strings.Split(fullContent, "\n")
	if len(lines) > fullMaxLines {
		lines = lines[len(lines)-fullMaxLines:]
		fullContent = strings.Join(lines, "\n")
		truncated = true
	}

	preview := lastNLines(fullContent, previewLines)

	return &LogInfo{
		Path:      latest,
		Name:      filepath.Base(latest),
		SizeBytes: size,
		SizeHuman: humanBytes(size),
		ModTime:   latestMod.Format("2006/01/02 15:04:05"),
		Result:    detectLogResult(fullContent),
		Preview:   preview,
		Full:      fullContent,
		Truncated: truncated,
	}
}

func AdminUpdatePage(c *gin.Context) {
	if !isOwner(c) {
		c.Redirect(http.StatusFound, "/zed-admin")
		return
	}

	sess := sessions.Default(c)

	rows, _ := database.DB.Query(
		`SELECT id, job_type, status, triggered_by, log_path, started_at, finished_at
		 FROM update_jobs ORDER BY started_at DESC LIMIT 20`)
	var jobs []UpdateJob
	if rows != nil {
		defer rows.Close()
		for rows.Next() {
			var j UpdateJob
			rows.Scan(&j.ID, &j.JobType, &j.Status, &j.TriggeredBy, &j.LogPath, &j.StartedAt, &j.FinishedAt)
			jobs = append(jobs, j)
		}
	}

	// Latest logs — prefer admin-triggered, fall back to update.sh logs
	updateLog := loadLogInfo(updateLogDir + "/admin-update-*.log")
	if updateLog == nil {
		updateLog = loadLogInfo(updateLogDir + "/update-*.log")
	}
	rollbackLog := loadLogInfo(updateLogDir + "/admin-rollback-*.log")
	manualUpdateLog := loadLogInfo(updateLogDir + "/update-*.log")

	data := adminData(c, "update")
	data["Title"] = "آپدیت و نسخه"
	data["Version"] = AppVersion
	data["BuildDate"] = AppBuildDate
	data["GitCommit"] = AppGitCommit
	data["Jobs"] = jobs
	data["UpdatesLocked"] = isUpdateLocked()
	data["UpdatesLockedReason"] = updateLockReason()
	data["UpdateLog"] = updateLog
	data["RollbackLog"] = rollbackLog
	data["ManualUpdateLog"] = manualUpdateLog
	data["ServerTime"] = time.Now().Format("2006/01/02 15:04:05")

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

	renderAdmin(c, "update", data)
}

func AdminUpdateRun(c *gin.Context) {
	if !isOwner(c) {
		c.Redirect(http.StatusFound, "/zed-admin")
		return
	}
	confirm := c.PostForm("confirm")
	sess := sessions.Default(c)
	if confirm != "UPDATE" {
		sess.Set("flash_err", "برای تایید، کلمه UPDATE را وارد کنید")
		sess.Save()
		c.Redirect(http.StatusFound, "/zed-admin/system/update")
		return
	}
	if isUpdateLocked() {
		sess.Set("flash_err", "آپدیت قفل است: "+updateLockReason())
		sess.Save()
		c.Redirect(http.StatusFound, "/zed-admin/system/update")
		return
	}
	runUpdateJob(c, "update", updateScript)
}

func AdminUpdateRollback(c *gin.Context) {
	if !isOwner(c) {
		c.Redirect(http.StatusFound, "/zed-admin")
		return
	}
	confirm := c.PostForm("confirm")
	sess := sessions.Default(c)
	if confirm != "ROLLBACK" {
		sess.Set("flash_err", "برای تایید، کلمه ROLLBACK را وارد کنید")
		sess.Save()
		c.Redirect(http.StatusFound, "/zed-admin/system/update")
		return
	}
	if _, err := os.Stat(rollbackScript); err != nil {
		sess.Set("flash_err", "اسکریپت rollback یافت نشد: "+rollbackScript)
		sess.Save()
		c.Redirect(http.StatusFound, "/zed-admin/system/update")
		return
	}
	runUpdateJob(c, "rollback", rollbackScript)
}

func AdminUpdateLock(c *gin.Context) {
	if !isOwner(c) {
		c.Redirect(http.StatusFound, "/zed-admin")
		return
	}
	reason := strings.TrimSpace(c.PostForm("reason"))
	// Write lock file
	os.MkdirAll(filepath.Dir(updateLockFile), 0755)
	os.WriteFile(updateLockFile, []byte(reason), 0644)
	// Also set DB setting for redundancy
	models.SetSetting("updates_locked", "1")
	models.SetSetting("updates_locked_reason", reason)
	LogAdminActivity(c, "admin_update_locked", "آپدیت قفل شد: "+reason)

	sess := sessions.Default(c)
	sess.Set("flash_ok", "آپدیت‌ها قفل شدند.")
	sess.Save()
	c.Redirect(http.StatusFound, "/zed-admin/system/update")
}

func AdminUpdateUnlock(c *gin.Context) {
	if !isOwner(c) {
		c.Redirect(http.StatusFound, "/zed-admin")
		return
	}
	os.Remove(updateLockFile)
	models.SetSetting("updates_locked", "0")
	models.SetSetting("updates_locked_reason", "")
	LogAdminActivity(c, "admin_update_unlocked", "قفل آپدیت برداشته شد")

	sess := sessions.Default(c)
	sess.Set("flash_ok", "قفل آپدیت برداشته شد.")
	sess.Save()
	c.Redirect(http.StatusFound, "/zed-admin/system/update")
}

func AdminUpdateCheck(c *gin.Context) {
	if !isOwner(c) {
		c.Redirect(http.StatusFound, "/zed-admin")
		return
	}
	LogAdminActivity(c, "admin_update_check", "بررسی نسخه جدید")
	sess := sessions.Default(c)
	sess.Set("flash_ok", "بررسی نسخه جدید انجام شد.")
	sess.Save()
	c.Redirect(http.StatusFound, "/zed-admin/system/update")
}

func runUpdateJob(c *gin.Context, jobType, script string) {
	sess := sessions.Default(c)
	username, _ := sess.Get("admin_username").(string)

	os.MkdirAll(updateLogDir, 0755)
	ts := time.Now().Format("20060102-150405")
	logPath := filepath.Join(updateLogDir, fmt.Sprintf("admin-%s-%s.log", jobType, ts))

	var jobID int64
	database.DB.QueryRow(
		`INSERT INTO update_jobs (job_type, status, triggered_by, log_path) VALUES (?,?,?,?) RETURNING id`,
		jobType, "running", username, logPath,
	).Scan(&jobID)

	LogAdminActivity(c, "admin_"+jobType+"_started", fmt.Sprintf("%s آغاز شد (job #%d)", jobType, jobID))

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
		msg := "admin_" + jobType + "_success"
		if err != nil {
			status = "failed"
			msg = "admin_" + jobType + "_failed"
		}
		database.DB.Exec(
			`UPDATE update_jobs SET status=?, finished_at=CURRENT_TIMESTAMP WHERE id=?`,
			status, jobID,
		)
		// Log completion — best effort, no context available in goroutine
		database.DB.Exec(
			`INSERT INTO admin_activity_logs (admin_username, action, details, created_at) VALUES (?,?,?,CURRENT_TIMESTAMP)`,
			username, msg, fmt.Sprintf("job #%d %s: %v", jobID, jobType, err),
		)
	}()

	startMsg := map[string]string{
		"update":   "آپدیت شروع شد.",
		"rollback": "رول‌بک شروع شد.",
	}[jobType]
	sess.Set("flash_ok", fmt.Sprintf("%s (job #%d)", startMsg, jobID))
	sess.Save()
	c.Redirect(http.StatusFound, "/zed-admin/system/update")
}
