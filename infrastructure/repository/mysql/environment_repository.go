package mysql

import (
	"context"
	"database/sql"
	"strings"
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

func (r *EnvironmentRepository) ListByProject(ctx context.Context, projectID, q string, limit, offset int) ([]entity.Environment, int, error) {
	if limit <= 0 || limit > 200 {
		limit = 50
	}
	if offset < 0 {
		offset = 0
	}
	q = strings.TrimSpace(q)
	const base = `
SELECT id, project_id, ` + "`key`" + `, name, sort_order, version, created_at, updated_at, deleted_at
FROM environments
WHERE project_id = ? AND deleted_at IS NULL
`
	args := []any{projectID}
	where := ""
	if q != "" {
		where = " AND (" + "`key`" + " LIKE ? OR name LIKE ?)"
		like := "%" + q + "%"
		args = append(args, like, like)
	}

	var total int
	countQ := "SELECT COUNT(1) FROM environments WHERE project_id = ? AND deleted_at IS NULL" + where
	if err := r.db.QueryRowContext(ctx, countQ, args...).Scan(&total); err != nil {
		return nil, 0, apperror.NewInfrastructureError("DB", "environments count failed", err)
	}

	listQ := base + where + " ORDER BY " + "`key`" + " ASC, name ASC LIMIT ? OFFSET ?"
	args = append(args, limit, offset)

	rows, err := r.db.QueryContext(ctx, listQ, args...)
	if err != nil {
		return nil, 0, apperror.NewInfrastructureError("DB", "environments list failed", err)
	}
	defer rows.Close()

	var out []entity.Environment
	for rows.Next() {
		var e entity.Environment
		var deletedAt sql.NullTime
		if err := rows.Scan(&e.ID, &e.ProjectID, &e.Key, &e.Name, &e.SortOrder, &e.Version, &e.CreatedAt, &e.UpdatedAt, &deletedAt); err != nil {
			return nil, 0, apperror.NewInfrastructureError("DB", "environments scan failed", err)
		}
		if deletedAt.Valid {
			t := deletedAt.Time
			e.DeletedAt = &t
		}
		out = append(out, e)
	}
	if err := rows.Err(); err != nil {
		return nil, 0, apperror.NewInfrastructureError("DB", "environments rows failed", err)
	}
	return out, total, nil
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

func (r *EnvironmentRepository) SoftDelete(ctx context.Context, projectID, environmentKey string) error {
	now := time.Now().UTC()
	const q = `
UPDATE environments
SET deleted_at = ?, updated_at = ?, version = version + 1
WHERE project_id = ? AND ` + "`key`" + ` = ? AND deleted_at IS NULL`
	res, err := r.db.ExecContext(ctx, q, now, now, projectID, environmentKey)
	if err != nil {
		return apperror.NewInfrastructureError("DB", "environment soft delete failed", err)
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return apperror.NewNotFoundError("Environment", sql.ErrNoRows)
	}
	return nil
}
