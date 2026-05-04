package di

import (
	"flag-gen/application/usecase"
	"flag-gen/domain/repository"
	"flag-gen/infrastructure/cache"
	"flag-gen/infrastructure/config"
	"flag-gen/infrastructure/snapshot"
)

func ProvideSnapshotWriter(cfg config.Config) *snapshot.Writer {
	return snapshot.NewWriter(cfg.Snapshot.Dir)
}

func ProvideSnapshotPointer(
	flags repository.FlagRepository,
	envs repository.EnvironmentRepository,
	meta repository.SnapshotMetadataRepository,
	writer *snapshot.Writer,
	scoped *cache.ScopedFlagCache,
) *usecase.SnapshotPointer {
	return usecase.NewSnapshotPointer(flags, envs, meta, writer, scoped)
}
