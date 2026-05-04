package mysql

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
	"time"

	"flag-gen/domain/repository"
	"flag-gen/domain/valueobject"
	apperror "flag-gen/pkg/error"
)

// AdminReadRepository implements repository.AdminReadRepository.
type AdminReadRepository struct {
	db *sql.DB
}

func NewAdminReadRepository(db *DB) *AdminReadRepository {
	return &AdminReadRepository{db: db.SQL()}
}

var _ repository.AdminReadRepository = (*AdminReadRepository)(nil)

func (r *AdminReadRepository) ListFlagVersions(ctx context.Context, scope valueobject.Scope, flagID string) ([]repository.FlagVersionSummary, error) {
	const q = `
SELECT fv.id, fv.version, fv.change_type, fv.created_at
FROM flag_versions fv
JOIN flags f ON f.id = fv.flag_id
JOIN environments e ON e.id = f.environment_id
WHERE f.project_id = ? AND e.` + "`key`" + ` = ? AND f.id = ? AND f.deleted_at IS NULL
ORDER BY fv.version DESC`

	rows, err := r.db.QueryContext(ctx, q, scope.ProjectID, scope.EnvironmentKey, flagID)
	if err != nil {
		return nil, apperror.NewInfrastructureError("DB", "list flag versions failed", err)
	}
	defer rows.Close()

	var out []repository.FlagVersionSummary
	for rows.Next() {
		var row repository.FlagVersionSummary
		var created time.Time
		if err := rows.Scan(&row.ID, &row.Version, &row.ChangeType, &created); err != nil {
			return nil, apperror.NewInfrastructureError("DB", "scan flag version failed", err)
		}
		row.CreatedAt = created.UTC().Format(time.RFC3339Nano)
		out = append(out, row)
	}
	return out, rows.Err()
}

func (r *AdminReadRepository) GetFlagVersionSnapshot(ctx context.Context, scope valueobject.Scope, flagID string, version int) ([]byte, error) {
	const q = `
SELECT fv.snapshot_json
FROM flag_versions fv
JOIN flags f ON f.id = fv.flag_id
JOIN environments e ON e.id = f.environment_id
WHERE f.project_id = ? AND e.` + "`key`" + ` = ? AND f.id = ? AND fv.version = ?`

	var snap []byte
	err := r.db.QueryRowContext(ctx, q, scope.ProjectID, scope.EnvironmentKey, flagID, version).Scan(&snap)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, apperror.NewNotFoundError("FlagVersion", err)
		}
		return nil, apperror.NewInfrastructureError("DB", "get flag version snapshot failed", err)
	}
	return snap, nil
}

func (r *AdminReadRepository) ListAuditEntries(ctx context.Context, projectID, entityType, entityID string, limit int) ([]repository.AuditEntrySummary, error) {
	if limit <= 0 || limit > 500 {
		limit = 100
	}
	var sb strings.Builder
	sb.WriteString(`
SELECT id, entity_type, entity_id, action, correlation_id, created_at
FROM audit_entries
WHERE project_id = ?`)
	args := []any{projectID}
	if entityType != "" {
		sb.WriteString(` AND entity_type = ?`)
		args = append(args, entityType)
	}
	if entityID != "" {
		sb.WriteString(` AND entity_id = ?`)
		args = append(args, entityID)
	}
	sb.WriteString(` ORDER BY created_at DESC LIMIT `)
	sb.WriteString(fmt.Sprintf("%d", limit))

	rows, err := r.db.QueryContext(ctx, sb.String(), args...)
	if err != nil {
		return nil, apperror.NewInfrastructureError("DB", "list audit failed", err)
	}
	defer rows.Close()

	var out []repository.AuditEntrySummary
	for rows.Next() {
		var row repository.AuditEntrySummary
		var corr sql.NullString
		var created time.Time
		if err := rows.Scan(&row.ID, &row.EntityType, &row.EntityID, &row.Action, &corr, &created); err != nil {
			return nil, apperror.NewInfrastructureError("DB", "scan audit failed", err)
		}
		if corr.Valid {
			s := corr.String
			row.CorrelationID = &s
		}
		row.CreatedAt = created.UTC().Format(time.RFC3339Nano)
		out = append(out, row)
	}
	return out, rows.Err()
}
