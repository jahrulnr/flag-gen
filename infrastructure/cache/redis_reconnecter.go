package cache

import (
	"context"
	"log"
	"sync"
	"time"

	"flag-gen/infrastructure/metrics"

	"github.com/redis/go-redis/v9"
)

// RedisReconnecter coalesces Redis connection failures into one background health probe.
// go-redis owns socket reconnects internally; this type keeps API calls non-blocking while
// emitting explicit reconnect metrics and logs for operators.
type RedisReconnecter struct {
	client  redis.UniversalClient
	backoff []time.Duration

	mu      sync.Mutex
	running bool
}

func NewRedisReconnecter(client redis.UniversalClient) *RedisReconnecter {
	return &RedisReconnecter{
		client: client,
		backoff: []time.Duration{
			100 * time.Millisecond,
			250 * time.Millisecond,
			500 * time.Millisecond,
			time.Second,
			2 * time.Second,
			5 * time.Second,
		},
	}
}

func (r *RedisReconnecter) Notify(ctx context.Context, cause error) {
	if r == nil || r.client == nil || cause == nil || !isRetryableRedis(cause) {
		return
	}

	r.mu.Lock()
	if r.running {
		r.mu.Unlock()
		metrics.Inc("redis_reconnect_events_coalesced_total", 1)
		return
	}
	r.running = true
	r.mu.Unlock()

	metrics.Inc("redis_reconnect_events_total", 1)
	_ = ctx
	go r.run(context.Background(), cause)
}

func (r *RedisReconnecter) run(ctx context.Context, cause error) {
	defer func() {
		r.mu.Lock()
		r.running = false
		r.mu.Unlock()
	}()

	log.Printf("redis reconnect probe started err=%v", cause)
	for i, d := range r.backoff {
		select {
		case <-ctx.Done():
			metrics.Inc("redis_reconnect_cancelled_total", 1)
			return
		case <-time.After(d):
		}

		metrics.Inc("redis_reconnect_attempts_total", 1)
		if err := r.client.Ping(ctx).Err(); err == nil {
			metrics.Inc("redis_reconnect_success_total", 1)
			log.Printf("redis reconnect probe succeeded attempt=%d", i+1)
			return
		} else if i == len(r.backoff)-1 {
			metrics.Inc("redis_reconnect_failures_total", 1)
			log.Printf("redis reconnect probe exhausted err=%v", err)
		}
	}
}
