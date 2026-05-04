package broker

import (
	"context"
	"time"

	"flag-gen/infrastructure/broadcast"
	"flag-gen/infrastructure/cache"
)

// DialBackoff reconnects the broadcast consumer with exponential backoff.
func DialBackoff(ctx context.Context, amqpURL, exchange string, hub *broadcast.Hub, scoped *cache.ScopedFlagCache) {
	dedupe := broadcast.NewEventDeduper(15 * time.Minute)
	backoff := time.Second
	for {
		select {
		case <-ctx.Done():
			return
		default:
		}
		err := RunFlagBroadcastConsumer(ctx, amqpURL, exchange, hub, scoped, dedupe)
		if err == nil || err == context.Canceled {
			return
		}
		time.Sleep(backoff)
		if backoff < 30*time.Second {
			backoff *= 2
		}
	}
}
