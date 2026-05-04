package repository

import (
	"context"

	"flag-gen/domain/entity"
	"flag-gen/domain/valueobject"
)

// SnapshotMetadataRepository tracks snapshot file metadata (the file body itself is not stored in DB).
type SnapshotMetadataRepository interface {
	GetByScope(ctx context.Context, scope valueobject.Scope) (*entity.SnapshotMetadata, error)
	Upsert(ctx context.Context, meta entity.SnapshotMetadata) error
	MarkInvalidByScope(ctx context.Context, scope valueobject.Scope) error
	// ListScopes returns distinct scopes that have snapshot_metadata rows (for consistency jobs).
	ListScopes(ctx context.Context) ([]valueobject.Scope, error)
}
