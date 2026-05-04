package handler

import (
	"encoding/json"
	"net/http"
	"strconv"

	"flag-gen/application/usecase"
	"flag-gen/delivery/http/response"
	"flag-gen/domain/repository"
	"flag-gen/domain/valueobject"

	"github.com/gin-gonic/gin"
)

type DashboardHandler struct {
	admin    repository.AdminReadRepository
	rollback *usecase.RollbackFlagUsecase
}

func NewDashboardHandler(admin repository.AdminReadRepository, rollback *usecase.RollbackFlagUsecase) *DashboardHandler {
	return &DashboardHandler{admin: admin, rollback: rollback}
}

func (h *DashboardHandler) ListFlagVersions(c *gin.Context) {
	scope := valueobject.Scope{ProjectID: c.Param("projectId"), EnvironmentKey: c.Param("environmentKey")}
	flagID := c.Param("flagId")
	rows, err := h.admin.ListFlagVersions(c.Request.Context(), scope, flagID)
	if err != nil {
		response.AbortErr(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"versions": rows, "correlationId": response.CorrelationID(c)})
}

func (h *DashboardHandler) DiffFlagVersions(c *gin.Context) {
	scope := valueobject.Scope{ProjectID: c.Param("projectId"), EnvironmentKey: c.Param("environmentKey")}
	flagID := c.Param("flagId")
	v1, err1 := strconv.Atoi(c.Query("v1"))
	v2, err2 := strconv.Atoi(c.Query("v2"))
	if err1 != nil || err2 != nil || v1 <= 0 || v2 <= 0 {
		response.AbortError(c, http.StatusBadRequest, "bad_request", "v1 and v2 query params required", nil, responseNow())
		return
	}
	s1, err := h.admin.GetFlagVersionSnapshot(c.Request.Context(), scope, flagID, v1)
	if err != nil {
		response.AbortErr(c, err)
		return
	}
	s2, err := h.admin.GetFlagVersionSnapshot(c.Request.Context(), scope, flagID, v2)
	if err != nil {
		response.AbortErr(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"v1":            v1,
		"v2":            v2,
		"snapshotJson1": rawJSONBytes(s1),
		"snapshotJson2": rawJSONBytes(s2),
		"correlationId": response.CorrelationID(c),
	})
}

func rawJSONBytes(b []byte) json.RawMessage {
	if b == nil {
		return json.RawMessage("null")
	}
	return json.RawMessage(b)
}

type rollbackRequest struct {
	TargetVersion int `json:"targetVersion"`
	Version       int `json:"version"`
}

func (h *DashboardHandler) Rollback(c *gin.Context) {
	var req rollbackRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.AbortError(c, http.StatusBadRequest, "bad_request", "invalid json body", nil, responseNow())
		return
	}
	scope := valueobject.Scope{ProjectID: c.Param("projectId"), EnvironmentKey: c.Param("environmentKey")}
	flagID := c.Param("flagId")
	out, err := h.rollback.Execute(c.Request.Context(), scope, flagID, req.TargetVersion, req.Version)
	if err != nil {
		response.AbortErr(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"flag": out, "correlationId": response.CorrelationID(c)})
}

func (h *DashboardHandler) ListAudit(c *gin.Context) {
	projectID := c.Query("projectId")
	if projectID == "" {
		projectID = c.Param("projectId")
	}
	entityType := c.Query("entityType")
	entityID := c.Query("entityId")
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "100"))
	rows, err := h.admin.ListAuditEntries(c.Request.Context(), projectID, entityType, entityID, limit)
	if err != nil {
		response.AbortErr(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"audit": rows, "correlationId": response.CorrelationID(c)})
}
