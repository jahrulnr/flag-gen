package broker

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"flag-gen/domain/valueobject"
	"flag-gen/infrastructure/broadcast"
	"flag-gen/infrastructure/cache"

	amqp "github.com/rabbitmq/amqp091-go"
)

func aggregateVersionFromHeaders(h amqp.Table) int {
	if h == nil {
		return 0
	}
	switch v := h["x-aggregate-version"].(type) {
	case int32:
		return int(v)
	case int64:
		return int(v)
	case int:
		return v
	case float64:
		return int(v)
	default:
		return 0
	}
}

// ProcessFlagBroadcastDelivery handles one AMQP delivery: dedupe, cache invalidation, hub fan-out.
func ProcessFlagBroadcastDelivery(
	ctx context.Context,
	d amqp.Delivery,
	hub *broadcast.Hub,
	scoped *cache.ScopedFlagCache,
	dedupe *broadcast.EventDeduper,
) bool {
	if hub == nil {
		return false
	}
	projectID, _ := d.Headers["x-project-id"].(string)
	_, _ = d.Headers["x-environment-id"].(string)
	eventID, _ := d.Headers["x-event-id"].(string)
	aggVer := aggregateVersionFromHeaders(d.Headers)

	if !dedupe.FirstTime(eventID, aggVer, d.Body) {
		return false
	}

	var payload map[string]any
	_ = json.Unmarshal(d.Body, &payload)
	envKey, _ := payload["environmentKey"].(string)
	if envKey == "" {
		return false
	}
	if projectID == "" {
		return false
	}

	scope := valueobject.Scope{ProjectID: projectID, EnvironmentKey: envKey}
	if scoped != nil {
		scoped.InvalidateScope(ctx, scope)
	}

	scopeKey := scope.CacheKey()
	if _, ok := payload["type"]; !ok {
		payload["type"] = d.RoutingKey
	}
	if _, ok := payload["eventId"]; !ok {
		payload["eventId"] = eventID
	}
	payload["aggregateVersion"] = aggVer
	hub.BroadcastJSON(scopeKey, eventID, payload)
	return true
}

// RunFlagBroadcastConsumer consumes flag.* events and forwards minimal payloads to the WebSocket hub.
func RunFlagBroadcastConsumer(ctx context.Context, amqpURL, exchange string, hub *broadcast.Hub, scoped *cache.ScopedFlagCache, dedupe *broadcast.EventDeduper) error {
	if amqpURL == "" || hub == nil {
		return fmt.Errorf("missing amqp url or hub")
	}
	if dedupe == nil {
		dedupe = broadcast.NewEventDeduper(15 * time.Minute)
	}
	conn, err := amqp.Dial(amqpURL)
	if err != nil {
		return err
	}
	defer conn.Close()

	ch, err := conn.Channel()
	if err != nil {
		return err
	}
	defer ch.Close()

	if err := ch.ExchangeDeclare(exchange, "topic", true, false, false, false, nil); err != nil {
		return err
	}

	q, err := ch.QueueDeclare("", false, true, true, false, nil)
	if err != nil {
		return err
	}
	for _, key := range []string{"flag.created", "flag.updated", "flag.deleted"} {
		if err := ch.QueueBind(q.Name, key, exchange, false, nil); err != nil {
			return err
		}
	}

	msgs, err := ch.Consume(q.Name, "flag-broadcast-ws", true, false, false, false, nil)
	if err != nil {
		return err
	}

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case d, ok := <-msgs:
			if !ok {
				return nil
			}
			_ = ProcessFlagBroadcastDelivery(ctx, d, hub, scoped, dedupe)
		}
	}
}
