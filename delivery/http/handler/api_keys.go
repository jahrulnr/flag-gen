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
)

type APIKeysHandler struct {
	db *sql.DB
}

func NewAPIKeysHandler(db *sql.DB) *APIKeysHandler {
	return &APIKeysHandler{db: db}
}

type apiKeyRequest struct {
	Name      string   `json:"name"`
	KeyType   string   `json:"keyType"`
	Scopes    []string `json:"scopes"`
	ExpiresAt *string  `json:"expiresAt"`
}

type apiKeyDTO struct {
	ID             string     `json:"id"`
	ProjectID      string     `json:"projectId"`
	EnvironmentID  *string    `json:"environmentId,omitempty"`
	Name           string     `json:"name"`
	KeyPrefix      string     `json:"keyPrefix"`
	KeyType        string     `json:"keyType"`
	Scopes         []string   `json:"scopes"`
	LastUsedAt     *time.Time `json:"lastUsedAt,omitempty"`
	ExpiresAt      *time.Time `json:"expiresAt,omitempty"`
	RevokedAt      *time.Time `json:"revokedAt,omitempty"`
	CreatedAt      time.Time  `json:"createdAt"`
	PlaintextToken string     `json:"token,omitempty"`
}

func (h *APIKeysHandler) List(c *gin.Context) {
	envID, err := h.environmentID(c)
	if err != nil {
		response.AbortError(c, http.StatusNotFound, "not_found", "environment not found", nil, responseNow())
		return
	}
	rows, err := h.db.QueryContext(c.Request.Context(), `
SELECT id, project_id, environment_id, name, key_prefix, key_type, scopes_json, last_used_at, expires_at, revoked_at, created_at
FROM api_keys
WHERE project_id = ? AND (environment_id = ? OR environment_id IS NULL)
ORDER BY created_at DESC`, c.Param("projectId"), envID)
	if err != nil {
		response.AbortError(c, http.StatusInternalServerError, "internal_error", "api key list failed", nil, responseNow())
		return
	}
	defer rows.Close()

	keys := make([]apiKeyDTO, 0)
	for rows.Next() {
		k, err := scanAPIKey(rows)
		if err != nil {
			response.AbortError(c, http.StatusInternalServerError, "internal_error", "api key scan failed", nil, responseNow())
			return
		}
		keys = append(keys, k)
	}
	c.JSON(http.StatusOK, gin.H{"apiKeys": keys, "correlationId": response.CorrelationID(c)})
}

func (h *APIKeysHandler) Create(c *gin.Context) {
	envID, err := h.environmentID(c)
	if err != nil {
		response.AbortError(c, http.StatusNotFound, "not_found", "environment not found", nil, responseNow())
		return
	}
	var req apiKeyRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.AbortError(c, http.StatusBadRequest, "bad_request", "invalid json body", nil, responseNow())
		return
	}
	k, err := h.insertKey(c, envID, req)
	if err != nil {
		response.AbortError(c, http.StatusBadRequest, "bad_request", err.Error(), nil, responseNow())
		return
	}
	c.JSON(http.StatusCreated, gin.H{"apiKey": k, "correlationId": response.CorrelationID(c)})
}

func (h *APIKeysHandler) Revoke(c *gin.Context) {
	_, err := h.db.ExecContext(c.Request.Context(), `
UPDATE api_keys SET revoked_at = COALESCE(revoked_at, ?), updated_at = ? WHERE id = ? AND project_id = ?`,
		time.Now().UTC(), time.Now().UTC(), c.Param("keyId"), c.Param("projectId"))
	if err != nil {
		response.AbortError(c, http.StatusInternalServerError, "internal_error", "api key revoke failed", nil, responseNow())
		return
	}
	c.Status(http.StatusNoContent)
}

func (h *APIKeysHandler) Rotate(c *gin.Context) {
	token := newAPIKeyToken()
	prefix := tokenPrefix(token)
	hash := sha256Hex(token)
	res, err := h.db.ExecContext(c.Request.Context(), `
UPDATE api_keys SET key_prefix = ?, key_hash = ?, updated_at = ?, last_used_at = NULL
WHERE id = ? AND project_id = ? AND revoked_at IS NULL`,
		prefix, hash, time.Now().UTC(), c.Param("keyId"), c.Param("projectId"))
	if err == nil {
		if n, _ := res.RowsAffected(); n == 0 {
			err = sql.ErrNoRows
		}
	}
	if err != nil {
		response.AbortError(c, http.StatusNotFound, "not_found", "api key not found", nil, responseNow())
		return
	}
	var k apiKeyDTO
	var scopesRaw []byte
	err = h.db.QueryRowContext(c.Request.Context(), `
SELECT id, project_id, environment_id, name, key_prefix, key_type, scopes_json, last_used_at, expires_at, revoked_at, created_at
FROM api_keys
WHERE id = ? AND project_id = ?`, c.Param("keyId"), c.Param("projectId")).Scan(
		&k.ID, &k.ProjectID, &k.EnvironmentID, &k.Name, &k.KeyPrefix, &k.KeyType, &scopesRaw, &k.LastUsedAt, &k.ExpiresAt, &k.RevokedAt, &k.CreatedAt)
	if err != nil {
		response.AbortError(c, http.StatusNotFound, "not_found", "api key not found", nil, responseNow())
		return
	}
	_ = json.Unmarshal(scopesRaw, &k.Scopes)
	k.PlaintextToken = token
	c.JSON(http.StatusOK, gin.H{"apiKey": k, "correlationId": response.CorrelationID(c)})
}

func (h *APIKeysHandler) insertKey(c *gin.Context, envID string, req apiKeyRequest) (apiKeyDTO, error) {
	if strings.TrimSpace(req.Name) == "" {
		req.Name = "API key"
	}
	if req.KeyType == "" {
		req.KeyType = "server"
	}
	if req.KeyType != "server" && req.KeyType != "broadcast" && req.KeyType != "public" {
		return apiKeyDTO{}, errString("invalid keyType")
	}
	if len(req.Scopes) == 0 {
		req.Scopes = defaultScopes(req.KeyType)
	}
	var expires any
	var expiresAt *time.Time
	if req.ExpiresAt != nil && *req.ExpiresAt != "" {
		t, err := time.Parse(time.RFC3339, *req.ExpiresAt)
		if err != nil {
			return apiKeyDTO{}, err
		}
		t = t.UTC()
		expires = t
		expiresAt = &t
	}
	token := newAPIKeyToken()
	prefix := tokenPrefix(token)
	scopesRaw, _ := json.Marshal(req.Scopes)
	id := uuid.NewString()
	now := time.Now().UTC()
	_, err := h.db.ExecContext(c.Request.Context(), `
INSERT INTO api_keys (id, project_id, environment_id, name, key_prefix, key_hash, key_type, scopes_json, expires_at, created_at, updated_at)
VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		id, c.Param("projectId"), envID, req.Name, prefix, sha256Hex(token), req.KeyType, scopesRaw, expires, now, now)
	if err != nil {
		return apiKeyDTO{}, err
	}
	return apiKeyDTO{
		ID: id, ProjectID: c.Param("projectId"), EnvironmentID: &envID, Name: req.Name,
		KeyPrefix: prefix, KeyType: req.KeyType, Scopes: req.Scopes, ExpiresAt: expiresAt,
		CreatedAt: now, PlaintextToken: token,
	}, nil
}

func (h *APIKeysHandler) environmentID(c *gin.Context) (string, error) {
	var id string
	err := h.db.QueryRowContext(c.Request.Context(), `
SELECT e.id
FROM environments e
WHERE e.project_id = ? AND e.`+"`key`"+` = ? AND e.deleted_at IS NULL
LIMIT 1`, c.Param("projectId"), c.Param("environmentKey")).Scan(&id)
	return id, err
}

type scanner interface {
	Scan(dest ...any) error
}

func scanAPIKey(s scanner) (apiKeyDTO, error) {
	var k apiKeyDTO
	var scopesRaw []byte
	err := s.Scan(&k.ID, &k.ProjectID, &k.EnvironmentID, &k.Name, &k.KeyPrefix, &k.KeyType, &scopesRaw, &k.LastUsedAt, &k.ExpiresAt, &k.RevokedAt, &k.CreatedAt)
	if err != nil {
		return k, err
	}
	_ = json.Unmarshal(scopesRaw, &k.Scopes)
	return k, nil
}

func defaultScopes(keyType string) []string {
	switch keyType {
	case "broadcast":
		return []string{"broadcast:subscribe"}
	case "public":
		return []string{"flags:read"}
	default:
		return []string{"flags:read", "flags:evaluate"}
	}
}

func newAPIKeyToken() string {
	var b [24]byte
	if _, err := rand.Read(b[:]); err != nil {
		panic(err)
	}
	return "fg_" + hex.EncodeToString(b[:])
}

func tokenPrefix(token string) string {
	if len(token) <= 16 {
		return token
	}
	return token[:16]
}

func sha256Hex(s string) string {
	sum := sha256.Sum256([]byte(s))
	return hex.EncodeToString(sum[:])
}

type errString string

func (e errString) Error() string { return string(e) }
