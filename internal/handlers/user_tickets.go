package handlers

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"zedproxy/internal/models"
)

var ticketCategories = []string{
	"مشکل اتصال",
	"مشکل پرداخت",
	"تمدید سرویس",
	"آموزش نصب",
	"درخواست تغییر سرویس",
	"سایر موارد",
}

func UserTicketsPage(c *gin.Context) {
	uid := currentUserID(c)
	tickets, _ := models.GetUserTickets(uid)
	renderUser(c, "tickets", map[string]interface{}{
		"Title":   "تیکت‌های پشتیبانی",
		"Tickets": tickets,
	})
}

func UserTicketNewPage(c *gin.Context) {
	renderUser(c, "ticket-new", map[string]interface{}{
		"Title":      "ثبت تیکت جدید",
		"Categories": ticketCategories,
	})
}

func UserTicketCreate(c *gin.Context) {
	uid := currentUserID(c)
	subject := strings.TrimSpace(c.PostForm("subject"))
	category := c.PostForm("category")
	message := strings.TrimSpace(c.PostForm("message"))

	data := map[string]interface{}{
		"Title":      "ثبت تیکت جدید",
		"Categories": ticketCategories,
		"Subject":    subject,
		"Category":   category,
		"Message":    message,
	}

	if subject == "" || message == "" {
		data["Error"] = "موضوع و پیام الزامی است"
		renderUser(c, "ticket-new", data)
		return
	}
	if category == "" {
		data["Error"] = "دسته‌بندی را انتخاب کنید"
		renderUser(c, "ticket-new", data)
		return
	}

	ticket, err := models.CreateSupportTicket(uid, subject, category, message)
	if err != nil {
		data["Error"] = "خطا در ثبت تیکت. لطفاً مجدداً تلاش کنید"
		renderUser(c, "ticket-new", data)
		return
	}

	models.CreateNotification(uid, "تیکت ثبت شد", "تیکت شما با شماره "+ticket.TicketNumber+" ثبت شد.", "info", "/user/tickets/"+ticket.TicketNumber)
	models.LogUserActivity(uid, "ticket_created", "تیکت جدید: "+ticket.TicketNumber, models.HashString(c.ClientIP()), c.Request.UserAgent())

	c.Redirect(http.StatusFound, "/user/tickets/"+ticket.TicketNumber)
}

func UserTicketDetailPage(c *gin.Context) {
	uid := currentUserID(c)
	number := c.Param("ticket_number")
	ticket, err := models.GetTicketByNumber(number, uid)
	if err != nil {
		c.Redirect(http.StatusFound, "/user/tickets")
		return
	}
	renderUser(c, "ticket-detail", map[string]interface{}{
		"Title":  "تیکت " + number,
		"Ticket": ticket,
	})
}

func UserTicketReply(c *gin.Context) {
	uid := currentUserID(c)
	number := c.Param("ticket_number")
	message := strings.TrimSpace(c.PostForm("message"))

	ticket, err := models.GetTicketByNumber(number, uid)
	if err != nil {
		c.Redirect(http.StatusFound, "/user/tickets")
		return
	}
	if ticket.Status == "closed" {
		c.Redirect(http.StatusFound, "/user/tickets/"+number)
		return
	}
	if message == "" {
		c.Redirect(http.StatusFound, "/user/tickets/"+number)
		return
	}

	models.AddTicketMessage(ticket.ID, uid, 0, "user", message)
	models.UpdateTicketStatus(ticket.ID, "waiting_admin")
	models.LogUserActivity(uid, "ticket_replied", "پاسخ به تیکت: "+number, models.HashString(c.ClientIP()), c.Request.UserAgent())

	c.Redirect(http.StatusFound, "/user/tickets/"+number)
}
