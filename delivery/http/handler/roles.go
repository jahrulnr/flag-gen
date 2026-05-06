package handler

import (
	"database/sql"
	"encoding/json"
	"net/http"
	"strings"
	"time"

	"flag-gen/delivery/http/response"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

type RolesHandler struct {
	db *sql.DB
}

func NewRolesHandler(db *sql.DB) *RolesHandler {
	return &RolesHandler{db: db}
}

type roleDTO struct {
	ID          string    `json:"id"`
	Key         string    `json:"key"`
	Name        string    `json:"name"`
	Description string    `json:"description"`
	Permissions []string  `json:"permissions"`
	SystemRole  bool      `json:"systemRole"`
	CreatedAt   time.Time `json:"createdAt"`
	UpdatedAt   time.Time `json:"updatedAt"`
}

type roleRequest struct {
	Key         string   `json:"key"`
	Name        string   `json:"name"`
	Description string   `json:"description"`
	Permissions []string `json:"permissions"`
}

func (h *RolesHandler) List(c *gin.Context) {
	rows, err := h.db.QueryContext(c.Request.Context(), `
SELECT id, `+"`key`"+`, name, COALESCE(description, ''), permissions_json, system_role, created_at, updated_at
FROM roles
ORDER BY system_role DESC, `+"`key`")
	if err != nil {
		response.AbortError(c, http.StatusInternalServerError, "internal_error", "role list failed", nil, responseNow())
		return
	}
	defer rows.Close()
	roles := make([]roleDTO, 0)
	for rows.Next() {
		r, err := scanRole(rows)
		if err != nil {
			response.AbortError(c, http.StatusInternalServerError, "internal_error", "role scan failed", nil, responseNow())
			return
		}
		roles = append(roles, r)
	}
	c.JSON(http.StatusOK, gin.H{"roles": roles, "correlationId": response.CorrelationID(c)})
}

func (h *RolesHandler) Create(c *gin.Context) {
	var req roleRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.AbortError(c, http.StatusBadRequest, "bad_request", "invalid json body", nil, responseNow())
		return
	}
	if err := validateRoleRequest(req); err != nil {
		response.AbortError(c, http.StatusUnprocessableEntity, "validation_error", err.Error(), nil, responseNow())
		return
	}
	now := time.Now().UTC()
	id := uuid.NewString()
	perms, _ := json.Marshal(req.Permissions)
	_, err := h.db.ExecContext(c.Request.Context(), `
INSERT INTO roles (id, `+"`key`"+`, name, description, permissions_json, system_role, created_at, updated_at)
VALUES (?, ?, ?, ?, ?, FALSE, ?, ?)`,
		id, normalizeRoleKey(req.Key), req.Name, nullableString(req.Description), perms, now, now)
	if err != nil {
		response.AbortError(c, http.StatusConflict, "conflict", "role already exists", nil, responseNow())
		return
	}
	c.JSON(http.StatusCreated, gin.H{
		"role": roleDTO{
			ID: id, Key: normalizeRoleKey(req.Key), Name: req.Name, Description: req.Description,
			Permissions: req.Permissions, SystemRole: false, CreatedAt: now, UpdatedAt: now,
		},
		"correlationId": response.CorrelationID(c),
	})
}

func (h *RolesHandler) Update(c *gin.Context) {
	var req roleRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.AbortError(c, http.StatusBadRequest, "bad_request", "invalid json body", nil, responseNow())
		return
	}
	if err := validateRoleRequest(req); err != nil {
		response.AbortError(c, http.StatusUnprocessableEntity, "validation_error", err.Error(), nil, responseNow())
		return
	}
	perms, _ := json.Marshal(req.Permissions)
	res, err := h.db.ExecContext(c.Request.Context(), `
UPDATE roles
SET name = ?, description = ?, permissions_json = ?, updated_at = ?
WHERE id = ?`,
		req.Name, nullableString(req.Description), perms, time.Now().UTC(), c.Param("roleId"))
	if err != nil {
		response.AbortError(c, http.StatusInternalServerError, "internal_error", "role update failed", nil, responseNow())
		return
	}
	if n, _ := res.RowsAffected(); n == 0 {
		response.AbortError(c, http.StatusNotFound, "not_found", "role not found", nil, responseNow())
		return
	}
	h.Get(c)
}

func (h *RolesHandler) Get(c *gin.Context) {
	var r roleDTO
	var raw []byte
	err := h.db.QueryRowContext(c.Request.Context(), `
SELECT id, `+"`key`"+`, name, COALESCE(description, ''), permissions_json, system_role, created_at, updated_at
FROM roles
WHERE id = ?`, c.Param("roleId")).Scan(&r.ID, &r.Key, &r.Name, &r.Description, &raw, &r.SystemRole, &r.CreatedAt, &r.UpdatedAt)
	if err != nil {
		response.AbortError(c, http.StatusNotFound, "not_found", "role not found", nil, responseNow())
		return
	}
	_ = json.Unmarshal(raw, &r.Permissions)
	c.JSON(http.StatusOK, gin.H{"role": r, "correlationId": response.CorrelationID(c)})
}

func (h *RolesHandler) Delete(c *gin.Context) {
	var system bool
	err := h.db.QueryRowContext(c.Request.Context(), `SELECT system_role FROM roles WHERE id = ?`, c.Param("roleId")).Scan(&system)
	if err != nil {
		response.AbortError(c, http.StatusNotFound, "not_found", "role not found", nil, responseNow())
		return
	}
	if system {
		response.AbortError(c, http.StatusConflict, "conflict", "system role cannot be deleted", nil, responseNow())
		return
	}
	res, err := h.db.ExecContext(c.Request.Context(), `DELETE FROM roles WHERE id = ?`, c.Param("roleId"))
	if err != nil {
		response.AbortError(c, http.StatusConflict, "conflict", "role is still assigned to users", nil, responseNow())
		return
	}
	if n, _ := res.RowsAffected(); n == 0 {
		response.AbortError(c, http.StatusNotFound, "not_found", "role not found", nil, responseNow())
		return
	}
	c.Status(http.StatusNoContent)
}

func scanRole(rows *sql.Rows) (roleDTO, error) {
	var r roleDTO
	var raw []byte
	err := rows.Scan(&r.ID, &r.Key, &r.Name, &r.Description, &raw, &r.SystemRole, &r.CreatedAt, &r.UpdatedAt)
	if err != nil {
		return r, err
	}
	_ = json.Unmarshal(raw, &r.Permissions)
	return r, nil
}

func validateRoleRequest(req roleRequest) error {
	if normalizeRoleKey(req.Key) == "" {
		return errString("role key is required")
	}
	if strings.TrimSpace(req.Name) == "" {
		return errString("role name is required")
	}
	if len(req.Permissions) == 0 {
		return errString("at least one permission is required")
	}
	return nil
}

func normalizeRoleKey(key string) string {
	key = strings.TrimSpace(strings.ToLower(key))
	key = strings.ReplaceAll(key, " ", "_")
	return key
}

func nullableString(v string) any {
	v = strings.TrimSpace(v)
	if v == "" {
		return nil
	}
	return v
}
