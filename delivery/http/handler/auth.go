package handler

import (
	"crypto/rand"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"net/http"
	"strings"
	"time"

	"flag-gen/delivery/http/response"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"
)

const DashboardSessionCookie = "flag_gen_session"

type AuthHandler struct {
	db *sql.DB
}

func NewAuthHandler(db *sql.DB) *AuthHandler {
	return &AuthHandler{db: db}
}

type loginRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

type authUserDTO struct {
	ID          string   `json:"id"`
	Email       string   `json:"email"`
	Name        string   `json:"name"`
	RoleID      string   `json:"roleId"`
	RoleKey     string   `json:"roleKey"`
	Role        string   `json:"role"`
	Permissions []string `json:"permissions"`
}

func (h *AuthHandler) Login(c *gin.Context) {
	var req loginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.AbortError(c, http.StatusBadRequest, "bad_request", "invalid json body", nil, responseNow())
		return
	}
	email := strings.TrimSpace(strings.ToLower(req.Email))
	if email == "" || req.Password == "" {
		response.AbortError(c, http.StatusUnauthorized, "unauthorized", "invalid credentials", nil, responseNow())
		return
	}

	var user authUserDTO
	var passwordHash string
	var permissionsRaw []byte
	err := h.db.QueryRowContext(c.Request.Context(), `
SELECT u.id, u.email, u.name, u.password_hash, r.id, r.`+"`key`"+`, r.name, r.permissions_json
FROM users u
JOIN roles r ON r.id = u.role_id
WHERE u.email = ? AND u.status = 'active'
LIMIT 1`, email).Scan(&user.ID, &user.Email, &user.Name, &passwordHash, &user.RoleID, &user.RoleKey, &user.Role, &permissionsRaw)
	if err != nil || bcrypt.CompareHashAndPassword([]byte(passwordHash), []byte(req.Password)) != nil {
		response.AbortError(c, http.StatusUnauthorized, "unauthorized", "invalid credentials", nil, responseNow())
		return
	}
	_ = json.Unmarshal(permissionsRaw, &user.Permissions)

	token := randomToken()
	tokenHash := hashToken(token)
	expires := time.Now().UTC().Add(12 * time.Hour)
	_, err = h.db.ExecContext(c.Request.Context(), `
INSERT INTO user_sessions (id, user_id, session_hash, user_agent, ip_address, expires_at, created_at)
VALUES (?, ?, ?, ?, ?, ?, ?)`,
		uuid.NewString(), user.ID, tokenHash, c.GetHeader("User-Agent"), c.ClientIP(), expires, time.Now().UTC())
	if err != nil {
		response.AbortError(c, http.StatusInternalServerError, "internal_error", "session create failed", nil, responseNow())
		return
	}

	setSessionCookie(c, tokenHash, expires)
	c.JSON(http.StatusOK, gin.H{"user": user, "correlationId": response.CorrelationID(c)})
}

func (h *AuthHandler) Me(c *gin.Context) {
	user, ok := c.Get("dashboardUser")
	if !ok {
		cookie, err := c.Cookie(DashboardSessionCookie)
		if err != nil || cookie == "" {
			response.AbortError(c, http.StatusUnauthorized, "unauthorized", "not authenticated", nil, responseNow())
			return
		}
		var u authUserDTO
		var permissionsRaw []byte
		err = h.db.QueryRowContext(c.Request.Context(), `
SELECT u.id, u.email, u.name, r.id, r.`+"`key`"+`, r.name, r.permissions_json
FROM user_sessions s
JOIN users u ON u.id = s.user_id
JOIN roles r ON r.id = u.role_id
WHERE s.session_hash = ?
  AND s.revoked_at IS NULL
  AND s.expires_at > UTC_TIMESTAMP(3)
  AND u.status = 'active'
LIMIT 1`, cookie).Scan(&u.ID, &u.Email, &u.Name, &u.RoleID, &u.RoleKey, &u.Role, &permissionsRaw)
		if err != nil {
			response.AbortError(c, http.StatusUnauthorized, "unauthorized", "not authenticated", nil, responseNow())
			return
		}
		_ = json.Unmarshal(permissionsRaw, &u.Permissions)
		user = u
	}
	c.JSON(http.StatusOK, gin.H{"user": user, "correlationId": response.CorrelationID(c)})
}

func (h *AuthHandler) Logout(c *gin.Context) {
	if cookie, err := c.Cookie(DashboardSessionCookie); err == nil && cookie != "" {
		_, _ = h.db.ExecContext(c.Request.Context(), `
UPDATE user_sessions SET revoked_at = ?, expires_at = LEAST(expires_at, ?) WHERE session_hash = ? AND revoked_at IS NULL`,
			time.Now().UTC(), time.Now().UTC(), cookie)
	}
	http.SetCookie(c.Writer, &http.Cookie{
		Name:     DashboardSessionCookie,
		Value:    "",
		Path:     "/",
		MaxAge:   -1,
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
		Secure:   c.Request.TLS != nil || strings.EqualFold(c.GetHeader("X-Forwarded-Proto"), "https"),
	})
	c.JSON(http.StatusOK, gin.H{"ok": true, "correlationId": response.CorrelationID(c)})
}

func randomToken() string {
	var b [32]byte
	if _, err := rand.Read(b[:]); err != nil {
		panic(err)
	}
	return hex.EncodeToString(b[:])
}

func hashToken(token string) string {
	sum := sha256.Sum256([]byte(token))
	return hex.EncodeToString(sum[:])
}

func setSessionCookie(c *gin.Context, token string, expires time.Time) {
	http.SetCookie(c.Writer, &http.Cookie{
		Name:     DashboardSessionCookie,
		Value:    token,
		Path:     "/",
		Expires:  expires,
		MaxAge:   int(time.Until(expires).Seconds()),
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
		Secure:   c.Request.TLS != nil || strings.EqualFold(c.GetHeader("X-Forwarded-Proto"), "https"),
	})
}
