package entity

import "time"

type SnapshotStatus string

const (
	SnapshotStatusValid   SnapshotStatus = "valid"
	SnapshotStatusInvalid SnapshotStatus = "invalid"
)

type SnapshotMetadata struct {
	ID string

	ProjectID      string
	EnvironmentID  string
	EnvironmentKey string

	ScopeVersion int
	Checksum     string
	FilePath     string

	SourceOutboxEventID *string

	GeneratedAt     time.Time
	LastValidatedAt *time.Time
	Status          SnapshotStatus
}
