package middleware

import (
	"net/http"

	"github.com/gin-contrib/sessions"
	"github.com/gin-gonic/gin"
)

// UserRequired redirects to /auth/login if no customer session exists.
func UserRequired() gin.HandlerFunc {
	return func(c *gin.Context) {
		sess := sessions.Default(c)
		uid := sess.Get("user_id")
		if uid == nil {
			c.Redirect(http.StatusFound, "/auth/login?next="+c.Request.URL.Path)
			c.Abort()
			return
		}
		c.Next()
	}
}
