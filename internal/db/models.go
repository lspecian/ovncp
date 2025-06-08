package db

import (
	"time"
)

// Resource represents a generic resource
type Resource struct {
	ID        string
	Type      string
	Data      interface{}
	CreatedAt time.Time
	UpdatedAt time.Time
}

// AuditLog represents an audit log entry
type AuditLog struct {
	ID        string
	UserID    string
	Action    string
	Resource  string
	Details   string
	Timestamp time.Time
	IP        string
	UserAgent string
}