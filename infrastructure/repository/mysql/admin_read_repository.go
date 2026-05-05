package mysql

import (
	"context"
	"database/sql"
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

func (r *AdminReadRepository) ListAuditEntries(ctx context.Context, projectID, entityType, entityID string, limit, offset int) ([]repository.AuditEntrySummary, int, error) {
	if limit <= 0 || limit > 500 {
		limit = 100
	}
	if offset < 0 {
		offset = 0
	}
	var sb strings.Builder
	sb.WriteString(`
SELECT ae.id, ae.entity_type, ae.entity_id, f.` + "`key`" + ` AS entity_key, f.name AS entity_name, ae.action, ae.correlation_id, ae.created_at
FROM audit_entries ae
LEFT JOIN flags f ON f.id = ae.entity_id AND ae.entity_type = 'flag' AND f.project_id = ae.project_id
WHERE ae.project_id = ?`)
	args := []any{projectID}
	if entityType != "" {
		sb.WriteString(` AND ae.entity_type = ?`)
		args = append(args, entityType)
	}
	if entityID != "" {
		sb.WriteString(` AND ae.entity_id = ?`)
		args = append(args, entityID)
	}

	var total int
	countQ := `
SELECT COUNT(1)
FROM audit_entries ae
WHERE ae.project_id = ?` + func() string {
		var w strings.Builder
		if entityType != "" {
			w.WriteString(` AND ae.entity_type = ?`)
		}
		if entityID != "" {
			w.WriteString(` AND ae.entity_id = ?`)
		}
		return w.String()
	}()
	if err := r.db.QueryRowContext(ctx, countQ, args...).Scan(&total); err != nil {
		return nil, 0, apperror.NewInfrastructureError("DB", "audit count failed", err)
	}

	sb.WriteString(` ORDER BY ae.created_at DESC, ae.id DESC LIMIT ? OFFSET ?`)
	args = append(args, limit, offset)

	rows, err := r.db.QueryContext(ctx, sb.String(), args...)
	if err != nil {
		return nil, 0, apperror.NewInfrastructureError("DB", "list audit failed", err)
	}
	defer rows.Close()

	var out []repository.AuditEntrySummary
	for rows.Next() {
		var row repository.AuditEntrySummary
		var entityKey sql.NullString
		var entityName sql.NullString
		var corr sql.NullString
		var created time.Time
		if err := rows.Scan(&row.ID, &row.EntityType, &row.EntityID, &entityKey, &entityName, &row.Action, &corr, &created); err != nil {
			return nil, 0, apperror.NewInfrastructureError("DB", "scan audit failed", err)
		}
		if entityKey.Valid {
			s := entityKey.String
			row.EntityKey = &s
		}
		if entityName.Valid {
			s := entityName.String
			row.EntityName = &s
		}
		if corr.Valid {
			s := corr.String
			row.CorrelationID = &s
		}
		row.CreatedAt = created.UTC().Format(time.RFC3339Nano)
		out = append(out, row)
	}
	if err := rows.Err(); err != nil {
		return nil, 0, apperror.NewInfrastructureError("DB", "audit rows failed", err)
	}
	return out, total, nil
}
