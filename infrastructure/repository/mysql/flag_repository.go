package mysql

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"time"

	"flag-gen/domain/entity"
	"flag-gen/domain/repository"
	"flag-gen/domain/valueobject"
	apperror "flag-gen/pkg/error"
	"flag-gen/pkg/requestctx"

	"github.com/google/uuid"
)

type FlagRepository struct {
	db *sql.DB
}

func NewFlagRepository(db *DB) *FlagRepository {
	return &FlagRepository{db: db.SQL()}
}

var _ repository.FlagRepository = (*FlagRepository)(nil)

func (r *FlagRepository) FindByKey(ctx context.Context, scope valueobject.Scope, flagKey string) (*entity.Flag, error) {
	const q = `
SELECT f.id, f.` + "`key`" + `, f.name, f.description, f.enabled, f.client_side,
       f.value_type, f.default_value_json, f.rollout_percentage, f.rollout_stickiness,
       f.version, f.created_at, f.updated_at, f.deleted_at
FROM flags f
JOIN environments e ON e.id = f.environment_id
WHERE f.project_id = ? AND e.` + "`key`" + ` = ? AND f.` + "`key`" + ` = ? AND f.deleted_at IS NULL
LIMIT 1`

	var f entity.Flag
	var desc sql.NullString
	var deleted sql.NullTime
	var valueType string
	var defaultJSON []byte
	var rolloutPct sql.NullFloat64
	var rolloutStickiness string

	err := r.db.QueryRowContext(ctx, q, scope.ProjectID, scope.EnvironmentKey, flagKey).Scan(
		&f.ID,
		&f.Key,
		&f.Name,
		&desc,
		&f.Enabled,
		&f.ClientSide,
		&valueType,
		&defaultJSON,
		&rolloutPct,
		&rolloutStickiness,
		&f.Version,
		&f.CreatedAt,
		&f.UpdatedAt,
		&deleted,
	)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, apperror.NewNotFoundError("Flag", err)
		}
		return nil, apperror.NewInfrastructureError("DB", "flag get failed", err)
	}
	if desc.Valid {
		f.Description = desc.String
	}
	if deleted.Valid {
		t := deleted.Time
		f.DeletedAt = &t
	}

	f.Value = entity.FlagValue{Type: entity.FlagValueType(valueType)}
	_ = json.Unmarshal(defaultJSON, &f.Value.Default)

	if rolloutPct.Valid {
		f.Rollout = &entity.RolloutConfig{Percentage: rolloutPct.Float64, Stickiness: rolloutStickiness}
	}

	rules, err := r.listRules(ctx, f.ID)
	if err != nil {
		return nil, err
	}
	f.Rules = rules

	return &f, nil
}

func (r *FlagRepository) ListByScope(ctx context.Context, scope valueobject.Scope) ([]entity.Flag, error) {
	const q = `
SELECT f.id, f.` + "`key`" + `, f.name, f.description, f.enabled, f.client_side,
       f.value_type, f.default_value_json, f.rollout_percentage, f.rollout_stickiness,
       f.version, f.created_at, f.updated_at, f.deleted_at
FROM flags f
JOIN environments e ON e.id = f.environment_id
WHERE f.project_id = ? AND e.` + "`key`" + ` = ? AND f.deleted_at IS NULL
ORDER BY f.created_at ASC`

	rows, err := r.db.QueryContext(ctx, q, scope.ProjectID, scope.EnvironmentKey)
	if err != nil {
		return nil, apperror.NewInfrastructureError("DB", "flags list failed", err)
	}
	defer rows.Close()

	var out []entity.Flag
	for rows.Next() {
		var f entity.Flag
		var desc sql.NullString
		var deleted sql.NullTime
		var valueType string
		var defaultJSON []byte
		var rolloutPct sql.NullFloat64
		var rolloutStickiness string

		if err := rows.Scan(
			&f.ID,
			&f.Key,
			&f.Name,
			&desc,
			&f.Enabled,
			&f.ClientSide,
			&valueType,
			&defaultJSON,
			&rolloutPct,
			&rolloutStickiness,
			&f.Version,
			&f.CreatedAt,
			&f.UpdatedAt,
			&deleted,
		); err != nil {
			return nil, apperror.NewInfrastructureError("DB", "flags scan failed", err)
		}
		if desc.Valid {
			f.Description = desc.String
		}
		if deleted.Valid {
			t := deleted.Time
			f.DeletedAt = &t
		}

		f.Value = entity.FlagValue{Type: entity.FlagValueType(valueType)}
		_ = json.Unmarshal(defaultJSON, &f.Value.Default)
		if rolloutPct.Valid {
			f.Rollout = &entity.RolloutConfig{Percentage: rolloutPct.Float64, Stickiness: rolloutStickiness}
		}

		// Rules are optional; load per flag (acceptable for MVP; can be optimized later).
		rules, err := r.listRules(ctx, f.ID)
		if err != nil {
			return nil, err
		}
		f.Rules = rules

		out = append(out, f)
	}
	if err := rows.Err(); err != nil {
		return nil, apperror.NewInfrastructureError("DB", "flags rows failed", err)
	}
	return out, nil
}

func (r *FlagRepository) FindByID(ctx context.Context, scope valueobject.Scope, flagID string) (*entity.Flag, error) {
	const q = `
SELECT f.id, f.` + "`key`" + `, f.name, f.description, f.enabled, f.client_side,
       f.value_type, f.default_value_json, f.rollout_percentage, f.rollout_stickiness,
       f.version, f.created_at, f.updated_at, f.deleted_at
FROM flags f
JOIN environments e ON e.id = f.environment_id
WHERE f.project_id = ? AND e.` + "`key`" + ` = ? AND f.id = ? AND f.deleted_at IS NULL
LIMIT 1`

	var f entity.Flag
	var desc sql.NullString
	var deleted sql.NullTime
	var valueType string
	var defaultJSON []byte
	var rolloutPct sql.NullFloat64
	var rolloutStickiness string

	err := r.db.QueryRowContext(ctx, q, scope.ProjectID, scope.EnvironmentKey, flagID).Scan(
		&f.ID,
		&f.Key,
		&f.Name,
		&desc,
		&f.Enabled,
		&f.ClientSide,
		&valueType,
		&defaultJSON,
		&rolloutPct,
		&rolloutStickiness,
		&f.Version,
		&f.CreatedAt,
		&f.UpdatedAt,
		&deleted,
	)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, apperror.NewNotFoundError("Flag", err)
		}
		return nil, apperror.NewInfrastructureError("DB", "flag get failed", err)
	}
	if desc.Valid {
		f.Description = desc.String
	}
	if deleted.Valid {
		t := deleted.Time
		f.DeletedAt = &t
	}
	f.Value = entity.FlagValue{Type: entity.FlagValueType(valueType)}
	_ = json.Unmarshal(defaultJSON, &f.Value.Default)
	if rolloutPct.Valid {
		f.Rollout = &entity.RolloutConfig{Percentage: rolloutPct.Float64, Stickiness: rolloutStickiness}
	}
	rules, err := r.listRules(ctx, f.ID)
	if err != nil {
		return nil, err
	}
	f.Rules = rules
	return &f, nil
}

func (r *FlagRepository) Create(ctx context.Context, scope valueobject.Scope, f entity.Flag) (entity.Flag, error) {
	envID, err := r.environmentIDByKey(ctx, scope.ProjectID, scope.EnvironmentKey)
	if err != nil {
		return entity.Flag{}, err
	}

	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return entity.Flag{}, apperror.NewInfrastructureError("DB", "begin tx failed", err)
	}
	defer func() { _ = tx.Rollback() }()

	now := time.Now().UTC()
	if f.CreatedAt.IsZero() {
		f.CreatedAt = now
	}
	if f.UpdatedAt.IsZero() {
		f.UpdatedAt = now
	}
	if f.Version == 0 {
		f.Version = 1
	}

	defaultJSON, _ := json.Marshal(f.Value.Default)
	var rolloutPct any = nil
	var stickiness string = "userId"
	if f.Rollout != nil {
		rolloutPct = f.Rollout.Percentage
		if f.Rollout.Stickiness != "" {
			stickiness = f.Rollout.Stickiness
		}
	}

	const q = `
INSERT INTO flags
  (id, project_id, environment_id, ` + "`key`" + `, name, description, enabled, client_side, value_type, default_value_json,
   rollout_percentage, rollout_stickiness, version, created_by, updated_by, created_at, updated_at, deleted_at)
VALUES
  (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, NULL, NULL, ?, ?, NULL)`

	if _, err := tx.ExecContext(ctx, q,
		f.ID,
		scope.ProjectID,
		envID,
		f.Key,
		f.Name,
		f.Description,
		f.Enabled,
		f.ClientSide,
		string(f.Value.Type),
		defaultJSON,
		rolloutPct,
		stickiness,
		f.Version,
		f.CreatedAt,
		f.UpdatedAt,
	); err != nil {
		return entity.Flag{}, apperror.NewInfrastructureError("DB", "flag insert failed", err)
	}

	// Replace rules wholesale for MVP.
	if err := r.replaceRules(ctx, tx, f.ID, f.Rules); err != nil {
		return entity.Flag{}, err
	}

	if err := r.insertFlagVersion(ctx, tx, scope.ProjectID, envID, f, "created"); err != nil {
		return entity.Flag{}, err
	}
	if err := r.insertAuditEntry(ctx, tx, scope.ProjectID, envID, f.ID, "created", nil, mustJSON(f)); err != nil {
		return entity.Flag{}, err
	}
	if err := r.insertOutbox(ctx, tx, scope.ProjectID, envID, f.ID, f.Key, f.Version, "flag.created"); err != nil {
		return entity.Flag{}, err
	}

	if err := tx.Commit(); err != nil {
		return entity.Flag{}, apperror.NewInfrastructureError("DB", "commit failed", err)
	}
	return f, nil
}

func (r *FlagRepository) Update(ctx context.Context, scope valueobject.Scope, f entity.Flag) (entity.Flag, error) {
	envID, err := r.environmentIDByKey(ctx, scope.ProjectID, scope.EnvironmentKey)
	if err != nil {
		return entity.Flag{}, err
	}

	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return entity.Flag{}, apperror.NewInfrastructureError("DB", "begin tx failed", err)
	}
	defer func() { _ = tx.Rollback() }()

	// Read old state for audit/version snapshot (best-effort, inside tx).
	old, _ := r.findByIDTx(ctx, tx, scope.ProjectID, envID, f.ID)

	now := time.Now().UTC()
	f.UpdatedAt = now

	defaultJSON, _ := json.Marshal(f.Value.Default)
	var rolloutPct any = nil
	var stickiness string = "userId"
	if f.Rollout != nil {
		rolloutPct = f.Rollout.Percentage
		if f.Rollout.Stickiness != "" {
			stickiness = f.Rollout.Stickiness
		}
	}

	const q = `
UPDATE flags
SET name = ?, description = ?, enabled = ?, client_side = ?, value_type = ?, default_value_json = ?,
    rollout_percentage = ?, rollout_stickiness = ?, version = version + 1, updated_at = ?
WHERE id = ? AND project_id = ? AND environment_id = ? AND deleted_at IS NULL AND version = ?`

	res, err := tx.ExecContext(ctx, q,
		f.Name,
		f.Description,
		f.Enabled,
		f.ClientSide,
		string(f.Value.Type),
		defaultJSON,
		rolloutPct,
		stickiness,
		f.UpdatedAt,
		f.ID,
		scope.ProjectID,
		envID,
		f.Version,
	)
	if err != nil {
		return entity.Flag{}, apperror.NewInfrastructureError("DB", "flag update failed", err)
	}
	aff, _ := res.RowsAffected()
	if aff == 0 {
		return entity.Flag{}, apperror.New(apperror.StatusConflict, "version_conflict", sql.ErrNoRows)
	}

	if err := r.replaceRules(ctx, tx, f.ID, f.Rules); err != nil {
		return entity.Flag{}, err
	}

	// bump in-memory version after successful DB update
	f.Version = f.Version + 1

	if err := r.insertFlagVersion(ctx, tx, scope.ProjectID, envID, f, "updated"); err != nil {
		return entity.Flag{}, err
	}
	var oldJSON []byte
	if old != nil {
		oldJSON = mustJSON(*old)
	}
	if err := r.insertAuditEntry(ctx, tx, scope.ProjectID, envID, f.ID, "updated", oldJSON, mustJSON(f)); err != nil {
		return entity.Flag{}, err
	}
	if err := r.insertOutbox(ctx, tx, scope.ProjectID, envID, f.ID, f.Key, f.Version, "flag.updated"); err != nil {
		return entity.Flag{}, err
	}

	if err := tx.Commit(); err != nil {
		return entity.Flag{}, apperror.NewInfrastructureError("DB", "commit failed", err)
	}
	return f, nil
}

func (r *FlagRepository) SoftDelete(ctx context.Context, scope valueobject.Scope, flagID string, expectedVersion int) error {
	envID, err := r.environmentIDByKey(ctx, scope.ProjectID, scope.EnvironmentKey)
	if err != nil {
		return err
	}

	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return apperror.NewInfrastructureError("DB", "begin tx failed", err)
	}
	defer func() { _ = tx.Rollback() }()

	old, _ := r.findByIDTx(ctx, tx, scope.ProjectID, envID, flagID)

	now := time.Now().UTC()
	const q = `
UPDATE flags
SET deleted_at = ?, updated_at = ?, version = version + 1
WHERE id = ? AND project_id = ? AND environment_id = ? AND deleted_at IS NULL AND version = ?`

	res, err := tx.ExecContext(ctx, q, now, now, flagID, scope.ProjectID, envID, expectedVersion)
	if err != nil {
		return apperror.NewInfrastructureError("DB", "flag delete failed", err)
	}
	aff, _ := res.RowsAffected()
	if aff == 0 {
		return apperror.New(apperror.StatusConflict, "version_conflict", sql.ErrNoRows)
	}

	// Write audit/outbox/version as best effort; if old is nil, still record delete event.
	if old != nil {
		old.Version = expectedVersion + 1
	}
	if err := r.insertAuditEntry(ctx, tx, scope.ProjectID, envID, flagID, "deleted", func() []byte {
		if old == nil {
			return nil
		}
		return mustJSON(*old)
	}(), nil); err != nil {
		return err
	}
	if err := r.insertOutbox(ctx, tx, scope.ProjectID, envID, flagID, func() string {
		if old != nil {
			return old.Key
		}
		return ""
	}(), expectedVersion+1, "flag.deleted"); err != nil {
		return err
	}

	if err := tx.Commit(); err != nil {
		return apperror.NewInfrastructureError("DB", "commit failed", err)
	}
	return nil
}

func (r *FlagRepository) environmentIDByKey(ctx context.Context, projectID, envKey string) (string, error) {
	const q = `
SELECT id
FROM environments
WHERE project_id = ? AND ` + "`key`" + ` = ? AND deleted_at IS NULL
LIMIT 1`
	var id string
	if err := r.db.QueryRowContext(ctx, q, projectID, envKey).Scan(&id); err != nil {
		if err == sql.ErrNoRows {
			return "", apperror.NewNotFoundError("Environment", err)
		}
		return "", apperror.NewInfrastructureError("DB", "environment lookup failed", err)
	}
	return id, nil
}

func (r *FlagRepository) listRules(ctx context.Context, flagID string) ([]entity.FlagRule, error) {
	const q = `
SELECT id, priority, attribute, operator, comparison_value_json, serve_value_json
FROM flag_rules
WHERE flag_id = ?
ORDER BY priority ASC`

	rows, err := r.db.QueryContext(ctx, q, flagID)
	if err != nil {
		return nil, apperror.NewInfrastructureError("DB", "flag rules list failed", err)
	}
	defer rows.Close()

	var out []entity.FlagRule
	for rows.Next() {
		var rrule entity.FlagRule
		var priority int
		var attr sql.NullString
		var op string
		var cmp []byte
		var serve []byte
		if err := rows.Scan(&rrule.ID, &priority, &attr, &op, &cmp, &serve); err != nil {
			return nil, apperror.NewInfrastructureError("DB", "flag rules scan failed", err)
		}
		rrule.Priority = priority
		if attr.Valid {
			rrule.Attribute = attr.String
		}
		rrule.Operator = entity.FlagRuleOperator(op)
		_ = json.Unmarshal(cmp, &rrule.Value)
		_ = json.Unmarshal(serve, &rrule.Serve)
		out = append(out, rrule)
	}
	if err := rows.Err(); err != nil {
		return nil, apperror.NewInfrastructureError("DB", "flag rules rows failed", err)
	}
	return out, nil
}

type execer interface {
	ExecContext(ctx context.Context, query string, args ...any) (sql.Result, error)
}

func (r *FlagRepository) replaceRules(ctx context.Context, ex execer, flagID string, rules []entity.FlagRule) error {
	if _, err := ex.ExecContext(ctx, `DELETE FROM flag_rules WHERE flag_id = ?`, flagID); err != nil {
		return apperror.NewInfrastructureError("DB", "flag rules delete failed", err)
	}
	if len(rules) == 0 {
		return nil
	}

	const q = `
INSERT INTO flag_rules (id, flag_id, priority, attribute, operator, comparison_value_json, serve_value_json, segment_id, created_at, updated_at)
VALUES (?, ?, ?, ?, ?, ?, ?, NULL, ?, ?)`

	now := time.Now().UTC()
	for _, rule := range rules {
		cmp, _ := json.Marshal(rule.Value)
		sv, _ := json.Marshal(rule.Serve)
		var attr any = nil
		if rule.Attribute != "" {
			attr = rule.Attribute
		}
		if _, err := ex.ExecContext(ctx, q, rule.ID, flagID, rule.Priority, attr, string(rule.Operator), cmp, sv, now, now); err != nil {
			return apperror.NewInfrastructureError("DB", "flag rule insert failed", err)
		}
	}
	return nil
}

func (r *FlagRepository) insertFlagVersion(ctx context.Context, ex execer, projectID, envID string, f entity.Flag, changeType string) error {
	const q = `
INSERT INTO flag_versions (id, flag_id, project_id, environment_id, version, change_type, snapshot_json, created_by, created_at)
VALUES (?, ?, ?, ?, ?, ?, ?, NULL, ?)`
	snap := mustJSON(f)
	_, err := ex.ExecContext(ctx, q, uuid.NewString(), f.ID, projectID, envID, f.Version, changeType, snap, time.Now().UTC())
	if err != nil {
		return apperror.NewInfrastructureError("DB", "flag version insert failed", err)
	}
	return nil
}

func (r *FlagRepository) insertAuditEntry(ctx context.Context, ex execer, projectID, envID, entityID, action string, oldJSON, newJSON []byte) error {
	const q = `
INSERT INTO audit_entries (id, project_id, environment_id, entity_type, entity_id, action, actor_id, correlation_id, old_value_json, new_value_json, created_at)
VALUES (?, ?, ?, 'flag', ?, ?, NULL, ?, ?, ?, ?)`

	cid, _ := ctx.Value(requestctx.CorrelationIDKey).(string)
	var old any = nil
	var newv any = nil
	if len(oldJSON) > 0 {
		old = oldJSON
	}
	if len(newJSON) > 0 {
		newv = newJSON
	}

	_, err := ex.ExecContext(ctx, q, uuid.NewString(), projectID, envID, entityID, action, cid, old, newv, time.Now().UTC())
	if err != nil {
		return apperror.NewInfrastructureError("DB", "audit insert failed", err)
	}
	return nil
}

func (r *FlagRepository) insertOutbox(ctx context.Context, ex execer, projectID, envID, flagID, flagKey string, version int, eventType string) error {
	const q = `
INSERT INTO outbox_events
  (id, aggregate_type, aggregate_id, aggregate_version, scope_version, project_id, environment_id, event_type, payload_json, status, retry_count, next_attempt_at, created_at)
VALUES
  (?, 'flag', ?, ?, 0, ?, ?, ?, ?, 'pending', 0, NULL, ?)`

	payload := map[string]any{
		"projectId":      projectID,
		"environmentId":  envID,
		"flagId":         flagID,
		"flagKey":        flagKey,
		"version":        version,
		"eventType":      eventType,
		"correlationId":  func() string { v, _ := ctx.Value(requestctx.CorrelationIDKey).(string); return v }(),
		"occurredAt":     time.Now().UTC().Format(time.RFC3339Nano),
	}
	b, _ := json.Marshal(payload)
	_, err := ex.ExecContext(ctx, q, uuid.NewString(), flagID, version, projectID, envID, eventType, b, time.Now().UTC())
	if err != nil {
		return apperror.NewInfrastructureError("DB", "outbox insert failed", err)
	}
	return nil
}

func mustJSON(v any) []byte {
	b, err := json.Marshal(v)
	if err != nil {
		// This should never happen for our domain structs.
		panic(err)
	}
	return b
}

func (r *FlagRepository) findByIDTx(ctx context.Context, tx *sql.Tx, projectID, envID, flagID string) (*entity.Flag, error) {
	const q = `
SELECT id, ` + "`key`" + `, name, description, enabled, client_side, value_type, default_value_json,
       rollout_percentage, rollout_stickiness, version, created_at, updated_at, deleted_at
FROM flags
WHERE id = ? AND project_id = ? AND environment_id = ?`

	var f entity.Flag
	var desc sql.NullString
	var deleted sql.NullTime
	var valueType string
	var defaultJSON []byte
	var rolloutPct sql.NullFloat64
	var rolloutStickiness string

	err := tx.QueryRowContext(ctx, q, flagID, projectID, envID).Scan(
		&f.ID,
		&f.Key,
		&f.Name,
		&desc,
		&f.Enabled,
		&f.ClientSide,
		&valueType,
		&defaultJSON,
		&rolloutPct,
		&rolloutStickiness,
		&f.Version,
		&f.CreatedAt,
		&f.UpdatedAt,
		&deleted,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, err
		}
		return nil, err
	}
	if desc.Valid {
		f.Description = desc.String
	}
	if deleted.Valid {
		t := deleted.Time
		f.DeletedAt = &t
	}
	f.Value = entity.FlagValue{Type: entity.FlagValueType(valueType)}
	_ = json.Unmarshal(defaultJSON, &f.Value.Default)
	if rolloutPct.Valid {
		f.Rollout = &entity.RolloutConfig{Percentage: rolloutPct.Float64, Stickiness: rolloutStickiness}
	}
	// load rules in tx not necessary for audit snapshot in MVP
	return &f, nil
}
