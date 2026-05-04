package handler

import (
	"net/http"

	"flag-gen/application/usecase"
	"flag-gen/delivery/http/response"
	"flag-gen/domain/valueobject"

	"github.com/gin-gonic/gin"
)

// PublicFlagsHandler exposes read-only flag lists for browser/client integrations.
type PublicFlagsHandler struct {
	list *usecase.ListFlagsUsecase
}

func NewPublicFlagsHandler(list *usecase.ListFlagsUsecase) *PublicFlagsHandler {
	return &PublicFlagsHandler{list: list}
}

func (h *PublicFlagsHandler) List(c *gin.Context) {
	scope := valueobject.Scope{
		ProjectID:      c.Param("projectId"),
		EnvironmentKey: c.Param("environmentKey"),
	}
	flags, err := h.list.Execute(c.Request.Context(), scope)
	if err != nil {
		response.AbortErr(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"flags": flags, "correlationId": response.CorrelationID(c)})
}
