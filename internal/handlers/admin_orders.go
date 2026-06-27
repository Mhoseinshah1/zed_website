package handlers

import (
	"fmt"
	"net/http"
	"strconv"

	"github.com/gin-contrib/sessions"
	"github.com/gin-gonic/gin"
	"zedproxy/internal/models"
)

func AdminOrdersPage(c *gin.Context) {
	status := c.Query("status")
	search := c.Query("search")
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	if page < 1 {
		page = 1
	}
	pageSize := 25

	orders, total, _ := models.GetAllSalesOrders(status, search, page, pageSize)
	pages := (total + pageSize - 1) / pageSize
	pageNums := make([]int, pages)
	for i := range pageNums {
		pageNums[i] = i + 1
	}

	data := adminData(c, "orders")
	data["Title"] = "سفارش‌ها"
	data["Orders"] = orders
	data["Total"] = total
	data["Page"] = page
	data["Pages"] = pages
	data["PageNums"] = pageNums
	data["StatusFilter"] = status
	data["Search"] = search
	data["Statuses"] = models.AllSalesOrderStatuses()
	renderAdmin(c, "orders", data)
}

func AdminOrderDetail(c *gin.Context) {
	idStr := c.Param("id")

	// Accept both numeric ID and public_id (ZP...)
	var order *models.SalesOrder
	var err error

	id, numErr := strconv.ParseInt(idStr, 10, 64)
	if numErr == nil {
		// numeric — fetch by ID
		orders, _, qErr := models.GetAllSalesOrders("", "", 1, 1000)
		if qErr == nil {
			for _, o := range orders {
				if o.ID == id {
					order = o
					break
				}
			}
		}
	} else {
		// treat as public_id
		order, err = models.GetSalesOrderByPublicID(idStr, 0)
	}

	if order == nil || err != nil {
		c.Redirect(http.StatusFound, "/zed-admin/orders")
		return
	}

	sess := sessions.Default(c)
	data := adminData(c, "orders")
	data["Title"] = "جزئیات سفارش"
	data["Order"] = order
	data["Statuses"] = models.AllSalesOrderStatuses()
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
	renderAdmin(c, "order-detail", data)
}

func AdminOrderUpdateStatus(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		c.Redirect(http.StatusFound, "/zed-admin/orders")
		return
	}
	status := c.PostForm("status")

	sess := sessions.Default(c)
	if err := models.UpdateSalesOrderStatus(id, status); err != nil {
		sess.Set("flash_err", "خطا در تغییر وضعیت: "+err.Error())
		sess.Save()
		c.Redirect(http.StatusFound, fmt.Sprintf("/zed-admin/orders/%d", id))
		return
	}

	// Fetch order to notify user
	if orders, _, qErr := models.GetAllSalesOrders("", "", 1, 1000); qErr == nil {
		for _, o := range orders {
			if o.ID == id {
				label := models.SalesOrderStatusLabel(status)
				notifyMsg := "وضعیت سفارش " + o.PublicID + " به «" + label + "» تغییر کرد."
				models.CreateNotification(o.UserID, "وضعیت سفارش تغییر کرد", notifyMsg, "order", "/user/orders/"+o.PublicID) //nolint:errcheck
				break
			}
		}
	}

	LogAdminActivity(c, "order_status_changed", fmt.Sprintf("order %s → %s", idStr, status))
	sess.Set("flash_ok", "وضعیت سفارش تغییر کرد.")
	sess.Save()
	c.Redirect(http.StatusFound, fmt.Sprintf("/zed-admin/orders/%d", id))
}

func AdminOrderUpdateNote(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		c.Redirect(http.StatusFound, "/zed-admin/orders")
		return
	}
	adminNote := c.PostForm("admin_note")
	manualSubURL := c.PostForm("manual_subscription_url")
	manualNote := c.PostForm("manual_service_note")

	sess := sessions.Default(c)
	if err := models.UpdateSalesOrderAdminFields(id, adminNote, manualSubURL, manualNote); err != nil {
		sess.Set("flash_err", "خطا در ذخیره یادداشت: "+err.Error())
	} else {
		LogAdminActivity(c, "order_note_updated", fmt.Sprintf("order %s", idStr))
		sess.Set("flash_ok", "اطلاعات سفارش ذخیره شد.")
	}
	sess.Save()
	c.Redirect(http.StatusFound, fmt.Sprintf("/zed-admin/orders/%d", id))
}
