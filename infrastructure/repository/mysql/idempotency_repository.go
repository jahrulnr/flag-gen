package mysql

import (
	"context"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"time"

	"flag-gen/domain/entity"
	"flag-gen/domain/repository"
	apperror "flag-gen/pkg/error"
)

type IdempotencyRepository struct {
	db *sql.DB
}

func NewIdempotencyRepository(db *DB) *IdempotencyRepository {
	return &IdempotencyRepository{db: db.SQL()}
}

var _ repository.IdempotencyRepository = (*IdempotencyRepository)(nil)

func (r *IdempotencyRepository) BeginProcessing(ctx context.Context, key string, actorID *string, projectID *string, requestHash string, lockFor time.Duration) (entity.IdempotencyRecord, error) {
	now := time.Now().UTC()
	lockUntil := now.Add(lockFor)

	// We use a single-row upsert to avoid race conditions. If a completed record exists with a different request hash,
	// return conflict at usecase/handler layer (by inspecting returned row).
	const q = `
INSERT INTO idempotency_keys
  (id, idempotency_key, actor_id, project_id, request_hash, status, response_code, response_json, locked_until, created_at, updated_at)
VALUES
  (?, ?, ?, ?, ?, 'processing', NULL, NULL, ?, ?, ?)
ON DUPLICATE KEY UPDATE
  -- If currently completed/failed, keep it as-is (locked_until not extended).
  locked_until = CASE WHEN status = 'processing' THEN VALUES(locked_until) ELSE locked_until END,
  updated_at = VALUES(updated_at)`

	id := newUUIDLike(key, now)

	var act any = nil
	if actorID != nil {
		act = *actorID
	}
	var proj any = nil
	if projectID != nil {
		proj = *projectID
	}

	if _, err := r.db.ExecContext(ctx, q, id, key, act, proj, requestHash, lockUntil, now, now); err != nil {
		return entity.IdempotencyRecord{}, apperror.NewInfrastructureError("DB", "idempotency begin failed", err)
	}

	return r.getByKey(ctx, key, actorID)
}

func (r *IdempotencyRepository) MarkCompleted(ctx context.Context, recordID string, responseCode int, responseJSON []byte) error {
	return r.markDone(ctx, recordID, entity.IdempotencyCompleted, responseCode, responseJSON)
}

func (r *IdempotencyRepository) MarkFailed(ctx context.Context, recordID string, responseCode int, responseJSON []byte) error {
	return r.markDone(ctx, recordID, entity.IdempotencyFailed, responseCode, responseJSON)
}

func (r *IdempotencyRepository) markDone(ctx context.Context, recordID string, status entity.IdempotencyStatus, responseCode int, responseJSON []byte) error {
	const q = `
UPDATE idempotency_keys
SET status = ?, response_code = ?, response_json = ?, locked_until = NULL, updated_at = ?
WHERE id = ?`

	now := time.Now().UTC()
	if _, err := r.db.ExecContext(ctx, q, string(status), responseCode, responseJSON, now, recordID); err != nil {
		return apperror.NewInfrastructureError("DB", "idempotency mark done failed", err)
	}
	return nil
}

func (r *IdempotencyRepository) getByKey(ctx context.Context, key string, actorID *string) (entity.IdempotencyRecord, error) {
	const q = `
SELECT id, idempotency_key, actor_id, project_id, request_hash, status, response_code, response_json, locked_until, created_at, updated_at
FROM idempotency_keys
WHERE idempotency_key = ? AND ((actor_id IS NULL AND ?) OR actor_id = ?)
ORDER BY created_at DESC
LIMIT 1`

	var rec entity.IdempotencyRecord
	var act sql.NullString
	var proj sql.NullString
	var code sql.NullInt64
	var resp sql.NullString
	var locked sql.NullTime

	var actorIsNil bool = actorID == nil
	var actorVal any = ""
	if actorID != nil {
		actorVal = *actorID
	}

	err := r.db.QueryRowContext(ctx, q, key, actorIsNil, actorVal).Scan(
		&rec.ID,
		&rec.IdempotencyKey,
		&act,
		&proj,
		&rec.RequestHash,
		&rec.Status,
		&code,
		&resp,
		&locked,
		&rec.CreatedAt,
		&rec.UpdatedAt,
	)
	if err != nil {
		return entity.IdempotencyRecord{}, apperror.NewInfrastructureError("DB", "idempotency read failed", err)
	}

	if act.Valid {
		rec.ActorID = &act.String
	}
	if proj.Valid {
		rec.ProjectID = &proj.String
	}
	if code.Valid {
		v := int(code.Int64)
		rec.ResponseCode = &v
	}
	if resp.Valid {
		rec.ResponseJSON = []byte(resp.String)
	}
	if locked.Valid {
		t := locked.Time
		rec.LockedUntil = &t
	}

	return rec, nil
}

func newUUIDLike(seed string, t time.Time) string {
	// Deterministic-ish placeholder. Real UUID generation can be swapped in later.
	sum := sha256.Sum256([]byte(seed + "\x1f" + t.Format(time.RFC3339Nano)))
	hexed := hex.EncodeToString(sum[:])
	// 32 chars (like UUID without dashes) is enough for CHAR(36) fields in dev-mode.
	return hexed[:32]
}
