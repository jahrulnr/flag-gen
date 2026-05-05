package mysql

import (
	"context"
	"database/sql"
	"errors"
	"strings"
	"time"

	"flag-gen/domain/entity"
	"flag-gen/domain/repository"
	apperror "flag-gen/pkg/error"

	mysqlerr "github.com/go-sql-driver/mysql"
)

type ProjectRepository struct {
	db *sql.DB
}

func NewProjectRepository(db *DB) *ProjectRepository {
	return &ProjectRepository{db: db.SQL()}
}

var _ repository.ProjectRepository = (*ProjectRepository)(nil)

func (r *ProjectRepository) List(ctx context.Context, q string, limit, offset int) ([]entity.Project, int, error) {
	if limit <= 0 || limit > 200 {
		limit = 50
	}
	if offset < 0 {
		offset = 0
	}
	q = strings.TrimSpace(q)
	const base = `
SELECT id, ` + "`key`" + `, name, description, created_by, created_at, updated_at, deleted_at
FROM projects
WHERE deleted_at IS NULL
`

	where := ""
	args := []any{}
	if q != "" {
		where = " AND (" + "`key`" + " LIKE ? OR name LIKE ?)"
		like := "%" + q + "%"
		args = append(args, like, like)
	}

	var total int
	countQ := "SELECT COUNT(1) FROM projects WHERE deleted_at IS NULL" + where
	if err := r.db.QueryRowContext(ctx, countQ, args...).Scan(&total); err != nil {
		return nil, 0, apperror.NewInfrastructureError("DB", "projects count failed", err)
	}

	listQ := base + where + " ORDER BY " + "`key`" + " ASC, name ASC LIMIT ? OFFSET ?"
	args = append(args, limit, offset)

	rows, err := r.db.QueryContext(ctx, listQ, args...)
	if err != nil {
		return nil, 0, apperror.NewInfrastructureError("DB", "projects list failed", err)
	}
	defer rows.Close()

	var out []entity.Project
	for rows.Next() {
		var p entity.Project
		var desc sql.NullString
		var createdBy sql.NullString
		var deletedAt sql.NullTime
		if err := rows.Scan(&p.ID, &p.Key, &p.Name, &desc, &createdBy, &p.CreatedAt, &p.UpdatedAt, &deletedAt); err != nil {
			return nil, 0, apperror.NewInfrastructureError("DB", "projects scan failed", err)
		}
		if desc.Valid {
			p.Description = &desc.String
		}
		if createdBy.Valid {
			p.CreatedBy = &createdBy.String
		}
		if deletedAt.Valid {
			t := deletedAt.Time
			p.DeletedAt = &t
		}
		out = append(out, p)
	}
	if err := rows.Err(); err != nil {
		return nil, 0, apperror.NewInfrastructureError("DB", "projects rows failed", err)
	}
	return out, total, nil
}

func (r *ProjectRepository) Create(ctx context.Context, p entity.Project) (entity.Project, error) {
	const q = `
INSERT INTO projects (id, ` + "`key`" + `, name, description, created_by, created_at, updated_at, deleted_at)
VALUES (?, ?, ?, ?, ?, ?, ?, NULL)`

	now := time.Now().UTC()
	if p.CreatedAt.IsZero() {
		p.CreatedAt = now
	}
	if p.UpdatedAt.IsZero() {
		p.UpdatedAt = now
	}

	var desc any = nil
	if p.Description != nil {
		desc = *p.Description
	}
	var createdBy any = nil
	if p.CreatedBy != nil {
		createdBy = *p.CreatedBy
	}

	if _, err := r.db.ExecContext(ctx, q, p.ID, p.Key, p.Name, desc, createdBy, p.CreatedAt, p.UpdatedAt); err != nil {
		if isMySQLDuplicateKey(err) {
			return entity.Project{}, apperror.New(apperror.StatusConflict, "Project key already exists. Choose a different key.", err)
		}
		return entity.Project{}, apperror.NewInfrastructureError("DB", "project insert failed", err)
	}
	return p, nil
}

func isMySQLDuplicateKey(err error) bool {
	var me *mysqlerr.MySQLError
	if errors.As(err, &me) {
		// 1062 = ER_DUP_ENTRY
		return me.Number == 1062
	}
	return false
}

func (r *ProjectRepository) FindByID(ctx context.Context, id string) (*entity.Project, error) {
	const q = `
SELECT id, ` + "`key`" + `, name, description, created_by, created_at, updated_at, deleted_at
FROM projects
WHERE id = ?`

	var p entity.Project
	var desc sql.NullString
	var createdBy sql.NullString
	var deletedAt sql.NullTime

	err := r.db.QueryRowContext(ctx, q, id).Scan(&p.ID, &p.Key, &p.Name, &desc, &createdBy, &p.CreatedAt, &p.UpdatedAt, &deletedAt)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, apperror.NewNotFoundError("Project", err)
		}
		return nil, apperror.NewInfrastructureError("DB", "project get failed", err)
	}
	if desc.Valid {
		p.Description = &desc.String
	}
	if createdBy.Valid {
		p.CreatedBy = &createdBy.String
	}
	if deletedAt.Valid {
		t := deletedAt.Time
		p.DeletedAt = &t
	}
	return &p, nil
}

func (r *ProjectRepository) SoftDelete(ctx context.Context, projectID string) error {
	now := time.Now().UTC()
	const q = `
UPDATE projects
SET deleted_at = ?, updated_at = ?
WHERE id = ? AND deleted_at IS NULL`
	res, err := r.db.ExecContext(ctx, q, now, now, projectID)
	if err != nil {
		return apperror.NewInfrastructureError("DB", "project soft delete failed", err)
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return apperror.NewNotFoundError("Project", sql.ErrNoRows)
	}
	return nil
}
