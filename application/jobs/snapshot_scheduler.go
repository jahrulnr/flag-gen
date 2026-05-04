package jobs

import (
	"context"
	"log"
	"time"

	"flag-gen/application/usecase"
	"flag-gen/domain/repository"
	"flag-gen/infrastructure/cache"
	"flag-gen/infrastructure/config"

	"github.com/redis/go-redis/v9"
)

// StartSnapshotConsistencyScheduler periodically reconciles snapshot files with DB state for scopes
// that already have snapshot_metadata. Uses local singleflight per scope and a Redis lock across replicas.
func StartSnapshotConsistencyScheduler(
	ctx context.Context,
	cfg config.Config,
	rdb redis.UniversalClient,
	meta repository.SnapshotMetadataRepository,
	ptr *usecase.SnapshotPointer,
	sf *cache.Singleflight,
) {
	if !cfg.Snapshot.SchedulerEnabled || ptr == nil || meta == nil || sf == nil {
		return
	}
	interval := cfg.Snapshot.SchedulerInterval
	if interval <= 0 {
		interval = 5 * time.Minute
	}
	lockTTL := cfg.Snapshot.SchedulerLockTTL
	if lockTTL <= 0 {
		lockTTL = 2 * time.Minute
	}
	go func() {
		t := time.NewTicker(interval)
		defer t.Stop()
		for {
			runSnapshotConsistencyTick(context.Background(), rdb, lockTTL, meta, ptr, sf)
			select {
			case <-ctx.Done():
				return
			case <-t.C:
			}
		}
	}()
}

func runSnapshotConsistencyTick(
	parent context.Context,
	rdb redis.UniversalClient,
	lockTTL time.Duration,
	meta repository.SnapshotMetadataRepository,
	ptr *usecase.SnapshotPointer,
	sf *cache.Singleflight,
) {
	ctx, cancel := context.WithTimeout(parent, 5*time.Minute)
	defer cancel()

	if rdb == nil {
		log.Print("snapshot scheduler: redis client nil, skip tick")
		return
	}
	if err := rdb.Ping(ctx).Err(); err != nil {
		log.Printf("snapshot scheduler: redis unavailable, skip tick: %v", err)
		return
	}

	scopes, err := meta.ListScopes(ctx)
	if err != nil {
		log.Printf("snapshot scheduler: list scopes failed: %v", err)
		return
	}

	for _, scope := range scopes {
		sc := scope
		sk := cache.ScopeKey(sc)
		_, _ = sf.Do("snapshot-sched:"+sk, func() (any, error) {
			token := cache.RandomLockToken()
			lockKey := "flag-gen:lock:snapshot-check:" + sk
			ok, err := cache.TryAcquireRedisLock(ctx, rdb, lockKey, token, lockTTL)
			if err != nil {
				log.Printf("snapshot scheduler: lock error scope=%s: %v", sk, err)
				return nil, nil
			}
			if !ok {
				return nil, nil
			}
			defer func() {
				if relErr := cache.ReleaseRedisLock(context.Background(), rdb, lockKey, token); relErr != nil {
					log.Printf("snapshot scheduler: unlock %s: %v", lockKey, relErr)
				}
			}()

			c2, c2cancel := context.WithTimeout(ctx, 2*time.Minute)
			defer c2cancel()
			if err := ptr.EnsureConsistent(c2, sc); err != nil {
				log.Printf("snapshot scheduler: reconcile %s: %v", sk, err)
			}
			return nil, nil
		})
	}
}
