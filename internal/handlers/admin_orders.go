package handlers

import (
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"zedproxy/internal/database"
)

type AdminOrder struct {
	ID            int64
	OrderNumber   string
	UserEmail     string
	UserPhone     string
	PlanName      string
	Amount        int64
	Currency      string
	PaymentStatus string
	OrderStatus   string
	CreatedAt     time.Time
}

func AdminOrdersPage(c *gin.Context) {
	search := c.Query("search")
	paymentStatus := c.Query("payment_status")
	orderStatus := c.Query("order_status")
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	if page < 1 {
		page = 1
	}
	pageSize := 25
	offset := (page - 1) * pageSize

	query := `SELECT o.id, o.order_number, COALESCE(u.email,''), COALESCE(u.phone,''),
		COALESCE(o.plan_name,''), o.amount, o.currency, o.payment_status, o.order_status, o.created_at
		FROM user_orders o
		LEFT JOIN users u ON u.id = o.user_id
		WHERE 1=1`
	countQuery := `SELECT COUNT(*) FROM user_orders o LEFT JOIN users u ON u.id = o.user_id WHERE 1=1`
	args := []interface{}{}

	if search != "" {
		query += " AND (o.order_number LIKE ? OR u.email LIKE ? OR u.phone LIKE ? OR o.plan_name LIKE ?)"
		countQuery += " AND (o.order_number LIKE ? OR u.email LIKE ? OR u.phone LIKE ? OR o.plan_name LIKE ?)"
		like := "%" + search + "%"
		args = append(args, like, like, like, like)
	}
	if paymentStatus != "" {
		query += " AND o.payment_status = ?"
		countQuery += " AND o.payment_status = ?"
		args = append(args, paymentStatus)
	}
	if orderStatus != "" {
		query += " AND o.order_status = ?"
		countQuery += " AND o.order_status = ?"
		args = append(args, orderStatus)
	}

	var total int
	database.DB.QueryRow(countQuery, args...).Scan(&total)

	query += " ORDER BY o.created_at DESC LIMIT ? OFFSET ?"
	rows, err := database.DB.Query(query, append(args, pageSize, offset)...)

	var orders []AdminOrder
	if err == nil {
		defer rows.Close()
		for rows.Next() {
			var o AdminOrder
			rows.Scan(&o.ID, &o.OrderNumber, &o.UserEmail, &o.UserPhone,
				&o.PlanName, &o.Amount, &o.Currency, &o.PaymentStatus, &o.OrderStatus, &o.CreatedAt)
			orders = append(orders, o)
		}
	}

	pages := (total + pageSize - 1) / pageSize

	data := adminData(c, "orders")
	data["Title"] = "سفارش‌ها"
	data["Orders"] = orders
	data["Total"] = total
	data["Page"] = page
	data["Pages"] = pages
	data["Search"] = search
	data["PaymentStatus"] = paymentStatus
	data["OrderStatus"] = orderStatus
	renderAdmin(c, "orders", data)
}

func AdminOrderDetail(c *gin.Context) {
	id := c.Param("id")
	var o AdminOrder
	err := database.DB.QueryRow(`SELECT o.id, o.order_number, COALESCE(u.email,''), COALESCE(u.phone,''),
		COALESCE(o.plan_name,''), o.amount, o.currency, o.payment_status, o.order_status, o.created_at
		FROM user_orders o LEFT JOIN users u ON u.id = o.user_id WHERE o.id = ?`, id).
		Scan(&o.ID, &o.OrderNumber, &o.UserEmail, &o.UserPhone, &o.PlanName, &o.Amount, &o.Currency,
			&o.PaymentStatus, &o.OrderStatus, &o.CreatedAt)
	if err != nil {
		c.Redirect(http.StatusFound, "/zed-admin/orders")
		return
	}
	data := adminData(c, "orders")
	data["Title"] = "جزئیات سفارش"
	data["Order"] = o
	renderAdmin(c, "orders", data)
}
