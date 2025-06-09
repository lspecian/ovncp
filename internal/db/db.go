package db

import (
	"database/sql"
	"embed"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/lspecian/ovncp/internal/config"
	_ "github.com/lib/pq"
	_ "github.com/mattn/go-sqlite3"
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
	var conn *sql.DB
	var err error

	switch cfg.Type {
	case "sqlite", "sqlite3":
		// Default to a local file in the data directory
		dbPath := cfg.Name
		if dbPath == "" || dbPath == "ovncp" {
			dbPath = "ovncp.db"
		}
		// Ensure the directory exists
		if dir := filepath.Dir(dbPath); dir != "." && dir != "" {
			os.MkdirAll(dir, 0755)
		}
		conn, err = sql.Open("sqlite3", dbPath)
	case "memory":
		// In-memory SQLite for testing
		conn, err = sql.Open("sqlite3", ":memory:")
	default: // postgres
		dsn := fmt.Sprintf("host=%s port=%s user=%s password=%s dbname=%s sslmode=%s",
			cfg.Host, cfg.Port, cfg.User, cfg.Password, cfg.Name, cfg.SSLMode)
		conn, err = sql.Open("postgres", dsn)
	}

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

//go:embed migrations/*.sql
var migrationFS embed.FS

// Migrate runs database migrations
func (db *DB) Migrate() error {
	migrationFiles := []string{
		"001_create_users_table.up.sql",
		"002_create_sessions_table.up.sql",
	}

	for _, file := range migrationFiles {
		// Try embedded migrations first
		content, err := migrationFS.ReadFile(fmt.Sprintf("migrations/%s", file))
		if err != nil {
			// Fallback to filesystem for development
			content, err = os.ReadFile(fmt.Sprintf("/app/migrations/%s", file))
			if err != nil {
				// Skip if file doesn't exist
				if os.IsNotExist(err) {
					continue
				}
				return fmt.Errorf("failed to read migration file %s: %w", file, err)
			}
		}

		// Adapt SQL for SQLite if needed
		sqlContent := string(content)
		if db.IsSQLite() {
			sqlContent = adaptPostgreSQLToSQLite(sqlContent)
		}

		if _, err := db.conn.Exec(sqlContent); err != nil {
			// Ignore errors if tables/indexes already exist
			if !strings.Contains(err.Error(), "already exists") && !strings.Contains(err.Error(), "duplicate") {
				return fmt.Errorf("failed to execute migration %s: %w", file, err)
			}
		}
	}

	return nil
}

// IsSQLite returns true if using SQLite database
func (db *DB) IsSQLite() bool {
	// Check driver name from connection
	return db.conn != nil && strings.Contains(fmt.Sprintf("%T", db.conn.Driver()), "sqlite")
}

// adaptPostgreSQLToSQLite converts PostgreSQL-specific syntax to SQLite
func adaptPostgreSQLToSQLite(sql string) string {
	// For SQLite, we'll use TEXT for UUID and generate them in the application
	sql = strings.ReplaceAll(sql, "UUID PRIMARY KEY DEFAULT gen_random_uuid()", "TEXT PRIMARY KEY")
	sql = strings.ReplaceAll(sql, "UUID", "TEXT")
	sql = strings.ReplaceAll(sql, "TIMESTAMP WITH TIME ZONE", "DATETIME")
	sql = strings.ReplaceAll(sql, "SERIAL", "INTEGER")
	sql = strings.ReplaceAll(sql, "BIGSERIAL", "INTEGER")
	sql = strings.ReplaceAll(sql, "BOOLEAN", "INTEGER")
	sql = strings.ReplaceAll(sql, "true", "1")
	sql = strings.ReplaceAll(sql, "false", "0")
	
	// Remove PostgreSQL-specific function and trigger definitions
	lines := strings.Split(sql, "\n")
	var result []string
	skipBlock := false
	
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		
		// Skip function and trigger blocks
		if strings.Contains(trimmed, "CREATE OR REPLACE FUNCTION") ||
			strings.Contains(trimmed, "CREATE TRIGGER") {
			skipBlock = true
			continue
		}
		
		// End of function block
		if skipBlock && (strings.HasPrefix(trimmed, "$$ language") || strings.HasPrefix(trimmed, "$$;")) {
			skipBlock = false
			continue
		}
		
		if !skipBlock {
			// Replace CURRENT_TIMESTAMP
			line = strings.ReplaceAll(line, "CURRENT_TIMESTAMP", "datetime('now')")
			result = append(result, line)
		}
	}
	
	return strings.Join(result, "\n")
}

// DB returns the underlying sql.DB connection
func (db *DB) DB() *sql.DB {
	return db.conn
}