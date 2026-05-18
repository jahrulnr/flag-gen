package flagclient

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
)

// EvalContext is the evaluation input (maps to flag-gen context in POST /flags/evaluate).
type EvalContext struct {
	UserID     string
	Attributes map[string]any
}

// EvaluationResult matches flag-gen evaluation result objects.
type EvaluationResult struct {
	Value         any     `json:"value"`
	Version       int     `json:"version"`
	Reason        string  `json:"reason"`
	MatchedRuleID *string `json:"matchedRuleId"`
	Source        string  `json:"source"`
}

// EvaluateResponse is the POST /flags/evaluate response body.
type EvaluateResponse struct {
	Results       map[string]EvaluationResult `json:"results"`
	Timestamp     string                      `json:"timestamp"`
	CorrelationID string                      `json:"correlationId"`
}

// EvaluateSingleResponse is the GET .../flags/:flagKey/evaluate response body.
type EvaluateSingleResponse struct {
	Result        EvaluationResult `json:"result"`
	Timestamp     string           `json:"timestamp,omitempty"`
	CorrelationID string           `json:"correlationId"`
}

// EvaluateFlags calls POST /projects/{pid}/environments/{ek}/flags/evaluate.
func (c *Client) EvaluateFlags(ctx context.Context, projectID, environmentKey string, evalCtx EvalContext, flagKeys []string) (*EvaluateResponse, error) {
	body := map[string]any{
		"context": map[string]any{
			"userId":     evalCtx.UserID,
			"attributes": evalCtx.Attributes,
		},
		"flags": flagKeys,
	}
	b, err := json.Marshal(body)
	if err != nil {
		return nil, err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost,
		fmt.Sprintf("%s/projects/%s/environments/%s/flags/evaluate", c.BaseURL, projectID, environmentKey),
		bytes.NewReader(b))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	if c.RuntimeKey != "" {
		req.Header.Set("X-API-Key", c.RuntimeKey)
	}
	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 300 {
		return nil, fmt.Errorf("evaluate: status %d", resp.StatusCode)
	}
	var out EvaluateResponse
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		return nil, err
	}
	return &out, nil
}

// EvaluateFlag is a convenience wrapper for a single flag key.
func (c *Client) EvaluateFlag(ctx context.Context, projectID, environmentKey string, evalCtx EvalContext, flagKey string) (*EvaluateResponse, error) {
	return c.EvaluateFlags(ctx, projectID, environmentKey, evalCtx, []string{flagKey})
}

// EvaluatePublicFlags calls POST /client/projects/.../flags/evaluate (clientSide flags only).
func (c *Client) EvaluatePublicFlags(ctx context.Context, projectID, environmentKey string, evalCtx EvalContext, flagKeys []string) (*EvaluateResponse, error) {
	body := map[string]any{
		"context": map[string]any{
			"userId":     evalCtx.UserID,
			"attributes": evalCtx.Attributes,
		},
		"flags": flagKeys,
	}
	b, err := json.Marshal(body)
	if err != nil {
		return nil, err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost,
		fmt.Sprintf("%s/client/projects/%s/environments/%s/flags/evaluate", c.BaseURL, projectID, environmentKey),
		bytes.NewReader(b))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	if c.PublicKey != "" {
		req.Header.Set("X-Public-Api-Key", c.PublicKey)
	}
	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 300 {
		return nil, fmt.Errorf("client evaluate: status %d", resp.StatusCode)
	}
	var out EvaluateResponse
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		return nil, err
	}
	return &out, nil
}
