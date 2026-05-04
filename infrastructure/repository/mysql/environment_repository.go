package mysql

import (
	"context"
	"database/sql"
	"time"

	"flag-gen/domain/entity"
	"flag-gen/domain/repository"
	apperror "flag-gen/pkg/error"
)

type EnvironmentRepository struct {
	db *sql.DB
}

func NewEnvironmentRepository(db *DB) *EnvironmentRepository {
	return &EnvironmentRepository{db: db.SQL()}
}

var _ repository.EnvironmentRepository = (*EnvironmentRepository)(nil)

func (r *EnvironmentRepository) ListByProject(ctx context.Context, projectID string) ([]entity.Environment, error) {
	const q = `
SELECT id, project_id, ` + "`key`" + `, name, sort_order, version, created_at, updated_at, deleted_at
FROM environments
WHERE project_id = ? AND deleted_at IS NULL
ORDER BY sort_order ASC, created_at ASC`

	rows, err := r.db.QueryContext(ctx, q, projectID)
	if err != nil {
		return nil, apperror.NewInfrastructureError("DB", "environments list failed", err)
	}
	defer rows.Close()

	var out []entity.Environment
	for rows.Next() {
		var e entity.Environment
		var deletedAt sql.NullTime
		if err := rows.Scan(&e.ID, &e.ProjectID, &e.Key, &e.Name, &e.SortOrder, &e.Version, &e.CreatedAt, &e.UpdatedAt, &deletedAt); err != nil {
			return nil, apperror.NewInfrastructureError("DB", "environments scan failed", err)
		}
		if deletedAt.Valid {
			t := deletedAt.Time
			e.DeletedAt = &t
		}
		out = append(out, e)
	}
	if err := rows.Err(); err != nil {
		return nil, apperror.NewInfrastructureError("DB", "environments rows failed", err)
	}
	return out, nil
}

func (r *EnvironmentRepository) Create(ctx context.Context, env entity.Environment) (entity.Environment, error) {
	const q = `
INSERT INTO environments (id, project_id, ` + "`key`" + `, name, sort_order, version, created_at, updated_at, deleted_at)
VALUES (?, ?, ?, ?, ?, ?, ?, ?, NULL)`

	now := time.Now().UTC()
	if env.CreatedAt.IsZero() {
		env.CreatedAt = now
	}
	if env.UpdatedAt.IsZero() {
		env.UpdatedAt = now
	}
	if env.Version == 0 {
		env.Version = 1
	}

	if _, err := r.db.ExecContext(ctx, q, env.ID, env.ProjectID, env.Key, env.Name, env.SortOrder, env.Version, env.CreatedAt, env.UpdatedAt); err != nil {
		return entity.Environment{}, apperror.NewInfrastructureError("DB", "environment insert failed", err)
	}
	return env, nil
}

func (r *EnvironmentRepository) FindByProjectAndKey(ctx context.Context, projectID, envKey string) (*entity.Environment, error) {
	const q = `
SELECT id, project_id, ` + "`key`" + `, name, sort_order, version, created_at, updated_at, deleted_at
FROM environments
WHERE project_id = ? AND ` + "`key`" + ` = ? AND deleted_at IS NULL`

	var e entity.Environment
	var deletedAt sql.NullTime
	err := r.db.QueryRowContext(ctx, q, projectID, envKey).Scan(&e.ID, &e.ProjectID, &e.Key, &e.Name, &e.SortOrder, &e.Version, &e.CreatedAt, &e.UpdatedAt, &deletedAt)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, apperror.NewNotFoundError("Environment", err)
		}
		return nil, apperror.NewInfrastructureError("DB", "environment get failed", err)
	}
	if deletedAt.Valid {
		t := deletedAt.Time
		e.DeletedAt = &t
	}
	return &e, nil
}
