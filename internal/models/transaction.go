package models

import (
	"time"
)

// TransactionOperation represents a single operation within a transaction
type TransactionOperation struct {
	ID         string                 `json:"id"`          // Client-provided ID for tracking
	Type       string                 `json:"type"`        // "create", "update", "delete"
	Resource   string                 `json:"resource"`    // "switch", "router", "port", "acl"
	ResourceID string                 `json:"resource_id,omitempty"` // Required for update/delete
	SwitchID   string                 `json:"switch_id,omitempty"`   // Required for port/acl creation
	Data       map[string]interface{} `json:"data,omitempty"`        // Resource data for create/update
}

// TransactionRequest represents a batch transaction request
type TransactionRequest struct {
	Operations []TransactionOperation `json:"operations"`
	DryRun     bool                   `json:"dry_run,omitempty"` // If true, validate but don't execute
}

// TransactionOperationResult represents the result of a single operation
type TransactionOperationResult struct {
	ID         string                 `json:"id"`
	Type       string                 `json:"type"`
	Resource   string                 `json:"resource"`
	ResourceID string                 `json:"resource_id,omitempty"`
	Success    bool                   `json:"success"`
	Error      string                 `json:"error,omitempty"`
	Data       map[string]interface{} `json:"data,omitempty"` // Created/updated resource data
}

// TransactionResponse represents the response for a transaction request
type TransactionResponse struct {
	TransactionID string                       `json:"transaction_id"`
	Success       bool                         `json:"success"`
	Results       []TransactionOperationResult `json:"results"`
	Error         string                       `json:"error,omitempty"`
	ExecutedAt    time.Time                    `json:"executed_at"`
}

// Validation constants
const (
	// Operation types
	OperationCreate = "create"
	OperationUpdate = "update"
	OperationDelete = "delete"

	// Resource types
	ResourceSwitch = "switch"
	ResourceRouter = "router"
	ResourcePort   = "port"
	ResourceACL    = "acl"
)

// ValidOperationTypes returns all valid operation types
func ValidOperationTypes() []string {
	return []string{OperationCreate, OperationUpdate, OperationDelete}
}

// ValidResourceTypes returns all valid resource types
func ValidResourceTypes() []string {
	return []string{ResourceSwitch, ResourceRouter, ResourcePort, ResourceACL}
}

// IsValidOperationType checks if the operation type is valid
func IsValidOperationType(opType string) bool {
	for _, valid := range ValidOperationTypes() {
		if opType == valid {
			return true
		}
	}
	return false
}

// IsValidResourceType checks if the resource type is valid
func IsValidResourceType(resource string) bool {
	for _, valid := range ValidResourceTypes() {
		if resource == valid {
			return true
		}
	}
	return false
}