package di

import (
	"flag-gen/domain/repository"
	"flag-gen/infrastructure/broker"
	"flag-gen/infrastructure/cache"
	"flag-gen/infrastructure/config"
	"flag-gen/infrastructure/outbox"
)

// ApplicationState is the composition root output used by cmd/app.
// It contains already-wired ports and infrastructure primitives.
type ApplicationState struct {
	Config config.Config

	Projects     repository.ProjectRepository
	Environments repository.EnvironmentRepository
	Segments     repository.SegmentRepository
	Idempotency  repository.IdempotencyRepository

	Flags     repository.FlagRepository
	Outbox    repository.OutboxRepository
	Snapshots repository.SnapshotMetadataRepository

	ScopedFlags *cache.ScopedFlagCache

	AMQPPublisher   *broker.AMQPPublisher
	OutboxDispatcher *outbox.Dispatcher
}
