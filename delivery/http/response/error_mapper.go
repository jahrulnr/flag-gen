package response

import (
	"errors"
	"net/http"
	"time"

	apperror "flag-gen/pkg/error"

	"github.com/gin-gonic/gin"
)

func AbortErr(c *gin.Context, err error) {
	if err == nil {
		AbortInternal(c, "unknown error", time.Now().UTC().Format(time.RFC3339Nano))
		return
	}

	ts := time.Now().UTC().Format(time.RFC3339Nano)

	var appErr apperror.IAppError
	if errors.As(err, &appErr) {
		AbortError(c, appErr.StatusCode(), classifyAppError(appErr), appErr.Error(), mapDetails(appErr), ts)
		return
	}

	AbortError(c, http.StatusInternalServerError, "internal_error", "internal server error", nil, ts)
}

func classifyAppError(err apperror.IAppError) string {
	// Prefer a stable code; message may include "[CODE] ...".
	// For MVP we expose generic buckets to clients.
	switch err.StatusCode() {
	case apperror.StatusNotFound:
		return "not_found"
	case apperror.StatusConflict:
		return "conflict"
	case apperror.StatusUnprocessableEntity:
		return "validation_error"
	case apperror.StatusTooManyRequests:
		return "rate_limited"
	case apperror.StatusUnauthorized:
		return "unauthorized"
	case apperror.StatusForbidden:
		return "forbidden"
	case apperror.StatusBadGateway:
		return "dependency_error"
	case apperror.StatusServiceUnavailable:
		return "degraded"
	default:
		if err.StatusCode() >= 400 && err.StatusCode() <= 499 {
			return "bad_request"
		}
		return "internal_error"
	}
}

func mapDetails(err apperror.IAppError) interface{} {
	// Keep details minimal to avoid leaking internals.
	return map[string]interface{}{
		"calledAt": err.CalledAt(),
	}
}

