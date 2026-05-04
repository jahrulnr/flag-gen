package handler

import (
	"net/http"

	"flag-gen/application/usecase"
	"flag-gen/delivery/http/response"
	"flag-gen/domain/entity"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

type ProjectsHandler struct {
	list   *usecase.ListProjectsUsecase
	create *usecase.CreateProjectUsecase
}

func NewProjectsHandler(list *usecase.ListProjectsUsecase, create *usecase.CreateProjectUsecase) *ProjectsHandler {
	return &ProjectsHandler{list: list, create: create}
}

type createProjectRequest struct {
	Key         string  `json:"key"`
	Name        string  `json:"name"`
	Description *string `json:"description"`
}

func (h *ProjectsHandler) List(c *gin.Context) {
	out, err := h.list.Execute(c.Request.Context())
	if err != nil {
		response.AbortErr(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"projects": out, "correlationId": response.CorrelationID(c)})
}

func (h *ProjectsHandler) Create(c *gin.Context) {
	var req createProjectRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.AbortError(c, http.StatusBadRequest, "bad_request", "invalid json body", nil, responseNow())
		return
	}
	p := entity.Project{
		ID:          uuid.NewString(),
		Key:         req.Key,
		Name:        req.Name,
		Description: req.Description,
	}
	out, err := h.create.Execute(c.Request.Context(), p)
	if err != nil {
		response.AbortErr(c, err)
		return
	}
	c.JSON(http.StatusCreated, gin.H{"project": out, "correlationId": response.CorrelationID(c)})
}

