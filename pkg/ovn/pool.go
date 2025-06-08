package ovn

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/lspecian/ovncp/internal/config"
	"go.uber.org/zap"
)

var (
	ErrPoolClosed    = errors.New("connection pool is closed")
	ErrPoolExhausted = errors.New("connection pool exhausted")
	ErrInvalidConn   = errors.New("invalid connection")
)

// ConnectionPool manages a pool of OVN client connections
type ConnectionPool struct {
	config   *PoolConfig
	conns    chan *poolConn
	factory  ConnectionFactory
	mu       sync.RWMutex
	closed   bool
	logger   *zap.Logger
	stats    *PoolStats
}

// PoolConfig contains connection pool configuration
type PoolConfig struct {
	MaxSize         int           // Maximum number of connections
	MinSize         int           // Minimum number of connections
	MaxIdleTime     time.Duration // Maximum idle time before connection is closed
	MaxLifetime     time.Duration // Maximum lifetime of a connection
	HealthCheckTime time.Duration // How often to health check connections
	WaitTimeout     time.Duration // Maximum time to wait for a connection
}

// DefaultPoolConfig returns a default pool configuration
func DefaultPoolConfig() *PoolConfig {
	return &PoolConfig{
		MaxSize:         10,
		MinSize:         2,
		MaxIdleTime:     5 * time.Minute,
		MaxLifetime:     30 * time.Minute,
		HealthCheckTime: 30 * time.Second,
		WaitTimeout:     5 * time.Second,
	}
}

// ConnectionFactory creates new OVN client connections
type ConnectionFactory func(ctx context.Context) (*Client, error)

// poolConn wraps a client connection with metadata
type poolConn struct {
	client      *Client
	createdAt   time.Time
	lastUsedAt  time.Time
	usageCount  int64
	inUse       bool
	mu          sync.Mutex
}

// PoolStats tracks connection pool statistics
type PoolStats struct {
	mu            sync.RWMutex
	TotalConns    int64
	ActiveConns   int64
	IdleConns     int64
	WaitCount     int64
	WaitDuration  time.Duration
	MaxWait       time.Duration
	Hits          int64
	Misses        int64
	Timeouts      int64
	BadConns      int64
}

// NewConnectionPool creates a new connection pool
func NewConnectionPool(cfg *PoolConfig, factory ConnectionFactory, logger *zap.Logger) (*ConnectionPool, error) {
	if cfg.MaxSize <= 0 {
		return nil, errors.New("max pool size must be positive")
	}
	if cfg.MinSize < 0 || cfg.MinSize > cfg.MaxSize {
		return nil, errors.New("invalid min pool size")
	}

	pool := &ConnectionPool{
		config:  cfg,
		conns:   make(chan *poolConn, cfg.MaxSize),
		factory: factory,
		logger:  logger,
		stats:   &PoolStats{},
	}

	// Initialize minimum connections
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	for i := 0; i < cfg.MinSize; i++ {
		conn, err := pool.createConn(ctx)
		if err != nil {
			pool.Close()
			return nil, fmt.Errorf("failed to create initial connection: %w", err)
		}
		pool.conns <- conn
		pool.stats.IdleConns++
	}
	pool.stats.TotalConns = int64(cfg.MinSize)

	// Start maintenance goroutine
	go pool.maintain()

	logger.Info("Connection pool created",
		zap.Int("min_size", cfg.MinSize),
		zap.Int("max_size", cfg.MaxSize))

	return pool, nil
}

// Get retrieves a connection from the pool
func (p *ConnectionPool) Get(ctx context.Context) (*Client, error) {
	p.mu.RLock()
	if p.closed {
		p.mu.RUnlock()
		return nil, ErrPoolClosed
	}
	p.mu.RUnlock()

	startTime := time.Now()
	p.stats.mu.Lock()
	p.stats.WaitCount++
	p.stats.mu.Unlock()

	// Try to get an existing connection
	select {
	case conn := <-p.conns:
		waitTime := time.Since(startTime)
		p.updateWaitStats(waitTime)

		// Check if connection is still valid
		if p.isConnValid(conn) {
			conn.mu.Lock()
			conn.lastUsedAt = time.Now()
			conn.usageCount++
			conn.inUse = true
			conn.mu.Unlock()

			p.stats.mu.Lock()
			p.stats.Hits++
			p.stats.IdleConns--
			p.stats.ActiveConns++
			p.stats.mu.Unlock()

			return conn.client, nil
		}

		// Connection is invalid, close it
		p.closeConn(conn)
		p.stats.mu.Lock()
		p.stats.BadConns++
		p.stats.mu.Unlock()

	case <-time.After(100 * time.Millisecond):
		// No immediate connection available
	}

	// Check if we can create a new connection
	p.stats.mu.RLock()
	canCreate := p.stats.TotalConns < int64(p.config.MaxSize)
	p.stats.mu.RUnlock()

	if canCreate {
		conn, err := p.createConn(ctx)
		if err != nil {
			p.stats.mu.Lock()
			p.stats.Misses++
			p.stats.mu.Unlock()
			return nil, fmt.Errorf("failed to create connection: %w", err)
		}

		conn.mu.Lock()
		conn.inUse = true
		conn.mu.Unlock()

		waitTime := time.Since(startTime)
		p.updateWaitStats(waitTime)

		p.stats.mu.Lock()
		p.stats.TotalConns++
		p.stats.ActiveConns++
		p.stats.mu.Unlock()

		return conn.client, nil
	}

	// Wait for a connection to become available
	timeoutCtx, cancel := context.WithTimeout(ctx, p.config.WaitTimeout)
	defer cancel()

	select {
	case conn := <-p.conns:
		waitTime := time.Since(startTime)
		p.updateWaitStats(waitTime)

		if p.isConnValid(conn) {
			conn.mu.Lock()
			conn.lastUsedAt = time.Now()
			conn.usageCount++
			conn.inUse = true
			conn.mu.Unlock()

			p.stats.mu.Lock()
			p.stats.Hits++
			p.stats.IdleConns--
			p.stats.ActiveConns++
			p.stats.mu.Unlock()

			return conn.client, nil
		}

		p.closeConn(conn)
		p.stats.mu.Lock()
		p.stats.BadConns++
		p.stats.mu.Unlock()
		return nil, ErrInvalidConn

	case <-timeoutCtx.Done():
		p.stats.mu.Lock()
		p.stats.Timeouts++
		p.stats.mu.Unlock()
		return nil, ErrPoolExhausted
	}
}

// Put returns a connection to the pool
func (p *ConnectionPool) Put(client *Client) error {
	p.mu.RLock()
	if p.closed {
		p.mu.RUnlock()
		client.Close()
		return ErrPoolClosed
	}
	p.mu.RUnlock()

	// Find the pool connection
	var conn *poolConn
	for {
		select {
		case c := <-p.conns:
			if c.client == client {
				conn = c
				break
			}
			// Put it back
			p.conns <- c
		default:
			// Connection not found in pool
			p.logger.Warn("Connection not found in pool")
			client.Close()
			return nil
		}
		if conn != nil {
			break
		}
	}

	conn.mu.Lock()
	conn.lastUsedAt = time.Now()
	conn.inUse = false
	conn.mu.Unlock()

	// Check if connection should be closed
	if !p.isConnValid(conn) {
		p.closeConn(conn)
		return nil
	}

	// Return to pool
	select {
	case p.conns <- conn:
		p.stats.mu.Lock()
		p.stats.ActiveConns--
		p.stats.IdleConns++
		p.stats.mu.Unlock()
	default:
		// Pool is full, close the connection
		p.closeConn(conn)
	}

	return nil
}

// Close closes all connections and the pool
func (p *ConnectionPool) Close() error {
	p.mu.Lock()
	if p.closed {
		p.mu.Unlock()
		return nil
	}
	p.closed = true
	p.mu.Unlock()

	// Close all connections
	close(p.conns)
	for conn := range p.conns {
		p.closeConn(conn)
	}

	p.logger.Info("Connection pool closed",
		zap.Int64("total_connections", p.stats.TotalConns))

	return nil
}

// Stats returns pool statistics
func (p *ConnectionPool) Stats() PoolStats {
	p.stats.mu.RLock()
	defer p.stats.mu.RUnlock()
	return *p.stats
}

// maintain performs periodic maintenance on the pool
func (p *ConnectionPool) maintain() {
	ticker := time.NewTicker(p.config.HealthCheckTime)
	defer ticker.Stop()

	for range ticker.C {
		p.mu.RLock()
		if p.closed {
			p.mu.RUnlock()
			return
		}
		p.mu.RUnlock()

		p.performMaintenance()
	}
}

// performMaintenance checks and maintains pool health
func (p *ConnectionPool) performMaintenance() {
	var connsToCheck []*poolConn
	
	// Collect idle connections to check
	for i := 0; i < len(p.conns); i++ {
		select {
		case conn := <-p.conns:
			connsToCheck = append(connsToCheck, conn)
		default:
			break
		}
	}

	// Check each connection
	validConns := 0
	for _, conn := range connsToCheck {
		if p.isConnValid(conn) && p.shouldKeepConn(conn) {
			// Health check
			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			if err := conn.client.Ping(ctx); err != nil {
				p.logger.Debug("Connection health check failed", zap.Error(err))
				p.closeConn(conn)
			} else {
				p.conns <- conn
				validConns++
			}
			cancel()
		} else {
			p.closeConn(conn)
		}
	}

	// Ensure minimum connections
	p.stats.mu.RLock()
	currentConns := p.stats.TotalConns
	p.stats.mu.RUnlock()

	if currentConns < int64(p.config.MinSize) {
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()

		for i := currentConns; i < int64(p.config.MinSize); i++ {
			conn, err := p.createConn(ctx)
			if err != nil {
				p.logger.Error("Failed to create connection during maintenance", zap.Error(err))
				break
			}
			p.conns <- conn
			p.stats.mu.Lock()
			p.stats.IdleConns++
			p.stats.mu.Unlock()
		}
	}
}

// createConn creates a new pooled connection
func (p *ConnectionPool) createConn(ctx context.Context) (*poolConn, error) {
	client, err := p.factory(ctx)
	if err != nil {
		return nil, err
	}

	return &poolConn{
		client:     client,
		createdAt:  time.Now(),
		lastUsedAt: time.Now(),
	}, nil
}

// closeConn closes a pooled connection
func (p *ConnectionPool) closeConn(conn *poolConn) {
	if conn == nil || conn.client == nil {
		return
	}

	conn.client.Close()
	
	p.stats.mu.Lock()
	p.stats.TotalConns--
	if conn.inUse {
		p.stats.ActiveConns--
	} else {
		p.stats.IdleConns--
	}
	p.stats.mu.Unlock()
}

// isConnValid checks if a connection is still valid
func (p *ConnectionPool) isConnValid(conn *poolConn) bool {
	if conn == nil || conn.client == nil {
		return false
	}

	// Check lifetime
	if time.Since(conn.createdAt) > p.config.MaxLifetime {
		return false
	}

	// Check if connection is closed
	if conn.client.IsClosed() {
		return false
	}

	return true
}

// shouldKeepConn checks if a connection should be kept in the pool
func (p *ConnectionPool) shouldKeepConn(conn *poolConn) bool {
	// Check idle time
	conn.mu.Lock()
	idleTime := time.Since(conn.lastUsedAt)
	conn.mu.Unlock()

	if idleTime > p.config.MaxIdleTime {
		return false
	}

	return true
}

// updateWaitStats updates wait time statistics
func (p *ConnectionPool) updateWaitStats(waitTime time.Duration) {
	p.stats.mu.Lock()
	p.stats.WaitDuration += waitTime
	if waitTime > p.stats.MaxWait {
		p.stats.MaxWait = waitTime
	}
	p.stats.mu.Unlock()
}

// PooledClient wraps a Client with automatic connection pooling
type PooledClient struct {
	pool   *ConnectionPool
	client *Client
}

// NewPooledClient creates a new pooled client
func NewPooledClient(cfg *config.OVNConfig, logger *zap.Logger) (*PooledClient, error) {
	poolCfg := DefaultPoolConfig()
	
	// Adjust pool size based on configuration
	if cfg.MaxConnections > 0 {
		poolCfg.MaxSize = cfg.MaxConnections
		poolCfg.MinSize = cfg.MaxConnections / 4
		if poolCfg.MinSize < 1 {
			poolCfg.MinSize = 1
		}
	}

	factory := func(ctx context.Context) (*Client, error) {
		client, err := NewClient(cfg)
		if err != nil {
			return nil, err
		}
		if err := client.Connect(ctx); err != nil {
			return nil, err
		}
		return client, nil
	}

	pool, err := NewConnectionPool(poolCfg, factory, logger)
	if err != nil {
		return nil, err
	}

	return &PooledClient{
		pool: pool,
	}, nil
}

// Execute runs a function with a pooled connection
func (pc *PooledClient) Execute(ctx context.Context, fn func(*Client) error) error {
	client, err := pc.pool.Get(ctx)
	if err != nil {
		return fmt.Errorf("failed to get connection from pool: %w", err)
	}
	defer pc.pool.Put(client)

	return fn(client)
}

// Stats returns pool statistics
func (pc *PooledClient) Stats() PoolStats {
	return pc.pool.Stats()
}

// Close closes the connection pool
func (pc *PooledClient) Close() error {
	return pc.pool.Close()
}