package middleware

import (
	"context"
	"net/http"
	"strings"
	"time"

	"flag-gen/delivery/http/response"
	"flag-gen/infrastructure/migrate"

	"github.com/gin-gonic/gin"
)

// SystemClassifier exposes cached DB lifecycle classification for the migration gate.
type SystemClassifier interface {
	ClassifyCached(ctx context.Context) (migrate.Classification, error)
}

// MigrationGate blocks protected routes when the database is not ready (see database-lifecycle.md).
func MigrationGate(svc SystemClassifier) gin.HandlerFunc {
	return func(c *gin.Context) {
		path := c.Request.URL.Path
		method := c.Request.Method

		if path == "/health" {
			c.Next()
			return
		}
		if path == "/app" || strings.HasPrefix(path, "/app/") {
			c.Next()
			return
		}

		cl, err := svc.ClassifyCached(c.Request.Context())
		if err != nil {
			ts := time.Now().UTC().Format(time.RFC3339Nano)
			response.AbortError(c, http.StatusServiceUnavailable, "system_status_error", err.Error(), nil, ts)
			c.Abort()
			return
		}

		if cl.State == migrate.StateMigrationOptional {
			c.Header("X-Schema-Migration-State", "optional")
		}

		if gateAllow(cl, path, method) {
			c.Next()
			return
		}

		code := http.StatusLocked
		errCode := "database_locked"
		if cl.State == migrate.StateDBUnreachable {
			code = http.StatusServiceUnavailable
			errCode = "db_unreachable"
		}

		ts := time.Now().UTC().Format(time.RFC3339Nano)
		response.AbortError(c, code, errCode, cl.Message, map[string]any{
			"databaseState": string(cl.State),
		}, ts)
		c.Abort()
	}
}

func gateAllow(cl migrate.Classification, path, method string) bool {
	if strings.HasPrefix(path, "/v1/public/") || strings.HasPrefix(path, "/api/public/") || strings.HasPrefix(path, "/v1/client/") || strings.HasPrefix(path, "/api/client/") {
		switch cl.State {
		case migrate.StateDBReady, migrate.StateMigrationOptional:
			return true
		default:
			return false
		}
	}
	if strings.HasPrefix(path, "/v1/system/") || strings.HasPrefix(path, "/api/system/") {
		if (path == "/v1/system/install" || path == "/api/system/install") && method == http.MethodPost {
			return cl.State == migrate.StateDBEmpty && cl.CanInstall
		}
		if (path == "/v1/system/migrate" || path == "/api/system/migrate") && method == http.MethodPost {
			return cl.CanMigrate
		}
		// GET status (and unknown system methods) allowed whenever connected enough to classify
		return true
	}
	if strings.HasPrefix(path, "/api/security/") {
		return true
	}

	switch cl.State {
	case migrate.StateDBUnreachable:
		return false
	case migrate.StateDBEmpty:
		return false
	case migrate.StateDBInvalid, migrate.StateDBAhead:
		return false
	case migrate.StateMigrationRequired:
		if cl.RuntimeReadCompatible && (method == http.MethodGet || (method == http.MethodPost && strings.HasSuffix(path, "/flags/evaluate")) || strings.HasPrefix(path, "/v1/broadcast/") || strings.HasPrefix(path, "/api/broadcast/")) {
			return true
		}
		return false
	case migrate.StateDBReady, migrate.StateMigrationOptional:
		return true
	default:
		return false
	}
}
