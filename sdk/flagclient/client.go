// Package flagclient is a minimal Go HTTP client for flag evaluation and public reads.
package flagclient

import (
	"bytes"
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
}

func New(baseURL string) *Client {
	return &Client{
		BaseURL: strings.TrimRight(baseURL, "/"),
		HTTPClient: &http.Client{
			Timeout: 15 * time.Second,
		},
	}
}

// Evaluate calls POST /projects/{pid}/environments/{ek}/flags/evaluate with a single flag key.
func (c *Client) Evaluate(ctx context.Context, projectID, environmentKey, userID, flagKey string) (map[string]any, error) {
	body := map[string]any{
		"context": map[string]any{"userId": userID},
		"flags":   []string{flagKey},
	}
	b, _ := json.Marshal(body)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost,
		fmt.Sprintf("%s/projects/%s/environments/%s/flags/evaluate", c.BaseURL, projectID, environmentKey),
		bytes.NewReader(b))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 300 {
		return nil, fmt.Errorf("evaluate: status %d", resp.StatusCode)
	}
	var out map[string]any
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		return nil, err
	}
	return out, nil
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
