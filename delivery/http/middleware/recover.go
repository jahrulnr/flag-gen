package middleware

import (
	"errors"
	"fmt"
	"net/http"
	"time"

	"flag-gen/delivery/http/response"

	"github.com/SALT-Indonesia/salt-pkg/logmanager"
	"github.com/gin-gonic/gin"
)

func RecoverMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		defer func() {
			if rec := recover(); rec != nil {
				errMessage := fmt.Sprintf("%v", rec)
				err := errors.New(errMessage)

				txn := logmanager.StartOtherSegment(
					logmanager.FromContext(c.Request.Context()),
					logmanager.OtherSegment{
						Name: "Recover Middleware",
					},
				)
				defer txn.End()

				txn.SetWebRequest(c.Request)

				resp := map[string]interface{}{
					"status":  false,
					"message": "Something went wrong, please try again later or you can contact the web administrator.",
				}

				txn.SetResponseValue(map[string]interface{}{
					"response":     resp,
					"errorMessage": errMessage,
				})
				txn.NoticeError(err)

				response.AbortError(
					c,
					http.StatusInternalServerError,
					"internal_error",
					resp["message"].(string),
					nil,
					time.Now().UTC().Format(time.RFC3339Nano),
				)
				return
			}
		}()

		c.Next()
	}
}
