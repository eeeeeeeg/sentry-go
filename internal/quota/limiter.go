package quota

import (
	"context"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
)

type Limiter struct {
	redis  *redis.Client
	window time.Duration
}

type Result struct {
	Allowed    bool
	Limit      int64
	Remaining  int64
	ResetAfter time.Duration
}

func NewLimiter(redis *redis.Client, window time.Duration) *Limiter {
	return &Limiter{
		redis:  redis,
		window: window,
	}
}

func (l *Limiter) Allow(ctx context.Context, name string, limit int64) (Result, error) {
	if limit <= 0 {
		return Result{Allowed: false, Limit: limit}, nil
	}

	key := l.windowKey(name, time.Now().UTC())
	count, err := l.redis.Incr(ctx, key).Result()
	if err != nil {
		return Result{}, err
	}
	if count == 1 {
		if err := l.redis.Expire(ctx, key, l.window+time.Second).Err(); err != nil {
			return Result{}, err
		}
	}

	remaining := limit - count
	if remaining < 0 {
		remaining = 0
	}

	ttl, err := l.redis.TTL(ctx, key).Result()
	if err != nil {
		return Result{}, err
	}

	return Result{
		Allowed:    count <= limit,
		Limit:      limit,
		Remaining:  remaining,
		ResetAfter: ttl,
	}, nil
}

func (l *Limiter) windowKey(name string, now time.Time) string {
	window := now.Unix() / int64(l.window.Seconds())
	return fmt.Sprintf("quota:%s:%d", name, window)
}
