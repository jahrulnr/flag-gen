package broker

import (
	"context"
	"testing"
	"time"

	"flag-gen/infrastructure/broadcast"

	amqp "github.com/rabbitmq/amqp091-go"
	"github.com/stretchr/testify/require"
)

func TestProcessFlagBroadcastDelivery_Dedupe(t *testing.T) {
	hub := broadcast.NewHub()
	dedupe := broadcast.NewEventDeduper(time.Hour)
	body := []byte(`{"environmentKey":"prod","projectId":"p1"}`)

	d := amqp.Delivery{
		RoutingKey: "flag.updated",
		Body:       body,
		Headers: amqp.Table{
			"x-project-id":        "pid",
			"x-event-id":        "evt-1",
			"x-aggregate-version": int32(3),
		},
	}
	ok1 := ProcessFlagBroadcastDelivery(context.Background(), d, hub, nil, dedupe)
	ok2 := ProcessFlagBroadcastDelivery(context.Background(), d, hub, nil, dedupe)
	require.True(t, ok1)
	require.False(t, ok2)
}
