package middleware

import (
	"database/sql"
	"encoding/json"
	"net/http"
	"strings"
	"time"

	"flag-gen/delivery/http/response"

	"github.com/gin-gonic/gin"
)

type DashboardUser struct {
	ID          string   `json:"id"`
	Email       string   `json:"email"`
	Name        string   `json:"name"`
	RoleID      string   `json:"roleId"`
	RoleKey     string   `json:"roleKey"`
	Role        string   `json:"role"`
	Permissions []string `json:"permissions"`
}

func DashboardSession(db *sql.DB, cookieName string) gin.HandlerFunc {
	if cookieName == "" {
		cookieName = "flag_gen_session"
	}
	return func(c *gin.Context) {
		if db == nil || isDashboardAuthExempt(c.Request.URL.Path, c.Request.Method) {
			c.Next()
			return
		}
		cookie, err := c.Cookie(cookieName)
		if err != nil || cookie == "" {
			response.AbortError(c, http.StatusUnauthorized, "unauthorized", "not authenticated", nil, time.Now().UTC().Format(time.RFC3339Nano))
			return
		}
		var user DashboardUser
		var permissionsRaw []byte
		err = db.QueryRowContext(c.Request.Context(), `
SELECT u.id, u.email, u.name, r.id, r.`+"`key`"+`, r.name, r.permissions_json
FROM user_sessions s
JOIN users u ON u.id = s.user_id
JOIN roles r ON r.id = u.role_id
WHERE s.session_hash = ?
  AND s.revoked_at IS NULL
  AND s.expires_at > UTC_TIMESTAMP(3)
  AND u.status = 'active'
LIMIT 1`, cookie).Scan(&user.ID, &user.Email, &user.Name, &user.RoleID, &user.RoleKey, &user.Role, &permissionsRaw)
		if err != nil {
			response.AbortError(c, http.StatusUnauthorized, "unauthorized", "not authenticated", nil, time.Now().UTC().Format(time.RFC3339Nano))
			return
		}
		_ = json.Unmarshal(permissionsRaw, &user.Permissions)
		c.Set("dashboardUser", user)
		c.Next()
	}
}

func DashboardPermission() gin.HandlerFunc {
	return func(c *gin.Context) {
		if isDashboardAuthExempt(c.Request.URL.Path, c.Request.Method) {
			c.Next()
			return
		}
		raw, ok := c.Get("dashboardUser")
		if !ok {
			response.AbortError(c, http.StatusUnauthorized, "unauthorized", "not authenticated", nil, time.Now().UTC().Format(time.RFC3339Nano))
			return
		}
		user, ok := raw.(DashboardUser)
		if !ok {
			response.AbortError(c, http.StatusUnauthorized, "unauthorized", "not authenticated", nil, time.Now().UTC().Format(time.RFC3339Nano))
			return
		}
		needed := dashboardPermissionFor(c.Request.URL.Path, c.Request.Method)
		if !dashboardHasPermission(user.Permissions, needed) {
			response.AbortError(c, http.StatusForbidden, "forbidden", "role permission denied", nil, time.Now().UTC().Format(time.RFC3339Nano))
			return
		}
		c.Next()
	}
}

func dashboardPermissionFor(path, method string) string {
	if strings.HasPrefix(path, "/api/users") || strings.HasPrefix(path, "/api/roles") {
		return "admin:*"
	}
	if strings.Contains(path, "/flags/evaluate") {
		return "flags:evaluate"
	}
	if method == http.MethodGet || method == http.MethodHead || method == http.MethodOptions {
		return "*:read"
	}
	return "*:write"
}

func dashboardHasPermission(perms []string, needed string) bool {
	for _, p := range perms {
		if p == "*" || p == needed {
			return true
		}
		if strings.HasSuffix(p, ":*") && strings.HasPrefix(needed, strings.TrimSuffix(p, "*")) {
			return true
		}
		if p == "*:read" && strings.HasSuffix(needed, ":read") {
			return true
		}
		if p == "*:write" && strings.HasSuffix(needed, ":write") {
			return true
		}
	}
	return false
}

func isDashboardAuthExempt(path, method string) bool {
	if strings.HasPrefix(path, "/api/auth/") || strings.HasPrefix(path, "/api/security/") {
		return true
	}
	if strings.HasPrefix(path, "/api/public/") {
		return true
	}
	if method == http.MethodGet && (path == "/api/system/status" || path == "/api/system/metrics") {
		return true
	}
	if path == "/api/system/install" {
		return true
	}
	return false
}
