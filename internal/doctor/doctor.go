package doctor

import (
	"fmt"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

// BaseDir is the production install directory.
const BaseDir = "/opt/zedproxy"

// Result holds the outcome of a single check.
type Result struct {
	Check   string
	OK      bool
	Message string
}

// RunDoctor runs all health checks and returns results.
func RunDoctor() []Result {
	var results []Result
	checks := []struct {
		name string
		fn   func() (bool, string)
	}{
		{"Service status", checkService},
		{"Port 8080", checkPort},
		{"Health endpoint", checkHealth},
		{"Database exists", checkDB},
		{"Templates exist", checkTemplates},
		{"Uploads dir", checkUploads},
		{"Logs dir", checkLogs},
		{"Backups dir", checkBackups},
		{"Binary metadata", checkBinaryMeta},
		{"Disk space", checkDisk},
	}
	for _, c := range checks {
		ok, msg := c.fn()
		results = append(results, Result{Check: c.name, OK: ok, Message: strings.TrimSpace(msg)})
	}
	return results
}

func checkService() (bool, string) {
	out, err := exec.Command("systemctl", "is-active", "zedproxy").Output()
	if err != nil {
		return false, "service not active"
	}
	s := strings.TrimSpace(string(out))
	return s == "active", s
}

func checkPort() (bool, string) {
	out, _ := exec.Command("ss", "-tlnp").Output()
	if strings.Contains(string(out), ":8080 ") {
		return true, "port 8080 listening"
	}
	return false, "port 8080 not listening"
}

func checkHealth() (bool, string) {
	c := &http.Client{Timeout: 3 * time.Second}
	resp, err := c.Get("http://127.0.0.1:8080/health")
	if err != nil {
		return false, err.Error()
	}
	defer resp.Body.Close()
	return resp.StatusCode == 200, fmt.Sprintf("HTTP %d", resp.StatusCode)
}

func checkDB() (bool, string) {
	p := filepath.Join(BaseDir, "data", "zedproxy.db")
	_, err := os.Stat(p)
	return err == nil, p
}

func checkTemplates() (bool, string) {
	p := filepath.Join(BaseDir, "templates")
	_, err := os.Stat(p)
	return err == nil, p
}

func checkUploads() (bool, string) {
	p := filepath.Join(BaseDir, "static", "uploads")
	_, err := os.Stat(p)
	return err == nil, p
}

func checkLogs() (bool, string) {
	p := filepath.Join(BaseDir, "logs")
	_, err := os.Stat(p)
	return err == nil, p
}

func checkBackups() (bool, string) {
	p := filepath.Join(BaseDir, "backups")
	_, err := os.Stat(p)
	return err == nil, p
}

func checkBinaryMeta() (bool, string) {
	exe, _ := os.Executable()
	out, _ := exec.Command(exe, "--version").Output()
	s := string(out)
	ok := !strings.Contains(s, "unknown") && !strings.Contains(s, "dev")
	return ok, strings.TrimSpace(s)
}

func checkDisk() (bool, string) {
	out, _ := exec.Command("df", "-h", BaseDir).Output()
	return true, strings.TrimSpace(string(out))
}

// PrintResults prints doctor results to stdout.
func PrintResults(results []Result) {
	for _, r := range results {
		sym := "[OK]"
		if !r.OK {
			sym = "[!!]"
		}
		fmt.Printf("%s %s: %s\n", sym, r.Check, r.Message)
	}
}

// RunRepair creates required directories and restarts the service.
func RunRepair() {
	dirs := []string{
		filepath.Join(BaseDir, "static/uploads"),
		filepath.Join(BaseDir, "static/uploads/images"),
		filepath.Join(BaseDir, "static/uploads/videos"),
		filepath.Join(BaseDir, "static/uploads/plans"),
		filepath.Join(BaseDir, "static/uploads/blog"),
		filepath.Join(BaseDir, "static/uploads/pages"),
		filepath.Join(BaseDir, "static/uploads/tmp"),
		filepath.Join(BaseDir, "logs"),
		filepath.Join(BaseDir, "backups"),
		filepath.Join(BaseDir, "data"),
	}
	for _, d := range dirs {
		if err := os.MkdirAll(d, 0755); err != nil {
			fmt.Printf("[!!] mkdir %s: %v\n", d, err)
		} else {
			fmt.Println("[OK] mkdir -p", d)
		}
	}

	exec.Command("systemctl", "reset-failed", "zedproxy").Run() //nolint:errcheck
	if err := exec.Command("systemctl", "restart", "zedproxy").Run(); err != nil {
		fmt.Printf("[!!] restart failed: %v\n", err)
	} else {
		fmt.Println("[OK] Service restarted")
	}
}
