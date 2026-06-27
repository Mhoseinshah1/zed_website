package handlers

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

// CheckoutCreate initiates the checkout flow for a product.
func CheckoutCreate(c *gin.Context) {
	// TODO: implement full checkout flow with NOWPayments
	c.JSON(http.StatusNotImplemented, gin.H{"error": "checkout not yet implemented"})
}

// CheckoutPayPage shows the payment page for an order.
func CheckoutPayPage(c *gin.Context) {
	c.JSON(http.StatusNotImplemented, gin.H{"error": "not implemented"})
}

// NOWPaymentsWebhook handles IPN callbacks from NOWPayments.
func NOWPaymentsWebhook(c *gin.Context) {
	// TODO: verify signature and update order status
	c.JSON(http.StatusOK, gin.H{"ok": true})
}

// CheckoutSuccess shows the success page after payment.
func CheckoutSuccess(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"message": "payment successful"})
}

// CheckoutCancel shows the cancel page after payment cancellation.
func CheckoutCancel(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"message": "payment cancelled"})
}
