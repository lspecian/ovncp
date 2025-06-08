package models

import (
	"time"
)

// Tenant represents an isolated namespace for resources
type Tenant struct {
	ID          string            `json:"id" db:"id"`
	Name        string            `json:"name" db:"name"`
	DisplayName string            `json:"display_name" db:"display_name"`
	Description string            `json:"description,omitempty" db:"description"`
	Status      TenantStatus      `json:"status" db:"status"`
	Type        TenantType        `json:"type" db:"type"`
	Parent      *string           `json:"parent,omitempty" db:"parent"`
	Metadata    map[string]string `json:"metadata,omitempty" db:"metadata"`
	Settings    TenantSettings    `json:"settings" db:"settings"`
	Quotas      TenantQuotas      `json:"quotas" db:"quotas"`
	CreatedAt   time.Time         `json:"created_at" db:"created_at"`
	UpdatedAt   time.Time         `json:"updated_at" db:"updated_at"`
	CreatedBy   string            `json:"created_by" db:"created_by"`
}

// TenantStatus represents the status of a tenant
type TenantStatus string

const (
	TenantStatusActive    TenantStatus = "active"
	TenantStatusSuspended TenantStatus = "suspended"
	TenantStatusDeleting  TenantStatus = "deleting"
	TenantStatusDeleted   TenantStatus = "deleted"
)

// TenantType represents the type of tenant
type TenantType string

const (
	TenantTypeOrganization TenantType = "organization"
	TenantTypeProject      TenantType = "project"
	TenantTypeEnvironment  TenantType = "environment"
)

// TenantSettings contains tenant-specific settings
type TenantSettings struct {
	DefaultNetworkType    string            `json:"default_network_type,omitempty"`
	NetworkNamePrefix     string            `json:"network_name_prefix,omitempty"`
	RequireApproval       bool              `json:"require_approval"`
	AllowExternalNetworks bool              `json:"allow_external_networks"`
	EnableAuditLogging    bool              `json:"enable_audit_logging"`
	CustomLabels          map[string]string `json:"custom_labels,omitempty"`
}

// TenantQuotas defines resource limits for a tenant
type TenantQuotas struct {
	MaxSwitches      int `json:"max_switches" db:"max_switches"`
	MaxRouters       int `json:"max_routers" db:"max_routers"`
	MaxPorts         int `json:"max_ports" db:"max_ports"`
	MaxACLs          int `json:"max_acls" db:"max_acls"`
	MaxLoadBalancers int `json:"max_load_balancers" db:"max_load_balancers"`
	MaxAddressSets   int `json:"max_address_sets" db:"max_address_sets"`
	MaxPortGroups    int `json:"max_port_groups" db:"max_port_groups"`
	MaxBackups       int `json:"max_backups" db:"max_backups"`
}

// TenantMembership represents a user's membership in a tenant
type TenantMembership struct {
	ID        string    `json:"id" db:"id"`
	TenantID  string    `json:"tenant_id" db:"tenant_id"`
	UserID    string    `json:"user_id" db:"user_id"`
	Role      string    `json:"role" db:"role"`
	CreatedAt time.Time `json:"created_at" db:"created_at"`
	CreatedBy string    `json:"created_by" db:"created_by"`
}

// TenantResource represents a resource owned by a tenant
type TenantResource struct {
	ResourceID   string    `json:"resource_id" db:"resource_id"`
	ResourceType string    `json:"resource_type" db:"resource_type"`
	TenantID     string    `json:"tenant_id" db:"tenant_id"`
	CreatedAt    time.Time `json:"created_at" db:"created_at"`
}

// ResourceUsage tracks current resource usage for a tenant
type ResourceUsage struct {
	TenantID         string    `json:"tenant_id" db:"tenant_id"`
	Switches         int       `json:"switches" db:"switches"`
	Routers          int       `json:"routers" db:"routers"`
	Ports            int       `json:"ports" db:"ports"`
	ACLs             int       `json:"acls" db:"acls"`
	LoadBalancers    int       `json:"load_balancers" db:"load_balancers"`
	AddressSets      int       `json:"address_sets" db:"address_sets"`
	PortGroups       int       `json:"port_groups" db:"port_groups"`
	Backups          int       `json:"backups" db:"backups"`
	LastUpdated      time.Time `json:"last_updated" db:"last_updated"`
}

// TenantInvitation represents an invitation to join a tenant
type TenantInvitation struct {
	ID          string    `json:"id" db:"id"`
	TenantID    string    `json:"tenant_id" db:"tenant_id"`
	Email       string    `json:"email" db:"email"`
	Role        string    `json:"role" db:"role"`
	Token       string    `json:"token" db:"token"`
	ExpiresAt   time.Time `json:"expires_at" db:"expires_at"`
	AcceptedAt  *time.Time `json:"accepted_at,omitempty" db:"accepted_at"`
	CreatedAt   time.Time `json:"created_at" db:"created_at"`
	CreatedBy   string    `json:"created_by" db:"created_by"`
}

// TenantAPIKey represents an API key scoped to a tenant
type TenantAPIKey struct {
	ID          string    `json:"id" db:"id"`
	TenantID    string    `json:"tenant_id" db:"tenant_id"`
	Name        string    `json:"name" db:"name"`
	Description string    `json:"description,omitempty" db:"description"`
	KeyHash     string    `json:"-" db:"key_hash"`
	Prefix      string    `json:"prefix" db:"prefix"`
	Scopes      []string  `json:"scopes" db:"scopes"`
	ExpiresAt   *time.Time `json:"expires_at,omitempty" db:"expires_at"`
	LastUsedAt  *time.Time `json:"last_used_at,omitempty" db:"last_used_at"`
	CreatedAt   time.Time `json:"created_at" db:"created_at"`
	CreatedBy   string    `json:"created_by" db:"created_by"`
}

// DefaultQuotas returns default quota values
func DefaultQuotas() TenantQuotas {
	return TenantQuotas{
		MaxSwitches:      100,
		MaxRouters:       50,
		MaxPorts:         1000,
		MaxACLs:          500,
		MaxLoadBalancers: 20,
		MaxAddressSets:   100,
		MaxPortGroups:    100,
		MaxBackups:       50,
	}
}

// UnlimitedQuotas returns unlimited quota values
func UnlimitedQuotas() TenantQuotas {
	return TenantQuotas{
		MaxSwitches:      -1,
		MaxRouters:       -1,
		MaxPorts:         -1,
		MaxACLs:          -1,
		MaxLoadBalancers: -1,
		MaxAddressSets:   -1,
		MaxPortGroups:    -1,
		MaxBackups:       -1,
	}
}

// IsWithinQuota checks if adding count resources would exceed quota
func (q TenantQuotas) IsWithinQuota(resourceType string, current, toAdd int) bool {
	var limit int
	
	switch resourceType {
	case "switch":
		limit = q.MaxSwitches
	case "router":
		limit = q.MaxRouters
	case "port":
		limit = q.MaxPorts
	case "acl":
		limit = q.MaxACLs
	case "load_balancer":
		limit = q.MaxLoadBalancers
	case "address_set":
		limit = q.MaxAddressSets
	case "port_group":
		limit = q.MaxPortGroups
	case "backup":
		limit = q.MaxBackups
	default:
		return true // Unknown resource type, allow
	}
	
	// -1 means unlimited
	if limit == -1 {
		return true
	}
	
	return (current + toAdd) <= limit
}