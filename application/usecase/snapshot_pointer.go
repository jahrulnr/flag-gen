package usecase

import (
	"context"
	"errors"
	"time"

	"flag-gen/domain/entity"
	"flag-gen/domain/repository"
	"flag-gen/domain/valueobject"
	"flag-gen/infrastructure/cache"
	"flag-gen/infrastructure/snapshot"
	apperror "flag-gen/pkg/error"

	"github.com/google/uuid"
)

// SnapshotPointer refreshes file snapshots and invalidates scoped caches after flag mutations.
type SnapshotPointer struct {
	flags  repository.FlagRepository
	envs   repository.EnvironmentRepository
	meta   repository.SnapshotMetadataRepository
	writer *snapshot.Writer
	scoped *cache.ScopedFlagCache
}

func NewSnapshotPointer(
	flags repository.FlagRepository,
	envs repository.EnvironmentRepository,
	meta repository.SnapshotMetadataRepository,
	writer *snapshot.Writer,
	scoped *cache.ScopedFlagCache,
) *SnapshotPointer {
	return &SnapshotPointer{
		flags:  flags,
		envs:   envs,
		meta:   meta,
		writer: writer,
		scoped: scoped,
	}
}

// AfterFlagMutation invalidates hot cache synchronously and schedules snapshot refresh asynchronously.
func (p *SnapshotPointer) AfterFlagMutation(ctx context.Context, scope valueobject.Scope) {
	if p == nil {
		return
	}
	if p.scoped != nil {
		p.scoped.InvalidateScope(ctx, scope)
	}
	if p.writer == nil {
		return
	}
	sc := scope
	go func() {
		c2, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
		defer cancel()
		_ = p.refresh(c2, sc)
	}()
}

// EnsureConsistent regenerates the on-disk snapshot when DB flag versions are ahead of snapshot_metadata
// or when metadata is marked invalid. No-op when writer is nil or metadata row is missing.
func (p *SnapshotPointer) EnsureConsistent(ctx context.Context, scope valueobject.Scope) error {
	if p == nil || p.writer == nil {
		return nil
	}
	meta, err := p.meta.GetByScope(ctx, scope)
	if err != nil {
		var appErr *apperror.AppError
		if errors.As(err, &appErr) && appErr.Code == apperror.StatusNotFound {
			return nil
		}
		return err
	}
	flags, err := p.flags.ListByScope(ctx, scope)
	if err != nil {
		return err
	}
	sv := 1
	for _, f := range flags {
		if f.Version > sv {
			sv = f.Version
		}
	}
	needs := sv > meta.ScopeVersion || meta.Status != entity.SnapshotStatusValid
	if !needs {
		return nil
	}
	return p.refresh(ctx, scope)
}

func (p *SnapshotPointer) refresh(ctx context.Context, scope valueobject.Scope) error {
	env, err := p.envs.FindByProjectAndKey(ctx, scope.ProjectID, scope.EnvironmentKey)
	if err != nil {
		return err
	}
	flags, err := p.flags.ListByScope(ctx, scope)
	if err != nil {
		return err
	}
	sv := 1
	for _, f := range flags {
		if f.Version > sv {
			sv = f.Version
		}
	}
	path, hex, err := p.writer.WriteAtomic(scope.ProjectID, scope.EnvironmentKey, sv, flags)
	if err != nil {
		return err
	}
	now := time.Now().UTC()
	meta := entity.SnapshotMetadata{
		ID:             uuid.NewString(),
		ProjectID:      scope.ProjectID,
		EnvironmentID:  env.ID,
		EnvironmentKey: scope.EnvironmentKey,
		ScopeVersion:   sv,
		Checksum:       hex,
		FilePath:       path,
		GeneratedAt:    now,
		Status:         entity.SnapshotStatusValid,
	}
	return p.meta.Upsert(ctx, meta)
}
