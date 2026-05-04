package handler

import (
	"net/http"

	"flag-gen/application/usecase"
	"flag-gen/delivery/http/response"
	"flag-gen/domain/entity"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

type EnvironmentsHandler struct {
	list   *usecase.ListEnvironmentsUsecase
	create *usecase.CreateEnvironmentUsecase
}

func NewEnvironmentsHandler(list *usecase.ListEnvironmentsUsecase, create *usecase.CreateEnvironmentUsecase) *EnvironmentsHandler {
	return &EnvironmentsHandler{list: list, create: create}
}

type createEnvironmentRequest struct {
	Key       string `json:"key"`
	Name      string `json:"name"`
	SortOrder int    `json:"sortOrder"`
}

func (h *EnvironmentsHandler) List(c *gin.Context) {
	projectID := c.Param("projectId")
	out, err := h.list.Execute(c.Request.Context(), projectID)
	if err != nil {
		response.AbortErr(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"environments": out, "correlationId": response.CorrelationID(c)})
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
