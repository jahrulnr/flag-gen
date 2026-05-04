package cache

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"time"

	"github.com/redis/go-redis/v9"
)

// TryAcquireRedisLock sets key to token with NX and EX ttl. false,nil means held by another owner.
func TryAcquireRedisLock(ctx context.Context, rdb redis.UniversalClient, key, token string, ttl time.Duration) (bool, error) {
	if rdb == nil || key == "" || token == "" || ttl <= 0 {
		return false, nil
	}
	return rdb.SetNX(ctx, key, token, ttl).Result()
}

// ReleaseRedisLock deletes key only if value still matches token (compare-and-delete).
func ReleaseRedisLock(ctx context.Context, rdb redis.UniversalClient, key, token string) error {
	if rdb == nil || key == "" || token == "" {
		return nil
	}
	const script = `
if redis.call("get", KEYS[1]) == ARGV[1] then
  return redis.call("del", KEYS[1])
else
  return 0
end`
	_, err := rdb.Eval(ctx, script, []string{key}, token).Result()
	return err
}

// RandomLockToken returns a random hex token for lock ownership.
func RandomLockToken() string {
	b := make([]byte, 16)
	_, _ = rand.Read(b)
	return hex.EncodeToString(b)
}
