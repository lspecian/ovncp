package services

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/lspecian/ovncp/internal/db"
	"github.com/lspecian/ovncp/internal/models"
	"go.uber.org/zap"
)

// TenantService handles tenant operations
type TenantService struct {
	db     *db.DB
	logger *zap.Logger
}

// NewTenantService creates a new tenant service
func NewTenantService(database *db.DB, logger *zap.Logger) *TenantService {
	return &TenantService{
		db:     database,
		logger: logger,
	}
}

// CreateTenant creates a new tenant
func (s *TenantService) CreateTenant(ctx context.Context, tenant *models.Tenant, createdBy string) (*models.Tenant, error) {
	// Validate tenant
	if err := s.validateTenant(tenant); err != nil {
		return nil, fmt.Errorf("invalid tenant: %w", err)
	}

	// Set defaults
	tenant.ID = uuid.New().String()
	tenant.Status = models.TenantStatusActive
	tenant.CreatedAt = time.Now()
	tenant.UpdatedAt = time.Now()
	tenant.CreatedBy = createdBy

	// Set default quotas if not provided
	if tenant.Quotas == (models.TenantQuotas{}) {
		tenant.Quotas = models.DefaultQuotas()
	}

	// Create tenant in database
	if err := s.db.CreateTenant(ctx, tenant); err != nil {
		return nil, fmt.Errorf("failed to create tenant: %w", err)
	}

	// Create default admin membership for creator
	membership := &models.TenantMembership{
		ID:        uuid.New().String(),
		TenantID:  tenant.ID,
		UserID:    createdBy,
		Role:      "admin",
		CreatedAt: time.Now(),
		CreatedBy: createdBy,
	}

	if err := s.db.CreateTenantMembership(ctx, membership); err != nil {
		s.logger.Error("Failed to create default membership",
			zap.String("tenant_id", tenant.ID),
			zap.Error(err))
	}

	s.logger.Info("Tenant created",
		zap.String("tenant_id", tenant.ID),
		zap.String("name", tenant.Name))

	return tenant, nil
}

// GetTenant retrieves a tenant by ID
func (s *TenantService) GetTenant(ctx context.Context, tenantID string) (*models.Tenant, error) {
	tenant, err := s.db.GetTenant(ctx, tenantID)
	if err != nil {
		return nil, fmt.Errorf("failed to get tenant: %w", err)
	}
	return tenant, nil
}

// ListTenants lists all tenants
func (s *TenantService) ListTenants(ctx context.Context, filter *models.TenantFilter) ([]*models.Tenant, error) {
	return s.db.ListTenants(ctx, filter)
}

// UpdateTenant updates a tenant
func (s *TenantService) UpdateTenant(ctx context.Context, tenantID string, updates *models.Tenant) (*models.Tenant, error) {
	// Get existing tenant
	existing, err := s.db.GetTenant(ctx, tenantID)
	if err != nil {
		return nil, fmt.Errorf("failed to get tenant: %w", err)
	}

	// Apply updates
	if updates.DisplayName != "" {
		existing.DisplayName = updates.DisplayName
	}
	if updates.Description != "" {
		existing.Description = updates.Description
	}
	if updates.Status != "" {
		existing.Status = updates.Status
	}
	// Update settings if any field is set
	if updates.Settings.DefaultNetworkType != "" || updates.Settings.NetworkNamePrefix != "" || 
		updates.Settings.RequireApproval || updates.Settings.AllowExternalNetworks || 
		updates.Settings.EnableAuditLogging || len(updates.Settings.CustomLabels) > 0 {
		existing.Settings = updates.Settings
	}
	if updates.Quotas != (models.TenantQuotas{}) {
		existing.Quotas = updates.Quotas
	}
	if updates.Metadata != nil {
		existing.Metadata = updates.Metadata
	}

	existing.UpdatedAt = time.Now()

	// Update in database
	if err := s.db.UpdateTenant(ctx, tenantID, existing); err != nil {
		return nil, fmt.Errorf("failed to update tenant: %w", err)
	}

	return existing, nil
}

// DeleteTenant marks a tenant for deletion
func (s *TenantService) DeleteTenant(ctx context.Context, tenantID string) error {
	// Check if tenant has resources
	usage, err := s.GetResourceUsage(ctx, tenantID)
	if err != nil {
		return fmt.Errorf("failed to check resource usage: %w", err)
	}

	if s.hasActiveResources(usage) {
		return fmt.Errorf("tenant has active resources, cannot delete")
	}

	// Mark tenant as deleting
	updates := &models.Tenant{
		Status: models.TenantStatusDeleting,
	}

	if _, err := s.UpdateTenant(ctx, tenantID, updates); err != nil {
		return err
	}

	// TODO: Implement async deletion of tenant data

	s.logger.Info("Tenant marked for deletion",
		zap.String("tenant_id", tenantID))

	return nil
}

// AddMember adds a user to a tenant
func (s *TenantService) AddMember(ctx context.Context, tenantID, userID, role, addedBy string) error {
	// Check if membership already exists
	existing, err := s.db.GetTenantMembership(ctx, tenantID, userID)
	if err == nil && existing != nil {
		return fmt.Errorf("user is already a member of this tenant")
	}

	membership := &models.TenantMembership{
		ID:        uuid.New().String(),
		TenantID:  tenantID,
		UserID:    userID,
		Role:      role,
		CreatedAt: time.Now(),
		CreatedBy: addedBy,
	}

	if err := s.db.CreateTenantMembership(ctx, membership); err != nil {
		return fmt.Errorf("failed to add member: %w", err)
	}

	s.logger.Info("Member added to tenant",
		zap.String("tenant_id", tenantID),
		zap.String("user_id", userID),
		zap.String("role", role))

	return nil
}

// GetMembership returns a user's membership in a tenant
func (s *TenantService) GetMembership(ctx context.Context, tenantID, userID string) (*models.TenantMembership, error) {
	return s.db.GetTenantMembership(ctx, tenantID, userID)
}

// ListMembers lists all members of a tenant
func (s *TenantService) ListMembers(ctx context.Context, tenantID string) ([]*models.TenantMembership, error) {
	return s.db.ListTenantMembers(ctx, tenantID)
}

// RemoveMember removes a user from a tenant
func (s *TenantService) RemoveMember(ctx context.Context, tenantID, userID string) error {
	// Check if this is the last admin
	members, err := s.db.ListTenantMembers(ctx, tenantID)
	if err != nil {
		return fmt.Errorf("failed to list members: %w", err)
	}

	adminCount := 0
	for _, member := range members {
		if member.Role == "admin" {
			adminCount++
		}
	}

	membership, err := s.db.GetTenantMembership(ctx, tenantID, userID)
	if err != nil {
		return fmt.Errorf("membership not found: %w", err)
	}

	if membership.Role == "admin" && adminCount <= 1 {
		return fmt.Errorf("cannot remove last admin from tenant")
	}

	if err := s.db.DeleteTenantMembership(ctx, tenantID, userID); err != nil {
		return fmt.Errorf("failed to remove member: %w", err)
	}

	s.logger.Info("Member removed from tenant",
		zap.String("tenant_id", tenantID),
		zap.String("user_id", userID))

	return nil
}

// UpdateMemberRole updates a member's role in a tenant
func (s *TenantService) UpdateMemberRole(ctx context.Context, tenantID, userID, newRole string) error {
	membership, err := s.db.GetTenantMembership(ctx, tenantID, userID)
	if err != nil {
		return fmt.Errorf("membership not found: %w", err)
	}

	// Check if this would remove the last admin
	if membership.Role == "admin" && newRole != "admin" {
		members, err := s.db.ListTenantMembers(ctx, tenantID)
		if err != nil {
			return fmt.Errorf("failed to list members: %w", err)
		}

		adminCount := 0
		for _, member := range members {
			if member.Role == "admin" {
				adminCount++
			}
		}

		if adminCount <= 1 {
			return fmt.Errorf("cannot demote last admin")
		}
	}

	membership.Role = newRole
	if err := s.db.UpdateTenantMembership(ctx, membership); err != nil {
		return fmt.Errorf("failed to update role: %w", err)
	}

	return nil
}

// GetResourceUsage returns current resource usage for a tenant
func (s *TenantService) GetResourceUsage(ctx context.Context, tenantID string) (*models.ResourceUsage, error) {
	return s.db.GetResourceUsage(ctx, tenantID)
}

// CheckQuota checks if a tenant can create more resources
func (s *TenantService) CheckQuota(ctx context.Context, tenantID, resourceType string, count int) error {
	tenant, err := s.GetTenant(ctx, tenantID)
	if err != nil {
		return fmt.Errorf("failed to get tenant: %w", err)
	}

	usage, err := s.GetResourceUsage(ctx, tenantID)
	if err != nil {
		return fmt.Errorf("failed to get resource usage: %w", err)
	}

	var current int
	switch resourceType {
	case "switch":
		current = usage.Switches
	case "router":
		current = usage.Routers
	case "port":
		current = usage.Ports
	case "acl":
		current = usage.ACLs
	case "load_balancer":
		current = usage.LoadBalancers
	case "address_set":
		current = usage.AddressSets
	case "port_group":
		current = usage.PortGroups
	case "backup":
		current = usage.Backups
	default:
		return fmt.Errorf("unknown resource type: %s", resourceType)
	}

	if !tenant.Quotas.IsWithinQuota(resourceType, current, count) {
		limit := s.getQuotaLimit(tenant.Quotas, resourceType)
		return fmt.Errorf("quota exceeded: %s (current: %d, limit: %d)", resourceType, current, limit)
	}

	return nil
}

// AssociateResource associates a resource with a tenant
func (s *TenantService) AssociateResource(ctx context.Context, tenantID, resourceID, resourceType string) error {
	resource := &models.TenantResource{
		ResourceID:   resourceID,
		ResourceType: resourceType,
		TenantID:     tenantID,
		CreatedAt:    time.Now(),
	}

	if err := s.db.CreateTenantResource(ctx, resource); err != nil {
		return fmt.Errorf("failed to associate resource: %w", err)
	}

	// Update resource usage
	if err := s.db.UpdateResourceUsage(ctx, tenantID, resourceType, 1); err != nil {
		s.logger.Error("Failed to update resource usage",
			zap.String("tenant_id", tenantID),
			zap.Error(err))
	}

	return nil
}

// DissociateResource removes a resource association
func (s *TenantService) DissociateResource(ctx context.Context, resourceID string) error {
	resource, err := s.db.GetTenantResource(ctx, resourceID)
	if err != nil {
		return fmt.Errorf("resource not found: %w", err)
	}

	if err := s.db.DeleteTenantResource(ctx, resourceID); err != nil {
		return fmt.Errorf("failed to dissociate resource: %w", err)
	}

	// Update resource usage
	if err := s.db.UpdateResourceUsage(ctx, resource.TenantID, resource.ResourceType, -1); err != nil {
		s.logger.Error("Failed to update resource usage",
			zap.String("tenant_id", resource.TenantID),
			zap.Error(err))
	}

	return nil
}

// GetResourceTenant returns the tenant ID for a resource
func (s *TenantService) GetResourceTenant(ctx context.Context, resourceID string) (string, error) {
	resource, err := s.db.GetTenantResource(ctx, resourceID)
	if err != nil {
		return "", fmt.Errorf("resource not found: %w", err)
	}
	return resource.TenantID, nil
}

// CreateInvitation creates an invitation to join a tenant
func (s *TenantService) CreateInvitation(ctx context.Context, tenantID, email, role, createdBy string) (*models.TenantInvitation, error) {
	// Generate secure token
	tokenBytes := make([]byte, 32)
	if _, err := rand.Read(tokenBytes); err != nil {
		return nil, fmt.Errorf("failed to generate token: %w", err)
	}
	token := base64.URLEncoding.EncodeToString(tokenBytes)

	invitation := &models.TenantInvitation{
		ID:        uuid.New().String(),
		TenantID:  tenantID,
		Email:     email,
		Role:      role,
		Token:     token,
		ExpiresAt: time.Now().Add(7 * 24 * time.Hour), // 7 days
		CreatedAt: time.Now(),
		CreatedBy: createdBy,
	}

	if err := s.db.CreateTenantInvitation(ctx, invitation); err != nil {
		return nil, fmt.Errorf("failed to create invitation: %w", err)
	}

	return invitation, nil
}

// AcceptInvitation accepts a tenant invitation
func (s *TenantService) AcceptInvitation(ctx context.Context, token, userID string) error {
	invitation, err := s.db.GetTenantInvitationByToken(ctx, token)
	if err != nil {
		return fmt.Errorf("invalid invitation token")
	}

	if invitation.AcceptedAt != nil {
		return fmt.Errorf("invitation already accepted")
	}

	if time.Now().After(invitation.ExpiresAt) {
		return fmt.Errorf("invitation expired")
	}

	// Add user to tenant
	if err := s.AddMember(ctx, invitation.TenantID, userID, invitation.Role, "system"); err != nil {
		return err
	}

	// Mark invitation as accepted
	now := time.Now()
	invitation.AcceptedAt = &now
	if err := s.db.UpdateTenantInvitation(ctx, invitation); err != nil {
		s.logger.Error("Failed to update invitation",
			zap.String("invitation_id", invitation.ID),
			zap.Error(err))
	}

	return nil
}

// CreateAPIKey creates a new API key for a tenant
func (s *TenantService) CreateAPIKey(ctx context.Context, tenantID string, key *models.TenantAPIKey) (string, error) {
	// Generate API key
	keyBytes := make([]byte, 32)
	if _, err := rand.Read(keyBytes); err != nil {
		return "", fmt.Errorf("failed to generate key: %w", err)
	}
	
	apiKey := base64.URLEncoding.EncodeToString(keyBytes)
	prefix := apiKey[:8]

	key.ID = uuid.New().String()
	key.TenantID = tenantID
	key.Prefix = prefix
	key.KeyHash = s.hashAPIKey(apiKey)
	key.CreatedAt = time.Now()

	if err := s.db.CreateTenantAPIKey(ctx, key); err != nil {
		return "", fmt.Errorf("failed to create API key: %w", err)
	}

	// Return the full key only once
	return fmt.Sprintf("ovncp_%s_%s", tenantID[:8], apiKey), nil
}

// ValidateAPIKey validates an API key and returns the associated tenant
func (s *TenantService) ValidateAPIKey(ctx context.Context, apiKey string) (*models.TenantAPIKey, error) {
	// Parse key format: ovncp_<tenant_prefix>_<key>
	parts := strings.Split(apiKey, "_")
	if len(parts) != 3 || parts[0] != "ovncp" {
		return nil, fmt.Errorf("invalid API key format")
	}

	keyHash := s.hashAPIKey(parts[2])
	
	key, err := s.db.GetTenantAPIKeyByHash(ctx, keyHash)
	if err != nil {
		return nil, fmt.Errorf("invalid API key")
	}

	// Check expiration
	if key.ExpiresAt != nil && time.Now().After(*key.ExpiresAt) {
		return nil, fmt.Errorf("API key expired")
	}

	// Update last used
	now := time.Now()
	key.LastUsedAt = &now
	if err := s.db.UpdateTenantAPIKey(ctx, key); err != nil {
		s.logger.Error("Failed to update API key last used",
			zap.String("key_id", key.ID),
			zap.Error(err))
	}

	return key, nil
}

// ListAPIKeys lists all API keys for a tenant
func (s *TenantService) ListAPIKeys(ctx context.Context, tenantID string) ([]*models.TenantAPIKey, error) {
	return s.db.ListTenantAPIKeys(ctx, tenantID)
}

// DeleteAPIKey deletes an API key
func (s *TenantService) DeleteAPIKey(ctx context.Context, tenantID, keyID string) error {
	// Verify the key belongs to the tenant
	key, err := s.db.GetTenantAPIKey(ctx, keyID)
	if err != nil {
		return fmt.Errorf("API key not found")
	}
	
	if key.TenantID != tenantID {
		return fmt.Errorf("API key does not belong to this tenant")
	}
	
	return s.db.DeleteTenantAPIKey(ctx, keyID)
}

// Helper functions

func (s *TenantService) validateTenant(tenant *models.Tenant) error {
	if tenant.Name == "" {
		return fmt.Errorf("tenant name is required")
	}

	if tenant.Type == "" {
		tenant.Type = models.TenantTypeProject
	}

	// Validate tenant name format (alphanumeric, dash, underscore)
	if !isValidTenantName(tenant.Name) {
		return fmt.Errorf("invalid tenant name format")
	}

	return nil
}

func (s *TenantService) hasActiveResources(usage *models.ResourceUsage) bool {
	return usage.Switches > 0 ||
		usage.Routers > 0 ||
		usage.Ports > 0 ||
		usage.ACLs > 0 ||
		usage.LoadBalancers > 0 ||
		usage.AddressSets > 0 ||
		usage.PortGroups > 0
}

func (s *TenantService) getQuotaLimit(quotas models.TenantQuotas, resourceType string) int {
	switch resourceType {
	case "switch":
		return quotas.MaxSwitches
	case "router":
		return quotas.MaxRouters
	case "port":
		return quotas.MaxPorts
	case "acl":
		return quotas.MaxACLs
	case "load_balancer":
		return quotas.MaxLoadBalancers
	case "address_set":
		return quotas.MaxAddressSets
	case "port_group":
		return quotas.MaxPortGroups
	case "backup":
		return quotas.MaxBackups
	default:
		return 0
	}
}

func (s *TenantService) hashAPIKey(key string) string {
	// In production, use bcrypt or similar
	// This is simplified for demonstration
	return fmt.Sprintf("hash_%s", key)
}

func isValidTenantName(name string) bool {
	// Simple validation: alphanumeric, dash, underscore, 3-63 chars
	if len(name) < 3 || len(name) > 63 {
		return false
	}
	
	for _, ch := range name {
		if !((ch >= 'a' && ch <= 'z') || (ch >= 'A' && ch <= 'Z') || 
			 (ch >= '0' && ch <= '9') || ch == '-' || ch == '_') {
			return false
		}
	}
	
	return true
}

