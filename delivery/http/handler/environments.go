package handler

import (
	"net/http"
	"strconv"

	"flag-gen/application/usecase"
	"flag-gen/delivery/http/response"
	"flag-gen/domain/entity"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

type EnvironmentsHandler struct {
	list   *usecase.ListEnvironmentsUsecase
	create *usecase.CreateEnvironmentUsecase
	delete *usecase.DeleteEnvironmentUsecase
}

func NewEnvironmentsHandler(
	list *usecase.ListEnvironmentsUsecase,
	create *usecase.CreateEnvironmentUsecase,
	delete *usecase.DeleteEnvironmentUsecase,
) *EnvironmentsHandler {
	return &EnvironmentsHandler{list: list, create: create, delete: delete}
}

type createEnvironmentRequest struct {
	Key       string `json:"key"`
	Name      string `json:"name"`
	SortOrder int    `json:"sortOrder"`
}

func (h *EnvironmentsHandler) List(c *gin.Context) {
	projectID := c.Param("projectId")
	q := c.Query("q")
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "50"))
	offset, _ := strconv.Atoi(c.DefaultQuery("offset", "0"))
	out, total, err := h.list.Execute(c.Request.Context(), projectID, q, limit, offset)
	if err != nil {
		response.AbortErr(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"environments":   out,
		"total":          total,
		"limit":          limit,
		"offset":         offset,
		"correlationId":  response.CorrelationID(c),
	})
}

func (h *EnvironmentsHandler) Create(c *gin.Context) {
	projectID := c.Param("projectId")
	var req createEnvironmentRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.AbortError(c, http.StatusBadRequest, "bad_request", "invalid json body", nil, responseNow())
		return
	}

	env := entity.Environment{
		ID:        uuid.NewString(),
		ProjectID: projectID,
		Key:       req.Key,
		Name:      req.Name,
		SortOrder: req.SortOrder,
	}
	out, err := h.create.Execute(c.Request.Context(), env)
	if err != nil {
		response.AbortErr(c, err)
		return
	}
	c.JSON(http.StatusCreated, gin.H{"environment": out, "correlationId": response.CorrelationID(c)})
}

func (h *EnvironmentsHandler) Delete(c *gin.Context) {
	projectID := c.Param("projectId")
	envKey := c.Param("environmentKey")
	if err := h.delete.Execute(c.Request.Context(), projectID, envKey); err != nil {
		response.AbortErr(c, err)
		return
	}
	c.Status(http.StatusNoContent)
}
