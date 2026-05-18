package handler

import (
	"context"
	"encoding/json"
	"net/http"
	"time"

	"flag-gen/domain/entity"
	"flag-gen/domain/repository"
	"flag-gen/domain/valueobject"
	"flag-gen/infrastructure/broadcast"
	"flag-gen/infrastructure/cache"
	"flag-gen/infrastructure/config"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
)

var wsUpgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	CheckOrigin: func(r *http.Request) bool {
		return true
	},
}

type BroadcastWSHandler struct {
	cfg    config.Config
	hub    *broadcast.Hub
	scoped *cache.ScopedFlagCache
	envs   repository.EnvironmentRepository
}

func NewBroadcastWSHandler(cfg config.Config, hub *broadcast.Hub, scoped *cache.ScopedFlagCache, envs repository.EnvironmentRepository) *BroadcastWSHandler {
	return &BroadcastWSHandler{cfg: cfg, hub: hub, scoped: scoped, envs: envs}
}

func (h *BroadcastWSHandler) Connect(c *gin.Context) {
	token := c.Query("token")
	if h.cfg.BroadcastDevToken != "" && token != h.cfg.BroadcastDevToken {
		c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}
	projectID := c.Query("projectId")
	envKey := c.Query("environmentKey")
	if projectID == "" || envKey == "" {
		c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": "projectId and environmentKey are required"})
		return
	}
	scope := valueobject.Scope{ProjectID: projectID, EnvironmentKey: envKey}
	scopeKey := scope.CacheKey()

	conn, err := wsUpgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		return
	}
	h.hub.Register(scopeKey, conn)

	ctx, cancel := context.WithTimeout(c.Request.Context(), 30*time.Second)
	flags, err := h.scoped.Get(ctx, scope)
	cancel()
	if err != nil {
		_ = conn.WriteMessage(websocket.TextMessage, []byte(`{"type":"error","message":"failed to load flags"}`))
		_ = conn.Close()
		h.hub.Unregister(scopeKey, conn)
		return
	}
	scopeVersion := 0
	if h.envs != nil {
		if env, err := h.envs.FindByProjectAndKey(ctx, projectID, envKey); err == nil && env != nil {
			scopeVersion = env.Version
		}
	}
	snap, _ := json.Marshal(map[string]any{
		"type":           "flags.snapshot",
		"projectId":      projectID,
		"environmentKey": envKey,
		"scopeVersion":   scopeVersion,
		"flags":          broadcastFlagDTOs(flags),
		"occurredAt":     time.Now().UTC().Format(time.RFC3339Nano),
		"correlationId":  c.GetHeader("X-Correlation-ID"),
	})
	_ = conn.WriteMessage(websocket.TextMessage, snap)

	go func() {
		defer func() {
			h.hub.Unregister(scopeKey, conn)
			_ = conn.Close()
		}()
		for {
			if _, _, err := conn.ReadMessage(); err != nil {
				return
			}
		}
	}()
}

func broadcastFlagDTOs(flags []entity.Flag) []map[string]any {
	out := make([]map[string]any, 0, len(flags))
	for _, f := range flags {
		item := map[string]any{
			"id":         f.ID,
			"key":        f.Key,
			"enabled":    f.Enabled,
			"clientSide": f.ClientSide,
			"value":      map[string]any{"type": string(f.Value.Type), "default": f.Value.Default},
			"rules":      broadcastRuleDTOs(f.Rules),
			"version":    f.Version,
		}
		if f.Rollout != nil {
			item["rollout"] = map[string]any{"percentage": f.Rollout.Percentage, "stickiness": f.Rollout.Stickiness}
		} else {
			item["rollout"] = nil
		}
		out = append(out, item)
	}
	return out
}

func broadcastRuleDTOs(rules []entity.FlagRule) []map[string]any {
	out := make([]map[string]any, 0, len(rules))
	for _, r := range rules {
		out = append(out, map[string]any{
			"id":        r.ID,
			"attribute": r.Attribute,
			"operator":  string(r.Operator),
			"value":     r.Value,
			"serve":     r.Serve,
			"priority":  r.Priority,
		})
	}
	return out
}
