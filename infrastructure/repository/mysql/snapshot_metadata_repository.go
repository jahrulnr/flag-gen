package mysql

import (
	"context"
	"database/sql"
	"time"

	"flag-gen/domain/entity"
	"flag-gen/domain/repository"
	"flag-gen/domain/valueobject"
	apperror "flag-gen/pkg/error"
)

type SnapshotMetadataRepository struct {
	db *sql.DB
}

func NewSnapshotMetadataRepository(db *DB) *SnapshotMetadataRepository {
	return &SnapshotMetadataRepository{db: db.SQL()}
}

var _ repository.SnapshotMetadataRepository = (*SnapshotMetadataRepository)(nil)

func (r *SnapshotMetadataRepository) GetByScope(ctx context.Context, scope valueobject.Scope) (*entity.SnapshotMetadata, error) {
	const q = `
SELECT sm.id, sm.project_id, sm.environment_id, e.` + "`key`" + `, sm.scope_version, sm.checksum, sm.file_path,
       sm.source_outbox_event_id, sm.generated_at, sm.last_validated_at, sm.status
FROM snapshot_metadata sm
JOIN environments e ON e.id = sm.environment_id
WHERE sm.project_id = ? AND e.` + "`key`" + ` = ?
LIMIT 1`

	var m entity.SnapshotMetadata
	var source sql.NullString
	var lastValidated sql.NullTime
	var status string
	err := r.db.QueryRowContext(ctx, q, scope.ProjectID, scope.EnvironmentKey).Scan(
		&m.ID,
		&m.ProjectID,
		&m.EnvironmentID,
		&m.EnvironmentKey,
		&m.ScopeVersion,
		&m.Checksum,
		&m.FilePath,
		&source,
		&m.GeneratedAt,
		&lastValidated,
		&status,
	)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, apperror.NewNotFoundError("SnapshotMetadata", err)
		}
		return nil, apperror.NewInfrastructureError("DB", "snapshot metadata get failed", err)
	}
	if source.Valid {
		m.SourceOutboxEventID = &source.String
	}
	if lastValidated.Valid {
		t := lastValidated.Time
		m.LastValidatedAt = &t
	}
	switch status {
	case string(entity.SnapshotStatusValid):
		m.Status = entity.SnapshotStatusValid
	case string(entity.SnapshotStatusInvalid):
		m.Status = entity.SnapshotStatusInvalid
	default:
		m.Status = entity.SnapshotStatusInvalid
	}
	return &m, nil
}

func (r *SnapshotMetadataRepository) Upsert(ctx context.Context, meta entity.SnapshotMetadata) error {
	const q = `
INSERT INTO snapshot_metadata
  (id, project_id, environment_id, scope_version, checksum, file_path, source_outbox_event_id, generated_at, last_validated_at, status)
VALUES
  (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
ON DUPLICATE KEY UPDATE
  scope_version = VALUES(scope_version),
  checksum = VALUES(checksum),
  file_path = VALUES(file_path),
  source_outbox_event_id = VALUES(source_outbox_event_id),
  generated_at = VALUES(generated_at),
  last_validated_at = VALUES(last_validated_at),
  status = VALUES(status)`

	if meta.GeneratedAt.IsZero() {
		meta.GeneratedAt = time.Now().UTC()
	}

	var src any = nil
	if meta.SourceOutboxEventID != nil {
		src = *meta.SourceOutboxEventID
	}
	var last any = nil
	if meta.LastValidatedAt != nil {
		last = *meta.LastValidatedAt
	}

	if _, err := r.db.ExecContext(ctx, q,
		meta.ID,
		meta.ProjectID,
		meta.EnvironmentID,
		meta.ScopeVersion,
		meta.Checksum,
		meta.FilePath,
		src,
		meta.GeneratedAt,
		last,
		string(meta.Status),
	); err != nil {
		return apperror.NewInfrastructureError("DB", "snapshot metadata upsert failed", err)
	}
	return nil
}
