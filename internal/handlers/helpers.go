package handlers

import (
	"encoding/json"
	"fmt"
	"html/template"
	"log"
	"net/http"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"zedproxy/internal/models"
)

var (
	tmplCache  = map[string]*template.Template{}
	tmplMu     sync.RWMutex
	tmplDir    string
	devMode    bool
	AppVersion = "dev"
)

func Init(templateDir string, dev bool) {
	tmplDir = templateDir
	devMode = dev
}

var funcMap = template.FuncMap{
	"safeHTML": func(s string) template.HTML { return template.HTML(s) },
	"safeURL":  func(s string) template.URL { return template.URL(s) },
	"add":      func(a, b int) int { return a + b },
	"mod":      func(a, b int) int { return a % b },
	"sub":      func(a, b int) int { return a - b },

	// dict creates a map from key/value pairs — used in templates
	"dict": func(values ...interface{}) (map[string]interface{}, error) {
		if len(values)%2 != 0 {
			return nil, fmt.Errorf("dict requires even number of args, got %d", len(values))
		}
		m := make(map[string]interface{}, len(values)/2)
		for i := 0; i < len(values); i += 2 {
			key, ok := values[i].(string)
			if !ok {
				return nil, fmt.Errorf("dict keys must be strings, got %T", values[i])
			}
			m[key] = values[i+1]
		}
		return m, nil
	},

	// default returns val if it is non-empty, else def
	"default": func(def, val interface{}) interface{} {
		if val == nil {
			return def
		}
		if s, ok := val.(string); ok && s == "" {
			return def
		}
		return val
	},

	// json encodes a value as a JSON string (safe for embedding in <script>)
	"json": func(v interface{}) (template.JS, error) {
		b, err := json.Marshal(v)
		if err != nil {
			return "", err
		}
		return template.JS(b), nil
	},

	// formatTime formats a time.Time value
	"formatTime": func(t time.Time) string {
		return t.Format("2006/01/02 15:04")
	},

	// seq generates a slice of ints 0..n-1, useful for range loops
	"seq": func(n int) []int {
		s := make([]int, n)
		for i := range s {
			s[i] = i
		}
		return s
	},

	// contains checks if a string contains a substring
	"contains": func(s, substr string) bool {
		return strings.Contains(s, substr)
	},

	// hasPrefix checks string prefix
	"hasPrefix": func(s, prefix string) bool {
		return strings.HasPrefix(s, prefix)
	},

	// ne is a not-equal comparison (Go templates have eq but not ne as a func)
	"ne": func(a, b interface{}) bool { return a != b },

	// coalesce returns the first non-empty string
	"coalesce": func(values ...string) string {
		for _, v := range values {
			if v != "" {
				return v
			}
		}
		return ""
	},
}

// getTemplate loads a public page template (base.html + page.html).
func getTemplate(name string) (*template.Template, error) {
	if !devMode {
		tmplMu.RLock()
		if t, ok := tmplCache[name]; ok {
			tmplMu.RUnlock()
			return t, nil
		}
		tmplMu.RUnlock()
	}

	base := filepath.Join(tmplDir, "layouts", "base.html")
	page := filepath.Join(tmplDir, "public", name+".html")

	t, err := template.New("base").Funcs(funcMap).ParseFiles(base, page)
	if err != nil {
		return nil, err
	}

	if !devMode {
		tmplMu.Lock()
		tmplCache[name] = t
		tmplMu.Unlock()
	}
	return t, nil
}

// getAdminTemplate loads an admin page (admin layout + page content).
func getAdminTemplate(name string) (*template.Template, error) {
	cacheKey := "admin_" + name
	if !devMode {
		tmplMu.RLock()
		if t, ok := tmplCache[cacheKey]; ok {
			tmplMu.RUnlock()
			return t, nil
		}
		tmplMu.RUnlock()
	}

	adminLayout := filepath.Join(tmplDir, "layouts", "admin.html")
	page := filepath.Join(tmplDir, "admin", name+".html")

	t, err := template.New("admin").Funcs(funcMap).ParseFiles(adminLayout, page)
	if err != nil {
		return nil, err
	}

	if !devMode {
		tmplMu.Lock()
		tmplCache[cacheKey] = t
		tmplMu.Unlock()
	}
	return t, nil
}

// getStandaloneTemplate loads a standalone public template (no base layout).
func getStandaloneTemplate(name string) (*template.Template, error) {
	cacheKey := "standalone_" + name
	if !devMode {
		tmplMu.RLock()
		if t, ok := tmplCache[cacheKey]; ok {
			tmplMu.RUnlock()
			return t, nil
		}
		tmplMu.RUnlock()
	}

	page := filepath.Join(tmplDir, "public", name+".html")
	t, err := template.New(name).Funcs(funcMap).ParseFiles(page)
	if err != nil {
		return nil, err
	}

	if !devMode {
		tmplMu.Lock()
		tmplCache[cacheKey] = t
		tmplMu.Unlock()
	}
	return t, nil
}

// renderAdminError writes a styled Persian admin error page instead of a blank screen.
func renderAdminError(c *gin.Context, errMsg string) {
	log.Printf("admin template error: %s", errMsg)
	c.Header("Content-Type", "text/html; charset=utf-8")
	c.Status(http.StatusInternalServerError)
	fmt.Fprintf(c.Writer, `<!DOCTYPE html>
<html lang="fa" dir="rtl">
<head><meta charset="UTF-8"><title>خطای سیستم</title>
<style>
body{background:#0d0d16;color:#f1f5f9;font-family:system-ui,sans-serif;display:flex;align-items:center;justify-content:center;min-height:100vh;margin:0;direction:rtl}
.card{background:#1e1e2e;border:1px solid #ef4444;border-radius:16px;padding:40px;max-width:600px;text-align:center}
.icon{font-size:48px;margin-bottom:16px}
h1{color:#ef4444;margin:0 0 12px}
p{color:#94a3b8;margin:0 0 8px;font-size:14px}
.detail{background:#0d0d16;border:1px solid #334155;border-radius:8px;padding:12px;margin-top:16px;font-family:monospace;font-size:12px;color:#64748b;text-align:left;word-break:break-all}
a{color:#06b6d4;text-decoration:none;margin-top:20px;display:inline-block}
</style></head>
<body>
<div class="card">
<div class="icon">⚠️</div>
<h1>خطای بارگذاری صفحه</h1>
<p>متأسفانه در بارگذاری این صفحه مشکلی پیش آمد.</p>
<p>لطفاً با مدیر سیستم تماس بگیرید یا مجدداً تلاش کنید.</p>
<div class="detail">%s</div>
<a href="/zed-admin">← بازگشت به داشبورد</a>
</div>
</body></html>`, template.HTMLEscapeString(errMsg))
}

func basePageData(pageName string) map[string]interface{} {
	settings := models.GetAllSettings()
	return map[string]interface{}{
		"Settings":    settings,
		"PageName":    pageName,
		"SiteName":    settings["site_name"],
		"SiteURL":     settings["site_url"],
		"TelegramBot": settings["telegram_bot"],
		"TelegramCh":  settings["telegram_channel"],
		"Support":     settings["telegram_support"],
		"SiteCSS":     buildSiteCSSVars(settings),
	}
}
