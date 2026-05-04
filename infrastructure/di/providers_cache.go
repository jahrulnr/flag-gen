package di

import (
	"flag-gen/domain/entity"
	"flag-gen/domain/repository"
	"flag-gen/infrastructure/cache"
	"flag-gen/infrastructure/config"
	"flag-gen/infrastructure/snapshot"
	"time"

	"github.com/dgraph-io/ristretto/v2"
	"github.com/redis/go-redis/v9"
)

func ProvideRistrettoCache() (*ristretto.Cache[string, []entity.Flag], error) {
	return ristretto.NewCache(&ristretto.Config[string, []entity.Flag]{
		NumCounters: 1e5,
		MaxCost:     1 << 20, // ~1MB cost units for MVP
		BufferItems: 64,
	})
}

func ProvideL1(cacheImpl *ristretto.Cache[string, []entity.Flag]) cache.L1 {
	return cache.NewRistrettoL1(cacheImpl)
}

func ProvideRedisUniversalClient(cfg config.Config) redis.UniversalClient {
	if cfg.Redis.ForceCluster {
		addrs := cfg.Redis.Addrs
		if len(addrs) == 0 && cfg.Redis.Addr != "" {
			addrs = []string{cfg.Redis.Addr}
		}
		return redis.NewClusterClient(&redis.ClusterOptions{
			Addrs:    addrs,
			Username: cfg.Redis.Username,
			Password: cfg.Redis.Password,
		})
	}

	return redis.NewClient(&redis.Options{
		Addr:     cfg.Redis.Addr,
		Username: cfg.Redis.Username,
		Password: cfg.Redis.Password,
		DB:       cfg.Redis.DB,
	})
}

func ProvideL2(c redis.UniversalClient) cache.L2 {
	return cache.NewRedisL2(c)
}

func ProvideSingleflight() *cache.Singleflight {
	return cache.NewSingleflight()
}

func ProvideSnapshotReader(cfg config.Config) *snapshot.Reader {
	return snapshot.NewReader(cfg.Snapshot.Dir)
}

func ProvideDegradedBreaker() *cache.DegradedBreaker {
	return cache.NewDegradedBreaker(5, 45*time.Second)
}

func ProvideNegativeFlagCache() (*cache.NegativeFlagCache, error) {
	return cache.NewNegativeFlagCache()
}

func ProvideScopedFlagCache(
	l1 cache.L1,
	l2 cache.L2,
	sf *cache.Singleflight,
	flags repository.FlagRepository,
	snapReader *snapshot.Reader,
	snapMeta repository.SnapshotMetadataRepository,
	br *cache.DegradedBreaker,
	neg *cache.NegativeFlagCache,
) *cache.ScopedFlagCache {
	return cache.NewScopedFlagCache(l1, l2, sf, flags, snapReader, snapMeta, br, neg)
}
