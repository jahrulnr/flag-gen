package handler

import (
	"context"
	"encoding/json"
	"net/http"
	"time"

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
}

func NewBroadcastWSHandler(cfg config.Config, hub *broadcast.Hub, scoped *cache.ScopedFlagCache) *BroadcastWSHandler {
	return &BroadcastWSHandler{cfg: cfg, hub: hub, scoped: scoped}
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
	snap, _ := json.Marshal(map[string]any{
		"type":           "flags.snapshot",
		"projectId":      projectID,
		"environmentKey": envKey,
		"flags":          flags,
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
