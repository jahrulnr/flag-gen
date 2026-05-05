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

type ProjectsHandler struct {
	list   *usecase.ListProjectsUsecase
	create *usecase.CreateProjectUsecase
	delete *usecase.DeleteProjectUsecase
}

func NewProjectsHandler(
	list *usecase.ListProjectsUsecase,
	create *usecase.CreateProjectUsecase,
	delete *usecase.DeleteProjectUsecase,
) *ProjectsHandler {
	return &ProjectsHandler{list: list, create: create, delete: delete}
}

type createProjectRequest struct {
	Key         string  `json:"key"`
	Name        string  `json:"name"`
	Description *string `json:"description"`
}

func (h *ProjectsHandler) List(c *gin.Context) {
	q := c.Query("q")
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "50"))
	offset, _ := strconv.Atoi(c.DefaultQuery("offset", "0"))
	out, total, err := h.list.Execute(c.Request.Context(), q, limit, offset)
	if err != nil {
		response.AbortErr(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"projects":       out,
		"total":          total,
		"limit":          limit,
		"offset":         offset,
		"correlationId":  response.CorrelationID(c),
	})
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

func (h *ProjectsHandler) Delete(c *gin.Context) {
	projectID := c.Param("projectId")
	if err := h.delete.Execute(c.Request.Context(), projectID); err != nil {
		response.AbortErr(c, err)
		return
	}
	c.Status(http.StatusNoContent)
}
