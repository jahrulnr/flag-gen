package flagclient

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestEvaluateFlags(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		require.Equal(t, http.MethodPost, r.Method)
		require.Equal(t, "/projects/p1/environments/dev/flags/evaluate", r.URL.Path)
		var body map[string]any
		require.NoError(t, json.NewDecoder(r.Body).Decode(&body))
		ctx := body["context"].(map[string]any)
		require.Equal(t, "u1", ctx["userId"])
		_, _ = w.Write([]byte(`{
			"results":{"checkout":{"value":true,"version":2,"reason":"default","matchedRuleId":null,"source":"database"}},
			"timestamp":"2026-01-01T00:00:00Z",
			"correlationId":"cid-1"
		}`))
	}))
	defer srv.Close()

	client := New(srv.URL)
	out, err := client.EvaluateFlags(context.Background(), "p1", "dev", EvalContext{
		UserID:     "u1",
		Attributes: map[string]any{"country": "ID"},
	}, []string{"checkout"})
	require.NoError(t, err)
	require.Equal(t, true, out.Results["checkout"].Value)
	require.Equal(t, "default", out.Results["checkout"].Reason)
	require.Equal(t, "cid-1", out.CorrelationID)
}
