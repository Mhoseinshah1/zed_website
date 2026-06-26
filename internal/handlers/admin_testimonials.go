package handlers

import (
	"net/http"
	"strconv"
	"time"

	"github.com/gin-contrib/sessions"
	"github.com/gin-gonic/gin"
	"zedproxy/internal/database"
)

type Testimonial struct {
	ID            int64
	CustomerAlias string
	Text          string
	Rating        int
	ServiceType   string
	IsActive      bool
	SortOrder     int
	CreatedAt     time.Time
	UpdatedAt     time.Time
}

func getTestimonialByID(id int) (*Testimonial, error) {
	var t Testimonial
	err := database.DB.QueryRow(`SELECT id, customer_alias, text, rating, service_type, is_active, sort_order, created_at, updated_at FROM testimonials WHERE id=?`, id).
		Scan(&t.ID, &t.CustomerAlias, &t.Text, &t.Rating, &t.ServiceType, &t.IsActive, &t.SortOrder, &t.CreatedAt, &t.UpdatedAt)
	return &t, err
}

func AdminTestimonialsPage(c *gin.Context) {
	rows, _ := database.DB.Query(`SELECT id, customer_alias, text, rating, service_type, is_active, sort_order, created_at, updated_at FROM testimonials ORDER BY sort_order, created_at DESC`)
	var items []Testimonial
	if rows != nil {
		defer rows.Close()
		for rows.Next() {
			var t Testimonial
			rows.Scan(&t.ID, &t.CustomerAlias, &t.Text, &t.Rating, &t.ServiceType, &t.IsActive, &t.SortOrder, &t.CreatedAt, &t.UpdatedAt)
			items = append(items, t)
		}
	}
	data := adminData(c, "testimonials")
	data["Title"] = "نظرات مشتریان"
	data["Testimonials"] = items
	renderAdmin(c, "testimonials", data)
}

func AdminTestimonialNew(c *gin.Context) {
	data := adminData(c, "testimonials")
	data["Title"] = "نظر جدید"
	data["Testimonial"] = nil
	renderAdmin(c, "testimonial-form", data)
}

func AdminTestimonialEdit(c *gin.Context) {
	id, _ := strconv.Atoi(c.Param("id"))
	t, err := getTestimonialByID(id)
	if err != nil {
		c.Redirect(http.StatusFound, "/zed-admin/testimonials")
		return
	}
	data := adminData(c, "testimonials")
	data["Title"] = "ویرایش نظر"
	data["Testimonial"] = t
	renderAdmin(c, "testimonial-form", data)
}

func AdminTestimonialSave(c *gin.Context) {
	idStr := c.PostForm("id")
	rating, _ := strconv.Atoi(c.PostForm("rating"))
	sortOrder, _ := strconv.Atoi(c.PostForm("sort_order"))
	isActive := c.PostForm("is_active") == "1"

	sess := sessions.Default(c)
	if idStr == "" || idStr == "0" {
		_, err := database.DB.Exec(`INSERT INTO testimonials (customer_alias, text, rating, service_type, is_active, sort_order) VALUES (?,?,?,?,?,?)`,
			c.PostForm("customer_alias"), c.PostForm("text"), rating, c.PostForm("service_type"), boolToInt(isActive), sortOrder)
		if err != nil {
			sess.AddFlash("خطا در ذخیره: "+err.Error(), "ok")
		} else {
			sess.AddFlash("نظر با موفقیت ثبت شد", "ok")
		}
	} else {
		_, err := database.DB.Exec(`UPDATE testimonials SET customer_alias=?, text=?, rating=?, service_type=?, is_active=?, sort_order=?, updated_at=CURRENT_TIMESTAMP WHERE id=?`,
			c.PostForm("customer_alias"), c.PostForm("text"), rating, c.PostForm("service_type"), boolToInt(isActive), sortOrder, idStr)
		if err != nil {
			sess.AddFlash("خطا در بروزرسانی: "+err.Error(), "ok")
		} else {
			sess.AddFlash("نظر با موفقیت بروزرسانی شد", "ok")
		}
	}
	sess.Save()
	c.Redirect(http.StatusFound, "/zed-admin/testimonials")
}

func AdminTestimonialDelete(c *gin.Context) {
	id := c.Param("id")
	database.DB.Exec(`DELETE FROM testimonials WHERE id=?`, id)
	sess := sessions.Default(c)
	sess.AddFlash("نظر حذف شد", "ok")
	sess.Save()
	c.Redirect(http.StatusFound, "/zed-admin/testimonials")
}

func AdminTestimonialToggle(c *gin.Context) {
	id := c.Param("id")
	database.DB.Exec(`UPDATE testimonials SET is_active = CASE WHEN is_active=1 THEN 0 ELSE 1 END, updated_at=CURRENT_TIMESTAMP WHERE id=?`, id)
	c.Redirect(http.StatusFound, "/zed-admin/testimonials")
}

func boolToInt(b bool) int {
	if b {
		return 1
	}
	return 0
}
