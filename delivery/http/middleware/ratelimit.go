package middleware

import (
	"net/http"
	"sync"

	"github.com/gin-gonic/gin"
	"golang.org/x/time/rate"
)

// RateLimit applies a token-bucket limiter per client IP (best-effort).
func RateLimit(requestsPerSecond float64, burst int) gin.HandlerFunc {
	if requestsPerSecond <= 0 {
		return func(c *gin.Context) { c.Next() }
	}
	if burst < 1 {
		burst = 1
	}
	var mu sync.Mutex
	byIP := make(map[string]*rate.Limiter)
	return func(c *gin.Context) {
		ip := c.ClientIP()
		mu.Lock()
		lim, ok := byIP[ip]
		if !ok {
			lim = rate.NewLimiter(rate.Limit(requestsPerSecond), burst)
			byIP[ip] = lim
		}
		mu.Unlock()
		if !lim.Allow() {
			c.AbortWithStatusJSON(http.StatusTooManyRequests, gin.H{"error": "rate_limited"})
			return
		}
		c.Next()
	}
}
