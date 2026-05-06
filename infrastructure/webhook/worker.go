package webhook

import (
	"bytes"
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"log"
	"net/http"
	"time"

	"flag-gen/domain/entity"
	"flag-gen/infrastructure/config"
	"flag-gen/infrastructure/metrics"
)

type Worker struct {
	cfg    config.WebhookConfig
	client *http.Client
	queue  chan entity.OutboxEvent
}

func NewWorker(cfg config.WebhookConfig) *Worker {
	if cfg.Timeout <= 0 {
		cfg.Timeout = 5 * time.Second
	}
	return &Worker{
		cfg:    cfg,
		client: &http.Client{Timeout: cfg.Timeout},
		queue:  make(chan entity.OutboxEvent, 256),
	}
}

func (w *Worker) Enabled() bool {
	return w != nil && w.cfg.Enabled && len(w.cfg.URLs) > 0
}

func (w *Worker) Enqueue(e entity.OutboxEvent) {
	if !w.Enabled() {
		return
	}
	select {
	case w.queue <- e:
		metrics.Inc("webhook_enqueued_total", 1)
	default:
		metrics.Inc("webhook_queue_dropped_total", 1)
	}
}

func (w *Worker) Run(ctx context.Context) error {
	if !w.Enabled() {
		return nil
	}
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case e := <-w.queue:
			w.deliver(ctx, e)
		}
	}
}

func (w *Worker) deliver(ctx context.Context, e entity.OutboxEvent) {
	for _, url := range w.cfg.URLs {
		if err := w.deliverOne(ctx, url, e); err != nil {
			log.Printf("webhook delivery failed event_id=%s url=%s err=%v", e.ID, url, err)
		}
	}
}

func (w *Worker) deliverOne(ctx context.Context, url string, e entity.OutboxEvent) error {
	var last error
	for attempt := 1; attempt <= 3; attempt++ {
		metrics.Inc("webhook_delivery_attempts_total", 1)
		req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(e.Payload))
		if err != nil {
			return err
		}
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("X-Flag-Gen-Event-ID", e.ID)
		req.Header.Set("X-Flag-Gen-Event-Type", e.EventType)
		if w.cfg.Secret != "" {
			req.Header.Set("X-Flag-Gen-Signature", signature(w.cfg.Secret, e.Payload))
		}
		resp, err := w.client.Do(req)
		if err == nil && resp != nil {
			_ = resp.Body.Close()
			if resp.StatusCode >= 200 && resp.StatusCode < 300 {
				metrics.Inc("webhook_delivery_success_total", 1)
				return nil
			}
		}
		if err != nil {
			last = err
		}
		metrics.Inc("webhook_delivery_failures_total", 1)
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(time.Duration(attempt) * 500 * time.Millisecond):
		}
	}
	return last
}

func signature(secret string, body []byte) string {
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write(body)
	return "sha256=" + hex.EncodeToString(mac.Sum(nil))
}
