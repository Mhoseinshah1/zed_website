package handlers

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"zedproxy/internal/models"
)

// POST /api/internal/telegram/connect-user
// Called by the Telegram customer bot after user clicks the connect link.
// Requires X-Internal-API-Key header matching the stored internal_api_key setting.
func APITelegramConnectUser(c *gin.Context) {
	// Validate internal API key
	key := c.GetHeader("X-Internal-API-Key")
	if key == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "missing api key"})
		return
	}

	storedKey := models.GetSetting("internal_api_key")
	if storedKey == "" || strings.TrimSpace(key) != storedKey {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid api key"})
		return
	}

	var req struct {
		Token           string `json:"token" binding:"required"`
		TelegramID      string `json:"telegram_id" binding:"required"`
		TelegramUsername string `json:"telegram_username"`
		FirstName       string `json:"first_name"`
		LastName        string `json:"last_name"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request body"})
		return
	}

	// Validate token
	userID, err := models.ValidateTelegramConnectToken(req.Token)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Connect Telegram to user account
	if err := models.ConnectUserTelegram(userID, req.TelegramID, req.TelegramUsername); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to connect telegram"})
		return
	}

	models.CreateNotification(userID, "تلگرام متصل شد", "حساب تلگرام شما با موفقیت به حساب کاربری متصل شد.", "success", "/user/connect-telegram")
	models.LogUserActivity(userID, "telegram_connected", "تلگرام متصل شد: @"+req.TelegramUsername, "", "")

	c.JSON(http.StatusOK, gin.H{"ok": true, "user_id": userID})
}
