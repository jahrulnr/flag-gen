//go:build wireinject

package di

import "github.com/google/wire"

func InitializeApplication() (*ApplicationState, error) {
	wire.Build(
		ProvideConfig,
		ProvideMySQLDB,
		ProvideProjectRepository,
		ProvideEnvironmentRepository,
		ProvideSegmentRepository,
		ProvideIdempotencyRepository,
		ProvideFlagRepository,
		ProvideOutboxRepository,
		ProvideSnapshotMetadataRepository,
		ProvideAdminReadRepository,
		ProvideBroadcastHub,
		ProvideRistrettoCache,
		ProvideL1,
		ProvideRedisUniversalClient,
		ProvideL2,
		ProvideSingleflight,
		ProvideSnapshotReader,
		ProvideSnapshotWriter,
		ProvideSnapshotPointer,
		ProvideDegradedBreaker,
		ProvideNegativeFlagCache,
		ProvideScopedFlagCache,
		ProvideAMQPPublisher,
		ProvideOutboxDispatcher,
		ProvideWebhookWorker,
		ProvideSystemService,
		ProvideApplicationState,
	)
	return &ApplicationState{}, nil
}
