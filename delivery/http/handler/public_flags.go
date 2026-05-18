package handler

import (
	"net/http"
	"time"

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
		"timestamp":     time.Now().UTC().Format(time.RFC3339Nano),
		"correlationId": response.CorrelationID(c),
	})
}

type publicEvaluateRequest struct {
	Context struct {
		UserID     string         `json:"userId"`
		Attributes map[string]any `json:"attributes"`
	} `json:"context"`
	Flags []string `json:"flags"`
}

// EvaluatePOST supports POST /client/.../flags/evaluate with attributes (clientSide flags only).
func (h *PublicFlagsHandler) EvaluatePOST(c *gin.Context) {
	scope := valueobject.Scope{
		ProjectID:      c.Param("projectId"),
		EnvironmentKey: c.Param("environmentKey"),
	}
	var req publicEvaluateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.AbortError(c, http.StatusBadRequest, "bad_request", "invalid json body", nil, time.Now().UTC().Format(time.RFC3339Nano))
		return
	}
	if len(req.Flags) == 0 {
		response.AbortError(c, http.StatusBadRequest, "bad_request", "flags array is required", nil, time.Now().UTC().Format(time.RFC3339Nano))
		return
	}

	flags, _, err := h.list.Execute(c.Request.Context(), scope, "", 500, 0)
	if err != nil {
		response.AbortErr(c, err)
		return
	}
	clientKeys := make(map[string]struct{}, len(flags))
	for _, f := range flags {
		if f.ClientSide {
			clientKeys[f.Key] = struct{}{}
		}
	}
	allowed := make([]string, 0, len(req.Flags))
	for _, key := range req.Flags {
		if _, ok := clientKeys[key]; ok {
			allowed = append(allowed, key)
		}
	}
	if len(allowed) == 0 {
		response.AbortError(c, http.StatusNotFound, "flag_not_found", "no clientSide flags matched", nil, time.Now().UTC().Format(time.RFC3339Nano))
		return
	}

	evalCtx := valueobject.EvaluationContext{UserID: req.Context.UserID, Attributes: req.Context.Attributes}
	out, err := h.eval.Execute(c.Request.Context(), scope, evalCtx, allowed)
	if err != nil {
		response.AbortErr(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"results":       mapEvaluationResults(out),
		"timestamp":     time.Now().UTC().Format(time.RFC3339Nano),
		"correlationId": response.CorrelationID(c),
	})
}
