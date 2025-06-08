package cache

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/redis/go-redis/v9"
	"go.uber.org/zap"
)

var (
	ErrCacheMiss = errors.New("cache miss")
	ErrCacheNil  = errors.New("cache: nil value")
)

// Cache interface defines cache operations
type Cache interface {
	Get(ctx context.Context, key string, dest interface{}) error
	Set(ctx context.Context, key string, value interface{}, ttl time.Duration) error
	Delete(ctx context.Context, keys ...string) error
	Exists(ctx context.Context, keys ...string) (int64, error)
	TTL(ctx context.Context, key string) (time.Duration, error)
	Clear(ctx context.Context, pattern string) error
	Close() error
}

// RedisCache implements Cache interface using Redis
type RedisCache struct {
	client *redis.Client
	logger *zap.Logger
	prefix string
	stats  *CacheStats
}

// CacheStats tracks cache statistics
type CacheStats struct {
	Hits       int64
	Misses     int64
	Sets       int64
	Deletes    int64
	Errors     int64
	Evictions  int64
}

// RedisConfig holds Redis configuration
type RedisConfig struct {
	Addr         string
	Password     string
	DB           int
	MaxRetries   int
	DialTimeout  time.Duration
	ReadTimeout  time.Duration
	WriteTimeout time.Duration
	PoolSize     int
	MinIdleConns int
	MaxIdleTime  time.Duration
	KeyPrefix    string
}

// DefaultRedisConfig returns default Redis configuration
func DefaultRedisConfig() *RedisConfig {
	return &RedisConfig{
		Addr:         "localhost:6379",
		Password:     "",
		DB:           0,
		MaxRetries:   3,
		DialTimeout:  5 * time.Second,
		ReadTimeout:  3 * time.Second,
		WriteTimeout: 3 * time.Second,
		PoolSize:     10,
		MinIdleConns: 5,
		MaxIdleTime:  5 * time.Minute,
		KeyPrefix:    "ovncp:",
	}
}

// NewRedisCache creates a new Redis cache instance
func NewRedisCache(cfg *RedisConfig, logger *zap.Logger) (*RedisCache, error) {
	client := redis.NewClient(&redis.Options{
		Addr:         cfg.Addr,
		Password:     cfg.Password,
		DB:           cfg.DB,
		MaxRetries:   cfg.MaxRetries,
		DialTimeout:  cfg.DialTimeout,
		ReadTimeout:  cfg.ReadTimeout,
		WriteTimeout: cfg.WriteTimeout,
		PoolSize:     cfg.PoolSize,
		MinIdleConns: cfg.MinIdleConns,
		ConnMaxIdleTime: cfg.MaxIdleTime,
	})

	// Test connection
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	
	if err := client.Ping(ctx).Err(); err != nil {
		return nil, fmt.Errorf("failed to connect to Redis: %w", err)
	}

	logger.Info("Connected to Redis cache",
		zap.String("addr", cfg.Addr),
		zap.Int("db", cfg.DB))

	return &RedisCache{
		client: client,
		logger: logger,
		prefix: cfg.KeyPrefix,
		stats:  &CacheStats{},
	}, nil
}

// Get retrieves a value from cache
func (c *RedisCache) Get(ctx context.Context, key string, dest interface{}) error {
	fullKey := c.prefix + key
	
	val, err := c.client.Get(ctx, fullKey).Result()
	if err == redis.Nil {
		c.stats.Misses++
		return ErrCacheMiss
	}
	if err != nil {
		c.stats.Errors++
		c.logger.Error("Cache get error", zap.String("key", key), zap.Error(err))
		return err
	}

	if err := json.Unmarshal([]byte(val), dest); err != nil {
		c.stats.Errors++
		c.logger.Error("Cache unmarshal error", zap.String("key", key), zap.Error(err))
		return err
	}

	c.stats.Hits++
	return nil
}

// Set stores a value in cache
func (c *RedisCache) Set(ctx context.Context, key string, value interface{}, ttl time.Duration) error {
	fullKey := c.prefix + key
	
	data, err := json.Marshal(value)
	if err != nil {
		c.stats.Errors++
		c.logger.Error("Cache marshal error", zap.String("key", key), zap.Error(err))
		return err
	}

	if err := c.client.Set(ctx, fullKey, data, ttl).Err(); err != nil {
		c.stats.Errors++
		c.logger.Error("Cache set error", zap.String("key", key), zap.Error(err))
		return err
	}

	c.stats.Sets++
	return nil
}

// Delete removes values from cache
func (c *RedisCache) Delete(ctx context.Context, keys ...string) error {
	if len(keys) == 0 {
		return nil
	}

	fullKeys := make([]string, len(keys))
	for i, key := range keys {
		fullKeys[i] = c.prefix + key
	}

	if err := c.client.Del(ctx, fullKeys...).Err(); err != nil {
		c.stats.Errors++
		c.logger.Error("Cache delete error", zap.Strings("keys", keys), zap.Error(err))
		return err
	}

	c.stats.Deletes += int64(len(keys))
	return nil
}

// Exists checks if keys exist in cache
func (c *RedisCache) Exists(ctx context.Context, keys ...string) (int64, error) {
	if len(keys) == 0 {
		return 0, nil
	}

	fullKeys := make([]string, len(keys))
	for i, key := range keys {
		fullKeys[i] = c.prefix + key
	}

	count, err := c.client.Exists(ctx, fullKeys...).Result()
	if err != nil {
		c.stats.Errors++
		c.logger.Error("Cache exists error", zap.Strings("keys", keys), zap.Error(err))
		return 0, err
	}

	return count, nil
}

// TTL returns the time-to-live of a key
func (c *RedisCache) TTL(ctx context.Context, key string) (time.Duration, error) {
	fullKey := c.prefix + key
	
	ttl, err := c.client.TTL(ctx, fullKey).Result()
	if err != nil {
		c.stats.Errors++
		c.logger.Error("Cache TTL error", zap.String("key", key), zap.Error(err))
		return 0, err
	}

	if ttl < 0 {
		return 0, ErrCacheMiss
	}

	return ttl, nil
}

// Clear removes all keys matching a pattern
func (c *RedisCache) Clear(ctx context.Context, pattern string) error {
	fullPattern := c.prefix + pattern
	
	// Use SCAN to find matching keys
	var cursor uint64
	var keys []string
	
	for {
		var err error
		var batch []string
		batch, cursor, err = c.client.Scan(ctx, cursor, fullPattern, 100).Result()
		if err != nil {
			c.stats.Errors++
			c.logger.Error("Cache scan error", zap.String("pattern", pattern), zap.Error(err))
			return err
		}
		
		keys = append(keys, batch...)
		
		if cursor == 0 {
			break
		}
	}

	if len(keys) > 0 {
		if err := c.client.Del(ctx, keys...).Err(); err != nil {
			c.stats.Errors++
			c.logger.Error("Cache clear error", zap.String("pattern", pattern), zap.Error(err))
			return err
		}
		c.stats.Deletes += int64(len(keys))
	}

	return nil
}

// Close closes the Redis connection
func (c *RedisCache) Close() error {
	return c.client.Close()
}

// Stats returns cache statistics
func (c *RedisCache) Stats() CacheStats {
	return *c.stats
}

// MemoryCache implements in-memory cache (for development/testing)
type MemoryCache struct {
	data   map[string]*memoryItem
	mu     sync.RWMutex
	logger *zap.Logger
	stats  *CacheStats
}

type memoryItem struct {
	value     []byte
	expiresAt time.Time
}

// NewMemoryCache creates a new in-memory cache
func NewMemoryCache(logger *zap.Logger) *MemoryCache {
	cache := &MemoryCache{
		data:   make(map[string]*memoryItem),
		logger: logger,
		stats:  &CacheStats{},
	}
	
	// Start cleanup goroutine
	go cache.cleanup()
	
	return cache
}

// Get retrieves a value from memory cache
func (m *MemoryCache) Get(ctx context.Context, key string, dest interface{}) error {
	m.mu.RLock()
	item, exists := m.data[key]
	m.mu.RUnlock()

	if !exists {
		m.stats.Misses++
		return ErrCacheMiss
	}

	if time.Now().After(item.expiresAt) {
		m.mu.Lock()
		delete(m.data, key)
		m.mu.Unlock()
		m.stats.Misses++
		m.stats.Evictions++
		return ErrCacheMiss
	}

	if err := json.Unmarshal(item.value, dest); err != nil {
		m.stats.Errors++
		return err
	}

	m.stats.Hits++
	return nil
}

// Set stores a value in memory cache
func (m *MemoryCache) Set(ctx context.Context, key string, value interface{}, ttl time.Duration) error {
	data, err := json.Marshal(value)
	if err != nil {
		m.stats.Errors++
		return err
	}

	m.mu.Lock()
	m.data[key] = &memoryItem{
		value:     data,
		expiresAt: time.Now().Add(ttl),
	}
	m.mu.Unlock()

	m.stats.Sets++
	return nil
}

// Delete removes values from memory cache
func (m *MemoryCache) Delete(ctx context.Context, keys ...string) error {
	m.mu.Lock()
	for _, key := range keys {
		delete(m.data, key)
	}
	m.mu.Unlock()

	m.stats.Deletes += int64(len(keys))
	return nil
}

// Exists checks if keys exist in memory cache
func (m *MemoryCache) Exists(ctx context.Context, keys ...string) (int64, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	var count int64
	for _, key := range keys {
		if item, exists := m.data[key]; exists {
			if time.Now().Before(item.expiresAt) {
				count++
			}
		}
	}

	return count, nil
}

// TTL returns the time-to-live of a key
func (m *MemoryCache) TTL(ctx context.Context, key string) (time.Duration, error) {
	m.mu.RLock()
	item, exists := m.data[key]
	m.mu.RUnlock()

	if !exists {
		return 0, ErrCacheMiss
	}

	ttl := time.Until(item.expiresAt)
	if ttl < 0 {
		return 0, ErrCacheMiss
	}

	return ttl, nil
}

// Clear removes all keys matching a pattern
func (m *MemoryCache) Clear(ctx context.Context, pattern string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	// Simple pattern matching (just prefix for now)
	keysToDelete := []string{}
	for key := range m.data {
		if strings.HasPrefix(key, pattern) {
			keysToDelete = append(keysToDelete, key)
		}
	}

	for _, key := range keysToDelete {
		delete(m.data, key)
	}

	m.stats.Deletes += int64(len(keysToDelete))
	return nil
}

// Close is a no-op for memory cache
func (m *MemoryCache) Close() error {
	return nil
}

// cleanup periodically removes expired items
func (m *MemoryCache) cleanup() {
	ticker := time.NewTicker(1 * time.Minute)
	defer ticker.Stop()

	for range ticker.C {
		m.mu.Lock()
		now := time.Now()
		for key, item := range m.data {
			if now.After(item.expiresAt) {
				delete(m.data, key)
				m.stats.Evictions++
			}
		}
		m.mu.Unlock()
	}
}

// Stats returns cache statistics
func (m *MemoryCache) Stats() CacheStats {
	return *m.stats
}