package mysql

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"time"

	"flag-gen/domain/entity"
	"flag-gen/domain/repository"
	apperror "flag-gen/pkg/error"
)

type OutboxRepository struct {
	db *sql.DB
}

func NewOutboxRepository(db *DB) *OutboxRepository {
	return &OutboxRepository{db: db.SQL()}
}

var _ repository.OutboxRepository = (*OutboxRepository)(nil)

func (r *OutboxRepository) ListPending(ctx context.Context, limit int) ([]entity.OutboxEvent, error) {
	if limit <= 0 {
		limit = 100
	}
	const q = `
SELECT id, aggregate_type, aggregate_id, aggregate_version, scope_version,
       project_id, environment_id, event_type, payload_json,
       status, retry_count, next_attempt_at, created_at, published_at
FROM outbox_events
WHERE status IN ('pending','failed')
  AND (next_attempt_at IS NULL OR next_attempt_at <= ?)
ORDER BY created_at ASC
LIMIT ?`

	rows, err := r.db.QueryContext(ctx, q, time.Now().UTC(), limit)
	if err != nil {
		return nil, apperror.NewInfrastructureError("DB", "outbox list pending failed", err)
	}
	defer rows.Close()

	var out []entity.OutboxEvent
	for rows.Next() {
		var e entity.OutboxEvent
		var envID sql.NullString
		var payload json.RawMessage
		var status string
		var next sql.NullTime
		var published sql.NullTime
		if err := rows.Scan(
			&e.ID,
			&e.AggregateType,
			&e.AggregateID,
			&e.AggregateVersion,
			&e.ScopeVersion,
			&e.ProjectID,
			&envID,
			&e.EventType,
			&payload,
			&status,
			&e.RetryCount,
			&next,
			&e.CreatedAt,
			&published,
		); err != nil {
			return nil, apperror.NewInfrastructureError("DB", "outbox scan failed", err)
		}
		if envID.Valid {
			v := envID.String
			e.EnvironmentID = &v
		}
		e.Payload = payload
		if next.Valid {
			t := next.Time
			e.NextAttemptAt = &t
		}
		if published.Valid {
			t := published.Time
			e.PublishedAt = &t
		}
		switch status {
		case string(entity.OutboxStatusPending):
			e.Status = entity.OutboxStatusPending
		case string(entity.OutboxStatusPublishing):
			e.Status = entity.OutboxStatusPublishing
		case string(entity.OutboxStatusPublished):
			e.Status = entity.OutboxStatusPublished
		case string(entity.OutboxStatusFailed):
			e.Status = entity.OutboxStatusFailed
		default:
			e.Status = entity.OutboxStatusFailed
		}
		out = append(out, e)
	}
	if err := rows.Err(); err != nil {
		return nil, apperror.NewInfrastructureError("DB", "outbox rows failed", err)
	}
	return out, nil
}

func (r *OutboxRepository) ClaimNext(ctx context.Context, workerID string, limit int, lockFor time.Duration) ([]entity.OutboxEvent, error) {
	if workerID == "" {
		workerID = "outbox-worker"
	}
	if limit <= 0 {
		limit = 100
	}
	if lockFor <= 0 {
		lockFor = 30 * time.Second
	}

	now := time.Now().UTC()
	lockExpiredBefore := now.Add(-lockFor)

	// Select a small candidate set first to reduce lock contention.
	const selectQ = `
SELECT id
FROM outbox_events
WHERE status IN ('pending','failed','publishing')
  AND (next_attempt_at IS NULL OR next_attempt_at <= ?)
  AND (
    status IN ('pending','failed')
    OR (status = 'publishing' AND (locked_at IS NULL OR locked_at <= ?))
  )
ORDER BY created_at ASC
LIMIT ?`

	rows, err := r.db.QueryContext(ctx, selectQ, now, lockExpiredBefore, limit)
	if err != nil {
		return nil, apperror.NewInfrastructureError("DB", "outbox select candidates failed", err)
	}
	defer rows.Close()

	var ids []string
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			return nil, apperror.NewInfrastructureError("DB", "outbox candidate scan failed", err)
		}
		ids = append(ids, id)
	}
	if err := rows.Err(); err != nil {
		return nil, apperror.NewInfrastructureError("DB", "outbox candidate rows failed", err)
	}
	if len(ids) == 0 {
		return nil, nil
	}

	// Claim each ID; only return successfully claimed events.
	const claimQ = `
UPDATE outbox_events
SET status = 'publishing', locked_by = ?, locked_at = ?
WHERE id = ?
  AND (next_attempt_at IS NULL OR next_attempt_at <= ?)
  AND (
    status IN ('pending','failed')
    OR (status = 'publishing' AND (locked_at IS NULL OR locked_at <= ?))
  )`

	var claimed []string
	for _, id := range ids {
		res, err := r.db.ExecContext(ctx, claimQ, workerID, now, id, now, lockExpiredBefore)
		if err != nil {
			return nil, apperror.NewInfrastructureError("DB", "outbox claim failed", err)
		}
		aff, _ := res.RowsAffected()
		if aff > 0 {
			claimed = append(claimed, id)
		}
	}
	if len(claimed) == 0 {
		return nil, nil
	}

	// Load full records for claimed IDs.
	const loadQ = `
SELECT id, aggregate_type, aggregate_id, aggregate_version, scope_version,
       project_id, environment_id, event_type, payload_json,
       status, retry_count, next_attempt_at, created_at, published_at
FROM outbox_events
WHERE id = ?`

	out := make([]entity.OutboxEvent, 0, len(claimed))
	for _, id := range claimed {
		var e entity.OutboxEvent
		var envID sql.NullString
		var payload json.RawMessage
		var status string
		var next sql.NullTime
		var published sql.NullTime

		err := r.db.QueryRowContext(ctx, loadQ, id).Scan(
			&e.ID,
			&e.AggregateType,
			&e.AggregateID,
			&e.AggregateVersion,
			&e.ScopeVersion,
			&e.ProjectID,
			&envID,
			&e.EventType,
			&payload,
			&status,
			&e.RetryCount,
			&next,
			&e.CreatedAt,
			&published,
		)
		if err != nil {
			if errors.Is(err, sql.ErrNoRows) {
				continue
			}
			return nil, apperror.NewInfrastructureError("DB", "outbox load claimed failed", err)
		}
		if envID.Valid {
			v := envID.String
			e.EnvironmentID = &v
		}
		e.Payload = payload
		if next.Valid {
			t := next.Time
			e.NextAttemptAt = &t
		}
		if published.Valid {
			t := published.Time
			e.PublishedAt = &t
		}
		switch status {
		case string(entity.OutboxStatusPending):
			e.Status = entity.OutboxStatusPending
		case string(entity.OutboxStatusPublishing):
			e.Status = entity.OutboxStatusPublishing
		case string(entity.OutboxStatusPublished):
			e.Status = entity.OutboxStatusPublished
		case string(entity.OutboxStatusFailed):
			e.Status = entity.OutboxStatusFailed
		default:
			e.Status = entity.OutboxStatusPublishing
		}
		out = append(out, e)
	}
	return out, nil
}

func (r *OutboxRepository) MarkPublished(ctx context.Context, eventID string) error {
	const q = `
UPDATE outbox_events
SET status = 'published', published_at = ?, last_error = NULL, next_attempt_at = NULL, locked_by = NULL, locked_at = NULL
WHERE id = ?`
	if _, err := r.db.ExecContext(ctx, q, time.Now().UTC(), eventID); err != nil {
		return apperror.NewInfrastructureError("DB", "outbox mark published failed", err)
	}
	return nil
}

func (r *OutboxRepository) MarkFailed(ctx context.Context, eventID string, lastError string, nextAttemptAt *time.Time) error {
	const q = `
UPDATE outbox_events
SET status = 'failed', retry_count = retry_count + 1, last_error = ?, next_attempt_at = ?, locked_by = NULL, locked_at = NULL
WHERE id = ?`

	var next any = nil
	if nextAttemptAt != nil {
		next = *nextAttemptAt
	}

	if _, err := r.db.ExecContext(ctx, q, lastError, next, eventID); err != nil {
		return apperror.NewInfrastructureError("DB", "outbox mark failed failed", err)
	}
	return nil
}
