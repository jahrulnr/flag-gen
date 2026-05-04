package migrate

import (
	"context"
	"database/sql"
	"fmt"
	"time"
)

// Migrator applies versioned SQL migrations and records them in schema_migrations.
type Migrator struct {
	db  *sql.DB
	dir string
}

func NewMigrator(db *sql.DB, migrationsDir string) *Migrator {
	return &Migrator{db: db, dir: migrationsDir}
}

// Pending returns migrations not yet recorded in schema_migrations (same version).
func (m *Migrator) Pending(ctx context.Context) ([]File, error) {
	all, err := DiscoverUpMigrations(m.dir)
	if err != nil {
		return nil, err
	}
	applied, err := m.appliedVersions(ctx)
	if err != nil {
		return nil, err
	}
	var pending []File
	for _, f := range all {
		if _, ok := applied[f.Version]; !ok {
			pending = append(pending, f)
		}
	}
	return pending, nil
}

func (m *Migrator) appliedVersions(ctx context.Context) (map[int64]struct{}, error) {
	out := make(map[int64]struct{})
	// Table may not exist yet on empty DB.
	var n int
	err := m.db.QueryRowContext(ctx, `
SELECT COUNT(*) FROM information_schema.tables
WHERE table_schema = DATABASE() AND table_name = 'schema_migrations'`).Scan(&n)
	if err != nil {
		return nil, err
	}
	if n == 0 {
		return out, nil
	}
	rows, err := m.db.QueryContext(ctx, `SELECT version FROM schema_migrations WHERE dirty = FALSE`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	for rows.Next() {
		var v int64
		if err := rows.Scan(&v); err != nil {
			return nil, err
		}
		out[v] = struct{}{}
	}
	return out, rows.Err()
}

// ApplyOne applies a single migration file in order (multiple statements) and records it.
func (m *Migrator) ApplyOne(ctx context.Context, appVersion string, f File) error {
	stmts := splitSQLStatements(f.SQL)
	if len(stmts) == 0 {
		return fmt.Errorf("migration %d is empty", f.Version)
	}
	for _, stmt := range stmts {
		if _, err := m.db.ExecContext(ctx, stmt); err != nil {
			return fmt.Errorf("migration %d (%s) failed: %w", f.Version, f.Name, err)
		}
	}
	start := time.Now().UTC()
	_, err := m.db.ExecContext(ctx, `
INSERT INTO schema_migrations (version, name, checksum, execution_ms, applied_by_app_version, applied_at, dirty)
VALUES (?, ?, ?, ?, ?, ?, FALSE)`,
		f.Version, f.Name, f.SHA256, int(time.Since(start).Milliseconds()), appVersion, time.Now().UTC())
	if err != nil {
		return fmt.Errorf("record migration %d: %w", f.Version, err)
	}
	return nil
}

// ApplyAllPending applies every pending migration in ascending version order.
func (m *Migrator) ApplyAllPending(ctx context.Context, appVersion string) ([]File, error) {
	pending, err := m.Pending(ctx)
	if err != nil {
		return nil, err
	}
	var applied []File
	for _, f := range pending {
		if err := m.ApplyOne(ctx, appVersion, f); err != nil {
			return applied, err
		}
		applied = append(applied, f)
	}
	return applied, nil
}

// MaxAppliedVersion returns the highest applied migration version, or 0 if none / table missing.
func (m *Migrator) MaxAppliedVersion(ctx context.Context) (int64, error) {
	var n int
	err := m.db.QueryRowContext(ctx, `
SELECT COUNT(*) FROM information_schema.tables
WHERE table_schema = DATABASE() AND table_name = 'schema_migrations'`).Scan(&n)
	if err != nil {
		return 0, err
	}
	if n == 0 {
		return 0, nil
	}
	var max sql.NullInt64
	err = m.db.QueryRowContext(ctx, `SELECT MAX(version) FROM schema_migrations WHERE dirty = FALSE`).Scan(&max)
	if err != nil {
		return 0, err
	}
	if !max.Valid {
		return 0, nil
	}
	return max.Int64, nil
}

// HasDirtyMigration reports whether any migration row is marked dirty.
func (m *Migrator) HasDirtyMigration(ctx context.Context) (bool, error) {
	var n int
	err := m.db.QueryRowContext(ctx, `
SELECT COUNT(*) FROM information_schema.tables
WHERE table_schema = DATABASE() AND table_name = 'schema_migrations'`).Scan(&n)
	if err != nil {
		return false, err
	}
	if n == 0 {
		return false, nil
	}
	var cnt int
	err = m.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM schema_migrations WHERE dirty = TRUE`).Scan(&cnt)
	if err != nil {
		return false, err
	}
	return cnt > 0, nil
}

// ValidateAppliedChecksums ensures on-disk migration checksums match recorded rows.
func (m *Migrator) ValidateAppliedChecksums(ctx context.Context) error {
	all, err := DiscoverUpMigrations(m.dir)
	if err != nil {
		return err
	}
	for _, f := range all {
		var checksum string
		err := m.db.QueryRowContext(ctx, `SELECT checksum FROM schema_migrations WHERE version = ? AND dirty = FALSE`, f.Version).Scan(&checksum)
		if err == sql.ErrNoRows {
			continue
		}
		if err != nil {
			return err
		}
		if checksum != f.SHA256 {
			return fmt.Errorf("checksum mismatch for migration %d: db=%s disk=%s", f.Version, checksum, f.SHA256)
		}
	}
	return nil
}
