package handlers

import (
	"fmt"
	"net/http"
	"strconv"

	"github.com/gin-contrib/sessions"
	"github.com/gin-gonic/gin"

	"zedproxy/internal/database"
)

// Product represents a product/plan for sale.
type Product struct {
	ID            int64
	Title         string
	Subtitle      string
	Description   string
	PriceIRR      int64
	PriceUSD      float64
	DurationDays  int
	TrafficGB     int
	DeviceLimit   int
	Category      string
	BadgeText     string
	IconPath      string
	IsActive      bool
	IsFeatured    bool
	SortOrder     int
	MarzbanPanelID *int64
	CreatedAt     string
	UpdatedAt     string
}

func getProducts() ([]Product, error) {
	rows, err := database.DB.Query(
		`SELECT id, title, subtitle, description, price_irr, price_usd, duration_days, traffic_gb,
		        device_limit, category, badge_text, icon_path, is_active, is_featured, sort_order, created_at
		 FROM products ORDER BY sort_order ASC, id DESC`,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var products []Product
	for rows.Next() {
		var p Product
		var active, featured int
		rows.Scan(&p.ID, &p.Title, &p.Subtitle, &p.Description, &p.PriceIRR, &p.PriceUSD,
			&p.DurationDays, &p.TrafficGB, &p.DeviceLimit, &p.Category, &p.BadgeText,
			&p.IconPath, &active, &featured, &p.SortOrder, &p.CreatedAt)
		p.IsActive = active == 1
		p.IsFeatured = featured == 1
		products = append(products, p)
	}
	return products, nil
}

func getProductByID(id int64) (*Product, error) {
	var p Product
	var active, featured int
	err := database.DB.QueryRow(
		`SELECT id, title, subtitle, description, price_irr, price_usd, duration_days, traffic_gb,
		        device_limit, category, badge_text, icon_path, is_active, is_featured, sort_order, created_at
		 FROM products WHERE id=?`, id,
	).Scan(&p.ID, &p.Title, &p.Subtitle, &p.Description, &p.PriceIRR, &p.PriceUSD,
		&p.DurationDays, &p.TrafficGB, &p.DeviceLimit, &p.Category, &p.BadgeText,
		&p.IconPath, &active, &featured, &p.SortOrder, &p.CreatedAt)
	if err != nil {
		return nil, err
	}
	p.IsActive = active == 1
	p.IsFeatured = featured == 1
	return &p, nil
}

// AdminProductsPage lists all products.
func AdminProductsPage(c *gin.Context) {
	products, err := getProducts()
	data := adminData(c, "products")
	data["Title"] = "محصولات"
	data["Section"] = "products"
	if err != nil {
		data["Error"] = err.Error()
	}
	data["Products"] = products

	sess := sessions.Default(c)
	if f := sess.Get("flash_ok"); f != nil {
		data["FlashOK"] = f.(string)
		sess.Delete("flash_ok")
		sess.Save()
	}

	t, err2 := getAdminTemplate("products")
	if err2 != nil {
		renderAdminError(c, fmt.Sprintf("template error: %v", err2))
		return
	}
	c.Header("Content-Type", "text/html; charset=utf-8")
	t.ExecuteTemplate(c.Writer, "admin", data) //nolint:errcheck
}

// AdminProductNew shows the new product form.
func AdminProductNew(c *gin.Context) {
	data := adminData(c, "products")
	data["Title"] = "محصول جدید"
	data["Section"] = "products"
	data["Product"] = &Product{IsActive: true, DurationDays: 30, TrafficGB: 10, DeviceLimit: 1, Category: "month_1"}
	data["IsNew"] = true

	t, err := getAdminTemplate("product-form")
	if err != nil {
		renderAdminError(c, fmt.Sprintf("template error: %v", err))
		return
	}
	c.Header("Content-Type", "text/html; charset=utf-8")
	t.ExecuteTemplate(c.Writer, "admin", data) //nolint:errcheck
}

// AdminProductEdit shows the edit product form.
func AdminProductEdit(c *gin.Context) {
	id, _ := strconv.ParseInt(c.Param("id"), 10, 64)
	p, err := getProductByID(id)
	if err != nil {
		c.Redirect(http.StatusFound, "/zed-admin/products")
		return
	}
	data := adminData(c, "products")
	data["Title"] = "ویرایش محصول"
	data["Section"] = "products"
	data["Product"] = p
	data["IsNew"] = false

	t, err := getAdminTemplate("product-form")
	if err != nil {
		renderAdminError(c, fmt.Sprintf("template error: %v", err))
		return
	}
	c.Header("Content-Type", "text/html; charset=utf-8")
	t.ExecuteTemplate(c.Writer, "admin", data) //nolint:errcheck
}

// AdminProductSave creates or updates a product.
func AdminProductSave(c *gin.Context) {
	sess := sessions.Default(c)

	idStr := c.PostForm("id")
	title := c.PostForm("title")
	subtitle := c.PostForm("subtitle")
	description := c.PostForm("description")
	priceIRR, _ := strconv.ParseInt(c.PostForm("price_irr"), 10, 64)
	priceUSD, _ := strconv.ParseFloat(c.PostForm("price_usd"), 64)
	durationDays, _ := strconv.Atoi(c.PostForm("duration_days"))
	trafficGB, _ := strconv.Atoi(c.PostForm("traffic_gb"))
	deviceLimit, _ := strconv.Atoi(c.PostForm("device_limit"))
	category := c.PostForm("category")
	badgeText := c.PostForm("badge_text")
	sortOrder, _ := strconv.Atoi(c.PostForm("sort_order"))
	isActive := 0
	if c.PostForm("is_active") == "1" {
		isActive = 1
	}
	isFeatured := 0
	if c.PostForm("is_featured") == "1" {
		isFeatured = 1
	}

	if title == "" {
		sess.Set("flash_err", "عنوان محصول اجباری است")
		sess.Save()
		c.Redirect(http.StatusFound, "/zed-admin/products")
		return
	}

	if idStr == "" || idStr == "0" {
		// new
		_, err := database.DB.Exec(
			`INSERT INTO products (title, subtitle, description, price_irr, price_usd, duration_days, traffic_gb, device_limit, category, badge_text, sort_order, is_active, is_featured)
			 VALUES (?,?,?,?,?,?,?,?,?,?,?,?,?)`,
			title, subtitle, description, priceIRR, priceUSD, durationDays, trafficGB, deviceLimit, category, badgeText, sortOrder, isActive, isFeatured,
		)
		if err != nil {
			sess.Set("flash_err", "خطا در ذخیره: "+err.Error())
		} else {
			sess.Set("flash_ok", "محصول ایجاد شد")
			LogAdminActivity(c, "product_created", "محصول جدید: "+title)
		}
	} else {
		id, _ := strconv.ParseInt(idStr, 10, 64)
		_, err := database.DB.Exec(
			`UPDATE products SET title=?, subtitle=?, description=?, price_irr=?, price_usd=?, duration_days=?, traffic_gb=?, device_limit=?, category=?, badge_text=?, sort_order=?, is_active=?, is_featured=?, updated_at=CURRENT_TIMESTAMP WHERE id=?`,
			title, subtitle, description, priceIRR, priceUSD, durationDays, trafficGB, deviceLimit, category, badgeText, sortOrder, isActive, isFeatured, id,
		)
		if err != nil {
			sess.Set("flash_err", "خطا در ذخیره: "+err.Error())
		} else {
			sess.Set("flash_ok", "محصول بروزرسانی شد")
			LogAdminActivity(c, "product_updated", "محصول ویرایش شد: "+title)
		}
	}
	sess.Save()
	c.Redirect(http.StatusFound, "/zed-admin/products")
}

// AdminProductDelete deletes a product.
func AdminProductDelete(c *gin.Context) {
	id, _ := strconv.ParseInt(c.Param("id"), 10, 64)
	database.DB.Exec("DELETE FROM products WHERE id=?", id) //nolint:errcheck
	LogAdminActivity(c, "product_deleted", fmt.Sprintf("product id=%d", id))
	sess := sessions.Default(c)
	sess.Set("flash_ok", "محصول حذف شد")
	sess.Save()
	c.Redirect(http.StatusFound, "/zed-admin/products")
}

// AdminProductToggle toggles product active status.
func AdminProductToggle(c *gin.Context) {
	id, _ := strconv.ParseInt(c.Param("id"), 10, 64)
	database.DB.Exec("UPDATE products SET is_active = CASE WHEN is_active=1 THEN 0 ELSE 1 END, updated_at=CURRENT_TIMESTAMP WHERE id=?", id) //nolint:errcheck
	c.JSON(http.StatusOK, gin.H{"ok": true})
}

