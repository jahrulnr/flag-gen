package middleware

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

// StaticAPIKey requires a header match when expected is non-empty.
func StaticAPIKey(headerName, expected string) gin.HandlerFunc {
	if expected == "" {
		return func(c *gin.Context) { c.Next() }
	}
	return func(c *gin.Context) {
		if c.GetHeader(headerName) != expected {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "invalid_api_key"})
			return
		}
		c.Next()
	}
}
