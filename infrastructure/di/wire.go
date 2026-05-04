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
		ProvideRistrettoCache,
		ProvideL1,
		ProvideRedisUniversalClient,
		ProvideL2,
		ProvideSingleflight,
		ProvideSnapshotReader,
		ProvideScopedFlagCache,
		ProvideAMQPPublisher,
		ProvideOutboxDispatcher,
		ProvideApplicationState,
	)
	return &ApplicationState{}, nil
}
