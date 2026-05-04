package handler

import (
	"net/http"

	"flag-gen/application/usecase"
	"flag-gen/delivery/http/response"
	"flag-gen/domain/valueobject"

	"github.com/gin-gonic/gin"
)

type EvaluateHandler struct {
	eval *usecase.EvaluateFlagsUsecase
}

func NewEvaluateHandler(eval *usecase.EvaluateFlagsUsecase) *EvaluateHandler {
	return &EvaluateHandler{eval: eval}
}

type evaluateRequest struct {
	Context struct {
		UserID     string         `json:"userId"`
		Attributes map[string]any `json:"attributes"`
	} `json:"context"`
	Flags []string `json:"flags"`
}

func (h *EvaluateHandler) Evaluate(c *gin.Context) {
	scope := valueobject.Scope{ProjectID: c.Param("projectId"), EnvironmentKey: c.Param("environmentKey")}

	var req evaluateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.AbortError(c, http.StatusBadRequest, "bad_request", "invalid json body", nil, responseNow())
		return
	}

	out, err := h.eval.Execute(
		c.Request.Context(),
		scope,
		valueobject.EvaluationContext{UserID: req.Context.UserID, Attributes: req.Context.Attributes},
		req.Flags,
	)
	if err != nil {
		response.AbortErr(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"results": out, "correlationId": response.CorrelationID(c)})
}

