package handler

import (
	"net/http"

	"flag-gen/application/usecase"
	"flag-gen/delivery/http/response"
	"flag-gen/infrastructure/metrics"

	"github.com/gin-gonic/gin"
)

type SystemHandler struct {
	svc *usecase.SystemService
}

func NewSystemHandler(svc *usecase.SystemService) *SystemHandler {
	return &SystemHandler{svc: svc}
}

func (h *SystemHandler) Status(c *gin.Context) {
	out, err := h.svc.Status(c.Request.Context(), response.CorrelationID(c))
	if err != nil {
		response.AbortErr(c, err)
		return
	}
	out.CorrelationID = response.CorrelationID(c)
	c.JSON(http.StatusOK, out)
}

func (h *SystemHandler) Install(c *gin.Context) {
	var req usecase.InstallRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.AbortError(c, http.StatusBadRequest, "bad_request", "invalid json body", nil, responseNow())
		return
	}
	out, err := h.svc.Install(c.Request.Context(), req)
	if err != nil {
		response.AbortErr(c, err)
		return
	}
	out.CorrelationID = response.CorrelationID(c)
	c.JSON(http.StatusCreated, out)
}

func (h *SystemHandler) Metrics(c *gin.Context) {
	c.Data(http.StatusOK, "text/plain; charset=utf-8", []byte(metrics.ExportText()))
}

func (h *SystemHandler) Migrate(c *gin.Context) {
	out, err := h.svc.Migrate(c.Request.Context())
	if err != nil {
		response.AbortErr(c, err)
		return
	}
	out.CorrelationID = response.CorrelationID(c)
	c.JSON(http.StatusOK, out)
}
