// nolint

package cache

import (
	"context"
	"testing"
	"time"

	"flag-gen/domain/entity"

	miniredis "github.com/alicebob/miniredis/v2"
	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/require"
)

func TestRedisL2_GetSet(t *testing.T) {
	s := miniredis.RunT(t)
	c := redis.NewClient(&redis.Options{Addr: s.Addr()})
	l2 := NewRedisL2(c)

	ctx := context.Background()
	flags := []entity.Flag{{Key: "k1"}}
	require.NoError(t, l2.Set(ctx, "scope", flags, 10*time.Second))

	got, ok, err := l2.Get(ctx, "scope")
	require.NoError(t, err)
	require.True(t, ok)
	require.Equal(t, "k1", got[0].Key)
}

func TestRedisL2_Get_CorruptJSONIsMiss(t *testing.T) {
	s := miniredis.RunT(t)
	c := redis.NewClient(&redis.Options{Addr: s.Addr()})
	l2 := NewRedisL2(c)

	ctx := context.Background()
	require.NoError(t, c.Set(ctx, "scope", "not-json", 0).Err())

	_, ok, err := l2.Get(ctx, "scope")
	require.NoError(t, err)
	require.False(t, ok)
}

func TestParseAddrs(t *testing.T) {
	require.Equal(t, []string{"a:1", "b:2"}, ParseAddrs(" a:1, b:2 "))
	require.Nil(t, ParseAddrs(""))
}
