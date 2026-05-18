package flagclient

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"

	"github.com/gorilla/websocket"
)

type Flag struct {
	Key        string `json:"key"`
	Enabled    bool   `json:"enabled"`
	ClientSide bool   `json:"clientSide"`
	Value      struct {
		Default any `json:"default"`
	} `json:"value"`
	Version int `json:"version"`
}

type ServerSDK struct {
	client         *Client
	projectID      string
	environmentKey string
	pollInterval   time.Duration

	mu    sync.RWMutex
	flags map[string]Flag
}

func NewServerSDK(client *Client, projectID, environmentKey string, pollInterval time.Duration) *ServerSDK {
	if pollInterval <= 0 {
		pollInterval = 30 * time.Second
	}
	return &ServerSDK{
		client:         client,
		projectID:      projectID,
		environmentKey: environmentKey,
		pollInterval:   pollInterval,
		flags:          make(map[string]Flag),
	}
}

func (s *ServerSDK) Refresh(ctx context.Context) error {
	flags, err := s.client.ListFlags(ctx, s.projectID, s.environmentKey)
	if err != nil {
		return err
	}
	next := make(map[string]Flag, len(flags))
	for _, f := range flags {
		next[f.Key] = f
	}
	s.mu.Lock()
	s.flags = next
	s.mu.Unlock()
	return nil
}

func (s *ServerSDK) StartPolling(ctx context.Context) {
	if s == nil {
		return
	}
	go func() {
		backoff := time.Second
		for {
			if err := s.Refresh(ctx); err != nil {
				select {
				case <-ctx.Done():
					return
				case <-time.After(backoff):
				}
				if backoff < 30*time.Second {
					backoff *= 2
				}
				continue
			}
			backoff = time.Second

			select {
			case <-ctx.Done():
				return
			case <-time.After(s.pollInterval):
			}
		}
	}()
}

func (s *ServerSDK) StartWebSocket(ctx context.Context) {
	if s == nil || s.client == nil {
		return
	}
	go func() {
		backoff := time.Second
		for {
			if err := s.runWebSocket(ctx); err != nil {
				log.Printf("flagclient: websocket: %v", err)
				select {
				case <-ctx.Done():
					return
				case <-time.After(backoff):
				}
				if backoff < 30*time.Second {
					backoff *= 2
				}
				continue
			}
			backoff = time.Second
		}
	}()
}

func (s *ServerSDK) runWebSocket(ctx context.Context) error {
	wsURL, err := broadcastURL(s.client.BaseURL, s.projectID, s.environmentKey, s.client.BroadcastDevToken)
	if err != nil {
		return err
	}
	header := http.Header{}
	if s.client.RuntimeKey != "" {
		header.Set("Authorization", "Bearer "+s.client.RuntimeKey)
	}
	conn, resp, err := websocket.DefaultDialer.DialContext(ctx, wsURL, header)
	if err != nil {
		if resp != nil {
			body, _ := io.ReadAll(io.LimitReader(resp.Body, 512))
			_ = resp.Body.Close()
			if resp.StatusCode == http.StatusUnauthorized {
				return fmt.Errorf("websocket unauthorized (check BROADCAST_DEV_TOKEN query token matches flag-gen): %s", strings.TrimSpace(string(body)))
			}
			return fmt.Errorf("websocket handshake %s: %s", resp.Status, strings.TrimSpace(string(body)))
		}
		return err
	}
	defer conn.Close()
	log.Printf("flagclient: websocket connected %s", wsURL)

	for {
		_, body, err := conn.ReadMessage()
		if err != nil {
			return err
		}
		s.applyBroadcast(body)
	}
}

func broadcastURL(baseURL, projectID, environmentKey, devToken string) (string, error) {
	u, err := url.Parse(baseURL)
	if err != nil {
		return "", err
	}
	switch u.Scheme {
	case "https":
		u.Scheme = "wss"
	default:
		u.Scheme = "ws"
	}
	u.Path = strings.TrimRight(u.Path, "/") + "/broadcast/flags"
	q := u.Query()
	q.Set("projectId", projectID)
	q.Set("environmentKey", environmentKey)
	if devToken != "" {
		q.Set("token", devToken)
	}
	u.RawQuery = q.Encode()
	return u.String(), nil
}

func (s *ServerSDK) applyBroadcast(body []byte) {
	var msg map[string]json.RawMessage
	if err := json.Unmarshal(body, &msg); err != nil {
		return
	}
	var typ string
	_ = json.Unmarshal(msg["type"], &typ)
	switch typ {
	case "flags.snapshot":
		var snap struct {
			Flags []Flag `json:"flags"`
		}
		if err := json.Unmarshal(body, &snap); err != nil {
			return
		}
		next := make(map[string]Flag, len(snap.Flags))
		for _, f := range snap.Flags {
			next[f.Key] = f
		}
		s.mu.Lock()
		s.flags = next
		s.mu.Unlock()
	case "flag.created", "flag.updated":
		f, ok := flagFromBroadcast(body)
		if !ok {
			return
		}
		s.mu.Lock()
		cur, exists := s.flags[f.Key]
		if !exists || f.Version >= cur.Version {
			s.flags[f.Key] = f
		}
		s.mu.Unlock()
	case "flag.deleted":
		var del struct {
			FlagKey string `json:"flagKey"`
			Version int    `json:"version"`
		}
		if err := json.Unmarshal(body, &del); err != nil || del.FlagKey == "" {
			return
		}
		s.mu.Lock()
		cur, exists := s.flags[del.FlagKey]
		if !exists || del.Version >= cur.Version {
			delete(s.flags, del.FlagKey)
		}
		s.mu.Unlock()
	}
}

func flagFromBroadcast(body []byte) (Flag, bool) {
	var msg struct {
		FlagKey    string `json:"flagKey"`
		Enabled    bool   `json:"enabled"`
		ClientSide bool   `json:"clientSide"`
		Value      struct {
			Default any `json:"default"`
		} `json:"value"`
		Version int `json:"version"`
	}
	if err := json.Unmarshal(body, &msg); err != nil || msg.FlagKey == "" {
		return Flag{}, false
	}
	var f Flag
	f.Key = msg.FlagKey
	f.Enabled = msg.Enabled
	f.ClientSide = msg.ClientSide
	f.Value.Default = msg.Value.Default
	f.Version = msg.Version
	return f, true
}

// Snapshot returns a copy of flags currently held in local cache (WebSocket / HTTP refresh).
func (s *ServerSDK) Snapshot() []Flag {
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := make([]Flag, 0, len(s.flags))
	for _, f := range s.flags {
		out = append(out, f)
	}
	return out
}

func (s *ServerSDK) Variation(flagKey string, fallback any) any {
	s.mu.RLock()
	f, ok := s.flags[flagKey]
	s.mu.RUnlock()
	if !ok || !f.Enabled {
		return fallback
	}
	if f.Value.Default == nil {
		return fallback
	}
	return f.Value.Default
}

func (c *Client) ListFlags(ctx context.Context, projectID, environmentKey string) ([]Flag, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet,
		fmt.Sprintf("%s/projects/%s/environments/%s/flags", c.BaseURL, projectID, environmentKey), nil)
	if err != nil {
		return nil, err
	}
	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 300 {
		return nil, fmt.Errorf("flags: status %d", resp.StatusCode)
	}
	var out struct {
		Flags []Flag `json:"flags"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		return nil, err
	}
	return out.Flags, nil
}
