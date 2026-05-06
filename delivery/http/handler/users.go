package handler

import (
	"database/sql"
	"net/http"
	"strings"
	"time"

	"flag-gen/delivery/http/response"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"
)

type UsersHandler struct {
	db *sql.DB
}

func NewUsersHandler(db *sql.DB) *UsersHandler {
	return &UsersHandler{db: db}
}

type userDTO struct {
	ID        string    `json:"id"`
	Email     string    `json:"email"`
	Name      string    `json:"name"`
	RoleID    string    `json:"roleId"`
	RoleKey   string    `json:"roleKey"`
	Role      string    `json:"role"`
	Status    string    `json:"status"`
	CreatedAt time.Time `json:"createdAt"`
}

type createUserRequest struct {
	Email    string `json:"email"`
	Name     string `json:"name"`
	Password string `json:"password"`
	RoleKey  string `json:"roleKey"`
	RoleID   string `json:"roleId"`
	Status   string `json:"status"`
}

type updateUserRequest struct {
	Name    string `json:"name"`
	RoleKey string `json:"roleKey"`
	RoleID  string `json:"roleId"`
	Status  string `json:"status"`
}

func (h *UsersHandler) List(c *gin.Context) {
	rows, err := h.db.QueryContext(c.Request.Context(), `
SELECT u.id, u.email, u.name, u.role_id, r.`+"`key`"+`, r.name, u.status, u.created_at
FROM users u
JOIN roles r ON r.id = u.role_id
ORDER BY u.created_at DESC`)
	if err != nil {
		response.AbortError(c, http.StatusInternalServerError, "internal_error", "user list failed", nil, responseNow())
		return
	}
	defer rows.Close()
	users := make([]userDTO, 0)
	for rows.Next() {
		var u userDTO
		if err := rows.Scan(&u.ID, &u.Email, &u.Name, &u.RoleID, &u.RoleKey, &u.Role, &u.Status, &u.CreatedAt); err != nil {
			response.AbortError(c, http.StatusInternalServerError, "internal_error", "user scan failed", nil, responseNow())
			return
		}
		users = append(users, u)
	}
	c.JSON(http.StatusOK, gin.H{"users": users, "correlationId": response.CorrelationID(c)})
}

func (h *UsersHandler) Create(c *gin.Context) {
	var req createUserRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.AbortError(c, http.StatusBadRequest, "bad_request", "invalid json body", nil, responseNow())
		return
	}
	req.Email = strings.TrimSpace(strings.ToLower(req.Email))
	if req.Email == "" || req.Name == "" || len(req.Password) < 8 {
		response.AbortError(c, http.StatusUnprocessableEntity, "validation_error", "email, name, and password length >= 8 are required", nil, responseNow())
		return
	}
	roleID, roleKey, roleName, err := h.resolveRole(c, req.RoleID, req.RoleKey, "readonly")
	if err != nil {
		response.AbortError(c, http.StatusUnprocessableEntity, "validation_error", "role not found", nil, responseNow())
		return
	}
	hash, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	if err != nil {
		response.AbortError(c, http.StatusInternalServerError, "internal_error", "password hash failed", nil, responseNow())
		return
	}
	now := time.Now().UTC()
	id := uuid.NewString()
	status := normalizeUserStatus(req.Status)
	_, err = h.db.ExecContext(c.Request.Context(), `
INSERT INTO users (id, email, name, password_hash, two_factor_enabled, status, role_id, created_at, updated_at)
VALUES (?, ?, ?, ?, FALSE, ?, ?, ?, ?)`, id, req.Email, req.Name, string(hash), status, roleID, now, now)
	if err != nil {
		response.AbortError(c, http.StatusConflict, "conflict", "user already exists", nil, responseNow())
		return
	}
	c.JSON(http.StatusCreated, gin.H{"user": userDTO{ID: id, Email: req.Email, Name: req.Name, RoleID: roleID, RoleKey: roleKey, Role: roleName, Status: status, CreatedAt: now}, "correlationId": response.CorrelationID(c)})
}

func (h *UsersHandler) Update(c *gin.Context) {
	var req updateUserRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.AbortError(c, http.StatusBadRequest, "bad_request", "invalid json body", nil, responseNow())
		return
	}
	roleID, _, _, err := h.resolveRole(c, req.RoleID, req.RoleKey, "")
	if err != nil {
		response.AbortError(c, http.StatusUnprocessableEntity, "validation_error", "role not found", nil, responseNow())
		return
	}
	status := normalizeUserStatus(req.Status)
	if status == "" {
		status = "active"
	}
	res, err := h.db.ExecContext(c.Request.Context(), `
UPDATE users SET name = COALESCE(NULLIF(?, ''), name), role_id = ?, status = ?, updated_at = ?
WHERE id = ?`, strings.TrimSpace(req.Name), roleID, status, time.Now().UTC(), c.Param("userId"))
	if err != nil {
		response.AbortError(c, http.StatusInternalServerError, "internal_error", "user update failed", nil, responseNow())
		return
	}
	if n, _ := res.RowsAffected(); n == 0 {
		response.AbortError(c, http.StatusNotFound, "not_found", "user not found", nil, responseNow())
		return
	}
	h.Get(c)
}

func (h *UsersHandler) Get(c *gin.Context) {
	var u userDTO
	err := h.db.QueryRowContext(c.Request.Context(), `
SELECT u.id, u.email, u.name, u.role_id, r.`+"`key`"+`, r.name, u.status, u.created_at
FROM users u
JOIN roles r ON r.id = u.role_id
WHERE u.id = ?`, c.Param("userId")).Scan(&u.ID, &u.Email, &u.Name, &u.RoleID, &u.RoleKey, &u.Role, &u.Status, &u.CreatedAt)
	if err != nil {
		response.AbortError(c, http.StatusNotFound, "not_found", "user not found", nil, responseNow())
		return
	}
	c.JSON(http.StatusOK, gin.H{"user": u, "correlationId": response.CorrelationID(c)})
}

func (h *UsersHandler) Disable(c *gin.Context) {
	_, err := h.db.ExecContext(c.Request.Context(), `
UPDATE users SET status = 'disabled', updated_at = ? WHERE id = ?`, time.Now().UTC(), c.Param("userId"))
	if err != nil {
		response.AbortError(c, http.StatusInternalServerError, "internal_error", "disable user failed", nil, responseNow())
		return
	}
	c.Status(http.StatusNoContent)
}

func (h *UsersHandler) resolveRole(c *gin.Context, roleID, roleKey, fallbackKey string) (string, string, string, error) {
	roleID = strings.TrimSpace(roleID)
	roleKey = strings.TrimSpace(strings.ToLower(roleKey))
	if roleID == "" && roleKey == "" {
		roleKey = fallbackKey
	}
	var id, key, name string
	var err error
	if roleID != "" {
		err = h.db.QueryRowContext(c.Request.Context(), `SELECT id, `+"`key`"+`, name FROM roles WHERE id = ?`, roleID).Scan(&id, &key, &name)
	} else {
		err = h.db.QueryRowContext(c.Request.Context(), `SELECT id, `+"`key`"+`, name FROM roles WHERE `+"`key`"+` = ?`, roleKey).Scan(&id, &key, &name)
	}
	return id, key, name, err
}

func normalizeUserStatus(status string) string {
	switch strings.ToLower(strings.TrimSpace(status)) {
	case "", "active":
		return "active"
	case "disabled", "inactive":
		return "disabled"
	default:
		return "active"
	}
}
