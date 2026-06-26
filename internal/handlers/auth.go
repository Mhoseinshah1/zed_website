package handlers

import (
	"net/http"
	"strings"

	"github.com/gin-contrib/sessions"
	"github.com/gin-gonic/gin"
	"golang.org/x/crypto/bcrypt"
	"zedproxy/internal/models"
)

// ─── Template helpers ─────────────────────────────────

func renderAuth(c *gin.Context, name string, data map[string]interface{}) {
	if data == nil {
		data = map[string]interface{}{}
	}
	// Merge base site data (CSS vars, settings)
	for k, v := range basePageData("auth") {
		if _, exists := data[k]; !exists {
			data[k] = v
		}
	}
	t, err := getAuthTemplate(name)
	if err != nil {
		c.String(http.StatusInternalServerError, "Template error: %v", err)
		return
	}
	c.Header("Content-Type", "text/html; charset=utf-8")
	if err := t.ExecuteTemplate(c.Writer, "base", data); err != nil {
		_ = err
	}
}

func isLoggedIn(c *gin.Context) bool {
	sess := sessions.Default(c)
	return sess.Get("user_id") != nil
}

func currentUserID(c *gin.Context) int64 {
	sess := sessions.Default(c)
	v := sess.Get("user_id")
	if v == nil {
		return 0
	}
	if id, ok := v.(int64); ok {
		return id
	}
	if id, ok := v.(int); ok {
		return int64(id)
	}
	return 0
}

// ─── Register ─────────────────────────────────────────

func AuthRegisterPage(c *gin.Context) {
	if isLoggedIn(c) {
		c.Redirect(http.StatusFound, "/user/dashboard")
		return
	}
	renderAuth(c, "register", map[string]interface{}{"Title": "ثبت‌نام"})
}

func AuthRegisterPost(c *gin.Context) {
	if isLoggedIn(c) {
		c.Redirect(http.StatusFound, "/user/dashboard")
		return
	}

	displayName := strings.TrimSpace(c.PostForm("display_name"))
	email := strings.ToLower(strings.TrimSpace(c.PostForm("email")))
	phone := strings.TrimSpace(c.PostForm("phone"))
	password := c.PostForm("password")
	passwordConfirm := strings.TrimSpace(c.PostForm("confirm_password"))

	data := map[string]interface{}{
		"Title":       "ثبت‌نام",
		"DisplayName": displayName,
		"Email":       email,
		"Phone":       phone,
	}
	fail := func(msg string) {
		data["Error"] = msg
		renderAuth(c, "register", data)
	}

	// Require at least email or phone
	if email == "" && phone == "" {
		fail("ایمیل یا شماره موبایل الزامی است")
		return
	}

	// Validate email format
	if email != "" && !strings.Contains(email, "@") {
		fail("فرمت ایمیل صحیح نیست")
		return
	}

	// Validate password
	if len(password) < 8 {
		fail("رمز عبور باید حداقل ۸ کاراکتر باشد")
		return
	}
	if password != passwordConfirm {
		fail("رمز عبور و تکرار آن یکسان نیستند")
		return
	}

	// Check duplicates
	if email != "" && models.EmailExists(email) {
		fail("این ایمیل قبلاً ثبت شده است")
		return
	}
	if phone != "" && models.PhoneExists(phone) {
		fail("این شماره موبایل قبلاً ثبت شده است")
		return
	}

	// Hash password
	hash, err := bcrypt.GenerateFromPassword([]byte(password), 12)
	if err != nil {
		fail("خطای سیستمی. لطفاً مجدداً تلاش کنید")
		return
	}

	user, err := models.CreateUser(email, phone, string(hash))
	if err != nil {
		fail("خطا در ثبت‌نام. لطفاً مجدداً تلاش کنید")
		return
	}

	// Set display name in profile if provided
	if displayName != "" {
		models.UpsertUserProfile(user.ID, "", "", displayName, "Asia/Tehran", "", "", "")
	}

	// Create welcome notification
	models.CreateNotification(user.ID, "خوش آمدید به ZedProxy!", "حساب کاربری شما با موفقیت ایجاد شد.", "success", "/user/dashboard")

	// Log activity
	models.LogUserActivity(user.ID, "register", "ثبت‌نام جدید", models.HashString(c.ClientIP()), c.Request.UserAgent())

	// Set session
	sess := sessions.Default(c)
	sess.Set("user_id", user.ID)
	sess.Set("user_role", user.Role)
	sess.Save()

	c.Redirect(http.StatusFound, "/user/dashboard")
}

// ─── Login ────────────────────────────────────────────

func AuthLoginPage(c *gin.Context) {
	if isLoggedIn(c) {
		c.Redirect(http.StatusFound, "/user/dashboard")
		return
	}
	next := c.Query("next")
	renderAuth(c, "login", map[string]interface{}{"Title": "ورود", "Next": next})
}

func AuthLoginPost(c *gin.Context) {
	if isLoggedIn(c) {
		c.Redirect(http.StatusFound, "/user/dashboard")
		return
	}

	identifier := strings.TrimSpace(c.PostForm("identifier"))
	password := c.PostForm("password")
	next := c.PostForm("next")

	data := map[string]interface{}{
		"Title":      "ورود",
		"Identifier": identifier,
		"Next":       next,
	}
	fail := func(msg string) {
		data["Error"] = msg
		renderAuth(c, "login", data)
	}

	if identifier == "" || password == "" {
		fail("ایمیل/موبایل و رمز عبور الزامی است")
		return
	}

	user, err := models.GetUserByEmailOrPhone(identifier)
	if err != nil {
		// Generic error to prevent user enumeration
		fail("اطلاعات وارد شده صحیح نیست")
		return
	}

	if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(password)); err != nil {
		models.LogUserActivity(user.ID, "login_failed", "ورود ناموفق", models.HashString(c.ClientIP()), c.Request.UserAgent())
		fail("اطلاعات وارد شده صحیح نیست")
		return
	}

	if user.IsBlocked() {
		fail("حساب کاربری شما مسدود شده است. با پشتیبانی تماس بگیرید")
		return
	}

	models.UpdateUserLastLogin(user.ID, models.HashString(c.ClientIP()))
	models.LogUserActivity(user.ID, "login", "ورود موفق", models.HashString(c.ClientIP()), c.Request.UserAgent())

	sess := sessions.Default(c)
	sess.Set("user_id", user.ID)
	sess.Set("user_role", user.Role)
	sess.Save()

	if next != "" && strings.HasPrefix(next, "/") && !strings.HasPrefix(next, "//") {
		c.Redirect(http.StatusFound, next)
		return
	}
	c.Redirect(http.StatusFound, "/user/dashboard")
}

// ─── Logout ───────────────────────────────────────────

func AuthLogout(c *gin.Context) {
	sess := sessions.Default(c)
	uid := currentUserID(c)
	if uid > 0 {
		models.LogUserActivity(uid, "logout", "خروج از حساب", models.HashString(c.ClientIP()), c.Request.UserAgent())
	}
	sess.Delete("user_id")
	sess.Delete("user_role")
	sess.Save()
	c.Redirect(http.StatusFound, "/auth/login")
}

// ─── Forgot Password ──────────────────────────────────

func AuthForgotPasswordPage(c *gin.Context) {
	if isLoggedIn(c) {
		c.Redirect(http.StatusFound, "/user/dashboard")
		return
	}
	renderAuth(c, "forgot-password", map[string]interface{}{"Title": "فراموشی رمز عبور"})
}

func AuthForgotPasswordPost(c *gin.Context) {
	identifier := strings.TrimSpace(c.PostForm("identifier"))
	data := map[string]interface{}{"Title": "فراموشی رمز عبور"}

	// Generic success message always — prevent user enumeration
	data["Success"] = "اگر این ایمیل یا شماره در سیستم ثبت شده باشد، لینک بازیابی برای شما ارسال می‌شود."

	if identifier == "" {
		renderAuth(c, "forgot-password", data)
		return
	}

	user, err := models.GetUserByEmailOrPhone(identifier)
	if err == nil && !user.IsBlocked() {
		token, err := models.CreatePasswordResetToken(user.ID)
		if err == nil {
			// In production: send email/SMS with token
			// For v1: show the link directly (dev convenience)
			_ = token
			// TODO: send via email
		}
	}

	renderAuth(c, "forgot-password", data)
}

// ─── Reset Password ───────────────────────────────────

func AuthResetPasswordPage(c *gin.Context) {
	if isLoggedIn(c) {
		c.Redirect(http.StatusFound, "/user/dashboard")
		return
	}
	token := c.Query("token")
	data := map[string]interface{}{"Title": "بازیابی رمز عبور", "Token": token}
	if token == "" {
		data["Error"] = "لینک بازیابی نامعتبر است"
	}
	renderAuth(c, "reset-password", data)
}

func AuthResetPasswordPost(c *gin.Context) {
	token := c.PostForm("token")
	password := c.PostForm("password")
	passwordConfirm := c.PostForm("password_confirm")

	data := map[string]interface{}{"Title": "بازیابی رمز عبور", "Token": token}
	fail := func(msg string) {
		data["Error"] = msg
		renderAuth(c, "reset-password", data)
	}

	if token == "" {
		fail("لینک بازیابی نامعتبر است")
		return
	}
	if len(password) < 8 {
		fail("رمز عبور باید حداقل ۸ کاراکتر باشد")
		return
	}
	if password != passwordConfirm {
		fail("رمز عبور و تکرار آن یکسان نیستند")
		return
	}

	userID, err := models.ValidatePasswordResetToken(token)
	if err != nil {
		fail("لینک بازیابی منقضی یا نامعتبر است")
		return
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(password), 12)
	if err != nil {
		fail("خطای سیستمی")
		return
	}
	if err := models.UpdateUserPassword(userID, string(hash)); err != nil {
		fail("خطا در تغییر رمز عبور")
		return
	}

	models.LogUserActivity(userID, "password_reset", "رمز عبور از طریق لینک بازیابی تغییر یافت", models.HashString(c.ClientIP()), c.Request.UserAgent())

	renderAuth(c, "reset-password", map[string]interface{}{
		"Title":   "بازیابی رمز عبور",
		"Success": "رمز عبور با موفقیت تغییر یافت. می‌توانید وارد شوید.",
	})
}
