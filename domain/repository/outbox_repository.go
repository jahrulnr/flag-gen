package repository

import (
	"context"
	"time"

	"flag-gen/domain/entity"
)

// OutboxRepository is a domain port (interface-only). Implementations live in infrastructure.
type OutboxRepository interface {
	ListPending(ctx context.Context, limit int) ([]entity.OutboxEvent, error)
	ClaimNext(ctx context.Context, workerID string, limit int, lockFor time.Duration) ([]entity.OutboxEvent, error)
	MarkPublished(ctx context.Context, eventID string) error
	MarkFailed(ctx context.Context, eventID string, lastError string, nextAttemptAt *time.Time) error
}
