package redis

import (
	"context"
	"fmt"
	"time"

	goredis "github.com/redis/go-redis/v9"
)

type RateLimiter struct {
	client *goredis.Client
}

func NewRateLimiter(client *goredis.Client) *RateLimiter {
	return &RateLimiter{client: client}
}

func (l *RateLimiter) Allow(ctx context.Context, key string, limit int, window time.Duration) (bool, error) {
	redisKey := "rate:" + key
	count, err := l.client.Incr(ctx, redisKey).Result()
	if err != nil {
		return false, fmt.Errorf("increment rate limit: %w", err)
	}
	if count == 1 {
		if err := l.client.Expire(ctx, redisKey, window).Err(); err != nil {
			return false, fmt.Errorf("expire rate limit: %w", err)
		}
	}
	return count <= int64(limit), nil
}
