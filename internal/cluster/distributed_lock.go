package cluster

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
	"go.uber.org/zap"
)

var (
	ErrLockAcquireFailed = errors.New("failed to acquire lock")
	ErrLockNotHeld       = errors.New("lock not held")
	ErrLockExpired       = errors.New("lock expired")
)

// DistributedLock implements a distributed lock using Redis
type DistributedLock struct {
	redis    *redis.Client
	key      string
	value    string
	ttl      time.Duration
	logger   *zap.Logger
}

// LockOptions configures lock behavior
type LockOptions struct {
	TTL         time.Duration
	RetryDelay  time.Duration
	MaxRetries  int
}

// DefaultLockOptions returns default lock options
func DefaultLockOptions() *LockOptions {
	return &LockOptions{
		TTL:        30 * time.Second,
		RetryDelay: 100 * time.Millisecond,
		MaxRetries: 10,
	}
}

// NewDistributedLock creates a new distributed lock
func NewDistributedLock(redis *redis.Client, key string, opts *LockOptions, logger *zap.Logger) *DistributedLock {
	if opts == nil {
		opts = DefaultLockOptions()
	}

	return &DistributedLock{
		redis:  redis,
		key:    fmt.Sprintf("ovncp:lock:%s", key),
		value:  uuid.New().String(),
		ttl:    opts.TTL,
		logger: logger,
	}
}

// Acquire attempts to acquire the lock
func (l *DistributedLock) Acquire(ctx context.Context) error {
	// Try to set the key only if it doesn't exist
	ok, err := l.redis.SetNX(ctx, l.key, l.value, l.ttl).Result()
	if err != nil {
		return fmt.Errorf("failed to acquire lock: %w", err)
	}

	if !ok {
		return ErrLockAcquireFailed
	}

	l.logger.Debug("Lock acquired", 
		zap.String("key", l.key),
		zap.String("value", l.value),
		zap.Duration("ttl", l.ttl))

	return nil
}

// AcquireWithRetry attempts to acquire the lock with retries
func (l *DistributedLock) AcquireWithRetry(ctx context.Context, opts *LockOptions) error {
	if opts == nil {
		opts = DefaultLockOptions()
	}

	for i := 0; i < opts.MaxRetries; i++ {
		err := l.Acquire(ctx)
		if err == nil {
			return nil
		}

		if err != ErrLockAcquireFailed {
			return err
		}

		// Wait before retry
		select {
		case <-time.After(opts.RetryDelay):
			// Continue to next retry
		case <-ctx.Done():
			return ctx.Err()
		}
	}

	return ErrLockAcquireFailed
}

// Release releases the lock
func (l *DistributedLock) Release(ctx context.Context) error {
	// Use Lua script to ensure atomic check-and-delete
	script := `
		if redis.call("get", KEYS[1]) == ARGV[1] then
			return redis.call("del", KEYS[1])
		else
			return 0
		end
	`

	result, err := l.redis.Eval(ctx, script, []string{l.key}, l.value).Result()
	if err != nil {
		return fmt.Errorf("failed to release lock: %w", err)
	}

	if result.(int64) == 0 {
		return ErrLockNotHeld
	}

	l.logger.Debug("Lock released", zap.String("key", l.key))
	return nil
}

// Extend extends the lock TTL
func (l *DistributedLock) Extend(ctx context.Context, ttl time.Duration) error {
	// Use Lua script to ensure atomic check-and-extend
	script := `
		if redis.call("get", KEYS[1]) == ARGV[1] then
			return redis.call("pexpire", KEYS[1], ARGV[2])
		else
			return 0
		end
	`

	millis := ttl.Milliseconds()
	result, err := l.redis.Eval(ctx, script, []string{l.key}, l.value, millis).Result()
	if err != nil {
		return fmt.Errorf("failed to extend lock: %w", err)
	}

	if result.(int64) == 0 {
		return ErrLockNotHeld
	}

	l.logger.Debug("Lock extended", 
		zap.String("key", l.key),
		zap.Duration("ttl", ttl))

	return nil
}

// IsHeld checks if the lock is currently held by this instance
func (l *DistributedLock) IsHeld(ctx context.Context) (bool, error) {
	val, err := l.redis.Get(ctx, l.key).Result()
	if err == redis.Nil {
		return false, nil
	}
	if err != nil {
		return false, err
	}

	return val == l.value, nil
}

// WithLock executes a function while holding the lock
func (l *DistributedLock) WithLock(ctx context.Context, fn func() error) error {
	if err := l.Acquire(ctx); err != nil {
		return err
	}
	defer l.Release(ctx)

	return fn()
}

// LockManager manages distributed locks
type LockManager struct {
	redis  *redis.Client
	logger *zap.Logger
	locks  map[string]*DistributedLock
	mu     sync.RWMutex
}

// NewLockManager creates a new lock manager
func NewLockManager(redis *redis.Client, logger *zap.Logger) *LockManager {
	return &LockManager{
		redis:  redis,
		logger: logger,
		locks:  make(map[string]*DistributedLock),
	}
}

// AcquireLock acquires a distributed lock
func (m *LockManager) AcquireLock(ctx context.Context, key string, opts *LockOptions) (*DistributedLock, error) {
	lock := NewDistributedLock(m.redis, key, opts, m.logger)
	
	if err := lock.AcquireWithRetry(ctx, opts); err != nil {
		return nil, err
	}

	m.mu.Lock()
	m.locks[key] = lock
	m.mu.Unlock()

	// Start auto-renewal goroutine
	go m.autoRenew(ctx, key, lock)

	return lock, nil
}

// ReleaseLock releases a distributed lock
func (m *LockManager) ReleaseLock(ctx context.Context, key string) error {
	m.mu.Lock()
	lock, exists := m.locks[key]
	if exists {
		delete(m.locks, key)
	}
	m.mu.Unlock()

	if !exists {
		return ErrLockNotHeld
	}

	return lock.Release(ctx)
}

// autoRenew automatically renews the lock before it expires
func (m *LockManager) autoRenew(ctx context.Context, key string, lock *DistributedLock) {
	ticker := time.NewTicker(lock.ttl / 2)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			m.mu.RLock()
			_, exists := m.locks[key]
			m.mu.RUnlock()

			if !exists {
				// Lock was released
				return
			}

			// Extend the lock
			if err := lock.Extend(ctx, lock.ttl); err != nil {
				m.logger.Error("Failed to extend lock", 
					zap.String("key", key),
					zap.Error(err))
				
				// Remove from manager if extension failed
				m.mu.Lock()
				delete(m.locks, key)
				m.mu.Unlock()
				return
			}

		case <-ctx.Done():
			return
		}
	}
}

// ReleaseAll releases all locks held by this manager
func (m *LockManager) ReleaseAll(ctx context.Context) {
	m.mu.Lock()
	locks := make(map[string]*DistributedLock)
	for k, v := range m.locks {
		locks[k] = v
	}
	m.locks = make(map[string]*DistributedLock)
	m.mu.Unlock()

	for key, lock := range locks {
		if err := lock.Release(ctx); err != nil {
			m.logger.Error("Failed to release lock",
				zap.String("key", key),
				zap.Error(err))
		}
	}
}