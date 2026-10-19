package cache

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/bubblevy/bookstore/pkg/logger"
	"github.com/redis/go-redis/v9"
)

var ErrCacheMiss = errors.New("cache: key not found")

// Cache defines the operations for caching data across the application.
type Cache interface {
	Get(ctx context.Context, key string, dest any) error
	Set(ctx context.Context, key string, value any, ttl time.Duration) error
	Delete(ctx context.Context, keys ...string) error
	DeletePrefix(ctx context.Context, prefix string) error
	Close() error
}

type Config struct {
	Host     string
	Port     string
	Password string
	DB       int
	Enabled  bool
}

type redisCache struct {
	client *redis.Client
}

// NewRedisCache initializes a production Redis connection.
// If Redis is disabled or fails to connect, it falls back to a NoopCache (fail-open) to prevent app crashes.
func NewRedisCache(cfg Config) (Cache, error) {
	if !cfg.Enabled || cfg.Host == "" {
		return &noopCache{}, nil
	}

	port := cfg.Port
	if port == "" {
		port = "6379"
	}
	addr := fmt.Sprintf("%s:%s", cfg.Host, port)

	rdb := redis.NewClient(&redis.Options{
		Addr:     addr,
		Password: cfg.Password,
		DB:       cfg.DB,
	})

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	if err := rdb.Ping(ctx).Err(); err != nil {
		logger.Warn("redis connection failed on startup, falling back to no-op cache", "addr", addr, "error", err)
		return &noopCache{}, nil
	}

	logger.Info("connected to redis cache successfully", "addr", addr, "db", cfg.DB)
	return &redisCache{client: rdb}, nil
}

// NewRedisCacheWithClient creates a Cache with a provided redis client (ideal for unit testing).
func NewRedisCacheWithClient(client *redis.Client) Cache {
	if client == nil {
		return &noopCache{}
	}
	return &redisCache{client: client}
}

// NewNoopCache returns an empty cache implementation that does nothing.
func NewNoopCache() Cache {
	return &noopCache{}
}

func (c *redisCache) Get(ctx context.Context, key string, dest any) error {
	val, err := c.client.Get(ctx, key).Bytes()
	if err != nil {
		if errors.Is(err, redis.Nil) {
			return ErrCacheMiss
		}
		logger.Warn("redis get failed", "key", key, "error", err)
		return ErrCacheMiss
	}

	if err := json.Unmarshal(val, dest); err != nil {
		logger.Warn("redis json unmarshal failed", "key", key, "error", err)
		return ErrCacheMiss
	}
	return nil
}

func (c *redisCache) Set(ctx context.Context, key string, value any, ttl time.Duration) error {
	bytes, err := json.Marshal(value)
	if err != nil {
		return fmt.Errorf("failed marshaling cache value: %w", err)
	}

	if err := c.client.Set(ctx, key, bytes, ttl).Err(); err != nil {
		logger.Warn("redis set failed", "key", key, "error", err)
		return nil // Fail-open: do not break business execution on cache write failure
	}
	return nil
}

func (c *redisCache) Delete(ctx context.Context, keys ...string) error {
	if len(keys) == 0 {
		return nil
	}
	if err := c.client.Del(ctx, keys...).Err(); err != nil {
		logger.Warn("redis delete failed", "error", err)
	}
	return nil
}

func (c *redisCache) DeletePrefix(ctx context.Context, prefix string) error {
	iter := c.client.Scan(ctx, 0, prefix+"*", 100).Iterator()
	var keys []string
	for iter.Next(ctx) {
		keys = append(keys, iter.Val())
		if len(keys) >= 100 {
			_ = c.client.Del(ctx, keys...).Err()
			keys = keys[:0]
		}
	}
	if err := iter.Err(); err != nil {
		logger.Warn("redis scan failed during delete prefix", "prefix", prefix, "error", err)
	}
	if len(keys) > 0 {
		_ = c.client.Del(ctx, keys...).Err()
	}
	return nil
}

func (c *redisCache) Close() error {
	if c.client != nil {
		return c.client.Close()
	}
	return nil
}

type noopCache struct{}

func (n *noopCache) Get(ctx context.Context, key string, dest any) error {
	return ErrCacheMiss
}

func (n *noopCache) Set(ctx context.Context, key string, value any, ttl time.Duration) error {
	return nil
}

func (n *noopCache) Delete(ctx context.Context, keys ...string) error {
	return nil
}

func (n *noopCache) DeletePrefix(ctx context.Context, prefix string) error {
	return nil
}

func (n *noopCache) Close() error {
	return nil
}
