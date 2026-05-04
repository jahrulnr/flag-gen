package cache

import (
	"context"
	"encoding/json"
	"errors"
	"net"
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
	var val string
	var err error
	for attempt := 0; attempt < 3; attempt++ {
		val, err = r.c.Get(ctx, scopeKey).Result()
		if err == nil || errors.Is(err, redis.Nil) {
			break
		}
		if isRetryableRedis(err) && attempt < 2 {
			time.Sleep(time.Duration(50*(attempt+1)) * time.Millisecond)
			continue
		}
		break
	}
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
	var last error
	for attempt := 0; attempt < 3; attempt++ {
		last = r.c.Set(ctx, scopeKey, string(b), ttl).Err()
		if last == nil || !isRetryableRedis(last) || attempt == 2 {
			break
		}
		time.Sleep(time.Duration(50*(attempt+1)) * time.Millisecond)
	}
	return last
}

func (r *RedisL2) Del(ctx context.Context, scopeKey string) error {
	var last error
	for attempt := 0; attempt < 3; attempt++ {
		last = r.c.Del(ctx, scopeKey).Err()
		if last == nil || !isRetryableRedis(last) || attempt == 2 {
			break
		}
		time.Sleep(time.Duration(50*(attempt+1)) * time.Millisecond)
	}
	return last
}

func isRetryableRedis(err error) bool {
	if err == nil {
		return false
	}
	if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
		return false
	}
	if errors.Is(err, redis.Nil) {
		return false
	}
	var ne net.Error
	if errors.As(err, &ne) && ne.Timeout() {
		return true
	}
	// Transient connection / IO issues
	var opErr *net.OpError
	return errors.As(err, &opErr)
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
