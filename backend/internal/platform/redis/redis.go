package redis

import (
	"context"
	"fmt"

	"github.com/gsnpeeps/gsnpeeps/backend/internal/config"
	goredis "github.com/redis/go-redis/v9"
)

type Client struct {
	client        *goredis.Client
	healthTimeout config.Redis
}

func Open(ctx context.Context, cfg config.Redis) (*Client, error) {
	options, err := goredis.ParseURL(cfg.URL)
	if err != nil {
		return nil, fmt.Errorf("parse redis configuration: %w", err)
	}
	options.PoolSize = cfg.PoolSize
	options.DialTimeout = cfg.DialTimeout
	options.ReadTimeout = cfg.ReadTimeout
	options.WriteTimeout = cfg.WriteTimeout

	client := &Client{client: goredis.NewClient(options), healthTimeout: cfg}
	if err := client.Ping(ctx); err != nil {
		_ = client.client.Close()
		return nil, err
	}
	return client, nil
}

func (c *Client) Ping(ctx context.Context) error {
	pingCtx, cancel := context.WithTimeout(ctx, c.healthTimeout.HealthTimeout)
	defer cancel()
	if err := c.client.Ping(pingCtx).Err(); err != nil {
		return fmt.Errorf("ping redis: %w", err)
	}
	return nil
}

func (c *Client) Raw() *goredis.Client { return c.client }
func (c *Client) Close() error         { return c.client.Close() }
