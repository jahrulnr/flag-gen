package handler

import (
	"net/http"

	"flag-gen/application/usecase"
	"flag-gen/delivery/http/response"
	"flag-gen/domain/entity"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

type SegmentsHandler struct {
	list   *usecase.ListSegmentsUsecase
	create *usecase.CreateSegmentUsecase
	update *usecase.UpdateSegmentUsecase
	del    *usecase.DeleteSegmentUsecase
}

func NewSegmentsHandler(list *usecase.ListSegmentsUsecase, create *usecase.CreateSegmentUsecase, update *usecase.UpdateSegmentUsecase, del *usecase.DeleteSegmentUsecase) *SegmentsHandler {
	return &SegmentsHandler{list: list, create: create, update: update, del: del}
}

type segmentUpsertRequest struct {
	Key   string          `json:"key"`
	Name  string          `json:"name"`
	Rules jsonRawNullable `json:"rules"`
}

// jsonRawNullable allows "rules" to be omitted or set to null; defaults to [].
type jsonRawNullable struct {
	Raw []byte
}

func (j *jsonRawNullable) UnmarshalJSON(b []byte) error {
	j.Raw = b
	return nil
}

func (h *SegmentsHandler) List(c *gin.Context) {
	projectID := c.Param("projectId")
	out, err := h.list.Execute(c.Request.Context(), projectID)
	if err != nil {
		response.AbortErr(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"segments": out, "correlationId": response.CorrelationID(c)})
}

func (h *SegmentsHandler) Create(c *gin.Context) {
	projectID := c.Param("projectId")
	var req segmentUpsertRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.AbortError(c, http.StatusBadRequest, "bad_request", "invalid json body", nil, responseNow())
		return
	}
	rules := req.Rules.Raw
	if len(rules) == 0 || string(rules) == "null" {
		rules = []byte("[]")
	}
	s := entity.Segment{
		ID:        uuid.NewString(),
		ProjectID: projectID,
		Key:       req.Key,
		Name:      req.Name,
		RulesJSON: rules,
	}
	out, err := h.create.Execute(c.Request.Context(), s)
	if err != nil {
		response.AbortErr(c, err)
		return
	}
	c.JSON(http.StatusCreated, gin.H{"segment": out, "correlationId": response.CorrelationID(c)})
}

func (h *SegmentsHandler) Update(c *gin.Context) {
	projectID := c.Param("projectId")
	segmentID := c.Param("segmentId")
	var req segmentUpsertRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.AbortError(c, http.StatusBadRequest, "bad_request", "invalid json body", nil, responseNow())
		return
	}
	rules := req.Rules.Raw
	if len(rules) == 0 || string(rules) == "null" {
		rules = []byte("[]")
	}
	s := entity.Segment{
		ID:        segmentID,
		ProjectID: projectID,
		Name:      req.Name,
		RulesJSON: rules,
	}
	out, err := h.update.Execute(c.Request.Context(), s)
	if err != nil {
		response.AbortErr(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"segment": out, "correlationId": response.CorrelationID(c)})
}

func (h *SegmentsHandler) Delete(c *gin.Context) {
	projectID := c.Param("projectId")
	segmentID := c.Param("segmentId")
	if err := h.del.Execute(c.Request.Context(), projectID, segmentID); err != nil {
		response.AbortErr(c, err)
		return
	}
	c.Status(http.StatusNoContent)
}
