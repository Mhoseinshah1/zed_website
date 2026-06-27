package handlers

import (
	"fmt"
	"net/http"

	"github.com/gin-contrib/sessions"
	"github.com/gin-gonic/gin"

	"zedproxy/internal/models"
	"zedproxy/internal/payment"
)

// AdminNOWPaymentsPage shows the NOWPayments settings page.
func AdminNOWPaymentsPage(c *gin.Context) {
	data := adminData(c, "nowpayments")
	data["Title"] = "درگاه NOWPayments"
	data["Section"] = "payments"
	settings := models.GetAllSettings()
	data["NPEnabled"] = settings["nowpayments_enabled"]
	data["NPSandbox"] = settings["nowpayments_sandbox"]
	data["NPAPIKeySet"] = settings["nowpayments_api_key"] != ""
	data["NPPayCurrency"] = settings["nowpayments_pay_currency"]
	data["NPSuccessURL"] = settings["nowpayments_success_url"]
	data["NPCancelURL"] = settings["nowpayments_cancel_url"]
	data["USDIRRRate"] = settings["usd_irr_rate_manual"]
	data["USDIRRProvider"] = settings["usd_irr_rate_provider"]

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

	t, err := getAdminTemplate("nowpayments")
	if err != nil {
		renderAdminError(c, fmt.Sprintf("template error: %v", err))
		return
	}
	c.Header("Content-Type", "text/html; charset=utf-8")
	t.ExecuteTemplate(c.Writer, "admin", data) //nolint:errcheck
}

// AdminNOWPaymentsSave saves the NOWPayments settings.
func AdminNOWPaymentsSave(c *gin.Context) {
	sess := sessions.Default(c)

	enabled := "0"
	if c.PostForm("nowpayments_enabled") == "1" {
		enabled = "1"
	}
	sandbox := "0"
	if c.PostForm("nowpayments_sandbox") == "1" {
		sandbox = "1"
	}

	models.SetSetting("nowpayments_enabled", enabled)
	models.SetSetting("nowpayments_sandbox", sandbox)

	// Only save API key if non-empty and not masked
	apiKey := c.PostForm("nowpayments_api_key")
	if apiKey != "" && apiKey != "***" {
		models.SetSetting("nowpayments_api_key", apiKey)
	}
	ipnSecret := c.PostForm("nowpayments_ipn_secret")
	if ipnSecret != "" && ipnSecret != "***" {
		models.SetSetting("nowpayments_ipn_secret", ipnSecret)
	}

	if v := c.PostForm("nowpayments_pay_currency"); v != "" {
		models.SetSetting("nowpayments_pay_currency", v)
	}
	if v := c.PostForm("nowpayments_success_url"); v != "" {
		models.SetSetting("nowpayments_success_url", v)
	}
	if v := c.PostForm("nowpayments_cancel_url"); v != "" {
		models.SetSetting("nowpayments_cancel_url", v)
	}
	if v := c.PostForm("usd_irr_rate_manual"); v != "" {
		models.SetSetting("usd_irr_rate_manual", v)
	}
	if v := c.PostForm("usd_irr_rate_provider"); v != "" {
		models.SetSetting("usd_irr_rate_provider", v)
	}

	LogAdminActivity(c, "nowpayments_settings_saved", "تنظیمات NOWPayments ذخیره شد")
	sess.Set("flash_ok", "تنظیمات NOWPayments ذخیره شد")
	sess.Save()
	c.Redirect(http.StatusFound, "/zed-admin/payments/nowpayments")
}

// AdminNOWPaymentsTest tests the NOWPayments API key.
func AdminNOWPaymentsTest(c *gin.Context) {
	apiKey := models.GetSetting("nowpayments_api_key")
	if apiKey == "" {
		c.JSON(http.StatusOK, gin.H{"ok": false, "error": "API key not configured"})
		return
	}
	sandbox := models.GetSetting("nowpayments_sandbox") == "1"
	client := payment.NewNOWPayments(apiKey, "", sandbox)
	if err := client.TestStatus(); err != nil {
		c.JSON(http.StatusOK, gin.H{"ok": false, "error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"ok": true, "description": "اتصال به NOWPayments موفق بود"})
}
