// Package flagclient is a minimal Go HTTP client for flag evaluation and public reads.
package flagclient

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"
)

type Client struct {
	BaseURL    string
	HTTPClient *http.Client
	PublicKey  string
	RuntimeKey string
	// BroadcastDevToken is sent as query token= on WebSocket /broadcast/flags (local dev).
	BroadcastDevToken string
}

func New(baseURL string) *Client {
	return &Client{
		BaseURL: strings.TrimRight(baseURL, "/"),
		HTTPClient: &http.Client{
			Timeout: 15 * time.Second,
		},
	}
}

// ListPublicFlags calls GET /public/projects/{pid}/environments/{ek}/flags when PUBLIC_READ_API_KEY is configured server-side.
func (c *Client) ListPublicFlags(ctx context.Context, projectID, environmentKey string) (map[string]any, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet,
		fmt.Sprintf("%s/public/projects/%s/environments/%s/flags", c.BaseURL, projectID, environmentKey), nil)
	if err != nil {
		return nil, err
	}
	if c.PublicKey != "" {
		req.Header.Set("X-Public-Api-Key", c.PublicKey)
	}
	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 300 {
		return nil, fmt.Errorf("public flags: status %d", resp.StatusCode)
	}
	var out map[string]any
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		return nil, err
	}
	return out, nil
}
