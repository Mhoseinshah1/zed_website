package handlers

import (
	"crypto/tls"
	"fmt"
	"net"
	"net/http"
	"net/smtp"
	"strings"

	"github.com/gin-contrib/sessions"
	"github.com/gin-gonic/gin"
	"zedproxy/internal/models"
)

func AdminEmailSettingsPage(c *gin.Context) {
	s := map[string]string{}
	keys := []string{
		"smtp_enabled", "smtp_host", "smtp_port", "smtp_username",
		"smtp_from_email", "smtp_from_name", "smtp_use_tls",
		"email_verification_code_ttl_minutes",
		"email_verification_resend_cooldown_seconds",
		"email_verification_max_attempts",
	}
	for _, k := range keys {
		s[k] = models.GetSetting(k)
	}
	// Indicate whether password is set without revealing it
	if models.GetSetting("smtp_password") != "" {
		s["smtp_password_set"] = "1"
	}

	renderAdmin(c, "email-settings", map[string]interface{}{
		"Title":   "تنظیمات ایمیل",
		"Section": "email-settings",
		"S":       s,
	})
}

func AdminEmailSettingsSave(c *gin.Context) {
	sess := sessions.Default(c)

	models.SetSetting("smtp_enabled", boolVal(c.PostForm("smtp_enabled")))
	models.SetSetting("smtp_host", strings.TrimSpace(c.PostForm("smtp_host")))
	models.SetSetting("smtp_port", strings.TrimSpace(c.PostForm("smtp_port")))
	models.SetSetting("smtp_username", strings.TrimSpace(c.PostForm("smtp_username")))
	models.SetSetting("smtp_from_email", strings.TrimSpace(c.PostForm("smtp_from_email")))
	models.SetSetting("smtp_from_name", strings.TrimSpace(c.PostForm("smtp_from_name")))
	models.SetSetting("smtp_use_tls", boolVal(c.PostForm("smtp_use_tls")))

	// Only replace password if a new one was typed
	if pw := c.PostForm("smtp_password"); pw != "" {
		models.SetSetting("smtp_password", pw)
	}

	if v := c.PostForm("email_verification_code_ttl_minutes"); v != "" {
		models.SetSetting("email_verification_code_ttl_minutes", v)
	}
	if v := c.PostForm("email_verification_resend_cooldown_seconds"); v != "" {
		models.SetSetting("email_verification_resend_cooldown_seconds", v)
	}
	if v := c.PostForm("email_verification_max_attempts"); v != "" {
		models.SetSetting("email_verification_max_attempts", v)
	}

	sess.Set("flash_ok", "تنظیمات ایمیل ذخیره شد")
	sess.Save()
	c.Redirect(http.StatusFound, "/zed-admin/settings/email")
}

func AdminEmailSettingsTest(c *gin.Context) {
	toEmail := strings.TrimSpace(c.PostForm("test_email"))

	s := map[string]string{}
	for _, k := range []string{"smtp_enabled", "smtp_host", "smtp_port", "smtp_username", "smtp_password", "smtp_from_email", "smtp_from_name", "smtp_use_tls"} {
		s[k] = models.GetSetting(k)
	}
	if s["smtp_password"] != "" {
		s["smtp_password_set"] = "1"
	}

	var result string
	var ok bool

	if s["smtp_enabled"] != "1" {
		result = "خطا: ارسال ایمیل غیرفعال است. ابتدا فعال کنید و ذخیره نمایید."
	} else if s["smtp_host"] == "" {
		result = "خطا: آدرس سرور SMTP تنظیم نشده است."
	} else if toEmail == "" {
		result = "خطا: آدرس ایمیل مقصد را وارد کنید."
	} else {
		err := sendTestEmail(s, toEmail)
		if err != nil {
			result = "خطا در ارسال: " + err.Error()
		} else {
			result = "ایمیل تست با موفقیت ارسال شد به " + toEmail
			ok = true
		}
	}

	renderAdmin(c, "email-settings", map[string]interface{}{
		"Title":      "تنظیمات ایمیل",
		"Section":    "email-settings",
		"S":          s,
		"TestResult": result,
		"TestOK":     ok,
	})
}

func sendTestEmail(s map[string]string, to string) error {
	host := s["smtp_host"]
	port := s["smtp_port"]
	if port == "" {
		port = "587"
	}
	addr := net.JoinHostPort(host, port)
	from := s["smtp_from_email"]
	if from == "" {
		from = s["smtp_username"]
	}
	fromName := s["smtp_from_name"]
	if fromName == "" {
		fromName = "ZedProxy"
	}
	user := s["smtp_username"]
	pass := s["smtp_password"]

	msg := fmt.Sprintf("From: %s <%s>\r\nTo: %s\r\nSubject: =?UTF-8?B?2KrYs9iqINib2KfbjNmHINin24zZhdmH2Ycg WmVkUHJveHk=?=\r\nContent-Type: text/plain; charset=UTF-8\r\n\r\nاتصال SMTP با موفقیت برقرار شد.",
		fromName, from, to)

	var auth smtp.Auth
	if user != "" && pass != "" {
		auth = smtp.PlainAuth("", user, pass, host)
	}

	useTLS := s["smtp_use_tls"] != "0"
	if useTLS {
		tlsConf := &tls.Config{ServerName: host}
		conn, err := tls.Dial("tcp", addr, tlsConf)
		if err != nil {
			// Fall back to STARTTLS
			cl, err2 := smtp.Dial(addr)
			if err2 != nil {
				return err2
			}
			defer cl.Close()
			if err2 = cl.StartTLS(tlsConf); err2 != nil {
				return err2
			}
			if auth != nil {
				if err2 = cl.Auth(auth); err2 != nil {
					return err2
				}
			}
			if err2 = cl.Mail(from); err2 != nil {
				return err2
			}
			if err2 = cl.Rcpt(to); err2 != nil {
				return err2
			}
			w, err2 := cl.Data()
			if err2 != nil {
				return err2
			}
			_, err2 = w.Write([]byte(msg))
			w.Close()
			return err2
		}
		cl, err := smtp.NewClient(conn, host)
		if err != nil {
			return err
		}
		defer cl.Close()
		if auth != nil {
			if err = cl.Auth(auth); err != nil {
				return err
			}
		}
		if err = cl.Mail(from); err != nil {
			return err
		}
		if err = cl.Rcpt(to); err != nil {
			return err
		}
		w, err := cl.Data()
		if err != nil {
			return err
		}
		_, err = w.Write([]byte(msg))
		w.Close()
		return err
	}

	return smtp.SendMail(addr, auth, from, []string{to}, []byte(msg))
}

func boolVal(v string) string {
	if v == "1" {
		return "1"
	}
	return "0"
}
