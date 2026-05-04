package handler

import (
	"errors"
	"net/http"
	"time"

	"flag-gen/application/usecase"
	"flag-gen/delivery/http/response"
	"flag-gen/domain/valueobject"
	"flag-gen/infrastructure/metrics"
	apperror "flag-gen/pkg/error"

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

	start := time.Now()
	out, err := h.eval.Execute(
		c.Request.Context(),
		scope,
		valueobject.EvaluationContext{UserID: req.Context.UserID, Attributes: req.Context.Attributes},
		req.Flags,
	)
	metrics.Inc("evaluate_requests_total", 1)
	metrics.AddLatencySample("evaluate", float64(time.Since(start).Milliseconds()))
	recordEvaluateOutcome(err)
	if err != nil {
		response.AbortErr(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"results": out, "correlationId": response.CorrelationID(c)})
}

// EvaluateGET supports GET .../flags/evaluate?flagKey=...&userId=... for simple dashboard/debug flows.
func (h *EvaluateHandler) EvaluateGET(c *gin.Context) {
	scope := valueobject.Scope{ProjectID: c.Param("projectId"), EnvironmentKey: c.Param("environmentKey")}
	flagKey := c.Query("flagKey")
	if flagKey == "" {
		response.AbortError(c, http.StatusBadRequest, "bad_request", "flagKey query param is required", nil, responseNow())
		return
	}
	userID := c.Query("userId")

	start := time.Now()
	out, err := h.eval.Execute(
		c.Request.Context(),
		scope,
		valueobject.EvaluationContext{UserID: userID, Attributes: nil},
		[]string{flagKey},
	)
	metrics.Inc("evaluate_requests_total", 1)
	metrics.AddLatencySample("evaluate", float64(time.Since(start).Milliseconds()))
	recordEvaluateOutcome(err)
	if err != nil {
		response.AbortErr(c, err)
		return
	}
	res := out[flagKey]
	c.JSON(http.StatusOK, gin.H{"result": res, "correlationId": response.CorrelationID(c)})
}

func recordEvaluateOutcome(err error) {
	if err == nil {
		return
	}
	var ae apperror.IAppError
	if !errors.As(err, &ae) || ae.StatusCode() != apperror.StatusServiceUnavailable {
		return
	}
	switch ae.Error() {
	case "circuit_open":
		metrics.Inc("evaluate_circuit_open_total", 1)
	case "service_degraded":
		metrics.Inc("evaluate_service_degraded_total", 1)
	default:
		metrics.Inc("evaluate_service_unavailable_total", 1)
	}
}
