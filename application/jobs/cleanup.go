package jobs

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"strconv"
	"time"

	"flag-gen/infrastructure/cache"

	"github.com/redis/go-redis/v9"
)

const softDeleteCleanupLockKey = "flag-gen:lock:cleanup:global"

// StartSoftDeleteCleanup periodically hard-deletes soft-deleted flags past retention, optionally prunes
// version/audit history per scope, and uses a Redis lock so only one replica runs a tick at a time.
func StartSoftDeleteCleanup(ctx context.Context, db *sql.DB, rdb redis.UniversalClient, interval, retention, lockTTL time.Duration, pruneMaxPerScope int) {
	if interval <= 0 || retention <= 0 {
		return
	}
	if lockTTL <= 0 {
		lockTTL = 10 * time.Minute
	}
	t := time.NewTicker(interval)
	defer t.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-t.C:
			runCleanupTick(context.Background(), db, rdb, retention, lockTTL, pruneMaxPerScope)
		}
	}
}

func runCleanupTick(ctx context.Context, db *sql.DB, rdb redis.UniversalClient, retention, lockTTL time.Duration, pruneMaxPerScope int) {
	cctx, cancel := context.WithTimeout(ctx, 10*time.Minute)
	defer cancel()

	if rdb != nil {
		if err := rdb.Ping(cctx).Err(); err != nil {
			log.Printf("cleanup: redis unavailable, skip tick: %v", err)
			return
		}
		token := cache.RandomLockToken()
		ok, err := cache.TryAcquireRedisLock(cctx, rdb, softDeleteCleanupLockKey, token, lockTTL)
		if err != nil {
			log.Printf("cleanup: lock error: %v", err)
			return
		}
		if !ok {
			return
		}
		defer func() {
			if relErr := cache.ReleaseRedisLock(context.Background(), rdb, softDeleteCleanupLockKey, token); relErr != nil {
				log.Printf("cleanup: unlock failed: %v", relErr)
			}
		}()
	}
	// When rdb is nil (e.g. local wiring without Redis), run retention/pruning without a distributed lock.

	sec := int(retention.Seconds())
	if sec < 1 {
		sec = 1
	}
	q := fmt.Sprintf(`
DELETE FROM flags
WHERE deleted_at IS NOT NULL AND deleted_at < DATE_SUB(UTC_TIMESTAMP(3), INTERVAL %s SECOND)`, strconv.Itoa(sec))
	if _, err := db.ExecContext(cctx, q); err != nil {
		log.Printf("cleanup: soft-delete purge failed: %v", err)
	}

	if pruneMaxPerScope > 0 {
		pruneExcessVersions(cctx, db, pruneMaxPerScope)
		pruneExcessAudit(cctx, db, pruneMaxPerScope)
	}
}

func pruneExcessVersions(ctx context.Context, db *sql.DB, keep int) {
	q := fmt.Sprintf(`
DELETE fv FROM flag_versions fv
INNER JOIN (
  SELECT id FROM (
    SELECT id, ROW_NUMBER() OVER (PARTITION BY project_id, environment_id ORDER BY created_at DESC) AS rn
    FROM flag_versions
  ) AS ranked WHERE ranked.rn > %d
) AS doomed ON doomed.id = fv.id`, keep)
	if _, err := db.ExecContext(ctx, q); err != nil {
		log.Printf("cleanup: prune flag_versions failed: %v", err)
	}
}

func pruneExcessAudit(ctx context.Context, db *sql.DB, keep int) {
	q := fmt.Sprintf(`
DELETE ae FROM audit_entries ae
INNER JOIN (
  SELECT id FROM (
    SELECT id, ROW_NUMBER() OVER (PARTITION BY project_id, environment_id ORDER BY created_at DESC) AS rn
    FROM audit_entries
    WHERE project_id IS NOT NULL AND environment_id IS NOT NULL
  ) AS ranked WHERE ranked.rn > %d
) AS doomed ON doomed.id = ae.id`, keep)
	if _, err := db.ExecContext(ctx, q); err != nil {
		log.Printf("cleanup: prune audit_entries failed: %v", err)
	}
}
