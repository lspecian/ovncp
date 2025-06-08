package db

import (
	"context"
	"fmt"
	"strings"
	"time"

	"gorm.io/gorm"
)

// QueryOptimizer provides optimized database queries
type QueryOptimizer struct {
	db *gorm.DB
}

// NewQueryOptimizer creates a new query optimizer
func NewQueryOptimizer(db *gorm.DB) *QueryOptimizer {
	return &QueryOptimizer{db: db}
}

// OptimizedQuery represents an optimized query builder
type OptimizedQuery struct {
	query      *gorm.DB
	selectCols []string
	indexes    []string
	hints      []string
}

// NewOptimizedQuery creates a new optimized query
func (q *QueryOptimizer) NewOptimizedQuery(model interface{}) *OptimizedQuery {
	return &OptimizedQuery{
		query: q.db.Model(model),
	}
}

// Select specifies columns to select (reduces data transfer)
func (oq *OptimizedQuery) Select(cols ...string) *OptimizedQuery {
	oq.selectCols = cols
	if len(cols) > 0 {
		oq.query = oq.query.Select(cols)
	}
	return oq
}

// UseIndex forces the use of specific indexes
func (oq *OptimizedQuery) UseIndex(indexes ...string) *OptimizedQuery {
	oq.indexes = indexes
	for range indexes {
		// TODO: Use clause.Index when available
		// oq.query = oq.query.Clauses(clause.Index{Name: idx})
	}
	return oq
}

// WithHint adds query hints
func (oq *OptimizedQuery) WithHint(hints ...string) *OptimizedQuery {
	oq.hints = hints
	return oq
}

// Paginate adds optimized pagination
func (oq *OptimizedQuery) Paginate(page, pageSize int) *OptimizedQuery {
	offset := (page - 1) * pageSize
	oq.query = oq.query.Offset(offset).Limit(pageSize)
	return oq
}

// Build returns the final query
func (oq *OptimizedQuery) Build() *gorm.DB {
	return oq.query
}

// Resource-specific optimized queries

// FindResourceByUUID finds a resource by UUID with minimal columns
func (q *QueryOptimizer) FindResourceByUUID(ctx context.Context, uuid string, dest interface{}) error {
	return q.db.WithContext(ctx).
		Where("ovn_uuid = ?", uuid).
		First(dest).Error
}

// FindResourcesByOwner finds resources owned by a user with pagination
func (q *QueryOptimizer) FindResourcesByOwner(ctx context.Context, ownerID string, resourceType string, page, pageSize int, dest interface{}) error {
	return q.db.WithContext(ctx).
		Where("owner_id = ? AND type = ?", ownerID, resourceType).
		Order("created_at DESC").
		Offset((page - 1) * pageSize).
		Limit(pageSize).
		Find(dest).Error
}

// CountResourcesByType counts resources efficiently
func (q *QueryOptimizer) CountResourcesByType(ctx context.Context, resourceType string) (int64, error) {
	var count int64
	err := q.db.WithContext(ctx).
		Model(&Resource{}).
		Where("type = ?", resourceType).
		Count(&count).Error
	return count, err
}

// BatchCreateResources creates multiple resources efficiently
func (q *QueryOptimizer) BatchCreateResources(ctx context.Context, resources []Resource) error {
	if len(resources) == 0 {
		return nil
	}
	
	// Use CreateInBatches for large datasets
	return q.db.WithContext(ctx).
		CreateInBatches(resources, 100).Error
}

// BatchUpdateResources updates multiple resources efficiently
func (q *QueryOptimizer) BatchUpdateResources(ctx context.Context, ids []string, updates map[string]interface{}) error {
	if len(ids) == 0 {
		return nil
	}
	
	// Add updated_at timestamp
	updates["updated_at"] = time.Now()
	
	return q.db.WithContext(ctx).
		Model(&Resource{}).
		Where("id IN ?", ids).
		Updates(updates).Error
}

// Audit log optimizations

// GetRecentAuditLogs retrieves recent audit logs efficiently
func (q *QueryOptimizer) GetRecentAuditLogs(ctx context.Context, limit int) ([]AuditLog, error) {
	var logs []AuditLog
	
	err := q.db.WithContext(ctx).
		Select("id", "timestamp", "user_id", "action", "resource_type", "resource_id", "status_code").
		Order("timestamp DESC").
		Limit(limit).
		Find(&logs).Error
		
	return logs, err
}

// GetAuditLogsByUser retrieves audit logs for a specific user
func (q *QueryOptimizer) GetAuditLogsByUser(ctx context.Context, userID string, startTime, endTime time.Time, limit int) ([]AuditLog, error) {
	var logs []AuditLog
	
	query := q.db.WithContext(ctx).
		Where("user_id = ?", userID)
		
	if !startTime.IsZero() {
		query = query.Where("timestamp >= ?", startTime)
	}
	
	if !endTime.IsZero() {
		query = query.Where("timestamp <= ?", endTime)
	}
	
	err := query.
		Order("timestamp DESC").
		Limit(limit).
		Find(&logs).Error
		
	return logs, err
}

// Metrics and analytics queries

// GetResourceMetrics retrieves resource counts by type
func (q *QueryOptimizer) GetResourceMetrics(ctx context.Context) (map[string]int64, error) {
	type result struct {
		Type  string
		Count int64
	}
	
	var results []result
	err := q.db.WithContext(ctx).
		Model(&Resource{}).
		Select("type, COUNT(*) as count").
		Group("type").
		Scan(&results).Error
		
	if err != nil {
		return nil, err
	}
	
	metrics := make(map[string]int64)
	for _, r := range results {
		metrics[r.Type] = r.Count
	}
	
	return metrics, nil
}

// GetUserActivityMetrics retrieves user activity metrics
func (q *QueryOptimizer) GetUserActivityMetrics(ctx context.Context, since time.Time) (map[string]interface{}, error) {
	type activityResult struct {
		UserID string
		Count  int64
	}
	
	var results []activityResult
	err := q.db.WithContext(ctx).
		Model(&AuditLog{}).
		Select("user_id, COUNT(*) as count").
		Where("timestamp >= ?", since).
		Group("user_id").
		Order("count DESC").
		Limit(10).
		Scan(&results).Error
		
	if err != nil {
		return nil, err
	}
	
	// Get total actions
	var totalActions int64
	err = q.db.WithContext(ctx).
		Model(&AuditLog{}).
		Where("timestamp >= ?", since).
		Count(&totalActions).Error
		
	if err != nil {
		return nil, err
	}
	
	metrics := map[string]interface{}{
		"top_users":     results,
		"total_actions": totalActions,
		"since":         since,
	}
	
	return metrics, nil
}

// Query optimization helpers

// ExplainQuery returns the query execution plan
func (q *QueryOptimizer) ExplainQuery(query *gorm.DB) ([]map[string]interface{}, error) {
	var results []map[string]interface{}
	
	// Get the SQL and arguments
	sql := query.ToSQL(func(tx *gorm.DB) *gorm.DB {
		return tx
	})
	
	// Run EXPLAIN ANALYZE
	explainSQL := fmt.Sprintf("EXPLAIN ANALYZE %s", sql)
	rows, err := q.db.Raw(explainSQL).Rows()
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	
	// Parse results
	for rows.Next() {
		var plan string
		if err := rows.Scan(&plan); err != nil {
			return nil, err
		}
		results = append(results, map[string]interface{}{
			"plan": plan,
		})
	}
	
	return results, nil
}

// AnalyzeTable updates table statistics for query optimization
func (q *QueryOptimizer) AnalyzeTable(tableName string) error {
	return q.db.Exec(fmt.Sprintf("ANALYZE %s", tableName)).Error
}

// VacuumTable performs vacuum on a table
func (q *QueryOptimizer) VacuumTable(tableName string, full bool) error {
	vacuumType := "VACUUM"
	if full {
		vacuumType = "VACUUM FULL"
	}
	return q.db.Exec(fmt.Sprintf("%s %s", vacuumType, tableName)).Error
}

// CreateIndexes creates optimized indexes for common queries
func (q *QueryOptimizer) CreateIndexes() error {
	indexes := []string{
		// Resources table indexes
		"CREATE INDEX IF NOT EXISTS idx_resources_type_created ON resources(type, created_at DESC)",
		"CREATE INDEX IF NOT EXISTS idx_resources_owner_type ON resources(owner_id, type)",
		"CREATE INDEX IF NOT EXISTS idx_resources_ovn_uuid ON resources(ovn_uuid)",
		"CREATE INDEX IF NOT EXISTS idx_resources_name ON resources(name)",
		
		// Audit logs table indexes
		"CREATE INDEX IF NOT EXISTS idx_audit_logs_user_timestamp ON audit_logs(user_id, timestamp DESC)",
		"CREATE INDEX IF NOT EXISTS idx_audit_logs_resource ON audit_logs(resource_type, resource_id)",
		"CREATE INDEX IF NOT EXISTS idx_audit_logs_action ON audit_logs(action)",
		"CREATE INDEX IF NOT EXISTS idx_audit_logs_timestamp ON audit_logs(timestamp DESC)",
		
		// Users table indexes
		"CREATE INDEX IF NOT EXISTS idx_users_email ON users(email)",
		"CREATE INDEX IF NOT EXISTS idx_users_provider ON users(provider, provider_id)",
		
		// Sessions table indexes (if exists)
		"CREATE INDEX IF NOT EXISTS idx_sessions_user_id ON sessions(user_id)",
		"CREATE INDEX IF NOT EXISTS idx_sessions_expires_at ON sessions(expires_at)",
	}
	
	for _, idx := range indexes {
		if err := q.db.Exec(idx).Error; err != nil {
			// Log error but continue with other indexes
			fmt.Printf("Failed to create index: %v\n", err)
		}
	}
	
	return nil
}

// OptimizeDatabase runs various database optimizations
func (q *QueryOptimizer) OptimizeDatabase(ctx context.Context) error {
	// Update statistics
	tables := []string{"resources", "audit_logs", "users"}
	for _, table := range tables {
		if err := q.AnalyzeTable(table); err != nil {
			return fmt.Errorf("failed to analyze table %s: %w", table, err)
		}
	}
	
	// Create/update indexes
	if err := q.CreateIndexes(); err != nil {
		return fmt.Errorf("failed to create indexes: %w", err)
	}
	
	return nil
}

// Connection pool optimization

// SetConnectionPoolConfig optimizes connection pool settings
func (q *QueryOptimizer) SetConnectionPoolConfig(maxOpen, maxIdle int, maxLifetime, idleTimeout time.Duration) {
	sqlDB, _ := q.db.DB()
	
	// Set maximum number of open connections
	sqlDB.SetMaxOpenConns(maxOpen)
	
	// Set maximum number of idle connections
	sqlDB.SetMaxIdleConns(maxIdle)
	
	// Set maximum lifetime of a connection
	sqlDB.SetConnMaxLifetime(maxLifetime)
	
	// Set maximum idle time
	sqlDB.SetConnMaxIdleTime(idleTimeout)
}

// GetConnectionStats returns database connection pool statistics
func (q *QueryOptimizer) GetConnectionStats() (map[string]interface{}, error) {
	sqlDB, err := q.db.DB()
	if err != nil {
		return nil, err
	}
	
	stats := sqlDB.Stats()
	
	return map[string]interface{}{
		"open_connections":  stats.OpenConnections,
		"in_use":           stats.InUse,
		"idle":             stats.Idle,
		"wait_count":       stats.WaitCount,
		"wait_duration":    stats.WaitDuration,
		"max_idle_closed":  stats.MaxIdleClosed,
		"max_lifetime_closed": stats.MaxLifetimeClosed,
	}, nil
}

// Batch operation helpers

// ChunkSlice splits a slice into chunks for batch processing
func ChunkSlice(slice []string, chunkSize int) [][]string {
	var chunks [][]string
	for i := 0; i < len(slice); i += chunkSize {
		end := i + chunkSize
		if end > len(slice) {
			end = len(slice)
		}
		chunks = append(chunks, slice[i:end])
	}
	return chunks
}

// BuildBulkInsertQuery builds an optimized bulk insert query
func BuildBulkInsertQuery(table string, columns []string, values [][]interface{}) (string, []interface{}) {
	if len(values) == 0 {
		return "", nil
	}
	
	// Build column list
	columnList := strings.Join(columns, ", ")
	
	// Build placeholders
	var placeholders []string
	var args []interface{}
	
	for i, row := range values {
		var rowPlaceholders []string
		for j := range columns {
			placeholder := fmt.Sprintf("$%d", i*len(columns)+j+1)
			rowPlaceholders = append(rowPlaceholders, placeholder)
			args = append(args, row[j])
		}
		placeholders = append(placeholders, fmt.Sprintf("(%s)", strings.Join(rowPlaceholders, ", ")))
	}
	
	query := fmt.Sprintf(
		"INSERT INTO %s (%s) VALUES %s",
		table,
		columnList,
		strings.Join(placeholders, ", "),
	)
	
	return query, args
}