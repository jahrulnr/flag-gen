package mysql

import (
	"context"
	"database/sql"
	"time"

	"flag-gen/domain/entity"
	"flag-gen/domain/repository"
	apperror "flag-gen/pkg/error"
)

type SegmentRepository struct {
	db *sql.DB
}

func NewSegmentRepository(db *DB) *SegmentRepository {
	return &SegmentRepository{db: db.SQL()}
}

var _ repository.SegmentRepository = (*SegmentRepository)(nil)

func (r *SegmentRepository) ListByProject(ctx context.Context, projectID string) ([]entity.Segment, error) {
	const q = `
SELECT id, project_id, ` + "`key`" + `, name, rules_json, version, created_by, updated_by, created_at, updated_at, deleted_at
FROM segments
WHERE project_id = ? AND deleted_at IS NULL
ORDER BY created_at ASC`

	rows, err := r.db.QueryContext(ctx, q, projectID)
	if err != nil {
		return nil, apperror.NewInfrastructureError("DB", "segments list failed", err)
	}
	defer rows.Close()

	var out []entity.Segment
	for rows.Next() {
		var s entity.Segment
		var rules []byte
		var createdBy sql.NullString
		var updatedBy sql.NullString
		var deletedAt sql.NullTime
		if err := rows.Scan(&s.ID, &s.ProjectID, &s.Key, &s.Name, &rules, &s.Version, &createdBy, &updatedBy, &s.CreatedAt, &s.UpdatedAt, &deletedAt); err != nil {
			return nil, apperror.NewInfrastructureError("DB", "segments scan failed", err)
		}
		s.RulesJSON = rules
		if createdBy.Valid {
			s.CreatedBy = &createdBy.String
		}
		if updatedBy.Valid {
			s.UpdatedBy = &updatedBy.String
		}
		if deletedAt.Valid {
			t := deletedAt.Time
			s.DeletedAt = &t
		}
		out = append(out, s)
	}
	if err := rows.Err(); err != nil {
		return nil, apperror.NewInfrastructureError("DB", "segments rows failed", err)
	}
	return out, nil
}

func (r *SegmentRepository) Create(ctx context.Context, s entity.Segment) (entity.Segment, error) {
	const q = `
INSERT INTO segments (id, project_id, ` + "`key`" + `, name, rules_json, version, created_by, updated_by, created_at, updated_at, deleted_at)
VALUES (?, ?, ?, ?, ?, ?, ?, NULL, ?, ?, NULL)`

	now := time.Now().UTC()
	if s.CreatedAt.IsZero() {
		s.CreatedAt = now
	}
	if s.UpdatedAt.IsZero() {
		s.UpdatedAt = now
	}
	if s.Version == 0 {
		s.Version = 1
	}

	var createdBy any = nil
	if s.CreatedBy != nil {
		createdBy = *s.CreatedBy
	}

	if _, err := r.db.ExecContext(ctx, q, s.ID, s.ProjectID, s.Key, s.Name, s.RulesJSON, s.Version, createdBy, s.CreatedAt, s.UpdatedAt); err != nil {
		return entity.Segment{}, apperror.NewInfrastructureError("DB", "segment insert failed", err)
	}
	return s, nil
}

func (r *SegmentRepository) Update(ctx context.Context, s entity.Segment) (entity.Segment, error) {
	const q = `
UPDATE segments
SET name = ?, rules_json = ?, version = version + 1, updated_by = ?, updated_at = ?
WHERE project_id = ? AND id = ? AND deleted_at IS NULL`

	now := time.Now().UTC()
	s.UpdatedAt = now

	var updatedBy any = nil
	if s.UpdatedBy != nil {
		updatedBy = *s.UpdatedBy
	}

	res, err := r.db.ExecContext(ctx, q, s.Name, s.RulesJSON, updatedBy, s.UpdatedAt, s.ProjectID, s.ID)
	if err != nil {
		return entity.Segment{}, apperror.NewInfrastructureError("DB", "segment update failed", err)
	}
	aff, _ := res.RowsAffected()
	if aff == 0 {
		return entity.Segment{}, apperror.NewNotFoundError("Segment", sql.ErrNoRows)
	}
	// caller can re-read if it needs the bumped version.
	return s, nil
}

func (r *SegmentRepository) SoftDelete(ctx context.Context, projectID, segmentID string) error {
	const q = `
UPDATE segments
SET deleted_at = ?, updated_at = ?
WHERE project_id = ? AND id = ? AND deleted_at IS NULL`

	now := time.Now().UTC()
	res, err := r.db.ExecContext(ctx, q, now, now, projectID, segmentID)
	if err != nil {
		return apperror.NewInfrastructureError("DB", "segment delete failed", err)
	}
	aff, _ := res.RowsAffected()
	if aff == 0 {
		return apperror.NewNotFoundError("Segment", sql.ErrNoRows)
	}
	return nil
}

