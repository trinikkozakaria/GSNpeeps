package postgres

import (
	"context"
	"fmt"

	"github.com/gsnpeeps/gsnpeeps/backend/internal/config"
	"github.com/jackc/pgx/v5/pgxpool"
)

type Client struct {
	pool          *pgxpool.Pool
	healthTimeout config.Postgres
}

func Open(ctx context.Context, cfg config.Postgres) (*Client, error) {
	poolConfig, err := pgxpool.ParseConfig(cfg.URL)
	if err != nil {
		return nil, fmt.Errorf("parse postgres configuration: %w", err)
	}
	poolConfig.MaxConns = cfg.MaxConnections
	poolConfig.MinConns = cfg.MinConnections
	poolConfig.MaxConnLifetime = cfg.MaxConnLifetime

	pool, err := pgxpool.NewWithConfig(ctx, poolConfig)
	if err != nil {
		return nil, fmt.Errorf("open postgres pool: %w", err)
	}
	client := &Client{pool: pool, healthTimeout: cfg}
	if err := client.Ping(ctx); err != nil {
		pool.Close()
		return nil, err
	}
	return client, nil
}

func (c *Client) Ping(ctx context.Context) error {
	pingCtx, cancel := context.WithTimeout(ctx, c.healthTimeout.HealthTimeout)
	defer cancel()
	if err := c.pool.Ping(pingCtx); err != nil {
		return fmt.Errorf("ping postgres: %w", err)
	}
	return nil
}

func (c *Client) Pool() *pgxpool.Pool { return c.pool }
func (c *Client) Close()              { c.pool.Close() }
