// Sample backend service: two integration paths —
//  1. Flags  — WebSocket (flags.snapshot) + local cache
//  2. Evaluate — HTTP POST/GET with full flag-gen evaluation response
package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"os/signal"
	"sort"
	"strings"
	"syscall"
	"time"

	"flag-gen/sdk/flagclient"
)

func main() {
	baseURL := env("FLAG_GEN_BASE_URL", "http://localhost:6123/v1")
	projectID := strings.TrimSpace(os.Getenv("FLAG_GEN_PROJECT_ID"))
	envKey := env("FLAG_GEN_ENVIRONMENT_KEY", "development")
	addr := env("EXAMPLE_HTTP_ADDR", ":8080")

	client := flagclient.New(baseURL)
	client.RuntimeKey = strings.TrimSpace(os.Getenv("FLAG_GEN_API_KEY"))
	client.BroadcastDevToken = strings.TrimSpace(os.Getenv("BROADCAST_DEV_TOKEN"))
	if client.BroadcastDevToken == "" {
		log.Fatal("BROADCAST_DEV_TOKEN is required (must match flag-gen .env, e.g. dev-broadcast-token)")
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	if projectID == "" {
		var err error
		projectID, err = resolveProjectID(ctx, baseURL)
		if err != nil {
			log.Fatalf("resolve project id: %v (set FLAG_GEN_PROJECT_ID)", err)
		}
	}

	// Primary sync: flag-gen pushes via WebSocket (flags.snapshot + flag.updated/deleted).
	// Requires flag-gen OUTBOX_WORKER + BROADCAST_CONSUMER + RabbitMQ — not manual polling.
	pollInterval := 30 * time.Second
	sdk := flagclient.NewServerSDK(client, projectID, envKey, pollInterval)
	if err := sdk.Refresh(ctx); err != nil {
		log.Printf("initial HTTP refresh before WS connect: %v", err)
	}
	sdk.StartWebSocket(ctx)
	if envBool("FLAG_GEN_POLL_FALLBACK", false) {
		log.Printf("HTTP poll fallback enabled (use only when WebSocket/RabbitMQ is down)")
		sdk.StartPolling(ctx)
	}

	mux := http.NewServeMux()
	mux.HandleFunc("GET /health", func(w http.ResponseWriter, _ *http.Request) {
		writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
	})

	// --- Sample: flags (local cache from WebSocket / HTTP list) ---
	mux.HandleFunc("GET /flags", func(w http.ResponseWriter, _ *http.Request) {
		writeJSON(w, http.StatusOK, map[string]any{
			"projectId":      projectID,
			"environmentKey": envKey,
			"source":         "local_cache",
			"flags":          snapshotDTOs(sdk.Snapshot()),
		})
	})
	mux.HandleFunc("GET /flags/{key}", func(w http.ResponseWriter, r *http.Request) {
		key := r.PathValue("key")
		flag := sdk.Variation(key, false)
		writeJSON(w, http.StatusOK, map[string]any{
			"key":   key,
			"value": flag,
		})
	})

	// --- Sample: evaluate (remote flag-gen, full evaluation payload) ---
	mux.HandleFunc("POST /evaluate", func(w http.ResponseWriter, r *http.Request) {
		body, err := io.ReadAll(r.Body)
		if err != nil {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid body"})
			return
		}
		var req struct {
			Context struct {
				UserID     string         `json:"userId"`
				Attributes map[string]any `json:"attributes"`
			} `json:"context"`
			Flags []string `json:"flags"`
		}
		if err := json.Unmarshal(body, &req); err != nil || len(req.Flags) == 0 {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "expected {context, flags}"})
			return
		}
		out, err := client.EvaluateFlags(ctx, projectID, envKey, flagclient.EvalContext{
			UserID:     req.Context.UserID,
			Attributes: req.Context.Attributes,
		}, req.Flags)
		if err != nil {
			writeJSON(w, http.StatusBadGateway, map[string]string{"error": err.Error()})
			return
		}
		writeJSON(w, http.StatusOK, out)
	})
	mux.HandleFunc("GET /evaluate/{key}", func(w http.ResponseWriter, r *http.Request) {
		key := r.PathValue("key")
		attrs, err := parseAttributesQuery(r.URL.Query().Get("attributes"))
		if err != nil {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
			return
		}
		out, err := client.EvaluateFlag(ctx, projectID, envKey, flagclient.EvalContext{
			UserID:     r.URL.Query().Get("userId"),
			Attributes: attrs,
		}, key)
		if err != nil {
			writeJSON(w, http.StatusBadGateway, map[string]string{"error": err.Error()})
			return
		}
		writeJSON(w, http.StatusOK, out)
	})

	log.Printf("flag-gen backend example: project=%s env=%s token=set", projectID, envKey)
	log.Printf("  flags:     GET %s/flags  (cache via WebSocket broadcast from flag-gen)", addr)
	log.Printf("  evaluate:  POST %s/evaluate  (flag-gen API)", addr)

	srv := &http.Server{Addr: addr, Handler: mux}
	go func() {
		<-ctx.Done()
		shutdown, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		_ = srv.Shutdown(shutdown)
	}()
	if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		log.Fatal(err)
	}
}

func snapshotDTOs(flags []flagclient.Flag) []map[string]any {
	sort.Slice(flags, func(i, j int) bool { return flags[i].Key < flags[j].Key })
	out := make([]map[string]any, 0, len(flags))
	for _, f := range flags {
		out = append(out, flagDTO(f))
	}
	return out
}

func flagDTO(f flagclient.Flag) map[string]any {
	return map[string]any{
		"key":        f.Key,
		"enabled":    f.Enabled,
		"clientSide": f.ClientSide,
		"value":      map[string]any{"default": f.Value.Default},
		"version":    f.Version,
	}
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

func parseAttributesQuery(raw string) (map[string]any, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return nil, nil
	}
	var attrs map[string]any
	if err := json.Unmarshal([]byte(raw), &attrs); err != nil {
		return nil, fmt.Errorf("attributes must be a JSON object")
	}
	return attrs, nil
}

func env(key, fallback string) string {
	if v := strings.TrimSpace(os.Getenv(key)); v != "" {
		return v
	}
	return fallback
}

func envBool(key string, fallback bool) bool {
	v := strings.TrimSpace(os.Getenv(key))
	if v == "" {
		return fallback
	}
	switch strings.ToLower(v) {
	case "1", "true", "yes", "on":
		return true
	default:
		return false
	}
}

func resolveProjectID(ctx context.Context, baseURL string) (string, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, strings.TrimRight(baseURL, "/")+"/projects", nil)
	if err != nil {
		return "", err
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 300 {
		return "", fmt.Errorf("projects: status %d", resp.StatusCode)
	}
	var out struct {
		Projects []struct {
			ID  string `json:"ID"`
			Key string `json:"Key"`
		} `json:"projects"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		return "", err
	}
	want := env("FLAG_GEN_PROJECT_KEY", "development")
	for _, p := range out.Projects {
		if p.ID != "" && (want == "" || p.Key == want) {
			return p.ID, nil
		}
	}
	if len(out.Projects) > 0 && out.Projects[0].ID != "" {
		return out.Projects[0].ID, nil
	}
	return "", fmt.Errorf("no projects found")
}
