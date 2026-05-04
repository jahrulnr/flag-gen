package di

import (
	"flag-gen/application/usecase"
	"flag-gen/domain/repository"
	"flag-gen/infrastructure/broadcast"
	"flag-gen/infrastructure/broker"
	"flag-gen/infrastructure/cache"
	"flag-gen/infrastructure/config"
	"flag-gen/infrastructure/outbox"
	mysqlrepo "flag-gen/infrastructure/repository/mysql"

	"github.com/redis/go-redis/v9"
)

// ApplicationState is the composition root output used by cmd/app.
// It contains already-wired ports and infrastructure primitives.
type ApplicationState struct {
	Config config.Config
	MySQL  *mysqlrepo.DB
	Redis  redis.UniversalClient

	Projects     repository.ProjectRepository
	Environments repository.EnvironmentRepository
	Segments     repository.SegmentRepository
	Idempotency  repository.IdempotencyRepository

	Flags     repository.FlagRepository
	Outbox    repository.OutboxRepository
	Snapshots repository.SnapshotMetadataRepository

	ScopedFlags  *cache.ScopedFlagCache
	Singleflight *cache.Singleflight

	AMQPPublisher    *broker.AMQPPublisher
	OutboxDispatcher *outbox.Dispatcher

	System *usecase.SystemService

	SnapshotPointer *usecase.SnapshotPointer

	AdminRead    repository.AdminReadRepository
	BroadcastHub *broadcast.Hub
}
