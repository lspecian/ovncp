package middleware

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/lspecian/ovncp/internal/db"
	"go.uber.org/zap"
)

// AuditEvent represents an audit log entry
type AuditEvent struct {
	ID            string                 `json:"id" db:"id"`
	Timestamp     time.Time              `json:"timestamp" db:"timestamp"`
	UserID        string                 `json:"user_id" db:"user_id"`
	UserEmail     string                 `json:"user_email" db:"user_email"`
	Action        string                 `json:"action" db:"action"`
	ResourceType  string                 `json:"resource_type" db:"resource_type"`
	ResourceID    string                 `json:"resource_id" db:"resource_id"`
	Method        string                 `json:"method" db:"method"`
	Path          string                 `json:"path" db:"path"`
	StatusCode    int                    `json:"status_code" db:"status_code"`
	IPAddress     string                 `json:"ip_address" db:"ip_address"`
	UserAgent     string                 `json:"user_agent" db:"user_agent"`
	RequestBody   json.RawMessage        `json:"request_body,omitempty" db:"request_body"`
	ResponseBody  json.RawMessage        `json:"response_body,omitempty" db:"response_body"`
	Error         string                 `json:"error,omitempty" db:"error"`
	Duration      time.Duration          `json:"duration" db:"duration"`
	Metadata      map[string]interface{} `json:"metadata,omitempty" db:"metadata"`
}

// AuditLogger interface for different audit backends
type AuditLogger interface {
	Log(event *AuditEvent) error
	Query(filter AuditFilter) ([]*AuditEvent, error)
}

// AuditFilter for querying audit logs
type AuditFilter struct {
	UserID       string
	ResourceType string
	ResourceID   string
	Action       string
	StartTime    time.Time
	EndTime      time.Time
	Limit        int
	Offset       int
}

// DatabaseAuditLogger stores audit logs in the database
type DatabaseAuditLogger struct {
	db     *db.DB
	logger *zap.Logger
}

// NewDatabaseAuditLogger creates a new database audit logger
func NewDatabaseAuditLogger(database *db.DB, logger *zap.Logger) *DatabaseAuditLogger {
	return &DatabaseAuditLogger{
		db:     database,
		logger: logger,
	}
}

// Log stores an audit event in the database
func (l *DatabaseAuditLogger) Log(event *AuditEvent) error {
	query := `
		INSERT INTO audit_logs (
			id, timestamp, user_id, user_email, action, resource_type,
			resource_id, method, path, status_code, ip_address, user_agent,
			request_body, response_body, error, duration, metadata
		) VALUES (
			$1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16, $17
		)`
	
	metadataJSON, err := json.Marshal(event.Metadata)
	if err != nil {
		return err
	}
	
	_, err = l.db.Exec(query,
		event.ID,
		event.Timestamp,
		event.UserID,
		event.UserEmail,
		event.Action,
		event.ResourceType,
		event.ResourceID,
		event.Method,
		event.Path,
		event.StatusCode,
		event.IPAddress,
		event.UserAgent,
		event.RequestBody,
		event.ResponseBody,
		event.Error,
		event.Duration,
		metadataJSON,
	)
	
	return err
}

// Query retrieves audit events based on filter criteria
func (l *DatabaseAuditLogger) Query(filter AuditFilter) ([]*AuditEvent, error) {
	query := `
		SELECT 
			id, timestamp, user_id, user_email, action, resource_type,
			resource_id, method, path, status_code, ip_address, user_agent,
			request_body, response_body, error, duration, metadata
		FROM audit_logs
		WHERE 1=1`
	
	args := []interface{}{}
	argCount := 0
	
	if filter.UserID != "" {
		argCount++
		query += " AND user_id = $" + string(argCount)
		args = append(args, filter.UserID)
	}
	
	if filter.ResourceType != "" {
		argCount++
		query += " AND resource_type = $" + string(argCount)
		args = append(args, filter.ResourceType)
	}
	
	if filter.ResourceID != "" {
		argCount++
		query += " AND resource_id = $" + string(argCount)
		args = append(args, filter.ResourceID)
	}
	
	if filter.Action != "" {
		argCount++
		query += " AND action = $" + string(argCount)
		args = append(args, filter.Action)
	}
	
	if !filter.StartTime.IsZero() {
		argCount++
		query += " AND timestamp >= $" + string(argCount)
		args = append(args, filter.StartTime)
	}
	
	if !filter.EndTime.IsZero() {
		argCount++
		query += " AND timestamp <= $" + string(argCount)
		args = append(args, filter.EndTime)
	}
	
	query += " ORDER BY timestamp DESC"
	
	if filter.Limit > 0 {
		argCount++
		query += " LIMIT $" + string(argCount)
		args = append(args, filter.Limit)
	}
	
	if filter.Offset > 0 {
		argCount++
		query += " OFFSET $" + string(argCount)
		args = append(args, filter.Offset)
	}
	
	rows, err := l.db.Query(query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	
	var events []*AuditEvent
	for rows.Next() {
		event := &AuditEvent{}
		var metadataJSON []byte
		
		err := rows.Scan(
			&event.ID,
			&event.Timestamp,
			&event.UserID,
			&event.UserEmail,
			&event.Action,
			&event.ResourceType,
			&event.ResourceID,
			&event.Method,
			&event.Path,
			&event.StatusCode,
			&event.IPAddress,
			&event.UserAgent,
			&event.RequestBody,
			&event.ResponseBody,
			&event.Error,
			&event.Duration,
			&metadataJSON,
		)
		if err != nil {
			return nil, err
		}
		
		if len(metadataJSON) > 0 {
			json.Unmarshal(metadataJSON, &event.Metadata)
		}
		
		events = append(events, event)
	}
	
	return events, nil
}

// AuditConfig holds audit middleware configuration
type AuditConfig struct {
	Enabled          bool
	Logger           AuditLogger
	LogRequestBody   bool
	LogResponseBody  bool
	MaxBodySize      int64
	ExcludePaths     []string
	SensitiveFields  []string // Fields to redact from logs
	IncludeResources []string // Resource types to audit
}

// responseWriter wraps gin.ResponseWriter to capture response body
type responseWriter struct {
	gin.ResponseWriter
	body *bytes.Buffer
}

func (w responseWriter) Write(b []byte) (int, error) {
	w.body.Write(b)
	return w.ResponseWriter.Write(b)
}

// Audit middleware for logging API operations
func Audit(cfg AuditConfig) gin.HandlerFunc {
	if !cfg.Enabled || cfg.Logger == nil {
		return func(c *gin.Context) { c.Next() }
	}
	
	// Compile exclude paths
	excludePaths := make(map[string]bool)
	for _, path := range cfg.ExcludePaths {
		excludePaths[path] = true
	}
	
	return func(c *gin.Context) {
		// Skip excluded paths
		if excludePaths[c.Request.URL.Path] {
			c.Next()
			return
		}
		
		// Skip health checks and metrics
		if strings.HasPrefix(c.Request.URL.Path, "/health") || 
		   strings.HasPrefix(c.Request.URL.Path, "/metrics") {
			c.Next()
			return
		}
		
		start := time.Now()
		
		// Capture request body
		var requestBody []byte
		if cfg.LogRequestBody && c.Request.Body != nil {
			requestBody, _ = io.ReadAll(c.Request.Body)
			c.Request.Body = io.NopCloser(bytes.NewBuffer(requestBody))
		}
		
		// Wrap response writer to capture response
		blw := &responseWriter{body: bytes.NewBufferString(""), ResponseWriter: c.Writer}
		c.Writer = blw
		
		// Process request
		c.Next()
		
		// Create audit event
		event := &AuditEvent{
			ID:          generateID(),
			Timestamp:   start,
			Method:      c.Request.Method,
			Path:        c.Request.URL.Path,
			StatusCode:  c.Writer.Status(),
			IPAddress:   c.ClientIP(),
			UserAgent:   c.Request.UserAgent(),
			Duration:    time.Since(start),
			Metadata:    make(map[string]interface{}),
		}
		
		// Add user information
		if user, exists := c.Get("user"); exists {
			if u, ok := user.(map[string]interface{}); ok {
				event.UserID = getString(u, "id")
				event.UserEmail = getString(u, "email")
			}
		}
		
		// Determine action and resource from path
		parseActionAndResource(c, event)
		
		// Add request body (redact sensitive fields)
		if cfg.LogRequestBody && len(requestBody) > 0 && int64(len(requestBody)) <= cfg.MaxBodySize {
			redactedBody := redactSensitiveData(requestBody, cfg.SensitiveFields)
			event.RequestBody = json.RawMessage(redactedBody)
		}
		
		// Add response body (redact sensitive fields)
		if cfg.LogResponseBody && blw.body.Len() > 0 && int64(blw.body.Len()) <= cfg.MaxBodySize {
			redactedBody := redactSensitiveData(blw.body.Bytes(), cfg.SensitiveFields)
			event.ResponseBody = json.RawMessage(redactedBody)
		}
		
		// Add error if present
		if len(c.Errors) > 0 {
			event.Error = c.Errors.String()
		}
		
		// Add request ID
		if requestID := c.GetString("request_id"); requestID != "" {
			event.Metadata["request_id"] = requestID
		}
		
		// Add query parameters
		if len(c.Request.URL.Query()) > 0 {
			event.Metadata["query_params"] = c.Request.URL.Query()
		}
		
		// Log audit event asynchronously
		go func() {
			if err := cfg.Logger.Log(event); err != nil {
				// Log error but don't fail the request
				if logger, ok := c.Get("logger"); ok {
					if l, ok := logger.(*zap.Logger); ok {
						l.Error("Failed to log audit event",
							zap.Error(err),
							zap.String("event_id", event.ID))
					}
				}
			}
		}()
	}
}

// Helper functions

func generateID() string {
	// In production, use a proper UUID generator
	return time.Now().Format("20060102150405") + "-" + randomString(8)
}

func randomString(n int) string {
	const letters = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
	b := make([]byte, n)
	for i := range b {
		b[i] = letters[time.Now().UnixNano()%int64(len(letters))]
	}
	return string(b)
}

func getString(m map[string]interface{}, key string) string {
	if v, ok := m[key]; ok {
		if s, ok := v.(string); ok {
			return s
		}
	}
	return ""
}

func parseActionAndResource(c *gin.Context, event *AuditEvent) {
	parts := strings.Split(strings.Trim(c.Request.URL.Path, "/"), "/")
	
	if len(parts) >= 2 && parts[0] == "api" {
		// API path format: /api/resource-type/resource-id
		switch c.Request.Method {
		case http.MethodGet:
			if len(parts) > 2 {
				event.Action = "read"
			} else {
				event.Action = "list"
			}
		case http.MethodPost:
			event.Action = "create"
		case http.MethodPut, http.MethodPatch:
			event.Action = "update"
		case http.MethodDelete:
			event.Action = "delete"
		default:
			event.Action = strings.ToLower(c.Request.Method)
		}
		
		if len(parts) > 1 {
			event.ResourceType = parts[1]
		}
		
		if len(parts) > 2 {
			event.ResourceID = parts[2]
		}
	}
}

func redactSensitiveData(data []byte, sensitiveFields []string) []byte {
	if len(sensitiveFields) == 0 {
		return data
	}
	
	var obj interface{}
	if err := json.Unmarshal(data, &obj); err != nil {
		return data // Return original if not JSON
	}
	
	redactValue(obj, sensitiveFields)
	
	redacted, err := json.Marshal(obj)
	if err != nil {
		return data
	}
	
	return redacted
}

func redactValue(v interface{}, sensitiveFields []string) {
	switch val := v.(type) {
	case map[string]interface{}:
		for k, v := range val {
			for _, field := range sensitiveFields {
				if strings.EqualFold(k, field) {
					val[k] = "[REDACTED]"
					break
				}
			}
			redactValue(v, sensitiveFields)
		}
	case []interface{}:
		for _, item := range val {
			redactValue(item, sensitiveFields)
		}
	}
}