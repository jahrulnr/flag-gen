package handler

import (
	"net/http"

	"flag-gen/delivery/http/middleware"
	"flag-gen/delivery/http/response"

	"github.com/gin-gonic/gin"
)

type SecurityHandler struct {
	csrf *middleware.CSRFProtection
}

func NewSecurityHandler(csrf *middleware.CSRFProtection) *SecurityHandler {
	return &SecurityHandler{csrf: csrf}
}

func (h *SecurityHandler) CSRF(c *gin.Context) {
	contract := h.csrf.Issue(c)
	c.JSON(http.StatusOK, gin.H{
		"csrf":          contract,
		"correlationId": response.CorrelationID(c),
	})
}
