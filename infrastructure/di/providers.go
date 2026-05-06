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
	"flag-gen/infrastructure/webhook"
	"time"

	"github.com/redis/go-redis/v9"
)

func ProvideConfig() (config.Config, error) {
	return config.Load()
}

func ProvideMySQLDB(cfg config.Config) (*mysqlrepo.DB, error) {
	return mysqlrepo.NewDB(cfg.MySQL.DSN)
}

func ProvideSystemService(db *mysqlrepo.DB, cfg config.Config) *usecase.SystemService {
	return usecase.NewSystemService(db.SQL(), cfg)
}

func ProvideFlagRepository(db *mysqlrepo.DB) repository.FlagRepository {
	return mysqlrepo.NewFlagRepository(db)
}

func ProvideProjectRepository(db *mysqlrepo.DB) repository.ProjectRepository {
	return mysqlrepo.NewProjectRepository(db)
}

func ProvideEnvironmentRepository(db *mysqlrepo.DB) repository.EnvironmentRepository {
	return mysqlrepo.NewEnvironmentRepository(db)
}

func ProvideSegmentRepository(db *mysqlrepo.DB) repository.SegmentRepository {
	return mysqlrepo.NewSegmentRepository(db)
}

func ProvideIdempotencyRepository(db *mysqlrepo.DB) repository.IdempotencyRepository {
	return mysqlrepo.NewIdempotencyRepository(db)
}

func ProvideOutboxRepository(db *mysqlrepo.DB) repository.OutboxRepository {
	return mysqlrepo.NewOutboxRepository(db)
}

func ProvideSnapshotMetadataRepository(db *mysqlrepo.DB) repository.SnapshotMetadataRepository {
	return mysqlrepo.NewSnapshotMetadataRepository(db)
}

func ProvideAMQPPublisher(cfg config.Config) *broker.AMQPPublisher {
	return broker.NewAMQPPublisher(cfg.RabbitMQ.AMQPURL, cfg.RabbitMQ.ExchangeName)
}

func ProvideOutboxDispatcher(cfg config.Config, outboxRepo repository.OutboxRepository, pub *broker.AMQPPublisher) *outbox.Dispatcher {
	interval := time.Duration(cfg.Outbox.IntervalMs) * time.Millisecond
	return outbox.NewDispatcher(
		"", // workerID default
		cfg.Outbox.BatchSize,
		interval,
		cfg.Outbox.LockFor,
		cfg.RabbitMQ.ExchangeName,
		cfg.RabbitMQ.PublishTimeout,
		outboxRepo,
		pub,
	)
}

func ProvideWebhookWorker(cfg config.Config) *webhook.Worker {
	return webhook.NewWorker(cfg.Webhook)
}

func ProvideApplicationState(
	cfg config.Config,
	db *mysqlrepo.DB,
	redisClient redis.UniversalClient,
	projects repository.ProjectRepository,
	envs repository.EnvironmentRepository,
	segments repository.SegmentRepository,
	idem repository.IdempotencyRepository,
	flags repository.FlagRepository,
	outbox repository.OutboxRepository,
	snapshots repository.SnapshotMetadataRepository,
	scopedFlags *cache.ScopedFlagCache,
	pub *broker.AMQPPublisher,
	dispatcher *outbox.Dispatcher,
	webhooks *webhook.Worker,
	system *usecase.SystemService,
	snapPtr *usecase.SnapshotPointer,
	admin repository.AdminReadRepository,
	hub *broadcast.Hub,
	sf *cache.Singleflight,
) *ApplicationState {
	return &ApplicationState{
		Config:           cfg,
		MySQL:            db,
		Redis:            redisClient,
		Projects:         projects,
		Environments:     envs,
		Segments:         segments,
		Idempotency:      idem,
		Flags:            flags,
		Outbox:           outbox,
		Snapshots:        snapshots,
		ScopedFlags:      scopedFlags,
		Singleflight:     sf,
		AMQPPublisher:    pub,
		OutboxDispatcher: dispatcher,
		WebhookWorker:    webhooks,
		System:           system,
		SnapshotPointer:  snapPtr,
		AdminRead:        admin,
		BroadcastHub:     hub,
	}
}
