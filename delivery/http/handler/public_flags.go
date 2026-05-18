package handler

import (
	"net/http"

	"flag-gen/application/usecase"
	"flag-gen/delivery/http/response"
	"flag-gen/domain/entity"
	"flag-gen/domain/valueobject"

	"github.com/gin-gonic/gin"
)

// PublicFlagsHandler exposes read-only flag lists for browser/client integrations.
type PublicFlagsHandler struct {
	list *usecase.ListFlagsUsecase
	eval *usecase.EvaluateFlagsUsecase
}

type publicFlagDTO struct {
	Key     string `json:"key"`
	Enabled bool   `json:"enabled"`
	Value   any    `json:"value"`
	Version int    `json:"version"`
}

func NewPublicFlagsHandler(list *usecase.ListFlagsUsecase, eval *usecase.EvaluateFlagsUsecase) *PublicFlagsHandler {
	return &PublicFlagsHandler{list: list, eval: eval}
}

func clientSideFlags(flags []entity.Flag) []publicFlagDTO {
	out := make([]publicFlagDTO, 0, len(flags))
	for _, f := range flags {
		if !f.ClientSide {
			continue
		}
		out = append(out, publicFlagDTO{
			Key:     f.Key,
			Enabled: f.Enabled,
			Value:   f.Value.Default,
			Version: f.Version,
		})
	}
	return out
}

func (h *PublicFlagsHandler) List(c *gin.Context) {
	scope := valueobject.Scope{
		ProjectID:      c.Param("projectId"),
		EnvironmentKey: c.Param("environmentKey"),
	}
	flags, _, err := h.list.Execute(c.Request.Context(), scope, "", 500, 0)
	if err != nil {
		response.AbortErr(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"flags": clientSideFlags(flags), "correlationId": response.CorrelationID(c)})
}

func (h *PublicFlagsHandler) Evaluate(c *gin.Context) {
	scope := valueobject.Scope{
		ProjectID:      c.Param("projectId"),
		EnvironmentKey: c.Param("environmentKey"),
	}
	flagKey := c.Param("flagKey")
	userID := c.Query("userId")
	res, err := h.eval.ExecuteClientSideSingle(c.Request.Context(), scope, valueobject.EvaluationContext{UserID: userID}, flagKey)
	if err != nil {
		response.AbortErr(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"result":        mapEvaluationResult(res),
		"correlationId": response.CorrelationID(c),
	})
}
