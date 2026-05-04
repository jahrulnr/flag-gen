package middleware

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"net/http"

	"flag-gen/pkg/requestctx"

	"github.com/gin-gonic/gin"
)

const CorrelationIDHeader = "X-Correlation-ID"

func CorrelationIDMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		cid := c.GetHeader(CorrelationIDHeader)
		if cid == "" {
			cid = newCorrelationID()
		}

		c.Set(CorrelationIDHeader, cid)
		c.Writer.Header().Set(CorrelationIDHeader, cid)
		c.Request = c.Request.WithContext(context.WithValue(c.Request.Context(), requestctx.CorrelationIDKey, cid))

		c.Next()
	}
}

func newCorrelationID() string {
	var b [16]byte
	if _, err := rand.Read(b[:]); err != nil {
		// fallback to something deterministic-ish but still unique enough per process
		return hex.EncodeToString([]byte(http.StatusText(http.StatusInternalServerError)))
	}
	return hex.EncodeToString(b[:])
}

