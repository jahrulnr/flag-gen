package flagclient

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"sync"
	"time"
)

type Flag struct {
	Key        string `json:"Key"`
	Enabled    bool   `json:"Enabled"`
	ClientSide bool   `json:"ClientSide"`
	Value      struct {
		Default any `json:"Default"`
	} `json:"Value"`
	Version int `json:"Version"`
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
