package response

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

type ErrorBody struct {
	Error         string      `json:"error"`
	Message       string      `json:"message"`
	Details       interface{} `json:"details,omitempty"`
	CorrelationID string      `json:"correlationId"`
	Timestamp     string      `json:"timestamp"`
}

func CorrelationID(c *gin.Context) string {
	// Keep this package dependency-free from middleware to avoid import cycles.
	const header = "X-Correlation-ID"
	if v, ok := c.Get(header); ok {
		if s, ok := v.(string); ok && s != "" {
			return s
		}
	}
	return c.GetHeader(header)
}

func AbortJSON(c *gin.Context, code int, body interface{}) {
	c.Header("Content-Type", "application/json")
	c.AbortWithStatusJSON(code, body)
}

func AbortError(c *gin.Context, code int, errCode string, message string, details interface{}, timestamp string) {
	AbortJSON(c, code, ErrorBody{
		Error:         errCode,
		Message:       message,
		Details:       details,
		CorrelationID: CorrelationID(c),
		Timestamp:     timestamp,
	})
}

func AbortInternal(c *gin.Context, message string, timestamp string) {
	AbortError(c, http.StatusInternalServerError, "internal_error", message, nil, timestamp)
}
