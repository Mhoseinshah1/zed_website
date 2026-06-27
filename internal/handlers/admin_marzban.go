package handlers

import (
	"fmt"
	"net/http"
	"strconv"

	"github.com/gin-contrib/sessions"
	"github.com/gin-gonic/gin"

	"zedproxy/internal/database"
	"zedproxy/internal/marzban"
	"zedproxy/internal/models"
)

// MarzbanPanel represents a configured Marzban panel.
type MarzbanPanel struct {
	ID           int64
	Name         string
	BaseURL      string
	Username     string
	IsEnabled    bool
	IsDefault    bool
	Notes        string
	LastTestedAt string
	LastTestOK   *int
	CreatedAt    string
}

func getMarzbanPanels() ([]MarzbanPanel, error) {
	rows, err := database.DB.Query(
		`SELECT id, name, base_url, username, is_enabled, is_default, notes, last_tested_at, last_test_ok, created_at FROM marzban_panels ORDER BY id`,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var panels []MarzbanPanel
	for rows.Next() {
		var p MarzbanPanel
		var enabled, isDefault int
		rows.Scan(&p.ID, &p.Name, &p.BaseURL, &p.Username, &enabled, &isDefault, &p.Notes, &p.LastTestedAt, &p.LastTestOK, &p.CreatedAt)
		p.IsEnabled = enabled == 1
		p.IsDefault = isDefault == 1
		panels = append(panels, p)
	}
	return panels, nil
}

// AdminMarzbanPage shows the Marzban integration overview page.
func AdminMarzbanPage(c *gin.Context) {
	panels, err := getMarzbanPanels()
	data := adminData(c, "marzban")
	data["Title"] = "یکپارچه‌سازی Marzban"
	data["Section"] = "integrations"
	if err != nil {
		data["Error"] = err.Error()
	}
	data["Panels"] = panels
	settings := models.GetAllSettings()
	data["MarzbanEnabled"] = settings["marzban_enabled"]

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

	t, err2 := getAdminTemplate("marzban")
	if err2 != nil {
		renderAdminError(c, fmt.Sprintf("template error: %v", err2))
		return
	}
	c.Header("Content-Type", "text/html; charset=utf-8")
	t.ExecuteTemplate(c.Writer, "admin", data) //nolint:errcheck
}

// AdminMarzbanSave saves Marzban global settings.
func AdminMarzbanSave(c *gin.Context) {
	sess := sessions.Default(c)
	enabled := "0"
	if c.PostForm("marzban_enabled") == "1" {
		enabled = "1"
	}
	models.SetSetting("marzban_enabled", enabled)
	LogAdminActivity(c, "marzban_settings_saved", "تنظیمات Marzban ذخیره شد")
	sess.Set("flash_ok", "تنظیمات Marzban ذخیره شد")
	sess.Save()
	c.Redirect(http.StatusFound, "/zed-admin/integrations/marzban")
}

// AdminMarzbanTest tests the default Marzban panel connection.
func AdminMarzbanTest(c *gin.Context) {
	rows, err := database.DB.Query(`SELECT base_url, username, password_enc FROM marzban_panels WHERE is_default=1 AND is_enabled=1 LIMIT 1`)
	if err != nil || rows == nil {
		c.JSON(http.StatusOK, gin.H{"ok": false, "error": "no default panel configured"})
		return
	}
	defer rows.Close()
	var baseURL, username, password string
	if !rows.Next() {
		c.JSON(http.StatusOK, gin.H{"ok": false, "error": "no default panel configured"})
		return
	}
	rows.Scan(&baseURL, &username, &password)
	rows.Close()

	client := marzban.New(baseURL, username, password)
	if err := client.TestConnection(); err != nil {
		c.JSON(http.StatusOK, gin.H{"ok": false, "error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"ok": true, "description": "اتصال به Marzban موفق بود"})
}

// AdminMarzbanPanelsPage lists all Marzban panels.
func AdminMarzbanPanelsPage(c *gin.Context) {
	panels, err := getMarzbanPanels()
	data := adminData(c, "marzban")
	data["Title"] = "پنل‌های Marzban"
	data["Section"] = "integrations"
	if err != nil {
		data["Error"] = err.Error()
	}
	data["Panels"] = panels
	t, err2 := getAdminTemplate("marzban")
	if err2 != nil {
		renderAdminError(c, fmt.Sprintf("template error: %v", err2))
		return
	}
	c.Header("Content-Type", "text/html; charset=utf-8")
	t.ExecuteTemplate(c.Writer, "admin", data) //nolint:errcheck
}

// AdminMarzbanPanelNew shows the new panel form.
func AdminMarzbanPanelNew(c *gin.Context) {
	data := adminData(c, "marzban")
	data["Title"] = "پنل جدید Marzban"
	data["Section"] = "integrations"
	data["Panel"] = &MarzbanPanel{IsEnabled: true}
	data["IsNew"] = true
	t, err := getAdminTemplate("marzban-panel-form")
	if err != nil {
		renderAdminError(c, fmt.Sprintf("template error: %v", err))
		return
	}
	c.Header("Content-Type", "text/html; charset=utf-8")
	t.ExecuteTemplate(c.Writer, "admin", data) //nolint:errcheck
}

// AdminMarzbanPanelEdit shows the edit panel form.
func AdminMarzbanPanelEdit(c *gin.Context) {
	id, _ := strconv.ParseInt(c.Param("id"), 10, 64)
	var p MarzbanPanel
	var enabled, isDefault int
	err := database.DB.QueryRow(
		`SELECT id, name, base_url, username, is_enabled, is_default, notes FROM marzban_panels WHERE id=?`, id,
	).Scan(&p.ID, &p.Name, &p.BaseURL, &p.Username, &enabled, &isDefault, &p.Notes)
	if err != nil {
		c.Redirect(http.StatusFound, "/zed-admin/integrations/marzban")
		return
	}
	p.IsEnabled = enabled == 1
	p.IsDefault = isDefault == 1

	data := adminData(c, "marzban")
	data["Title"] = "ویرایش پنل Marzban"
	data["Section"] = "integrations"
	data["Panel"] = &p
	data["IsNew"] = false
	t, err2 := getAdminTemplate("marzban-panel-form")
	if err2 != nil {
		renderAdminError(c, fmt.Sprintf("template error: %v", err2))
		return
	}
	c.Header("Content-Type", "text/html; charset=utf-8")
	t.ExecuteTemplate(c.Writer, "admin", data) //nolint:errcheck
}

// AdminMarzbanPanelSave creates or updates a Marzban panel.
func AdminMarzbanPanelSave(c *gin.Context) {
	sess := sessions.Default(c)
	idStr := c.PostForm("id")
	name := c.PostForm("name")
	baseURL := c.PostForm("base_url")
	username := c.PostForm("username")
	password := c.PostForm("password")
	notes := c.PostForm("notes")
	isEnabled := 0
	if c.PostForm("is_enabled") == "1" {
		isEnabled = 1
	}
	isDefault := 0
	if c.PostForm("is_default") == "1" {
		isDefault = 1
	}

	if name == "" || baseURL == "" || username == "" {
		sess.Set("flash_err", "نام، آدرس و نام کاربری اجباری هستند")
		sess.Save()
		c.Redirect(http.StatusFound, "/zed-admin/integrations/marzban")
		return
	}

	if idStr == "" || idStr == "0" {
		if password == "" {
			sess.Set("flash_err", "رمز عبور برای پنل جدید اجباری است")
			sess.Save()
			c.Redirect(http.StatusFound, "/zed-admin/integrations/marzban")
			return
		}
		_, err := database.DB.Exec(
			`INSERT INTO marzban_panels (name, base_url, username, password_enc, is_enabled, is_default, notes) VALUES (?,?,?,?,?,?,?)`,
			name, baseURL, username, password, isEnabled, isDefault, notes,
		)
		if err != nil {
			sess.Set("flash_err", "خطا: "+err.Error())
		} else {
			sess.Set("flash_ok", "پنل Marzban ایجاد شد")
			LogAdminActivity(c, "marzban_panel_created", "پنل جدید: "+name)
		}
	} else {
		id, _ := strconv.ParseInt(idStr, 10, 64)
		if password != "" {
			database.DB.Exec( //nolint:errcheck
				`UPDATE marzban_panels SET name=?, base_url=?, username=?, password_enc=?, is_enabled=?, is_default=?, notes=?, updated_at=CURRENT_TIMESTAMP WHERE id=?`,
				name, baseURL, username, password, isEnabled, isDefault, notes, id,
			)
		} else {
			database.DB.Exec( //nolint:errcheck
				`UPDATE marzban_panels SET name=?, base_url=?, username=?, is_enabled=?, is_default=?, notes=?, updated_at=CURRENT_TIMESTAMP WHERE id=?`,
				name, baseURL, username, isEnabled, isDefault, notes, id,
			)
		}
		sess.Set("flash_ok", "پنل Marzban بروزرسانی شد")
		LogAdminActivity(c, "marzban_panel_updated", "پنل ویرایش شد: "+name)
	}
	sess.Save()
	c.Redirect(http.StatusFound, "/zed-admin/integrations/marzban")
}

// AdminMarzbanPanelTest tests a specific Marzban panel's connection.
func AdminMarzbanPanelTest(c *gin.Context) {
	id, _ := strconv.ParseInt(c.Param("id"), 10, 64)
	var baseURL, username, password string
	err := database.DB.QueryRow(
		`SELECT base_url, username, password_enc FROM marzban_panels WHERE id=?`, id,
	).Scan(&baseURL, &username, &password)
	if err != nil {
		c.JSON(http.StatusOK, gin.H{"ok": false, "error": "panel not found"})
		return
	}

	client := marzban.New(baseURL, username, password)
	testErr := client.TestConnection()
	testOK := 0
	if testErr == nil {
		testOK = 1
	}
	database.DB.Exec( //nolint:errcheck
		`UPDATE marzban_panels SET last_tested_at=CURRENT_TIMESTAMP, last_test_ok=? WHERE id=?`, testOK, id,
	)

	if testErr != nil {
		c.JSON(http.StatusOK, gin.H{"ok": false, "error": testErr.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"ok": true, "description": "اتصال موفق بود"})
}

// AdminMarzbanPanelDelete deletes a Marzban panel.
func AdminMarzbanPanelDelete(c *gin.Context) {
	id, _ := strconv.ParseInt(c.Param("id"), 10, 64)
	database.DB.Exec("DELETE FROM marzban_panels WHERE id=?", id) //nolint:errcheck
	LogAdminActivity(c, "marzban_panel_deleted", fmt.Sprintf("panel id=%d", id))
	sess := sessions.Default(c)
	sess.Set("flash_ok", "پنل حذف شد")
	sess.Save()
	c.Redirect(http.StatusFound, "/zed-admin/integrations/marzban")
}
