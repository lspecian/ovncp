package db

import (
	"database/sql"
	"fmt"
	"time"

	"github.com/lspecian/ovncp/internal/config"
	_ "github.com/lib/pq"
)

// DB represents the database connection
type DB struct {
	conn *sql.DB
}

// Exec executes a query without returning any rows
func (db *DB) Exec(query string, args ...interface{}) (sql.Result, error) {
	return db.conn.Exec(query, args...)
}

// Query executes a query that returns rows
func (db *DB) Query(query string, args ...interface{}) (*sql.Rows, error) {
	return db.conn.Query(query, args...)
}

// New creates a new database connection
func New(cfg *config.DatabaseConfig) (*DB, error) {
	dsn := fmt.Sprintf("host=%s port=%s user=%s password=%s dbname=%s sslmode=%s",
		cfg.Host, cfg.Port, cfg.User, cfg.Password, cfg.Name, cfg.SSLMode)
	
	conn, err := sql.Open("postgres", dsn)
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}
	
	// Set connection pool settings with defaults
	maxOpen := cfg.MaxOpenConns
	if maxOpen == 0 {
		maxOpen = 25
	}
	conn.SetMaxOpenConns(maxOpen)
	
	maxIdle := cfg.MaxIdleConns
	if maxIdle == 0 {
		maxIdle = 5
	}
	conn.SetMaxIdleConns(maxIdle)
	
	maxLifetime := cfg.ConnMaxLifetime
	if maxLifetime == 0 {
		maxLifetime = 5 * time.Minute
	}
	conn.SetConnMaxLifetime(maxLifetime)
	
	// Test the connection
	if err := conn.Ping(); err != nil {
		return nil, fmt.Errorf("failed to ping database: %w", err)
	}
	
	return &DB{conn: conn}, nil
}

// Close closes the database connection
func (db *DB) Close() error {
	return db.conn.Close()
}

// Migrate runs database migrations
func (db *DB) Migrate() error {
	// TODO: Implement migration logic
	return nil
}