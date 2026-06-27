package models

import (
	"crypto/rand"
	"fmt"
	"time"

	"zedproxy/internal/database"
)

// SalesOrder represents a product order placed through the website.
type SalesOrder struct {
	ID                   int64
	PublicID             string
	UserID               int64
	ProductID            int64
	ProductTitleSnapshot string
	Status               string
	PriceIRR             int64
	PriceUSD             float64
	PaymentGateway       string
	PaymentID            string
	PaymentStatus        string
	SubscriptionURL      string
	ManualSubscriptionURL string
	ManualServiceNote    string
	AdminNote            string
	// Joined fields
	UserEmail string
	UserPhone string
	CreatedAt time.Time
	UpdatedAt time.Time
}

// StatusLabel returns a Persian label for an order status.
func (o *SalesOrder) StatusLabel() string {
	return SalesOrderStatusLabel(o.Status)
}

// SalesOrderStatusLabel maps order status codes to Persian labels.
func SalesOrderStatusLabel(status string) string {
	switch status {
	case "pending_payment":
		return "در انتظار پرداخت"
	case "payment_review":
		return "در حال بررسی پرداخت"
	case "paid":
		return "پرداخت شده"
	case "provisioning_pending":
		return "در انتظار فعال‌سازی"
	case "completed":
		return "تکمیل شده"
	case "canceled", "cancelled":
		return "لغو شده"
	case "failed":
		return "ناموفق"
	default:
		return status
	}
}

// AllSalesOrderStatuses returns all valid status values.
func AllSalesOrderStatuses() []string {
	return []string{
		"pending_payment",
		"payment_review",
		"paid",
		"provisioning_pending",
		"completed",
		"canceled",
		"failed",
	}
}

func generatePublicOrderID() string {
	b := make([]byte, 4)
	rand.Read(b) //nolint:errcheck
	return fmt.Sprintf("ZP%08X", b)
}

// CreateSalesOrder inserts a new order and returns the public_id.
func CreateSalesOrder(userID, productID int64, productTitle string, priceIRR int64, priceUSD float64) (string, error) {
	publicID := generatePublicOrderID()
	_, err := database.DB.Exec(
		`INSERT INTO orders
		 (public_id, user_id, product_id, product_title_snapshot, status, price_irr, price_usd, created_at, updated_at)
		 VALUES (?, ?, ?, ?, 'pending_payment', ?, ?, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP)`,
		publicID, userID, productID, productTitle, priceIRR, priceUSD,
	)
	if err != nil {
		return "", err
	}
	return publicID, nil
}

const salesOrderCols = `o.id, o.public_id, o.user_id, o.product_id,
	COALESCE(o.product_title_snapshot,''), o.status, o.price_irr, COALESCE(o.price_usd,0),
	COALESCE(o.payment_gateway,''), COALESCE(o.payment_id,''), COALESCE(o.payment_status,''),
	COALESCE(o.subscription_url,''), COALESCE(o.manual_subscription_url,''),
	COALESCE(o.manual_service_note,''), COALESCE(o.admin_note,''),
	COALESCE(u.email,''), COALESCE(u.phone,''),
	o.created_at, o.updated_at`

func scanSalesOrder(row interface{ Scan(...interface{}) error }) (*SalesOrder, error) {
	o := &SalesOrder{}
	err := row.Scan(
		&o.ID, &o.PublicID, &o.UserID, &o.ProductID,
		&o.ProductTitleSnapshot, &o.Status, &o.PriceIRR, &o.PriceUSD,
		&o.PaymentGateway, &o.PaymentID, &o.PaymentStatus,
		&o.SubscriptionURL, &o.ManualSubscriptionURL,
		&o.ManualServiceNote, &o.AdminNote,
		&o.UserEmail, &o.UserPhone,
		&o.CreatedAt, &o.UpdatedAt,
	)
	return o, err
}

// GetSalesOrderByPublicID fetches an order for a specific user (user=0 means admin — no user filter).
func GetSalesOrderByPublicID(publicID string, userID int64) (*SalesOrder, error) {
	q := `SELECT ` + salesOrderCols + ` FROM orders o
		  LEFT JOIN users u ON u.id = o.user_id
		  WHERE o.public_id = ?`
	args := []interface{}{publicID}
	if userID > 0 {
		q += " AND o.user_id = ?"
		args = append(args, userID)
	}
	return scanSalesOrder(database.DB.QueryRow(q, args...))
}

// GetSalesOrdersByUserID returns all orders for a user, newest first.
func GetSalesOrdersByUserID(userID int64) ([]*SalesOrder, error) {
	rows, err := database.DB.Query(
		`SELECT `+salesOrderCols+` FROM orders o
		 LEFT JOIN users u ON u.id = o.user_id
		 WHERE o.user_id = ? ORDER BY o.created_at DESC`, userID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var list []*SalesOrder
	for rows.Next() {
		o, err := scanSalesOrder(rows)
		if err == nil {
			list = append(list, o)
		}
	}
	return list, nil
}

// GetAllSalesOrders returns a paginated, optionally filtered list of all orders.
func GetAllSalesOrders(status, search string, page, pageSize int) ([]*SalesOrder, int, error) {
	if page < 1 {
		page = 1
	}
	offset := (page - 1) * pageSize

	base := ` FROM orders o LEFT JOIN users u ON u.id = o.user_id WHERE 1=1`
	args := []interface{}{}

	if status != "" {
		base += " AND o.status = ?"
		args = append(args, status)
	}
	if search != "" {
		like := "%" + search + "%"
		base += ` AND (o.public_id LIKE ? OR u.email LIKE ? OR u.phone LIKE ? OR o.product_title_snapshot LIKE ?)`
		args = append(args, like, like, like, like)
	}

	var total int
	database.DB.QueryRow("SELECT COUNT(*)"+base, args...).Scan(&total) //nolint:errcheck

	rows, err := database.DB.Query(
		"SELECT "+salesOrderCols+base+" ORDER BY o.created_at DESC LIMIT ? OFFSET ?",
		append(args, pageSize, offset)...,
	)
	if err != nil {
		return nil, total, err
	}
	defer rows.Close()
	var list []*SalesOrder
	for rows.Next() {
		o, err := scanSalesOrder(rows)
		if err == nil {
			list = append(list, o)
		}
	}
	return list, total, nil
}

// UpdateSalesOrderStatus changes an order's status and updates updated_at.
func UpdateSalesOrderStatus(id int64, status string) error {
	_, err := database.DB.Exec(
		`UPDATE orders SET status=?, updated_at=CURRENT_TIMESTAMP WHERE id=?`,
		status, id,
	)
	return err
}

// UpdateSalesOrderAdminFields saves admin note and optional manual fulfillment fields.
func UpdateSalesOrderAdminFields(id int64, adminNote, manualSubURL, manualNote string) error {
	_, err := database.DB.Exec(
		`UPDATE orders SET admin_note=?, manual_subscription_url=?, manual_service_note=?, updated_at=CURRENT_TIMESTAMP WHERE id=?`,
		adminNote, manualSubURL, manualNote, id,
	)
	return err
}
