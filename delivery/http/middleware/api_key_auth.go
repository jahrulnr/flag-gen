package middleware

import (
	"database/sql"
	"encoding/json"
	"net/http"
	"strings"
	"sync"
	"time"

	"flag-gen/delivery/http/response"

	"github.com/gin-gonic/gin"
	"golang.org/x/time/rate"
)

type APIKeyPrincipal struct {
	ID            string
	ProjectID     string
	EnvironmentID sql.NullString
	KeyType       string
	Scopes        []string
}

func RequireAPIKeyScope(db *sql.DB, scope string, requireWhenMissing bool) gin.HandlerFunc {
	return func(c *gin.Context) {
		token := bearerOrHeader(c)
		if token == "" {
			if requireWhenMissing {
				response.AbortError(c, http.StatusUnauthorized, "unauthorized", "missing api key", nil, time.Now().UTC().Format(time.RFC3339Nano))
				return
			}
			c.Next()
			return
		}
		principal, ok := lookupAPIKey(c, db, token)
		if !ok {
			response.AbortError(c, http.StatusUnauthorized, "unauthorized", "invalid api key", nil, time.Now().UTC().Format(time.RFC3339Nano))
			return
		}
		if !matchesRouteScope(c, db, principal) {
			response.AbortError(c, http.StatusForbidden, "forbidden", "api key scope denied", nil, time.Now().UTC().Format(time.RFC3339Nano))
			return
		}
		if !hasScope(principal.Scopes, scope) {
			response.AbortError(c, http.StatusForbidden, "forbidden", "api key scope denied", nil, time.Now().UTC().Format(time.RFC3339Nano))
			return
		}
		c.Set("apiKey", principal)
		c.Next()
	}
}

func matchesRouteScope(c *gin.Context, db *sql.DB, p APIKeyPrincipal) bool {
	if routeProject := c.Param("projectId"); routeProject != "" && routeProject != p.ProjectID {
		return false
	}
	if !p.EnvironmentID.Valid {
		return true
	}
	envKey := c.Param("environmentKey")
	if envKey == "" {
		return true
	}
	var id string
	err := db.QueryRowContext(c.Request.Context(), `
SELECT id FROM environments WHERE project_id = ? AND `+"`key`"+` = ? AND deleted_at IS NULL LIMIT 1`,
		p.ProjectID, envKey).Scan(&id)
	return err == nil && id == p.EnvironmentID.String
}

func APIKeyRateLimit(requestsPerSecond float64, burst int) gin.HandlerFunc {
	if requestsPerSecond <= 0 {
		return func(c *gin.Context) { c.Next() }
	}
	var mu sync.Mutex
	limiters := make(map[string]*rate.Limiter)
	return func(c *gin.Context) {
		key := bearerOrHeader(c)
		if key == "" {
			key = c.ClientIP()
		}
		mu.Lock()
		lim, ok := limiters[key]
		if !ok {
			lim = rate.NewLimiter(rate.Limit(requestsPerSecond), burst)
			limiters[key] = lim
		}
		mu.Unlock()
		if !lim.Allow() {
			response.AbortError(c, http.StatusTooManyRequests, "rate_limited", "rate limit exceeded", nil, time.Now().UTC().Format(time.RFC3339Nano))
			return
		}
		c.Next()
	}
}

func bearerOrHeader(c *gin.Context) string {
	if v := strings.TrimSpace(c.GetHeader("X-API-Key")); v != "" {
		return v
	}
	auth := strings.TrimSpace(c.GetHeader("Authorization"))
	if strings.HasPrefix(strings.ToLower(auth), "bearer ") {
		return strings.TrimSpace(auth[7:])
	}
	return ""
}

func lookupAPIKey(c *gin.Context, db *sql.DB, token string) (APIKeyPrincipal, bool) {
	if db == nil {
		return APIKeyPrincipal{}, false
	}
	var p APIKeyPrincipal
	var scopesRaw []byte
	err := db.QueryRowContext(c.Request.Context(), `
SELECT id, project_id, environment_id, key_type, scopes_json
FROM api_keys
WHERE key_hash = SHA2(?, 256)
  AND revoked_at IS NULL
  AND (expires_at IS NULL OR expires_at > UTC_TIMESTAMP(3))
LIMIT 1`, token).Scan(&p.ID, &p.ProjectID, &p.EnvironmentID, &p.KeyType, &scopesRaw)
	if err != nil {
		return APIKeyPrincipal{}, false
	}
	_ = json.Unmarshal(scopesRaw, &p.Scopes)
	_, _ = db.ExecContext(c.Request.Context(), `UPDATE api_keys SET last_used_at = ? WHERE id = ?`, time.Now().UTC(), p.ID)
	return p, true
}

func hasScope(scopes []string, needed string) bool {
	for _, s := range scopes {
		if s == needed || s == "*" {
			return true
		}
		if strings.HasSuffix(s, ":*") && strings.HasPrefix(needed, strings.TrimSuffix(s, "*")) {
			return true
		}
	}
	return false
}
