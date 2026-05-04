package cache

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"time"

	"flag-gen/domain/entity"

	"github.com/redis/go-redis/v9"
)

type RedisL2 struct {
	c redis.UniversalClient
}

func NewRedisL2(c redis.UniversalClient) *RedisL2 {
	return &RedisL2{c: c}
}

func (r *RedisL2) Get(ctx context.Context, scopeKey string) ([]entity.Flag, bool, error) {
	val, err := r.c.Get(ctx, scopeKey).Result()
	if err != nil {
		if errors.Is(err, redis.Nil) {
			return nil, false, nil
		}
		return nil, false, err
	}

	var flags []entity.Flag
	if err := json.Unmarshal([]byte(val), &flags); err != nil {
		// Treat corrupted cache value as miss.
		return nil, false, nil
	}
	return flags, true, nil
}

func (r *RedisL2) Set(ctx context.Context, scopeKey string, flags []entity.Flag, ttl time.Duration) error {
	b, err := json.Marshal(flags)
	if err != nil {
		return err
	}
	return r.c.Set(ctx, scopeKey, string(b), ttl).Err()
}

// ParseAddrs parses REDIS_ADDRS=host1:6379,host2:6379 into []string.
func ParseAddrs(raw string) []string {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return nil
	}
	parts := strings.Split(raw, ",")
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p != "" {
			out = append(out, p)
		}
	}
	return out
}
