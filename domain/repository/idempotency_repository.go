package repository

import (
	"context"
	"time"

	"flag-gen/domain/entity"
)

// IdempotencyRepository stores idempotency keys and (optional) cached responses.
// ActorID may be nil in dev-mode.
type IdempotencyRepository interface {
	// BeginProcessing creates or locks an idempotency record for a request.
	// If a completed record exists with matching request hash, it is returned.
	BeginProcessing(ctx context.Context, key string, actorID *string, projectID *string, requestHash string, lockFor time.Duration) (entity.IdempotencyRecord, error)
	MarkCompleted(ctx context.Context, recordID string, responseCode int, responseJSON []byte) error
	MarkFailed(ctx context.Context, recordID string, responseCode int, responseJSON []byte) error
}

