package broadcast

import (
	"encoding/json"
	"sync"

	"github.com/gorilla/websocket"
)

// Hub fans out JSON messages to WebSocket clients subscribed by scope key "projectId|environmentKey".
type Hub struct {
	mu   sync.RWMutex
	subs map[string]map[*websocket.Conn]struct{}
}

func NewHub() *Hub {
	return &Hub{
		subs: make(map[string]map[*websocket.Conn]struct{}),
	}
}

func (h *Hub) Register(scopeKey string, c *websocket.Conn) {
	h.mu.Lock()
	defer h.mu.Unlock()
	if h.subs[scopeKey] == nil {
		h.subs[scopeKey] = make(map[*websocket.Conn]struct{})
	}
	h.subs[scopeKey][c] = struct{}{}
}

func (h *Hub) Unregister(scopeKey string, c *websocket.Conn) {
	h.mu.Lock()
	defer h.mu.Unlock()
	if m, ok := h.subs[scopeKey]; ok {
		delete(m, c)
		if len(m) == 0 {
			delete(h.subs, scopeKey)
		}
	}
}

// BroadcastJSON sends a payload to all subscribers of scopeKey.
// eventID is reserved for future dedupe metadata on the wire.
func (h *Hub) BroadcastJSON(scopeKey string, eventID string, v any) {
	b, err := json.Marshal(v)
	if err != nil {
		return
	}
	h.mu.RLock()
	m := h.subs[scopeKey]
	conns := make([]*websocket.Conn, 0, len(m))
	for c := range m {
		conns = append(conns, c)
	}
	h.mu.RUnlock()

	for _, c := range conns {
		_ = c.WriteMessage(websocket.TextMessage, b)
	}
	_ = eventID // forwarded in payload by publisher; clients may dedupe.
}
