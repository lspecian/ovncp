package middleware

import (
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/lspecian/ovncp/internal/logging"
	"go.opentelemetry.io/otel/trace"
	"go.uber.org/zap"
)

// Logging middleware for structured request logging
func Logging(logger *logging.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		// Start timer
		start := time.Now()
		path := c.Request.URL.Path
		raw := c.Request.URL.RawQuery

		// Process request
		c.Next()

		// Calculate latency
		latency := time.Since(start)

		// Get trace ID if available
		var traceID string
		if span := trace.SpanFromContext(c.Request.Context()); span.SpanContext().IsValid() {
			traceID = span.SpanContext().TraceID().String()
		}

		// Get user info if available
		var userID, userEmail string
		if user, exists := c.Get("user"); exists {
			if u, ok := user.(map[string]interface{}); ok {
				if id, ok := u["id"].(string); ok {
					userID = id
				}
				if email, ok := u["email"].(string); ok {
					userEmail = email
				}
			}
		}

		// Build fields
		fields := []zap.Field{
			zap.Int("status", c.Writer.Status()),
			zap.String("method", c.Request.Method),
			zap.String("path", path),
			zap.String("ip", c.ClientIP()),
			zap.Duration("latency", latency),
			zap.String("user_agent", c.Request.UserAgent()),
		}

		if raw != "" {
			fields = append(fields, zap.String("query", raw))
		}

		if traceID != "" {
			fields = append(fields, zap.String("trace_id", traceID))
		}

		if userID != "" {
			fields = append(fields, zap.String("user_id", userID))
		}

		if userEmail != "" {
			fields = append(fields, zap.String("user_email", userEmail))
		}

		// Add error if exists
		if len(c.Errors) > 0 {
			// Log all errors
			for _, e := range c.Errors {
				fields = append(fields, zap.Error(e.Err))
			}
		}

		// Log based on status code
		statusCode := c.Writer.Status()
		switch {
		case statusCode >= 500:
			logger.Error("Server error", fields...)
		case statusCode >= 400:
			logger.Warn("Client error", fields...)
		case statusCode >= 300:
			logger.Info("Redirection", fields...)
		default:
			logger.Info("Request completed", fields...)
		}
	}
}

// ErrorLogger returns a gin error logger that uses our structured logger
func ErrorLogger(logger *logging.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Next()

		// Log any errors that occurred during request processing
		for _, err := range c.Errors {
			// Get trace ID if available
			var traceID string
			if span := trace.SpanFromContext(c.Request.Context()); span.SpanContext().IsValid() {
				traceID = span.SpanContext().TraceID().String()
			}

			fields := []zap.Field{
				zap.Error(err.Err),
				zap.Uint64("type", uint64(err.Type)),
				zap.String("method", c.Request.Method),
				zap.String("path", c.Request.URL.Path),
			}

			if traceID != "" {
				fields = append(fields, zap.String("trace_id", traceID))
			}

			// Add meta data if available
			if err.Meta != nil {
				fields = append(fields, zap.Any("meta", err.Meta))
			}

			logger.Error("Request error", fields...)
		}
	}
}

// GetLoggerFromContext gets logger from gin context
func GetLoggerFromContext(c *gin.Context) *logging.Logger {
	if logger, exists := c.Get("logger"); exists {
		return logger.(*logging.Logger)
	}
	// Return a nop logger if not found
	return logging.NewNopLogger()
}

// WithLogger adds a logger to the gin context
func WithLogger(logger *logging.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		// Add trace ID if available
		if span := trace.SpanFromContext(c.Request.Context()); span.SpanContext().IsValid() {
			logger = logger.WithTraceID(span.SpanContext().TraceID().String())
		}

		// Add user info if available
		if user, exists := c.Get("user"); exists {
			if u, ok := user.(map[string]interface{}); ok {
				userID := ""
				userEmail := ""
				if id, ok := u["id"].(string); ok {
					userID = id
				}
				if email, ok := u["email"].(string); ok {
					userEmail = email
				}
				logger = logger.WithUser(userID, userEmail)
			}
		}

		c.Set("logger", logger)
		c.Next()
	}
}

// LoggerConfig defines the config for Logger middleware
type LoggerConfig struct {
	Logger       *zap.Logger
	SkipPaths    []string
	SkipPatterns []string
}

// LoggerWithConfig returns a gin.HandlerFunc (middleware) with config
func LoggerWithConfig(cfg LoggerConfig) gin.HandlerFunc {
	return func(c *gin.Context) {
		// Skip logging for certain paths
		for _, path := range cfg.SkipPaths {
			if c.Request.URL.Path == path {
				c.Next()
				return
			}
		}

		// Skip logging for patterns
		for _, pattern := range cfg.SkipPatterns {
			if strings.Contains(c.Request.URL.Path, pattern) {
				c.Next()
				return
			}
		}

		// Use the logging middleware
		start := time.Now()
		path := c.Request.URL.Path
		raw := c.Request.URL.RawQuery

		// Process request
		c.Next()

		// Log only errors or slow requests
		latency := time.Since(start)
		if c.Writer.Status() >= 400 || latency > time.Second {
			if raw != "" {
				path = path + "?" + raw
			}

			cfg.Logger.Info("Request",
				zap.String("method", c.Request.Method),
				zap.String("path", path),
				zap.Int("status", c.Writer.Status()),
				zap.Duration("latency", latency),
				zap.String("client_ip", c.ClientIP()),
				zap.String("user_agent", c.Request.UserAgent()),
				zap.String("request_id", c.GetString("request_id")),
			)
		}
	}
}