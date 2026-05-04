// nolint
package outbox

import (
	"context"
	"errors"
	"testing"
	"time"

	"flag-gen/domain/entity"
	"flag-gen/domain/repository"

	amqp "github.com/rabbitmq/amqp091-go"
	"github.com/stretchr/testify/require"
)

type fakeOutboxRepo struct {
	events  []entity.OutboxEvent
	claimed bool

	published []string
	failed    []string
}

func (f *fakeOutboxRepo) ListPending(context.Context, int) ([]entity.OutboxEvent, error) {
	return nil, errors.New("unused")
}

func (f *fakeOutboxRepo) ClaimNext(ctx context.Context, workerID string, limit int, lockFor time.Duration) ([]entity.OutboxEvent, error) {
	_ = ctx
	_ = workerID
	_ = limit
	_ = lockFor
	if f.claimed {
		return nil, nil
	}
	f.claimed = true
	return f.events, nil
}

func (f *fakeOutboxRepo) MarkPublished(ctx context.Context, eventID string) error {
	_ = ctx
	f.published = append(f.published, eventID)
	return nil
}

func (f *fakeOutboxRepo) MarkFailed(ctx context.Context, eventID string, lastError string, nextAttemptAt *time.Time) error {
	_ = ctx
	_ = lastError
	_ = nextAttemptAt
	f.failed = append(f.failed, eventID)
	return nil
}

var _ repository.OutboxRepository = (*fakeOutboxRepo)(nil)

type fakePublisher struct {
	err error
}

func (f *fakePublisher) Publish(ctx context.Context, routingKey string, body []byte, headers amqp.Table, timeout time.Duration) error {
	_ = ctx
	_ = routingKey
	_ = body
	_ = headers
	_ = timeout
	return f.err
}

func TestBackoffDuration(t *testing.T) {
	require.Equal(t, 500*time.Millisecond, backoffDuration(1))
	require.Equal(t, 1*time.Second, backoffDuration(2))
	require.Equal(t, 2*time.Second, backoffDuration(3))
	require.Equal(t, 60*time.Second, backoffDuration(100))
}

func TestDispatcher_Tick_MarkPublishedOnSuccess(t *testing.T) {
	repo := &fakeOutboxRepo{events: []entity.OutboxEvent{{ID: "e1", EventType: "flag.updated", Payload: []byte(`{"x":1}`)}}}
	pub := &fakePublisher{err: nil}
	d := NewDispatcher("w1", 10, 1*time.Millisecond, 5*time.Second, "flaggen.events", 1*time.Second, repo, pub)

	require.NoError(t, d.tick(context.Background()))
	require.Equal(t, []string{"e1"}, repo.published)
	require.Empty(t, repo.failed)
}

func TestDispatcher_Tick_MarkFailedOnError(t *testing.T) {
	repo := &fakeOutboxRepo{events: []entity.OutboxEvent{{ID: "e1", EventType: "flag.updated", Payload: []byte(`{"x":1}`)}}}
	pub := &fakePublisher{err: errors.New("publish failed")}
	d := NewDispatcher("w1", 10, 1*time.Millisecond, 5*time.Second, "flaggen.events", 1*time.Second, repo, pub)

	require.NoError(t, d.tick(context.Background()))
	require.Empty(t, repo.published)
	require.Equal(t, []string{"e1"}, repo.failed)
}
