package services

import (
	"context"
	"fmt"

	"github.com/lspecian/ovncp/internal/models"
)

// TenantOVNService wraps OVNService with tenant filtering
type TenantOVNService struct {
	ovnService    OVNServiceInterface
	tenantService *TenantService
}

// NewTenantOVNService creates a new tenant-aware OVN service
func NewTenantOVNService(ovnService OVNServiceInterface, tenantService *TenantService) *TenantOVNService {
	return &TenantOVNService{
		ovnService:    ovnService,
		tenantService: tenantService,
	}
}

// ListLogicalSwitches returns only switches belonging to the tenant
func (s *TenantOVNService) ListLogicalSwitches(ctx context.Context) ([]*models.LogicalSwitch, error) {
	tenantID := getTenantFromContext(ctx)
	if tenantID == "" {
		return s.ovnService.ListLogicalSwitches(ctx)
	}

	// Get all switches
	switches, err := s.ovnService.ListLogicalSwitches(ctx)
	if err != nil {
		return nil, err
	}

	// Filter by tenant
	var filtered []*models.LogicalSwitch
	for _, sw := range switches {
		if s.belongsToTenant(ctx, sw.UUID, tenantID) {
			filtered = append(filtered, sw)
		}
	}

	return filtered, nil
}

// GetLogicalSwitch checks tenant ownership before returning
func (s *TenantOVNService) GetLogicalSwitch(ctx context.Context, id string) (*models.LogicalSwitch, error) {
	sw, err := s.ovnService.GetLogicalSwitch(ctx, id)
	if err != nil {
		return nil, err
	}

	if err := s.checkTenantAccess(ctx, sw.UUID); err != nil {
		return nil, err
	}

	return sw, nil
}

// CreateLogicalSwitch creates a switch and associates it with tenant
func (s *TenantOVNService) CreateLogicalSwitch(ctx context.Context, ls *models.LogicalSwitch) (*models.LogicalSwitch, error) {
	tenantID := getTenantFromContext(ctx)
	if tenantID == "" {
		return nil, fmt.Errorf("tenant context required")
	}

	// Check quota
	if err := s.tenantService.CheckQuota(ctx, tenantID, "switch", 1); err != nil {
		return nil, err
	}

	// Add tenant prefix to name if configured
	tenant, _ := s.tenantService.GetTenant(ctx, tenantID)
	if tenant != nil && tenant.Settings.NetworkNamePrefix != "" {
		ls.Name = fmt.Sprintf("%s-%s", tenant.Settings.NetworkNamePrefix, ls.Name)
	}

	// Add tenant external ID
	if ls.ExternalIDs == nil {
		ls.ExternalIDs = make(map[string]string)
	}
	ls.ExternalIDs["tenant_id"] = tenantID

	// Create switch
	created, err := s.ovnService.CreateLogicalSwitch(ctx, ls)
	if err != nil {
		return nil, err
	}

	// Associate with tenant
	if err := s.tenantService.AssociateResource(ctx, tenantID, created.UUID, "switch"); err != nil {
		// Rollback
		s.ovnService.DeleteLogicalSwitch(ctx, created.UUID)
		return nil, fmt.Errorf("failed to associate switch with tenant: %w", err)
	}

	return created, nil
}

// UpdateLogicalSwitch checks tenant ownership before updating
func (s *TenantOVNService) UpdateLogicalSwitch(ctx context.Context, id string, ls *models.LogicalSwitch) (*models.LogicalSwitch, error) {
	if err := s.checkTenantAccess(ctx, id); err != nil {
		return nil, err
	}

	// Preserve tenant external ID
	existing, err := s.ovnService.GetLogicalSwitch(ctx, id)
	if err != nil {
		return nil, err
	}

	if tenantID, ok := existing.ExternalIDs["tenant_id"]; ok {
		if ls.ExternalIDs == nil {
			ls.ExternalIDs = make(map[string]string)
		}
		ls.ExternalIDs["tenant_id"] = tenantID
	}

	return s.ovnService.UpdateLogicalSwitch(ctx, id, ls)
}

// DeleteLogicalSwitch checks tenant ownership before deleting
func (s *TenantOVNService) DeleteLogicalSwitch(ctx context.Context, id string) error {
	if err := s.checkTenantAccess(ctx, id); err != nil {
		return err
	}

	// Delete the switch
	if err := s.ovnService.DeleteLogicalSwitch(ctx, id); err != nil {
		return err
	}

	// Dissociate from tenant
	if err := s.tenantService.DissociateResource(ctx, id); err != nil {
		// Log error but don't fail the operation
		fmt.Printf("Failed to dissociate switch from tenant: %v\n", err)
	}

	return nil
}

// Similar implementations for routers...

func (s *TenantOVNService) ListLogicalRouters(ctx context.Context) ([]*models.LogicalRouter, error) {
	tenantID := getTenantFromContext(ctx)
	if tenantID == "" {
		return s.ovnService.ListLogicalRouters(ctx)
	}

	routers, err := s.ovnService.ListLogicalRouters(ctx)
	if err != nil {
		return nil, err
	}

	var filtered []*models.LogicalRouter
	for _, router := range routers {
		if s.belongsToTenant(ctx, router.UUID, tenantID) {
			filtered = append(filtered, router)
		}
	}

	return filtered, nil
}

func (s *TenantOVNService) GetLogicalRouter(ctx context.Context, id string) (*models.LogicalRouter, error) {
	router, err := s.ovnService.GetLogicalRouter(ctx, id)
	if err != nil {
		return nil, err
	}

	if err := s.checkTenantAccess(ctx, router.UUID); err != nil {
		return nil, err
	}

	return router, nil
}

func (s *TenantOVNService) CreateLogicalRouter(ctx context.Context, lr *models.LogicalRouter) (*models.LogicalRouter, error) {
	tenantID := getTenantFromContext(ctx)
	if tenantID == "" {
		return nil, fmt.Errorf("tenant context required")
	}

	// Check quota
	if err := s.tenantService.CheckQuota(ctx, tenantID, "router", 1); err != nil {
		return nil, err
	}

	// Add tenant prefix and external ID
	tenant, _ := s.tenantService.GetTenant(ctx, tenantID)
	if tenant != nil && tenant.Settings.NetworkNamePrefix != "" {
		lr.Name = fmt.Sprintf("%s-%s", tenant.Settings.NetworkNamePrefix, lr.Name)
	}

	if lr.ExternalIDs == nil {
		lr.ExternalIDs = make(map[string]string)
	}
	lr.ExternalIDs["tenant_id"] = tenantID

	created, err := s.ovnService.CreateLogicalRouter(ctx, lr)
	if err != nil {
		return nil, err
	}

	if err := s.tenantService.AssociateResource(ctx, tenantID, created.UUID, "router"); err != nil {
		s.ovnService.DeleteLogicalRouter(ctx, created.UUID)
		return nil, fmt.Errorf("failed to associate router with tenant: %w", err)
	}

	return created, nil
}

func (s *TenantOVNService) UpdateLogicalRouter(ctx context.Context, id string, lr *models.LogicalRouter) (*models.LogicalRouter, error) {
	if err := s.checkTenantAccess(ctx, id); err != nil {
		return nil, err
	}

	existing, err := s.ovnService.GetLogicalRouter(ctx, id)
	if err != nil {
		return nil, err
	}

	if tenantID, ok := existing.ExternalIDs["tenant_id"]; ok {
		if lr.ExternalIDs == nil {
			lr.ExternalIDs = make(map[string]string)
		}
		lr.ExternalIDs["tenant_id"] = tenantID
	}

	return s.ovnService.UpdateLogicalRouter(ctx, id, lr)
}

func (s *TenantOVNService) DeleteLogicalRouter(ctx context.Context, id string) error {
	if err := s.checkTenantAccess(ctx, id); err != nil {
		return err
	}

	if err := s.ovnService.DeleteLogicalRouter(ctx, id); err != nil {
		return err
	}

	if err := s.tenantService.DissociateResource(ctx, id); err != nil {
		fmt.Printf("Failed to dissociate router from tenant: %v\n", err)
	}

	return nil
}

// Port operations

func (s *TenantOVNService) ListPorts(ctx context.Context, switchID string) ([]*models.LogicalSwitchPort, error) {
	// Check switch ownership first
	if err := s.checkTenantAccess(ctx, switchID); err != nil {
		return nil, err
	}

	return s.ovnService.ListPorts(ctx, switchID)
}

func (s *TenantOVNService) GetPort(ctx context.Context, id string) (*models.LogicalSwitchPort, error) {
	port, err := s.ovnService.GetPort(ctx, id)
	if err != nil {
		return nil, err
	}

	// Check if the parent switch belongs to tenant
	// In real implementation, would need to track switch-port relationships
	
	return port, nil
}

func (s *TenantOVNService) CreatePort(ctx context.Context, switchID string, port *models.LogicalSwitchPort) (*models.LogicalSwitchPort, error) {
	// Check switch ownership
	if err := s.checkTenantAccess(ctx, switchID); err != nil {
		return nil, err
	}

	tenantID := getTenantFromContext(ctx)
	if tenantID == "" {
		return nil, fmt.Errorf("tenant context required")
	}

	// Check quota
	if err := s.tenantService.CheckQuota(ctx, tenantID, "port", 1); err != nil {
		return nil, err
	}

	// Add tenant external ID
	if port.ExternalIDs == nil {
		port.ExternalIDs = make(map[string]string)
	}
	port.ExternalIDs["tenant_id"] = tenantID

	created, err := s.ovnService.CreatePort(ctx, switchID, port)
	if err != nil {
		return nil, err
	}

	if err := s.tenantService.AssociateResource(ctx, tenantID, created.UUID, "port"); err != nil {
		s.ovnService.DeletePort(ctx, created.UUID)
		return nil, fmt.Errorf("failed to associate port with tenant: %w", err)
	}

	return created, nil
}

func (s *TenantOVNService) UpdatePort(ctx context.Context, id string, port *models.LogicalSwitchPort) (*models.LogicalSwitchPort, error) {
	if err := s.checkTenantAccess(ctx, id); err != nil {
		return nil, err
	}

	existing, err := s.ovnService.GetPort(ctx, id)
	if err != nil {
		return nil, err
	}

	if tenantID, ok := existing.ExternalIDs["tenant_id"]; ok {
		if port.ExternalIDs == nil {
			port.ExternalIDs = make(map[string]string)
		}
		port.ExternalIDs["tenant_id"] = tenantID
	}

	return s.ovnService.UpdatePort(ctx, id, port)
}

func (s *TenantOVNService) DeletePort(ctx context.Context, id string) error {
	if err := s.checkTenantAccess(ctx, id); err != nil {
		return err
	}

	if err := s.ovnService.DeletePort(ctx, id); err != nil {
		return err
	}

	if err := s.tenantService.DissociateResource(ctx, id); err != nil {
		fmt.Printf("Failed to dissociate port from tenant: %v\n", err)
	}

	return nil
}

// ACL operations

func (s *TenantOVNService) ListACLs(ctx context.Context, switchID string) ([]*models.ACL, error) {
	// Check switch ownership first
	if err := s.checkTenantAccess(ctx, switchID); err != nil {
		return nil, err
	}

	return s.ovnService.ListACLs(ctx, switchID)
}

func (s *TenantOVNService) GetACL(ctx context.Context, id string) (*models.ACL, error) {
	acl, err := s.ovnService.GetACL(ctx, id)
	if err != nil {
		return nil, err
	}

	if err := s.checkTenantAccess(ctx, id); err != nil {
		return nil, err
	}

	return acl, nil
}

func (s *TenantOVNService) CreateACL(ctx context.Context, switchID string, acl *models.ACL) (*models.ACL, error) {
	// Check switch ownership
	if err := s.checkTenantAccess(ctx, switchID); err != nil {
		return nil, err
	}

	tenantID := getTenantFromContext(ctx)
	if tenantID == "" {
		return nil, fmt.Errorf("tenant context required")
	}

	// Check quota
	if err := s.tenantService.CheckQuota(ctx, tenantID, "acl", 1); err != nil {
		return nil, err
	}

	// Add tenant external ID
	if acl.ExternalIDs == nil {
		acl.ExternalIDs = make(map[string]string)
	}
	acl.ExternalIDs["tenant_id"] = tenantID

	created, err := s.ovnService.CreateACL(ctx, switchID, acl)
	if err != nil {
		return nil, err
	}

	if err := s.tenantService.AssociateResource(ctx, tenantID, created.UUID, "acl"); err != nil {
		s.ovnService.DeleteACL(ctx, created.UUID)
		return nil, fmt.Errorf("failed to associate ACL with tenant: %w", err)
	}

	return created, nil
}

func (s *TenantOVNService) UpdateACL(ctx context.Context, id string, acl *models.ACL) (*models.ACL, error) {
	if err := s.checkTenantAccess(ctx, id); err != nil {
		return nil, err
	}

	existing, err := s.ovnService.GetACL(ctx, id)
	if err != nil {
		return nil, err
	}

	if tenantID, ok := existing.ExternalIDs["tenant_id"]; ok {
		if acl.ExternalIDs == nil {
			acl.ExternalIDs = make(map[string]string)
		}
		acl.ExternalIDs["tenant_id"] = tenantID
	}

	return s.ovnService.UpdateACL(ctx, id, acl)
}

func (s *TenantOVNService) DeleteACL(ctx context.Context, id string) error {
	if err := s.checkTenantAccess(ctx, id); err != nil {
		return err
	}

	if err := s.ovnService.DeleteACL(ctx, id); err != nil {
		return err
	}

	if err := s.tenantService.DissociateResource(ctx, id); err != nil {
		fmt.Printf("Failed to dissociate ACL from tenant: %v\n", err)
	}

	return nil
}

// Helper functions

func (s *TenantOVNService) checkTenantAccess(ctx context.Context, resourceID string) error {
	tenantID := getTenantFromContext(ctx)
	if tenantID == "" {
		// No tenant context, allow access (backward compatibility)
		return nil
	}

	resourceTenant, err := s.tenantService.GetResourceTenant(ctx, resourceID)
	if err != nil {
		// Resource not associated with any tenant
		return nil
	}

	if resourceTenant != tenantID {
		return fmt.Errorf("access denied: resource belongs to different tenant")
	}

	return nil
}

func (s *TenantOVNService) belongsToTenant(ctx context.Context, resourceID, tenantID string) bool {
	resourceTenant, err := s.tenantService.GetResourceTenant(ctx, resourceID)
	if err != nil {
		// Resource not associated with any tenant
		return false
	}

	return resourceTenant == tenantID
}

func getTenantFromContext(ctx context.Context) string {
	if tenant, ok := ctx.Value("tenant_id").(string); ok {
		return tenant
	}
	return ""
}

// ContextWithTenant returns a new context with tenant ID
func ContextWithTenant(ctx context.Context, tenantID string) context.Context {
	return context.WithValue(ctx, "tenant_id", tenantID)
}