package handlers

import (
	"net/http"
	"strings"

	"github.com/gin-contrib/sessions"
	"github.com/gin-gonic/gin"
	"zedproxy/internal/models"
)

func AdminSupportTicketsPage(c *gin.Context) {
	status := c.Query("status")
	category := c.Query("category")
	page := 1

	tickets, total, _ := models.ListAllTickets(models.TicketListFilter{
		Status:   status,
		Category: category,
		Page:     page,
		PageSize: 30,
	})

	data := adminData(c, "support")
	data["Title"] = "تیکت‌های پشتیبانی"
	data["Tickets"] = tickets
	data["Total"] = total
	data["FilterStatus"] = status
	data["FilterCategory"] = category
	renderAdmin(c, "support-tickets", data)
}

func AdminSupportTicketDetailPage(c *gin.Context) {
	number := c.Param("ticket_number")
	ticket, err := models.GetTicketByNumber(number, 0)
	if err != nil {
		c.Redirect(http.StatusFound, "/zed-admin/support/tickets")
		return
	}

	sess := sessions.Default(c)
	data := adminData(c, "support")
	data["Title"] = "تیکت " + number
	data["Ticket"] = ticket

	if f := sess.Get("flash_ok"); f != nil {
		data["FlashOK"] = f.(string)
		sess.Delete("flash_ok")
		sess.Save()
	}
	renderAdmin(c, "support-ticket-detail", data)
}

func AdminSupportTicketReply(c *gin.Context) {
	number := c.Param("ticket_number")
	message := strings.TrimSpace(c.PostForm("message"))

	ticket, err := models.GetTicketByNumber(number, 0)
	sess := sessions.Default(c)
	if err != nil {
		c.Redirect(http.StatusFound, "/zed-admin/support/tickets")
		return
	}
	if message == "" {
		c.Redirect(http.StatusFound, "/zed-admin/support/tickets/"+number)
		return
	}

	adminID := int64(0)
	if v := sess.Get("admin_id"); v != nil {
		if id, ok := v.(int); ok {
			adminID = int64(id)
		}
	}

	models.AddTicketMessage(ticket.ID, 0, adminID, "admin", message)
	models.UpdateTicketStatus(ticket.ID, "waiting_user")
	models.CreateNotification(ticket.UserID, "پاسخ تیکت", "تیکت شما پاسخ داده شد: "+ticket.TicketNumber, "info", "/user/tickets/"+number)

	sess.Set("flash_ok", "پاسخ ارسال شد")
	sess.Save()
	c.Redirect(http.StatusFound, "/zed-admin/support/tickets/"+number)
}

func AdminSupportTicketSetStatus(c *gin.Context) {
	number := c.Param("ticket_number")
	status := c.PostForm("status")

	validStatuses := map[string]bool{"open": true, "waiting_admin": true, "waiting_user": true, "closed": true}
	if !validStatuses[status] {
		c.Redirect(http.StatusFound, "/zed-admin/support/tickets/"+number)
		return
	}

	ticket, err := models.GetTicketByNumber(number, 0)
	sess := sessions.Default(c)
	if err != nil {
		c.Redirect(http.StatusFound, "/zed-admin/support/tickets")
		return
	}

	models.UpdateTicketStatus(ticket.ID, status)

	statusLabels := map[string]string{
		"open":          "باز",
		"waiting_admin": "در انتظار پشتیبانی",
		"waiting_user":  "در انتظار کاربر",
		"closed":        "بسته",
	}
	sess.Set("flash_ok", "وضعیت تیکت به «"+statusLabels[status]+"» تغییر یافت")
	sess.Save()
	c.Redirect(http.StatusFound, "/zed-admin/support/tickets/"+number)
}
