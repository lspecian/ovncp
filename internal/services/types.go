package services

import (
	"time"
	
	"github.com/lspecian/ovncp/internal/models"
)

// ListOptions represents options for listing resources
type ListOptions struct {
	Limit    int
	Offset   int
	Page     int
	PageSize int
	Filter   map[string]string
	Filters  map[string]string // Alias for Filter
}

// Topology represents the network topology
type Topology struct {
	Switches     []*models.LogicalSwitch
	Routers      []*models.LogicalRouter
	Ports        []*models.LogicalSwitchPort
	RouterPorts  []*models.LogicalRouterPort
	ACLs         []*models.ACL
	Connections  []Connection
	Timestamp    time.Time
}

// Connection represents a connection between network elements
type Connection struct {
	From     string `json:"from"`
	To       string `json:"to"`
	Type     string `json:"type"`
	PortID   string `json:"port_id,omitempty"`
}

// TransactionOp represents a single operation in a transaction
type TransactionOp struct {
	Operation    string      `json:"operation"`      // create, update, delete
	Table        string      `json:"table"`          // switch, router, port, acl
	ID           string      `json:"id,omitempty"`
	Data         interface{} `json:"data"`
	ResourceType string      `json:"resource_type,omitempty"` // For batch operations
	ResourceID   string      `json:"resource_id,omitempty"`   // For batch operations
}