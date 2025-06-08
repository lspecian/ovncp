package logging

import (
	"os"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

// Config holds logging configuration
type Config struct {
	Level      string // debug, info, warn, error
	Format     string // json, console
	OutputPath string // stdout, stderr, or file path
}

// Logger wraps zap logger with additional functionality
type Logger struct {
	*zap.Logger
}

// NewLogger creates a new structured logger
func NewLogger(cfg Config) (*Logger, error) {
	// Parse log level
	level, err := zapcore.ParseLevel(cfg.Level)
	if err != nil {
		return nil, err
	}

	// Create encoder config
	encoderConfig := zapcore.EncoderConfig{
		TimeKey:        "timestamp",
		LevelKey:       "level",
		NameKey:        "logger",
		CallerKey:      "caller",
		FunctionKey:    zapcore.OmitKey,
		MessageKey:     "message",
		StacktraceKey:  "stacktrace",
		LineEnding:     zapcore.DefaultLineEnding,
		EncodeLevel:    zapcore.LowercaseLevelEncoder,
		EncodeTime:     zapcore.ISO8601TimeEncoder,
		EncodeDuration: zapcore.SecondsDurationEncoder,
		EncodeCaller:   zapcore.ShortCallerEncoder,
	}

	// Create encoder based on format
	var encoder zapcore.Encoder
	switch cfg.Format {
	case "console":
		encoderConfig.EncodeLevel = zapcore.CapitalColorLevelEncoder
		encoder = zapcore.NewConsoleEncoder(encoderConfig)
	default: // json
		encoder = zapcore.NewJSONEncoder(encoderConfig)
	}

	// Create output writer
	var writer zapcore.WriteSyncer
	switch cfg.OutputPath {
	case "stdout":
		writer = zapcore.AddSync(os.Stdout)
	case "stderr":
		writer = zapcore.AddSync(os.Stderr)
	default:
		file, err := os.OpenFile(cfg.OutputPath, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0644)
		if err != nil {
			return nil, err
		}
		writer = zapcore.AddSync(file)
	}

	// Create core
	core := zapcore.NewCore(encoder, writer, level)

	// Create logger with additional options
	logger := zap.New(core,
		zap.AddCaller(),
		zap.AddStacktrace(zapcore.ErrorLevel),
		zap.AddCallerSkip(1),
	)

	return &Logger{Logger: logger}, nil
}

// With creates a child logger with additional fields
func (l *Logger) With(fields ...zap.Field) *Logger {
	return &Logger{Logger: l.Logger.With(fields...)}
}

// WithTraceID adds trace ID to logger context
func (l *Logger) WithTraceID(traceID string) *Logger {
	return l.With(zap.String("trace_id", traceID))
}

// WithUser adds user context to logger
func (l *Logger) WithUser(userID, userEmail string) *Logger {
	return l.With(
		zap.String("user_id", userID),
		zap.String("user_email", userEmail),
	)
}

// WithResource adds resource context to logger
func (l *Logger) WithResource(resourceType, resourceID, resourceName string) *Logger {
	fields := []zap.Field{
		zap.String("resource_type", resourceType),
	}
	if resourceID != "" {
		fields = append(fields, zap.String("resource_id", resourceID))
	}
	if resourceName != "" {
		fields = append(fields, zap.String("resource_name", resourceName))
	}
	return l.With(fields...)
}

// WithOperation adds operation context to logger
func (l *Logger) WithOperation(operation string) *Logger {
	return l.With(zap.String("operation", operation))
}

// WithError adds error context to logger
func (l *Logger) WithError(err error) *Logger {
	return l.With(zap.Error(err))
}

// LogHTTPRequest logs HTTP request details
func (l *Logger) LogHTTPRequest(method, path string, statusCode int, duration float64, fields ...zap.Field) {
	baseFields := []zap.Field{
		zap.String("http_method", method),
		zap.String("http_path", path),
		zap.Int("http_status_code", statusCode),
		zap.Float64("duration_seconds", duration),
	}
	allFields := append(baseFields, fields...)

	if statusCode >= 500 {
		l.Error("HTTP request failed", allFields...)
	} else if statusCode >= 400 {
		l.Warn("HTTP request client error", allFields...)
	} else {
		l.Info("HTTP request completed", allFields...)
	}
}

// LogOVNOperation logs OVN operation details
func (l *Logger) LogOVNOperation(operation, resource string, success bool, duration float64, fields ...zap.Field) {
	baseFields := []zap.Field{
		zap.String("ovn_operation", operation),
		zap.String("ovn_resource", resource),
		zap.Bool("success", success),
		zap.Float64("duration_seconds", duration),
	}
	allFields := append(baseFields, fields...)

	if success {
		l.Info("OVN operation completed", allFields...)
	} else {
		l.Error("OVN operation failed", allFields...)
	}
}

// LogDBQuery logs database query details
func (l *Logger) LogDBQuery(queryType, table string, success bool, duration float64, fields ...zap.Field) {
	baseFields := []zap.Field{
		zap.String("db_query_type", queryType),
		zap.String("db_table", table),
		zap.Bool("success", success),
		zap.Float64("duration_seconds", duration),
	}
	allFields := append(baseFields, fields...)

	if success {
		l.Debug("Database query completed", allFields...)
	} else {
		l.Error("Database query failed", allFields...)
	}
}

// LogTransaction logs transaction details
func (l *Logger) LogTransaction(transactionID string, operationCount int, success bool, fields ...zap.Field) {
	baseFields := []zap.Field{
		zap.String("transaction_id", transactionID),
		zap.Int("operation_count", operationCount),
		zap.Bool("success", success),
	}
	allFields := append(baseFields, fields...)

	if success {
		l.Info("Transaction completed", allFields...)
	} else {
		l.Error("Transaction failed", allFields...)
	}
}

// LogAudit logs audit events
func (l *Logger) LogAudit(userID, action, resourceType, resourceID string, success bool, fields ...zap.Field) {
	baseFields := []zap.Field{
		zap.String("audit_user_id", userID),
		zap.String("audit_action", action),
		zap.String("audit_resource_type", resourceType),
		zap.String("audit_resource_id", resourceID),
		zap.Bool("audit_success", success),
	}
	allFields := append(baseFields, fields...)

	l.Info("Audit event", allFields...)
}

// Helper functions for common logging patterns

// NewNopLogger returns a no-op logger for testing
func NewNopLogger() *Logger {
	return &Logger{Logger: zap.NewNop()}
}

// NewDevelopmentLogger returns a development logger with console output
func NewDevelopmentLogger() (*Logger, error) {
	return NewLogger(Config{
		Level:      "debug",
		Format:     "console",
		OutputPath: "stdout",
	})
}

// NewProductionLogger returns a production logger with JSON output
func NewProductionLogger(level string) (*Logger, error) {
	return NewLogger(Config{
		Level:      level,
		Format:     "json",
		OutputPath: "stdout",
	})
}

// Field constructors for common fields
func UserField(userID string) zap.Field {
	return zap.String("user_id", userID)
}

func ResourceField(resourceType, resourceID string) []zap.Field {
	return []zap.Field{
		zap.String("resource_type", resourceType),
		zap.String("resource_id", resourceID),
	}
}

func OperationField(operation string) zap.Field {
	return zap.String("operation", operation)
}

func DurationField(duration float64) zap.Field {
	return zap.Float64("duration_seconds", duration)
}

func TraceField(traceID string) zap.Field {
	return zap.String("trace_id", traceID)
}

func ErrorField(err error) zap.Field {
	return zap.Error(err)
}