package outbox

import (
	"context"
	"encoding/json"
	"fmt"
	"math"
	"time"

	"flag-gen/domain/entity"
	"flag-gen/domain/repository"

	amqp "github.com/rabbitmq/amqp091-go"
)

type Publisher interface {
	Publish(ctx context.Context, routingKey string, body []byte, headers amqp.Table, timeout time.Duration) error
}

type Dispatcher struct {
	workerID string
	batch    int
	interval time.Duration
	lockFor  time.Duration

	outboxRepo repository.OutboxRepository
	publisher  Publisher

	exchange string
	timeout  time.Duration
}

func NewDispatcher(workerID string, batch int, interval time.Duration, lockFor time.Duration, exchange string, timeout time.Duration, outboxRepo repository.OutboxRepository, publisher Publisher) *Dispatcher {
	if workerID == "" {
		workerID = "outbox-worker"
	}
	if batch <= 0 {
		batch = 100
	}
	if interval <= 0 {
		interval = 500 * time.Millisecond
	}
	if lockFor <= 0 {
		lockFor = 30 * time.Second
	}
	if exchange == "" {
		exchange = "flaggen.events"
	}
	if timeout <= 0 {
		timeout = 5 * time.Second
	}
	return &Dispatcher{
		workerID:  workerID,
		batch:     batch,
		interval:  interval,
		lockFor:   lockFor,
		outboxRepo: outboxRepo,
		publisher:  publisher,
		exchange:   exchange,
		timeout:    timeout,
	}
}

func (d *Dispatcher) Run(ctx context.Context) error {
	t := time.NewTicker(d.interval)
	defer t.Stop()

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-t.C:
			_ = d.tick(ctx)
		}
	}
}

func (d *Dispatcher) tick(ctx context.Context) error {
	events, err := d.outboxRepo.ClaimNext(ctx, d.workerID, d.batch, d.lockFor)
	if err != nil {
		return err
	}
	for _, e := range events {
		if err := d.publishOne(ctx, e); err != nil {
			next := time.Now().UTC().Add(backoffDuration(e.RetryCount + 1))
			_ = d.outboxRepo.MarkFailed(ctx, e.ID, err.Error(), &next)
			continue
		}
		_ = d.outboxRepo.MarkPublished(ctx, e.ID)
	}
	return nil
}

func (d *Dispatcher) publishOne(ctx context.Context, e entity.OutboxEvent) error {
	if d.publisher == nil {
		return fmt.Errorf("missing amqp publisher")
	}

	headers := amqp.Table{
		"x-event-id":          e.ID,
		"x-aggregate-type":    e.AggregateType,
		"x-aggregate-id":      e.AggregateID,
		"x-aggregate-version": e.AggregateVersion,
		"x-project-id":        e.ProjectID,
	}
	if e.EnvironmentID != nil {
		headers["x-environment-id"] = *e.EnvironmentID
	}

	// If payload contains correlationId, forward it as header too.
	var payload map[string]any
	if err := json.Unmarshal(e.Payload, &payload); err == nil {
		if v, ok := payload["correlationId"].(string); ok && v != "" {
			headers["x-correlation-id"] = v
		}
	}

	return d.publisher.Publish(ctx, e.EventType, e.Payload, headers, d.timeout)
}

func backoffDuration(attempt int) time.Duration {
	// Exponential backoff with cap. attempt starts at 1.
	if attempt < 1 {
		attempt = 1
	}
	min := 500 * time.Millisecond
	max := 60 * time.Second
	// min * 2^(attempt-1) with overflow protection.
	shift := attempt - 1
	if shift > 30 {
		return max
	}
	mul := math.Pow(2, float64(shift))
	dFloat := float64(min) * mul
	if dFloat > float64(max) {
		return max
	}
	d := time.Duration(dFloat)
	if d > max {
		return max
	}
	return d
}

