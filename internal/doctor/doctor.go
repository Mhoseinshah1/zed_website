package doctor

import (
	"archive/zip"
	"database/sql"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	_ "github.com/mattn/go-sqlite3"
)

// BaseDir is the production install directory.
const BaseDir = "/opt/zedproxy"

// Result holds the outcome of a single check.
type Result struct {
	Check   string
	OK      bool
	Message string
}

// Report holds all results for a doctor/repair run.
type Report struct {
	Mode      string // "doctor" or "repair"
	StartedAt time.Time
	Hostname  string
	Results   []Result
	Issues    []string
	Actions   []string
	LogPath   string
	DryRun    bool
}

func newReport(mode string, dryRun bool) *Report {
	host, _ := os.Hostname()
	ts := time.Now().Format("20060102-150405")
	logDir := filepath.Join(BaseDir, "logs")
	os.MkdirAll(logDir, 0755) //nolint:errcheck
	logPath := filepath.Join(logDir, mode+"-"+ts+".log")
	return &Report{
		Mode:      mode,
		StartedAt: time.Now(),
		Hostname:  host,
		LogPath:   logPath,
		DryRun:    dryRun,
	}
}

func (r *Report) add(check string, ok bool, msg string) {
	r.Results = append(r.Results, Result{Check: check, OK: ok, Message: strings.TrimSpace(msg)})
	if !ok {
		r.Issues = append(r.Issues, check+": "+strings.TrimSpace(msg))
	}
}

func (r *Report) action(msg string) {
	r.Actions = append(r.Actions, msg)
}

// WriteLog writes the full report to the log file.
func (r *Report) WriteLog() {
	f, err := os.Create(r.LogPath)
	if err != nil {
		return
	}
	defer f.Close()
	fmt.Fprintf(f, "=== ZedProxy %s Report ===\n", strings.ToUpper(r.Mode))
	fmt.Fprintf(f, "Started:  %s\n", r.StartedAt.Format("2006-01-02 15:04:05"))
	fmt.Fprintf(f, "Hostname: %s\n", r.Hostname)
	if r.DryRun {
		fmt.Fprintln(f, "Mode:     DRY-RUN")
	}
	fmt.Fprintln(f, "")
	fmt.Fprintln(f, "--- CHECKS ---")
	for _, res := range r.Results {
		sym := "[OK]"
		if !res.OK {
			sym = "[!!]"
		}
		fmt.Fprintf(f, "%s %s: %s\n", sym, res.Check, res.Message)
	}
	if len(r.Issues) > 0 {
		fmt.Fprintln(f, "\n--- ISSUES DETECTED ---")
		for _, iss := range r.Issues {
			fmt.Fprintln(f, " * "+iss)
		}
	}
	if len(r.Actions) > 0 {
		fmt.Fprintln(f, "\n--- ACTIONS TAKEN ---")
		for _, act := range r.Actions {
			fmt.Fprintln(f, " > "+act)
		}
	}
	fmt.Fprintf(f, "\nCompleted: %s\n", time.Now().Format("2006-01-02 15:04:05"))
}

// Print prints the report to stdout.
func (r *Report) Print() {
	fmt.Printf("\n=== ZedProxy %s ===\n", strings.ToUpper(r.Mode))
	if r.DryRun {
		fmt.Println("[DRY-RUN] No changes will be made")
	}
	fmt.Println("")
	okCount, failCount := 0, 0
	for _, res := range r.Results {
		sym := "[OK]"
		if !res.OK {
			sym = "[!!]"
			failCount++
		} else {
			okCount++
		}
		fmt.Printf("  %s  %-32s %s\n", sym, res.Check, res.Message)
	}
	fmt.Printf("\n  Passed: %d  Failed: %d\n", okCount, failCount)
	if len(r.Actions) > 0 {
		fmt.Println("\n  Actions:")
		for _, a := range r.Actions {
			fmt.Println("    >", a)
		}
	}
	fmt.Printf("\n  Log: %s\n\n", r.LogPath)
}

// ============================================================
// DOCTOR
// ============================================================

// RunDoctor runs all health checks and returns a Report.
func RunDoctor() *Report {
	r := newReport("doctor", false)
	runChecks(r)
	r.WriteLog()
	return r
}

func runChecks(r *Report) {
	checkServiceStatus(r)
	checkPort8080(r)
	checkPort80443(r)
	checkHTTP(r)
	checkFiles(r)
	checkPermissions(r)
	checkDatabase(r)
	checkTemplates(r)
	checkUpdateScripts(r)
	checkRecentLogs(r)
	checkDisk(r)
	checkNginx(r)
}

// ---- individual checks ----

func checkServiceStatus(r *Report) {
	out, err := exec.Command("systemctl", "is-active", "zedproxy").Output()
	status := strings.TrimSpace(string(out))
	if err != nil || status != "active" {
		r.add("Service zedproxy", false, "not active: "+status)
		// Grab recent journal
		jout, _ := exec.Command("journalctl", "-u", "zedproxy", "-n", "20", "--no-pager").Output()
		r.add("Service journal (last 20 lines)", false, "\n"+string(jout))
	} else {
		r.add("Service zedproxy", true, "active")
	}
}

func checkPort8080(r *Report) {
	out, _ := exec.Command("ss", "-tlnp").Output()
	lines := string(out)
	if !strings.Contains(lines, ":8080 ") && !strings.Contains(lines, ":8080\t") {
		r.add("Port 8080", false, "nothing listening on 8080")
		return
	}
	// Check if it's zedproxy or something else
	if strings.Contains(lines, "zedproxy") {
		r.add("Port 8080", true, "zedproxy listening")
		return
	}
	// Could be monitorix or other
	fuserOut, _ := exec.Command("fuser", "8080/tcp").Output()
	pid := strings.TrimSpace(string(fuserOut))
	if pid == "" {
		pid = "unknown"
	}
	// Try to identify
	lsofOut, _ := exec.Command("sh", "-c", "lsof -i :8080 -n -P 2>/dev/null | head -5").Output()
	detail := string(lsofOut)
	if strings.Contains(strings.ToLower(detail), "monitorix") {
		r.add("Port 8080", false, "occupied by monitorix (pid "+pid+")")
	} else {
		r.add("Port 8080", false, "occupied by unknown process (pid "+pid+"): "+strings.TrimSpace(detail))
	}
}

func checkPort80443(r *Report) {
	out, _ := exec.Command("ss", "-tlnp").Output()
	lines := string(out)
	has80 := strings.Contains(lines, ":80 ") || strings.Contains(lines, ":80\t")
	has443 := strings.Contains(lines, ":443 ") || strings.Contains(lines, ":443\t")
	r.add("Port 80 (nginx)", has80, boolMsg(has80, "listening", "not listening"))
	r.add("Port 443 (nginx/TLS)", has443, boolMsg(has443, "listening", "not listening"))
}

func checkHTTP(r *Report) {
	c := &http.Client{Timeout: 4 * time.Second}
	for _, path := range []string{"/health", "/version", "/"} {
		url := "http://127.0.0.1:8080" + path
		resp, err := c.Get(url)
		if err != nil {
			r.add("HTTP "+path, false, err.Error())
			continue
		}
		resp.Body.Close()
		ok := resp.StatusCode < 500
		r.add("HTTP "+path, ok, fmt.Sprintf("HTTP %d", resp.StatusCode))
	}
}

func checkFiles(r *Report) {
	type fileCheck struct {
		path string
		exec bool
	}
	checks := []fileCheck{
		{BaseDir, false},
		{filepath.Join(BaseDir, "zedproxy"), true},
		{filepath.Join(BaseDir, ".env"), false},
		{filepath.Join(BaseDir, "data", "zedproxy.db"), false},
		{filepath.Join(BaseDir, "templates"), false},
		{filepath.Join(BaseDir, "static"), false},
		{filepath.Join(BaseDir, "static", "uploads"), false},
		{filepath.Join(BaseDir, "logs"), false},
		{filepath.Join(BaseDir, "backups"), false},
	}
	for _, fc := range checks {
		info, err := os.Stat(fc.path)
		if err != nil {
			r.add("Path "+fc.path, false, "missing")
			continue
		}
		if fc.exec {
			mode := info.Mode()
			if mode&0111 == 0 {
				r.add("Path "+fc.path, false, "exists but not executable")
				continue
			}
		}
		r.add("Path "+fc.path, true, "ok")
	}
}

func checkPermissions(r *Report) {
	dirs := []string{
		filepath.Join(BaseDir, "data"),
		filepath.Join(BaseDir, "static", "uploads"),
		filepath.Join(BaseDir, "logs"),
		filepath.Join(BaseDir, "backups"),
	}
	for _, d := range dirs {
		info, err := os.Stat(d)
		if err != nil {
			r.add("Perm "+d, false, "missing")
			continue
		}
		mode := info.Mode().Perm()
		// Check writable by owner at minimum
		ok := mode&0200 != 0
		r.add("Perm "+d, ok, fmt.Sprintf("%04o", mode))
	}
}

func checkDatabase(r *Report) {
	dbPath := filepath.Join(BaseDir, "data", "zedproxy.db")
	if _, err := os.Stat(dbPath); err != nil {
		r.add("DB integrity", false, "database file missing")
		return
	}
	db, err := sql.Open("sqlite3", dbPath+"?mode=ro")
	if err != nil {
		r.add("DB integrity", false, "open failed: "+err.Error())
		return
	}
	defer db.Close()

	// integrity_check
	var intResult string
	if err := db.QueryRow("PRAGMA integrity_check").Scan(&intResult); err != nil {
		r.add("DB integrity", false, "integrity_check failed: "+err.Error())
	} else {
		r.add("DB integrity", intResult == "ok", intResult)
	}

	// required tables
	tables := []string{"admins", "settings", "users"}
	for _, t := range tables {
		var name string
		err := db.QueryRow("SELECT name FROM sqlite_master WHERE type='table' AND name=?", t).Scan(&name)
		r.add("DB table "+t, err == nil, boolMsg(err == nil, "exists", "missing"))
	}

	// admin user exists
	var count int
	db.QueryRow("SELECT COUNT(*) FROM admins").Scan(&count) //nolint:errcheck
	r.add("DB admin user", count > 0, fmt.Sprintf("%d admin(s)", count))
}

func checkTemplates(r *Report) {
	tmplDir := filepath.Join(BaseDir, "templates")
	if _, err := os.Stat(tmplDir); err != nil {
		r.add("Templates dir", false, "missing")
		return
	}
	r.add("Templates dir", true, tmplDir)

	// Check for known template error patterns in recent logs
	logDir := filepath.Join(BaseDir, "logs")
	if logEntries, err := os.ReadDir(logDir); err == nil {
		errPatterns := []string{"Template error", "unexpected EOF", "unexpected <define>", "panic:", "nil pointer"}
		recent := []os.DirEntry{}
		for _, e := range logEntries {
			if !e.IsDir() {
				recent = append(recent, e)
			}
		}
		// Check last 3 log files
		start := len(recent) - 3
		if start < 0 {
			start = 0
		}
		for _, e := range recent[start:] {
			content, err := os.ReadFile(filepath.Join(logDir, e.Name()))
			if err != nil {
				continue
			}
			for _, pat := range errPatterns {
				if strings.Contains(string(content), pat) {
					r.add("Template/runtime log "+e.Name(), false, "contains '"+pat+"'")
				}
			}
		}
	}
}

func checkUpdateScripts(r *Report) {
	scripts := []string{
		filepath.Join(BaseDir, "update.sh"),
		filepath.Join(BaseDir, "rollback.sh"),
		filepath.Join(BaseDir, "manage.sh"),
	}
	for _, s := range scripts {
		if _, err := os.Stat(s); err != nil {
			r.add("Script "+filepath.Base(s), false, "missing")
			continue
		}
		out, err := exec.Command("bash", "-n", s).CombinedOutput()
		if err != nil {
			r.add("Script "+filepath.Base(s), false, "syntax error: "+string(out))
		} else {
			r.add("Script "+filepath.Base(s), true, "syntax ok")
		}
	}
}

func checkRecentLogs(r *Report) {
	// Check journalctl for panics
	out, _ := exec.Command("journalctl", "-u", "zedproxy", "-n", "50", "--no-pager").Output()
	content := string(out)
	for _, pat := range []string{"panic:", "nil pointer", "template:", "bind: address already in use"} {
		if strings.Contains(strings.ToLower(content), strings.ToLower(pat)) {
			r.add("Journal pattern '"+pat+"'", false, "found in recent logs")
		}
	}
}

func checkNginx(r *Report) {
	out, err := exec.Command("systemctl", "is-active", "nginx").Output()
	status := strings.TrimSpace(string(out))
	r.add("Nginx service", err == nil && status == "active", status)

	// nginx -t
	testOut, testErr := exec.Command("nginx", "-t").CombinedOutput()
	r.add("Nginx config test", testErr == nil, strings.TrimSpace(string(testOut)))
}

func checkDisk(r *Report) {
	out, _ := exec.Command("df", "-h", BaseDir).Output()
	lines := strings.Split(strings.TrimSpace(string(out)), "\n")
	if len(lines) >= 2 {
		r.add("Disk space", true, lines[1])
	}
}

func boolMsg(ok bool, yes, no string) string {
	if ok {
		return yes
	}
	return no
}

// ============================================================
// REPAIR
// ============================================================

// RunRepair performs safe automated repairs. dryRun=true shows what would change.
func RunRepair(dryRun bool) *Report {
	r := newReport("repair", dryRun)

	// Run all checks first
	runChecks(r)

	// Only repair if issues found
	if len(r.Issues) == 0 {
		r.action("No issues found — no repair needed")
		r.Print()
		r.WriteLog()
		sendTelegram(r, "complete")
		return r
	}

	// Create backup first (before any repair action)
	if !dryRun {
		backupPath, err := createRepairBackup()
		if err != nil {
			r.action("[!!] Backup failed: " + err.Error())
			fmt.Println("[!!] Backup failed:", err, "— aborting repair for safety")
			r.WriteLog()
			return r
		}
		r.action("Backup created: " + backupPath)
		sendTelegram(r, "backup_created")
	} else {
		r.action("[dry-run] Would create backup ZIP in " + filepath.Join(BaseDir, "backups"))
	}

	// Repair: create missing directories
	repairDirs(r, dryRun)

	// Repair: fix permissions
	repairPermissions(r, dryRun)

	// Repair: port 8080 conflicts
	repairPort8080(r, dryRun)

	// Repair: service restart
	repairService(r, dryRun)

	// Repair: nginx reload (only if config test passes)
	repairNginx(r, dryRun)

	// Final health check
	if !dryRun {
		time.Sleep(2 * time.Second)
		c := &http.Client{Timeout: 5 * time.Second}
		resp, err := c.Get("http://127.0.0.1:8080/health")
		if err == nil {
			resp.Body.Close()
			r.action(fmt.Sprintf("Post-repair health check: HTTP %d", resp.StatusCode))
		} else {
			r.action("Post-repair health check: " + err.Error())
		}
	}

	r.Print()
	r.WriteLog()
	sendTelegram(r, "complete")
	return r
}

func repairDirs(r *Report, dryRun bool) {
	dirs := []string{
		filepath.Join(BaseDir, "data"),
		filepath.Join(BaseDir, "static", "uploads"),
		filepath.Join(BaseDir, "logs"),
		filepath.Join(BaseDir, "backups"),
	}
	for _, d := range dirs {
		if _, err := os.Stat(d); err == nil {
			continue
		}
		if dryRun {
			r.action("[dry-run] Would create dir: " + d)
			continue
		}
		if err := os.MkdirAll(d, 0755); err != nil {
			r.action("[!!] mkdir " + d + ": " + err.Error())
		} else {
			r.action("Created dir: " + d)
		}
	}
}

func repairPermissions(r *Report, dryRun bool) {
	dirs := []string{
		filepath.Join(BaseDir, "data"),
		filepath.Join(BaseDir, "static", "uploads"),
		filepath.Join(BaseDir, "logs"),
		filepath.Join(BaseDir, "backups"),
	}
	for _, d := range dirs {
		info, err := os.Stat(d)
		if err != nil {
			continue
		}
		mode := info.Mode().Perm()
		if mode&0200 == 0 {
			if dryRun {
				r.action("[dry-run] Would chmod 0755 " + d)
				continue
			}
			if err := os.Chmod(d, 0755); err != nil {
				r.action("[!!] chmod " + d + ": " + err.Error())
			} else {
				r.action("Fixed permissions: " + d)
			}
		}
	}
}

func repairPort8080(r *Report, dryRun bool) {
	// Check if something other than zedproxy holds 8080
	lsofOut, _ := exec.Command("sh", "-c", "lsof -i :8080 -n -P 2>/dev/null").Output()
	detail := strings.ToLower(string(lsofOut))

	if strings.Contains(detail, "monitorix") {
		if dryRun {
			r.action("[dry-run] Would stop and disable monitorix (occupying port 8080)")
			return
		}
		exec.Command("systemctl", "stop", "monitorix").Run()   //nolint:errcheck
		exec.Command("systemctl", "disable", "monitorix").Run() //nolint:errcheck
		r.action("Stopped and disabled monitorix (was occupying port 8080)")
		return
	}

	// Check if a stale zedproxy process occupies 8080
	fuserOut, _ := exec.Command("fuser", "8080/tcp").Output()
	pids := strings.Fields(string(fuserOut))
	for _, pid := range pids {
		commOut, _ := exec.Command("cat", "/proc/"+pid+"/comm").Output()
		comm := strings.TrimSpace(string(commOut))
		if comm == "zedproxy" {
			if dryRun {
				r.action("[dry-run] Would kill stale zedproxy process (pid " + pid + ")")
				continue
			}
			exec.Command("kill", pid).Run() //nolint:errcheck
			r.action("Killed stale zedproxy process (pid " + pid + ")")
		}
		// Unknown process: do NOT kill
		if comm != "zedproxy" && comm != "" {
			r.action("[!!] Port 8080 occupied by '" + comm + "' (pid " + pid + ") — manual action required")
		}
	}
}

func repairService(r *Report, dryRun bool) {
	if dryRun {
		r.action("[dry-run] Would reset-failed and restart zedproxy service")
		return
	}
	exec.Command("systemctl", "reset-failed", "zedproxy").Run() //nolint:errcheck
	if err := exec.Command("systemctl", "restart", "zedproxy").Run(); err != nil {
		r.action("[!!] Service restart failed: " + err.Error())
	} else {
		r.action("Service zedproxy restarted")
	}
}

func repairNginx(r *Report, dryRun bool) {
	// Only reload if config test passes
	testOut, testErr := exec.Command("nginx", "-t").CombinedOutput()
	if testErr != nil {
		r.action("[!!] Nginx config test failed — not reloading: " + string(testOut))
		return
	}
	if dryRun {
		r.action("[dry-run] Would reload nginx (config test passed)")
		return
	}
	if err := exec.Command("nginx", "-s", "reload").Run(); err != nil {
		r.action("[!!] Nginx reload failed: " + err.Error())
	} else {
		r.action("Nginx reloaded (config test passed)")
	}
}

// ============================================================
// BACKUP
// ============================================================

func createRepairBackup() (string, error) {
	ts := time.Now().Format("20060102-150405")
	backupDir := filepath.Join(BaseDir, "backups")
	if err := os.MkdirAll(backupDir, 0755); err != nil {
		return "", err
	}
	zipPath := filepath.Join(backupDir, "repair-backup-"+ts+".zip")

	f, err := os.Create(zipPath)
	if err != nil {
		return "", err
	}
	defer f.Close()

	zw := zip.NewWriter(f)
	defer zw.Close()

	// Items to backup
	items := []struct {
		src  string
		name string
	}{
		{filepath.Join(BaseDir, "data", "zedproxy.db"), "data/zedproxy.db"},
		{filepath.Join(BaseDir, ".env"), ".env"},
		{filepath.Join(BaseDir, "zedproxy"), "zedproxy"},
		{filepath.Join(BaseDir, "update.sh"), "update.sh"},
		{filepath.Join(BaseDir, "rollback.sh"), "rollback.sh"},
		{filepath.Join(BaseDir, "manage.sh"), "manage.sh"},
	}

	for _, item := range items {
		if err := addFileToZip(zw, item.src, item.name); err != nil {
			// non-fatal — log and continue
			fmt.Fprintf(os.Stderr, "[warn] backup: skip %s: %v\n", item.src, err)
		}
	}

	// Add templates directory
	addDirToZip(zw, filepath.Join(BaseDir, "templates"), "templates")

	// Add uploads directory (skip large files > 50MB)
	addDirToZipSized(zw, filepath.Join(BaseDir, "static", "uploads"), "uploads", 50*1024*1024)

	// Info file (no secrets)
	w, _ := zw.Create("backup-info.txt")
	fmt.Fprintf(w, "ZedProxy Repair Backup\nCreated: %s\nHostname: %s\n", time.Now().Format("2006-01-02 15:04:05"), hostnameStr())

	return zipPath, nil
}

func addFileToZip(zw *zip.Writer, src, name string) error {
	data, err := os.ReadFile(src)
	if err != nil {
		return err
	}
	w, err := zw.Create(name)
	if err != nil {
		return err
	}
	_, err = w.Write(data)
	return err
}

func addDirToZip(zw *zip.Writer, dir, prefix string) {
	filepath.Walk(dir, func(path string, info os.FileInfo, err error) error { //nolint:errcheck
		if err != nil || info.IsDir() {
			return nil
		}
		rel, _ := filepath.Rel(dir, path)
		w, err := zw.Create(filepath.Join(prefix, rel))
		if err != nil {
			return nil
		}
		f, err := os.Open(path)
		if err != nil {
			return nil
		}
		defer f.Close()
		io.Copy(w, f) //nolint:errcheck
		return nil
	})
}

func addDirToZipSized(zw *zip.Writer, dir, prefix string, maxBytes int64) {
	filepath.Walk(dir, func(path string, info os.FileInfo, err error) error { //nolint:errcheck
		if err != nil || info.IsDir() {
			return nil
		}
		if info.Size() > maxBytes {
			return nil
		}
		rel, _ := filepath.Rel(dir, path)
		w, err := zw.Create(filepath.Join(prefix, rel))
		if err != nil {
			return nil
		}
		f, err := os.Open(path)
		if err != nil {
			return nil
		}
		defer f.Close()
		io.Copy(w, f) //nolint:errcheck
		return nil
	})
}

func hostnameStr() string {
	h, _ := os.Hostname()
	return h
}

// ============================================================
// TELEGRAM
// ============================================================

// TelegramSender is a function type for injecting Telegram send capability.
// Avoids import cycles — caller (main) injects it via SetTelegramSender.
var TelegramSender func(title, message, category string)

// SetTelegramSender injects the Telegram send function from main/telegram package.
func SetTelegramSender(fn func(title, message, category string)) {
	TelegramSender = fn
}

func sendTelegram(r *Report, stage string) {
	if TelegramSender == nil {
		return
	}
	var title, msg, cat string
	issueCount := len(r.Issues)
	actionCount := len(r.Actions)

	switch stage {
	case "backup_created":
		cat = "backups"
		title = "بکاپ ایمنی ایجاد شد"
		msg = fmt.Sprintf("قبل از تعمیر، بکاپ ایجاد شد.\nسرور: %s", r.Hostname)
	case "complete":
		if r.Mode == "doctor" {
			if issueCount == 0 {
				cat = "system_status"
				title = "Doctor: سیستم سالم است"
				msg = fmt.Sprintf("سرور: %s\nهمه چک‌ها موفق بودند.", r.Hostname)
			} else {
				cat = "critical_alerts"
				title = "Doctor: مشکل پیدا شد"
				msg = fmt.Sprintf("سرور: %s\nمشکلات: %d مورد\n", r.Hostname, issueCount)
				for i, iss := range r.Issues {
					if i >= 5 {
						msg += fmt.Sprintf("... و %d مشکل دیگر", issueCount-5)
						break
					}
					msg += "• " + iss + "\n"
				}
			}
		} else {
			if actionCount > 0 {
				cat = "maintenance"
				title = "Repair: تعمیر انجام شد"
				msg = fmt.Sprintf("سرور: %s\nمشکلات: %d\nاقدامات: %d\n", r.Hostname, issueCount, actionCount)
				for i, act := range r.Actions {
					if i >= 5 {
						msg += fmt.Sprintf("... و %d اقدام دیگر", actionCount-5)
						break
					}
					msg += "• " + act + "\n"
				}
			} else {
				cat = "system_status"
				title = "Repair: مشکلی برای تعمیر نبود"
				msg = fmt.Sprintf("سرور: %s\nسیستم سالم است.", r.Hostname)
			}
		}
	}
	if title != "" {
		TelegramSender(title, msg, cat)
	}
}

// ============================================================
// LEGACY COMPAT (still used by older callers)
// ============================================================

// PrintResults prints doctor results to stdout (legacy format).
func PrintResults(results []Result) {
	for _, r := range results {
		sym := "[OK]"
		if !r.OK {
			sym = "[!!]"
		}
		fmt.Printf("%s %s: %s\n", sym, r.Check, r.Message)
	}
}
