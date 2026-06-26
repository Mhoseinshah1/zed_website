package handlers

import (
	"html/template"
	"path/filepath"
	"sync"

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
}

// getTemplate loads a public page template (base.html + page.html).
// The entry template block is "base".
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
// The entry template block is "admin".
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
// The entry template block name must match the {{define "name"}} in the file.
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
	}
}
