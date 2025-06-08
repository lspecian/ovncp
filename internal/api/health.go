package api

import (
	"context"
	"net/http"
	"runtime"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/lspecian/ovncp/internal/db"
	"github.com/lspecian/ovncp/pkg/ovn"
	"github.com/redis/go-redis/v9"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

// HealthCheck represents the health status of a component
type HealthCheck struct {
	Status    string                 `json:"status"`
	Timestamp time.Time              `json:"timestamp"`
	Details   map[string]interface{} `json:"details,omitempty"`
}

// HealthService handles health checks
type HealthService struct {
	db        *gorm.DB
	ovnClient *ovn.Client
	redis     *redis.Client
	logger    *zap.Logger
}

// NewHealthService creates a new health service
func NewHealthService(db *gorm.DB, ovnClient *ovn.Client, redis *redis.Client, logger *zap.Logger) *HealthService {
	return &HealthService{
		db:        db,
		ovnClient: ovnClient,
		redis:     redis,
		logger:    logger,
	}
}

// RegisterHealthRoutes registers health check routes
func (h *HealthService) RegisterHealthRoutes(router *gin.Engine) {
	health := router.Group("/health")
	{
		health.GET("/", h.handleHealthCheck)
		health.GET("/live", h.handleLivenessCheck)
		health.GET("/ready", h.handleReadinessCheck)
		health.GET("/startup", h.handleStartupCheck)
	}
}

// handleHealthCheck performs a comprehensive health check
func (h *HealthService) handleHealthCheck(c *gin.Context) {
	ctx, cancel := context.WithTimeout(c.Request.Context(), 5*time.Second)
	defer cancel()

	health := &HealthCheck{
		Status:    "healthy",
		Timestamp: time.Now(),
		Details:   make(map[string]interface{}),
	}

	// Check database
	dbHealth := h.checkDatabase(ctx)
	health.Details["database"] = dbHealth
	if dbHealth["status"] != "healthy" {
		health.Status = "unhealthy"
	}

	// Check OVN
	ovnHealth := h.checkOVN(ctx)
	health.Details["ovn"] = ovnHealth
	if ovnHealth["status"] != "healthy" {
		health.Status = "unhealthy"
	}

	// Check Redis
	redisHealth := h.checkRedis(ctx)
	health.Details["redis"] = redisHealth
	if redisHealth["status"] != "healthy" {
		health.Status = "degraded" // Redis failure doesn't make the service unhealthy
	}

	// System metrics
	health.Details["system"] = h.getSystemMetrics()

	statusCode := http.StatusOK
	if health.Status == "unhealthy" {
		statusCode = http.StatusServiceUnavailable
	}

	c.JSON(statusCode, health)
}

// handleLivenessCheck checks if the service is alive
func (h *HealthService) handleLivenessCheck(c *gin.Context) {
	// Simple check - if we can handle requests, we're alive
	c.JSON(http.StatusOK, gin.H{
		"status":    "alive",
		"timestamp": time.Now(),
	})
}

// handleReadinessCheck checks if the service is ready to handle requests
func (h *HealthService) handleReadinessCheck(c *gin.Context) {
	ctx, cancel := context.WithTimeout(c.Request.Context(), 3*time.Second)
	defer cancel()

	// Check critical dependencies
	ready := true
	details := make(map[string]interface{})

	// Check database
	if err := h.db.WithContext(ctx).Raw("SELECT 1").Error; err != nil {
		ready = false
		details["database"] = "not ready"
	} else {
		details["database"] = "ready"
	}

	// Check OVN
	if err := h.ovnClient.Ping(ctx); err != nil {
		ready = false
		details["ovn"] = "not ready"
	} else {
		details["ovn"] = "ready"
	}

	statusCode := http.StatusOK
	status := "ready"
	if !ready {
		statusCode = http.StatusServiceUnavailable
		status = "not ready"
	}

	c.JSON(statusCode, gin.H{
		"status":    status,
		"timestamp": time.Now(),
		"details":   details,
	})
}

// handleStartupCheck checks if the service has completed startup
func (h *HealthService) handleStartupCheck(c *gin.Context) {
	// Check if all components are initialized
	ctx, cancel := context.WithTimeout(c.Request.Context(), 10*time.Second)
	defer cancel()

	startup := true
	details := make(map[string]interface{})

	// Check database migrations
	var count int64
	if err := h.db.WithContext(ctx).Model(&db.Resource{}).Count(&count).Error; err != nil {
		startup = false
		details["database_migrations"] = "incomplete"
	} else {
		details["database_migrations"] = "complete"
	}

	// Check OVN connection
	if !h.ovnClient.IsConnected() {
		startup = false
		details["ovn_connection"] = "not established"
	} else {
		details["ovn_connection"] = "established"
	}

	statusCode := http.StatusOK
	status := "started"
	if !startup {
		statusCode = http.StatusServiceUnavailable
		status = "starting"
	}

	c.JSON(statusCode, gin.H{
		"status":    status,
		"timestamp": time.Now(),
		"details":   details,
	})
}

// checkDatabase performs database health check
func (h *HealthService) checkDatabase(ctx context.Context) map[string]interface{} {
	start := time.Now()
	status := "healthy"
	var err error

	// Test connection
	sqlDB, _ := h.db.DB()
	if sqlDB != nil {
		err = sqlDB.PingContext(ctx)
	}

	latency := time.Since(start).Milliseconds()

	result := map[string]interface{}{
		"status":  status,
		"latency": latency,
	}

	if err != nil {
		status = "unhealthy"
		result["status"] = status
		result["error"] = err.Error()
		return result
	}

	// Get connection pool stats
	if sqlDB != nil {
		stats := sqlDB.Stats()
		result["connections"] = map[string]interface{}{
			"open":            stats.OpenConnections,
			"in_use":          stats.InUse,
			"idle":            stats.Idle,
			"wait_count":      stats.WaitCount,
			"wait_duration":   stats.WaitDuration.Milliseconds(),
			"max_idle_closed": stats.MaxIdleClosed,
			"max_lifetime_closed": stats.MaxLifetimeClosed,
		}
	}

	return result
}

// checkOVN performs OVN health check
func (h *HealthService) checkOVN(ctx context.Context) map[string]interface{} {
	start := time.Now()
	status := "healthy"
	
	result := map[string]interface{}{
		"status": status,
	}

	// Check connection
	if !h.ovnClient.IsConnected() {
		result["status"] = "unhealthy"
		result["error"] = "not connected"
		return result
	}

	// Ping OVN
	err := h.ovnClient.Ping(ctx)
	latency := time.Since(start).Milliseconds()
	result["latency"] = latency

	if err != nil {
		result["status"] = "unhealthy"
		result["error"] = err.Error()
		return result
	}

	// Get connection info
	if connInfo := h.ovnClient.GetConnectionInfo(); connInfo != nil {
		result["connection"] = connInfo
	}

	return result
}

// checkRedis performs Redis health check
func (h *HealthService) checkRedis(ctx context.Context) map[string]interface{} {
	if h.redis == nil {
		return map[string]interface{}{
			"status": "disabled",
		}
	}

	start := time.Now()
	status := "healthy"

	result := map[string]interface{}{
		"status": status,
	}

	// Ping Redis
	err := h.redis.Ping(ctx).Err()
	latency := time.Since(start).Milliseconds()
	result["latency"] = latency

	if err != nil {
		result["status"] = "unhealthy"
		result["error"] = err.Error()
		return result
	}

	// Get Redis info
	info, err := h.redis.Info(ctx, "server", "clients", "memory", "stats").Result()
	if err == nil {
		// Parse some basic info
		result["info"] = parseRedisInfo(info)
	}

	return result
}

// getSystemMetrics returns system-level metrics
func (h *HealthService) getSystemMetrics() map[string]interface{} {
	metrics := make(map[string]interface{})

	// Get memory stats
	var m runtime.MemStats
	runtime.ReadMemStats(&m)
	
	metrics["memory"] = map[string]interface{}{
		"alloc_mb":       m.Alloc / 1024 / 1024,
		"total_alloc_mb": m.TotalAlloc / 1024 / 1024,
		"sys_mb":         m.Sys / 1024 / 1024,
		"num_gc":         m.NumGC,
		"gc_cpu_percent": m.GCCPUFraction * 100,
	}

	// Goroutines
	metrics["goroutines"] = runtime.NumGoroutine()

	// Uptime
	metrics["uptime_seconds"] = time.Since(startTime).Seconds()

	return metrics
}

// parseRedisInfo parses Redis INFO output
func parseRedisInfo(info string) map[string]interface{} {
	result := make(map[string]interface{})
	
	lines := strings.Split(info, "\r\n")
	for _, line := range lines {
		if strings.Contains(line, ":") {
			parts := strings.SplitN(line, ":", 2)
			key := strings.TrimSpace(parts[0])
			value := strings.TrimSpace(parts[1])
			
			// Parse some important metrics
			switch key {
			case "redis_version", "redis_mode", "process_id", "uptime_in_seconds",
			     "connected_clients", "used_memory_human", "total_connections_received",
			     "total_commands_processed", "instantaneous_ops_per_sec":
				result[key] = value
			}
		}
	}
	
	return result
}

var startTime = time.Now()

// HealthMiddleware adds health check headers
func HealthMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		// Add health check headers for load balancers
		c.Header("X-Health-Check", "1")
		c.Next()
	}
}