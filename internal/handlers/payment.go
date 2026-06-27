package handlers

import (
	"net/http"
	"strconv"

	"github.com/gin-contrib/sessions"
	"github.com/gin-gonic/gin"
	"zedproxy/internal/models"
)

// CheckoutPage shows the checkout confirmation page for a product.
// GET /checkout/:product_id
func CheckoutPage(c *gin.Context) {
	idStr := c.Param("product_id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		c.Redirect(http.StatusFound, "/plans")
		return
	}

	product, err := getProductByID(id)
	if err != nil || !product.IsActive {
		c.Status(http.StatusNotFound)
		data := basePageData("plans")
		data["Title"] = "محصول یافت نشد"
		renderPage(c, "404", data)
		return
	}

	data := basePageData("plans")
	data["Title"] = "تایید سفارش - " + product.Title
	data["Product"] = product
	data["LoggedIn"] = isLoggedIn(c)
	renderPage(c, "checkout", data)
}

// CheckoutCreate processes the order submission.
// POST /checkout/:product_id
func CheckoutCreate(c *gin.Context) {
	if !isLoggedIn(c) {
		c.Redirect(http.StatusFound, "/auth/login?next=/checkout/"+c.Param("product_id"))
		return
	}

	idStr := c.Param("product_id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		c.Redirect(http.StatusFound, "/plans")
		return
	}

	product, err := getProductByID(id)
	if err != nil || !product.IsActive {
		c.Redirect(http.StatusFound, "/plans")
		return
	}

	uid := currentUserID(c)
	if uid == 0 {
		c.Redirect(http.StatusFound, "/auth/login")
		return
	}

	publicID, err := models.CreateSalesOrder(uid, product.ID, product.Title, product.PriceIRR, product.PriceUSD)
	if err != nil {
		sess := sessions.Default(c)
		sess.Set("flash_err", "خطا در ثبت سفارش. دوباره امتحان کنید.")
		sess.Save()
		c.Redirect(http.StatusFound, "/checkout/"+idStr)
		return
	}

	sess := sessions.Default(c)
	sess.Set("flash_ok", "سفارش شما ثبت شد.")
	sess.Save()
	c.Redirect(http.StatusFound, "/user/orders/"+publicID)
}

// CheckoutPayPage is a stub for the future payment gateway step.
// GET /checkout/:order_id/pay
func CheckoutPayPage(c *gin.Context) {
	c.Redirect(http.StatusFound, "/user/orders")
}

// NOWPaymentsWebhook handles IPN callbacks from NOWPayments.
func NOWPaymentsWebhook(c *gin.Context) {
	// TODO: verify signature and update order status
	c.JSON(http.StatusOK, gin.H{"ok": true})
}

// CheckoutSuccess shows the success page after payment.
func CheckoutSuccess(c *gin.Context) {
	c.Redirect(http.StatusFound, "/user/orders")
}

// CheckoutCancel shows the cancel page after payment cancellation.
func CheckoutCancel(c *gin.Context) {
	c.Redirect(http.StatusFound, "/user/orders")
}
