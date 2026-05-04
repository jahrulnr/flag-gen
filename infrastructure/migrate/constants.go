package migrate

// Application and schema compatibility constants (see docs/assessment/database-lifecycle.md).
const (
	AppVersion = "0.1.0"

	// SchemaTargetVersion is the latest migration version shipped with this binary.
	SchemaTargetVersion int64 = 202604280001

	// SchemaMinCompatibleVersion is the oldest schema this binary can safely run against.
	SchemaMinCompatibleVersion int64 = 202604280001

	// SchemaBlockingBelowVersion: DB schema version strictly below this requires migration before full use.
	SchemaBlockingBelowVersion int64 = 202604280001
)
