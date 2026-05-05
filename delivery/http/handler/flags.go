package handler

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"io"
	"net/http"
	"strconv"
	"strings"

	"flag-gen/application/usecase"
	"flag-gen/delivery/http/response"
	"flag-gen/domain/entity"
	"flag-gen/domain/repository"
	"flag-gen/domain/valueobject"
	apperror "flag-gen/pkg/error"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

type FlagsHandler struct {
	list   *usecase.ListFlagsUsecase
	get    *usecase.GetFlagUsecase
	create *usecase.CreateFlagUsecase
	update *usecase.UpdateFlagUsecase
	del    *usecase.DeleteFlagUsecase

	idem repository.IdempotencyRepository
}

func NewFlagsHandler(
	list *usecase.ListFlagsUsecase,
	get *usecase.GetFlagUsecase,
	create *usecase.CreateFlagUsecase,
	update *usecase.UpdateFlagUsecase,
	del *usecase.DeleteFlagUsecase,
	idem repository.IdempotencyRepository,
) *FlagsHandler {
	return &FlagsHandler{
		list:   list,
		get:    get,
		create: create,
		update: update,
		del:    del,
		idem:   idem,
	}
}

type flagValueDTO struct {
	Type    string `json:"type"`
	Default any    `json:"default"`
}

type rolloutDTO struct {
	Percentage float64 `json:"percentage"`
	Stickiness string  `json:"stickiness"`
}

type flagRuleDTO struct {
	ID        string `json:"id"`
	Attribute string `json:"attribute"`
	Operator  string `json:"operator"`
	Value     any    `json:"value"`
	Serve     any    `json:"serve"`
	Priority  int    `json:"priority"`
}

type createFlagRequest struct {
	Key         string        `json:"key"`
	Name        string        `json:"name"`
	Description string        `json:"description"`
	Enabled     bool          `json:"enabled"`
	ClientSide  bool          `json:"clientSide"`
	Value       flagValueDTO  `json:"value"`
	Rules       []flagRuleDTO `json:"rules"`
	Rollout     *rolloutDTO   `json:"rollout"`
}

type updateFlagRequest struct {
	Version     int           `json:"version"`
	Key         *string       `json:"key,omitempty"`
	Name        string        `json:"name"`
	Description string        `json:"description"`
	Enabled     bool          `json:"enabled"`
	ClientSide  bool          `json:"clientSide"`
	Value       flagValueDTO  `json:"value"`
	Rules       []flagRuleDTO `json:"rules"`
	Rollout     *rolloutDTO   `json:"rollout"`
}

func (h *FlagsHandler) List(c *gin.Context) {
	scope := valueobject.Scope{ProjectID: c.Param("projectId"), EnvironmentKey: c.Param("environmentKey")}
	q := c.Query("q")
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "50"))
	offset, _ := strconv.Atoi(c.DefaultQuery("offset", "0"))
	out, total, err := h.list.Execute(c.Request.Context(), scope, q, limit, offset)
	if err != nil {
		response.AbortErr(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"flags":         out,
		"total":         total,
		"limit":         limit,
		"offset":        offset,
		"correlationId": response.CorrelationID(c),
	})
}

func (h *FlagsHandler) GetByKey(c *gin.Context) {
	scope := valueobject.Scope{ProjectID: c.Param("projectId"), EnvironmentKey: c.Param("environmentKey")}
	key := c.Param("flagKey")
	out, err := h.get.Execute(c.Request.Context(), scope, key)
	if err != nil {
		response.AbortErr(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"flag": out, "correlationId": response.CorrelationID(c)})
}

func (h *FlagsHandler) Create(c *gin.Context) {
	scope := valueobject.Scope{ProjectID: c.Param("projectId"), EnvironmentKey: c.Param("environmentKey")}

	body, raw, err := readBody(c)
	if err != nil {
		response.AbortError(c, http.StatusBadRequest, "bad_request", "invalid request body", nil, responseNow())
		return
	}
	c.Request.Body = io.NopCloser(body)

	var req createFlagRequest
	if err := json.Unmarshal(raw, &req); err != nil {
		response.AbortError(c, http.StatusBadRequest, "bad_request", "invalid json body", nil, responseNow())
		return
	}

	if err := h.withIdempotency(c, scope.ProjectID, raw, func() (int, any, error) {
		flag := mapCreateFlag(req)
		out, err := h.create.Execute(c.Request.Context(), scope, flag)
		if err != nil {
			return 0, nil, err
		}
		return http.StatusCreated, gin.H{"flag": out, "correlationId": response.CorrelationID(c)}, nil
	}); err != nil {
		response.AbortErr(c, err)
	}
}

func (h *FlagsHandler) Update(c *gin.Context) {
	scope := valueobject.Scope{ProjectID: c.Param("projectId"), EnvironmentKey: c.Param("environmentKey")}
	flagID := c.Param("flagKey")

	body, raw, err := readBody(c)
	if err != nil {
		response.AbortError(c, http.StatusBadRequest, "bad_request", "invalid request body", nil, responseNow())
		return
	}
	c.Request.Body = io.NopCloser(body)

	var req updateFlagRequest
	if err := json.Unmarshal(raw, &req); err != nil {
		response.AbortError(c, http.StatusBadRequest, "bad_request", "invalid json body", nil, responseNow())
		return
	}

	if err := h.withIdempotency(c, scope.ProjectID, raw, func() (int, any, error) {
		flag := mapUpdateFlag(flagID, req)
		out, err := h.update.Execute(c.Request.Context(), scope, flag)
		if err != nil {
			return 0, nil, err
		}
		return http.StatusOK, gin.H{"flag": out, "correlationId": response.CorrelationID(c)}, nil
	}); err != nil {
		response.AbortErr(c, err)
	}
}

func (h *FlagsHandler) Delete(c *gin.Context) {
	scope := valueobject.Scope{ProjectID: c.Param("projectId"), EnvironmentKey: c.Param("environmentKey")}
	flagID := c.Param("flagKey")
	versionStr := c.Query("version")
	if versionStr == "" {
		versionStr = c.GetHeader("If-Match")
	}
	v, _ := strconv.Atoi(versionStr)

	raw := []byte("delete:" + flagID + ":" + strconv.Itoa(v))
	if err := h.withIdempotency(c, scope.ProjectID, raw, func() (int, any, error) {
		if err := h.del.Execute(c.Request.Context(), scope, flagID, v); err != nil {
			return 0, nil, err
		}
		return http.StatusNoContent, nil, nil
	}); err != nil {
		response.AbortErr(c, err)
	}
}

func mapCreateFlag(req createFlagRequest) entity.Flag {
	f := entity.Flag{
		ID:          uuid.NewString(),
		Key:         req.Key,
		Name:        req.Name,
		Description: req.Description,
		Enabled:     req.Enabled,
		ClientSide:  req.ClientSide,
		Value: entity.FlagValue{
			Type:    entity.FlagValueType(req.Value.Type),
			Default: req.Value.Default,
		},
		Version: 1,
	}
	for _, r := range req.Rules {
		f.Rules = append(f.Rules, entity.FlagRule{
			ID:        firstNonEmpty(r.ID, uuid.NewString()),
			Attribute: r.Attribute,
			Operator:  entity.FlagRuleOperator(r.Operator),
			Value:     r.Value,
			Serve:     r.Serve,
			Priority:  r.Priority,
		})
	}
	if req.Rollout != nil {
		f.Rollout = &entity.RolloutConfig{Percentage: req.Rollout.Percentage, Stickiness: req.Rollout.Stickiness}
	}
	return f
}

func mapUpdateFlag(flagID string, req updateFlagRequest) entity.Flag {
	f := entity.Flag{
		ID:          flagID,
		Name:        req.Name,
		Description: req.Description,
		Enabled:     req.Enabled,
		ClientSide:  req.ClientSide,
		Value: entity.FlagValue{
			Type:    entity.FlagValueType(req.Value.Type),
			Default: req.Value.Default,
		},
		Version: req.Version,
	}
	if req.Key != nil {
		f.Key = *req.Key
	}
	for _, r := range req.Rules {
		f.Rules = append(f.Rules, entity.FlagRule{
			ID:        firstNonEmpty(r.ID, uuid.NewString()),
			Attribute: r.Attribute,
			Operator:  entity.FlagRuleOperator(r.Operator),
			Value:     r.Value,
			Serve:     r.Serve,
			Priority:  r.Priority,
		})
	}
	if req.Rollout != nil {
		f.Rollout = &entity.RolloutConfig{Percentage: req.Rollout.Percentage, Stickiness: req.Rollout.Stickiness}
	}
	return f
}

func firstNonEmpty(a, b string) string {
	if a != "" {
		return a
	}
	return b
}

func readBody(c *gin.Context) (io.Reader, []byte, error) {
	raw, err := io.ReadAll(c.Request.Body)
	if err != nil {
		return nil, nil, err
	}
	return strings.NewReader(string(raw)), raw, nil
}

func (h *FlagsHandler) withIdempotency(c *gin.Context, projectID string, rawRequest []byte, fn func() (int, any, error)) error {
	key := c.GetHeader("Idempotency-Key")
	if key == "" || h.idem == nil {
		code, body, err := fn()
		if err != nil {
			return err
		}
		if code == http.StatusNoContent {
			c.Status(code)
			return nil
		}
		c.JSON(code, body)
		return nil
	}

	sum := sha256.Sum256(rawRequest)
	reqHash := hex.EncodeToString(sum[:])

	rec, err := h.idem.BeginProcessing(c.Request.Context(), key, nil, &projectID, reqHash, 30_000_000_000)
	if err != nil {
		return err
	}

	// If already completed, replay response.
	if rec.Status == entity.IdempotencyCompleted && rec.ResponseCode != nil && len(rec.ResponseJSON) > 0 {
		var anyBody any
		_ = json.Unmarshal(rec.ResponseJSON, &anyBody)
		c.JSON(*rec.ResponseCode, anyBody)
		return nil
	}

	code, body, err := fn()
	if err != nil {
		_ = h.idem.MarkFailed(c.Request.Context(), rec.ID, apperror.StatusInternalServerError, []byte(`{"error":"failed"}`))
		return err
	}

	if code == http.StatusNoContent {
		_ = h.idem.MarkCompleted(c.Request.Context(), rec.ID, code, []byte(`{}`))
		c.Status(code)
		return nil
	}

	respJSON, _ := json.Marshal(body)
	_ = h.idem.MarkCompleted(c.Request.Context(), rec.ID, code, respJSON)
	c.JSON(code, body)
	return nil
}
