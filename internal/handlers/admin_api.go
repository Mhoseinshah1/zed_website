package handlers

import (
	"crypto/rand"
	"encoding/hex"

	"github.com/gin-contrib/sessions"
	"github.com/gin-gonic/gin"
	"zedproxy/internal/models"
)

func AdminAPIPage(c *gin.Context) {
	settings := models.GetAllSettings()
	apiKey := settings["internal_api_key"]
	maskedKey := ""
	if len(apiKey) > 8 {
		maskedKey = apiKey[:8] + "••••••••••••••••••••••••"
	} else if apiKey != "" {
		maskedKey = "••••••••"
	}

	data := adminData(c, "integrations")
	data["Title"] = "اتصالات و API"
	data["APIKeyMasked"] = maskedKey
	data["APIKeySet"] = apiKey != ""
	renderAdmin(c, "api-integrations", data)
}

func AdminAPIRegenerateKey(c *gin.Context) {
	b := make([]byte, 32)
	rand.Read(b)
	newKey := hex.EncodeToString(b)
	models.SetSetting("internal_api_key", newKey)
	LogAdminActivity(c, "api_key_regenerate", "کلید API داخلی بازسازی شد")
	sess := sessions.Default(c)
	sess.AddFlash("کلید API جدید با موفقیت تولید شد", "ok")
	sess.Save()
	c.Redirect(302, "/zed-admin/integrations/api")
}
