package middleware

import (
	"net/http"
	"sync"
	"time"

	"github.com/gin-contrib/sessions"
	"github.com/gin-gonic/gin"
)

func AdminRequired() gin.HandlerFunc {
	return func(c *gin.Context) {
		session := sessions.Default(c)
		user := session.Get("admin_id")
		if user == nil {
			c.Redirect(http.StatusFound, "/zed-admin/login")
			c.Abort()
			return
		}
		c.Next()
	}
}

// Simple in-memory rate limiter for login
type rateLimiter struct {
	mu       sync.Mutex
	attempts map[string][]time.Time
}

var loginLimiter = &rateLimiter{attempts: make(map[string][]time.Time)}

func RateLimit(maxAttempts int, window time.Duration) gin.HandlerFunc {
	return func(c *gin.Context) {
		ip := c.ClientIP()
		loginLimiter.mu.Lock()
		now := time.Now()
		var valid []time.Time
		for _, t := range loginLimiter.attempts[ip] {
			if now.Sub(t) < window {
				valid = append(valid, t)
			}
		}
		loginLimiter.attempts[ip] = valid
		if len(valid) >= maxAttempts {
			loginLimiter.mu.Unlock()
			c.JSON(http.StatusTooManyRequests, gin.H{"error": "تعداد تلاش‌های شما بیش از حد مجاز است. لطفاً کمی صبر کنید."})
			c.Abort()
			return
		}
		loginLimiter.attempts[ip] = append(loginLimiter.attempts[ip], now)
		loginLimiter.mu.Unlock()
		c.Next()
	}
}
