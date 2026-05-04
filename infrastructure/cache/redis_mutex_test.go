// nolint

package cache

import (
	"context"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/require"
)

func TestTryAcquireRedisLock_AndRelease(t *testing.T) {
	s := miniredis.RunT(t)
	t.Cleanup(s.Close)
	rdb := redis.NewClient(&redis.Options{Addr: s.Addr()})
	ctx := context.Background()

	token := RandomLockToken()
	ok, err := TryAcquireRedisLock(ctx, rdb, "flag-gen:lock:test:1", token, 30*time.Second)
	require.NoError(t, err)
	require.True(t, ok)

	ok2, err := TryAcquireRedisLock(ctx, rdb, "flag-gen:lock:test:1", "other", 30*time.Second)
	require.NoError(t, err)
	require.False(t, ok2)

	require.NoError(t, ReleaseRedisLock(ctx, rdb, "flag-gen:lock:test:1", token))

	ok3, err := TryAcquireRedisLock(ctx, rdb, "flag-gen:lock:test:1", "next", 30*time.Second)
	require.NoError(t, err)
	require.True(t, ok3)
}
