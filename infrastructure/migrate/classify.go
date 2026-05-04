package migrate

import (
	"context"
	"database/sql"
	"fmt"
)

// DatabaseState matches docs/assessment/api-contract.md and database-lifecycle.md.
type DatabaseState string

const (
	StateDBUnreachable     DatabaseState = "db_unreachable"
	StateDBEmpty           DatabaseState = "db_empty"
	StateDBReady           DatabaseState = "db_ready"
	StateMigrationOptional DatabaseState = "migration_optional"
	StateMigrationRequired DatabaseState = "migration_required"
	StateDBAhead           DatabaseState = "db_ahead"
	StateDBInvalid         DatabaseState = "db_invalid"
)

// Classification is the full database lifecycle snapshot for API responses.
type Classification struct {
	State DatabaseState

	Connected bool
	Empty     bool

	CurrentSchemaVersion int64
	TargetSchemaVersion  int64
	MinCompatible        int64
	BlockingBelow        int64

	Dirty bool

	CanInstall bool
	CanMigrate bool

	RuntimeReadCompatible bool
	Message               string
}

func tableExists(ctx context.Context, db *sql.DB, table string) (bool, error) {
	var n int
	err := db.QueryRowContext(ctx, `
SELECT COUNT(*) FROM information_schema.tables
WHERE table_schema = DATABASE() AND table_name = ?`, table).Scan(&n)
	return n > 0, err
}

// Classify inspects the database and returns lifecycle classification.
func Classify(ctx context.Context, db *sql.DB, migrationsDir string) (Classification, error) {
	out := Classification{
		TargetSchemaVersion:   SchemaTargetVersion,
		MinCompatible:         SchemaMinCompatibleVersion,
		BlockingBelow:         SchemaBlockingBelowVersion,
		RuntimeReadCompatible: true,
	}
	if err := db.PingContext(ctx); err != nil {
		out.State = StateDBUnreachable
		out.Message = "Cannot connect to MySQL."
		return out, nil
	}
	out.Connected = true

	hasSM, err := tableExists(ctx, db, "schema_migrations")
	if err != nil {
		return Classification{}, err
	}
	hasUsers, err := tableExists(ctx, db, "users")
	if err != nil {
		return Classification{}, err
	}

	if !hasSM {
		if !hasUsers {
			out.State = StateDBEmpty
			out.Empty = true
			out.CanInstall = true
			out.Message = "Database is empty; run install to bootstrap schema."
			return out, nil
		}
		out.State = StateDBInvalid
		out.Message = "Schema metadata missing but other tables exist."
		return out, nil
	}

	m := NewMigrator(db, migrationsDir)
	dirty, err := m.HasDirtyMigration(ctx)
	if err != nil {
		return Classification{}, err
	}
	if dirty {
		out.State = StateDBInvalid
		out.Dirty = true
		out.Message = "A migration failed previously (dirty state)."
		return out, nil
	}

	if err := m.ValidateAppliedChecksums(ctx); err != nil {
		out.State = StateDBInvalid
		out.Message = fmt.Sprintf("Migration checksum validation failed: %v", err)
		return out, nil
	}

	maxV, err := m.MaxAppliedVersion(ctx)
	if err != nil {
		return Classification{}, err
	}
	out.CurrentSchemaVersion = maxV

	if maxV == 0 && !hasUsers {
		out.State = StateDBEmpty
		out.Empty = true
		out.CanInstall = true
		out.Message = "No migrations applied; database appears empty."
		return out, nil
	}
	if maxV == 0 && hasUsers {
		out.State = StateDBInvalid
		out.Message = "schema_migrations has no applied versions but users table exists."
		return out, nil
	}

	if maxV > SchemaTargetVersion {
		out.State = StateDBAhead
		out.CanMigrate = false
		out.RuntimeReadCompatible = false
		out.Message = "Database schema is newer than this application binary."
		return out, nil
	}

	pending, err := m.Pending(ctx)
	if err != nil {
		return Classification{}, err
	}
	if len(pending) == 0 {
		out.State = StateDBReady
		out.CanMigrate = false
		out.Message = "Database schema is up to date."
		return out, nil
	}

	out.CanMigrate = true

	// If applied schema is strictly below the blocking threshold, migrations are required.
	if maxV < SchemaBlockingBelowVersion {
		out.State = StateMigrationRequired
		out.RuntimeReadCompatible = maxV >= SchemaMinCompatibleVersion
		out.Message = "Database migration is required before dashboard management can continue."
		return out, nil
	}

	out.State = StateMigrationOptional
	out.RuntimeReadCompatible = true
	out.Message = "Optional migrations are available; recommended to migrate."
	return out, nil
}
