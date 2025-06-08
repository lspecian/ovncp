package models

// TenantFilter represents filters for listing tenants
type TenantFilter struct {
	UserID string
	Type   TenantType
	Status TenantStatus
	Parent string
}