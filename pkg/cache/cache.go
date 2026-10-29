package cache

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"sync"
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
// If Redis is disabled or fails to connect, it falls back to an in-memory TTL cache to maintain high performance.
func NewRedisCache(cfg Config) (Cache, error) {
	if !cfg.Enabled || cfg.Host == "" {
		logger.Info("redis cache disabled, using high-performance in-memory cache")
		return NewMemoryCache(), nil
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
		logger.Warn("redis connection failed on startup, falling back to high-performance in-memory cache", "addr", addr, "error", err)
		return NewMemoryCache(), nil
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

type memoryItem struct {
	data      []byte
	expiresAt time.Time
}

type memoryCache struct {
	mu       sync.RWMutex
	items    map[string]memoryItem
	stopChan chan struct{}
}

// NewMemoryCache creates an in-memory TTL cache with periodic cleanup of expired entries.
func NewMemoryCache() Cache {
	mc := &memoryCache{
		items:    make(map[string]memoryItem),
		stopChan: make(chan struct{}),
	}

	go func() {
		ticker := time.NewTicker(1 * time.Minute)
		defer ticker.Stop()
		for {
			select {
			case <-ticker.C:
				mc.mu.Lock()
				now := time.Now()
				for k, v := range mc.items {
					if !v.expiresAt.IsZero() && now.After(v.expiresAt) {
						delete(mc.items, k)
					}
				}
				mc.mu.Unlock()
			case <-mc.stopChan:
				return
			}
		}
	}()

	return mc
}

func (m *memoryCache) Get(ctx context.Context, key string, dest any) error {
	m.mu.RLock()
	item, ok := m.items[key]
	m.mu.RUnlock()

	if !ok {
		return ErrCacheMiss
	}

	if !item.expiresAt.IsZero() && time.Now().After(item.expiresAt) {
		m.mu.Lock()
		delete(m.items, key)
		m.mu.Unlock()
		return ErrCacheMiss
	}

	return json.Unmarshal(item.data, dest)
}

func (m *memoryCache) Set(ctx context.Context, key string, value any, ttl time.Duration) error {
	bytes, err := json.Marshal(value)
	if err != nil {
		return fmt.Errorf("failed marshaling cache value: %w", err)
	}

	var expiresAt time.Time
	if ttl > 0 {
		expiresAt = time.Now().Add(ttl)
	}

	m.mu.Lock()
	m.items[key] = memoryItem{
		data:      bytes,
		expiresAt: expiresAt,
	}
	m.mu.Unlock()
	return nil
}

func (m *memoryCache) Delete(ctx context.Context, keys ...string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	for _, k := range keys {
		delete(m.items, k)
	}
	return nil
}

func (m *memoryCache) DeletePrefix(ctx context.Context, prefix string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	for k := range m.items {
		if strings.HasPrefix(k, prefix) {
			delete(m.items, k)
		}
	}
	return nil
}

func (m *memoryCache) Close() error {
	select {
	case <-m.stopChan:
	default:
		close(m.stopChan)
	}
	return nil
}
