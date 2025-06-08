package middleware

import (
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/lspecian/ovncp/internal/metrics"
)

// Metrics middleware for recording HTTP metrics
func Metrics() gin.HandlerFunc {
	return func(c *gin.Context) {
		// Skip metrics endpoint to avoid recursion
		if c.Request.URL.Path == "/metrics" {
			c.Next()
			return
		}

		// Record request start time
		start := time.Now()

		// Get request size
		requestSize := c.Request.ContentLength
		if requestSize < 0 {
			requestSize = 0
		}

		// Process request
		c.Next()

		// Calculate duration
		duration := time.Since(start).Seconds()

		// Get response status
		status := strconv.Itoa(c.Writer.Status())

		// Get response size
		responseSize := c.Writer.Size()
		if responseSize < 0 {
			responseSize = 0
		}

		// Normalize endpoint for metrics (remove IDs)
		endpoint := normalizeEndpoint(c.Request.URL.Path)

		// Record metrics
		metrics.RecordHTTPRequest(
			c.Request.Method,
			endpoint,
			status,
			duration,
			requestSize,
			int64(responseSize),
		)
	}
}

// normalizeEndpoint removes dynamic parts from URL paths for consistent metrics
func normalizeEndpoint(path string) string {
	// Common patterns to normalize
	patterns := map[string]string{
		// API v1 endpoints
		"/api/v1/switches/":        "/api/v1/switches/:id",
		"/api/v1/routers/":         "/api/v1/routers/:id",
		"/api/v1/ports/":           "/api/v1/ports/:id",
		"/api/v1/acls/":            "/api/v1/acls/:id",
		"/api/v1/load-balancers/":  "/api/v1/load-balancers/:id",
		"/api/v1/nat/":             "/api/v1/nat/:id",
		"/api/v1/dhcp/":            "/api/v1/dhcp/:id",
		"/api/v1/address-sets/":    "/api/v1/address-sets/:id",
		"/api/v1/port-groups/":     "/api/v1/port-groups/:id",
		"/api/v1/logical-flows/":   "/api/v1/logical-flows/:id",
	}

	// Check if path starts with any pattern
	for prefix, normalized := range patterns {
		if len(path) > len(prefix) && path[:len(prefix)] == prefix {
			return normalized
		}
	}

	// Return original path if no pattern matches
	return path
}

// RecordOVNMetrics is a helper to record OVN operation metrics
func RecordOVNMetrics(operation, resource string) func(error) {
	start := time.Now()
	return func(err error) {
		duration := time.Since(start).Seconds()
		status := "success"
		if err != nil {
			status = "error"
			metrics.RecordError("ovn", err.Error())
		}
		metrics.RecordOVNOperation(operation, resource, status, duration)
	}
}

// RecordDBMetrics is a helper to record database operation metrics
func RecordDBMetrics(queryType, table string) func(error) {
	start := time.Now()
	return func(err error) {
		duration := time.Since(start).Seconds()
		status := "success"
		if err != nil {
			status = "error"
			metrics.RecordError("database", err.Error())
		}
		metrics.RecordDBQuery(queryType, table, status, duration)
	}
}