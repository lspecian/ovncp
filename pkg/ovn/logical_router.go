package ovn

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"

	"github.com/lspecian/ovncp/internal/models"
	"github.com/lspecian/ovncp/pkg/ovn/nbdb"
)

// ListLogicalRouters returns all logical routers
func (c *Client) ListLogicalRouters(ctx context.Context) ([]*models.LogicalRouter, error) {
	if !c.IsConnected() {
		return nil, fmt.Errorf("client not connected")
	}

	// Get all logical routers from database
	lrList := []nbdb.LogicalRouter{}
	err := c.nbClient.List(ctx, &lrList)
	if err != nil {
		return nil, fmt.Errorf("failed to list logical routers: %w", err)
	}

	// Convert to our model
	result := make([]*models.LogicalRouter, 0, len(lrList))
	for i := range lrList {
		result = append(result, convertLogicalRouter(&lrList[i]))
	}

	return result, nil
}

// GetLogicalRouter returns a specific logical router by UUID or name
func (c *Client) GetLogicalRouter(ctx context.Context, id string) (*models.LogicalRouter, error) {
	if !c.IsConnected() {
		return nil, fmt.Errorf("client not connected")
	}

	// First, list all routers and find by UUID or name
	lrList := []nbdb.LogicalRouter{}
	err := c.nbClient.List(ctx, &lrList)
	if err != nil {
		return nil, fmt.Errorf("failed to list logical routers: %w", err)
	}

	// Look for match by UUID or name
	for i := range lrList {
		if lrList[i].UUID == id || lrList[i].Name == id {
			return convertLogicalRouter(&lrList[i]), nil
		}
	}

	return nil, fmt.Errorf("logical router %s not found", id)
}

// CreateLogicalRouter creates a new logical router
func (c *Client) CreateLogicalRouter(ctx context.Context, lr *models.LogicalRouter) (*models.LogicalRouter, error) {
	if !c.IsConnected() {
		return nil, fmt.Errorf("client not connected")
	}

	// Generate UUID if not provided
	if lr.UUID == "" {
		lr.UUID = uuid.New().String()
	}

	// Add timestamps to external_ids
	now := time.Now()
	if lr.ExternalIDs == nil {
		lr.ExternalIDs = make(map[string]string)
	}
	lr.ExternalIDs["created_at"] = now.Format(time.RFC3339)
	lr.ExternalIDs["updated_at"] = now.Format(time.RFC3339)

	// Create the OVN logical router
	ovnLR := &nbdb.LogicalRouter{
		UUID:        lr.UUID,
		Name:        lr.Name,
		Options:     lr.Options,
		ExternalIDs: lr.ExternalIDs,
	}

	// Add static routes if provided
	if len(lr.StaticRoutes) > 0 {
		staticRoutes := make([]string, 0, len(lr.StaticRoutes))
		for _, route := range lr.StaticRoutes {
			// Create static route entries
			sr := &nbdb.LogicalRouterStaticRoute{
				UUID:       uuid.New().String(),
				IPPrefix:   route.IPPrefix,
				Nexthop:    route.Nexthop,
				OutputPort: route.OutputPort,
				Policy:     route.Policy,
			}
			
			// Create the static route
			ops, err := c.nbClient.Create(sr)
			if err != nil {
				return nil, fmt.Errorf("failed to create static route: %w", err)
			}
			
			// Execute the transaction
			results, err := c.Transact(ctx, ops...)
			if err != nil {
				return nil, fmt.Errorf("failed to create static route: %w", err)
			}
			
			// Check results
			for _, result := range results {
				if result.Error != "" {
					return nil, fmt.Errorf("static route transaction error: %s", result.Error)
				}
			}
			
			staticRoutes = append(staticRoutes, sr.UUID)
		}
		ovnLR.StaticRoutes = staticRoutes
	}

	// Create the transaction
	ops, err := c.nbClient.Create(ovnLR)
	if err != nil {
		return nil, fmt.Errorf("failed to create logical router operations: %w", err)
	}

	// Execute the transaction
	results, err := c.Transact(ctx, ops...)
	if err != nil {
		return nil, fmt.Errorf("failed to create logical router: %w", err)
	}

	// Check results
	for _, result := range results {
		if result.Error != "" {
			return nil, fmt.Errorf("transaction error: %s", result.Error)
		}
	}

	// Set timestamps
	lr.CreatedAt = now
	lr.UpdatedAt = now

	return lr, nil
}

// UpdateLogicalRouter updates an existing logical router
func (c *Client) UpdateLogicalRouter(ctx context.Context, id string, updates *models.LogicalRouter) (*models.LogicalRouter, error) {
	if !c.IsConnected() {
		return nil, fmt.Errorf("client not connected")
	}

	// Get the existing router
	existing, err := c.GetLogicalRouter(ctx, id)
	if err != nil {
		return nil, err
	}

	// Update timestamp
	now := time.Now()
	if updates.ExternalIDs == nil {
		updates.ExternalIDs = existing.ExternalIDs
	}
	if updates.ExternalIDs == nil {
		updates.ExternalIDs = make(map[string]string)
	}
	updates.ExternalIDs["updated_at"] = now.Format(time.RFC3339)

	// Create the OVN logical router with updates
	ovnLR := &nbdb.LogicalRouter{
		UUID:        existing.UUID,
		Name:        existing.Name,
		Options:     existing.Options,
		ExternalIDs: existing.ExternalIDs,
	}

	// Apply updates
	if updates.Name != "" {
		ovnLR.Name = updates.Name
	}
	if updates.Options != nil {
		ovnLR.Options = updates.Options
	}
	if updates.ExternalIDs != nil {
		ovnLR.ExternalIDs = updates.ExternalIDs
	}

	// Create the transaction
	ops, err := c.nbClient.Where(ovnLR).Update(ovnLR, &ovnLR.Name, &ovnLR.Options, &ovnLR.ExternalIDs)
	if err != nil {
		return nil, fmt.Errorf("failed to create update operations: %w", err)
	}

	// Execute the transaction
	results, err := c.Transact(ctx, ops...)
	if err != nil {
		return nil, fmt.Errorf("failed to update logical router: %w", err)
	}

	// Check results
	for _, result := range results {
		if result.Error != "" {
			return nil, fmt.Errorf("transaction error: %s", result.Error)
		}
	}

	// Get the updated router
	return c.GetLogicalRouter(ctx, existing.UUID)
}

// DeleteLogicalRouter deletes a logical router
func (c *Client) DeleteLogicalRouter(ctx context.Context, id string) error {
	if !c.IsConnected() {
		return fmt.Errorf("client not connected")
	}

	// Get the existing router
	existing, err := c.GetLogicalRouter(ctx, id)
	if err != nil {
		return err
	}

	// Check if router has ports
	if len(existing.Ports) > 0 {
		return fmt.Errorf("cannot delete router: router has %d ports attached", len(existing.Ports))
	}

	// Create the OVN logical router for deletion
	ovnLR := &nbdb.LogicalRouter{
		UUID: existing.UUID,
	}

	// Create the transaction
	ops, err := c.nbClient.Where(ovnLR).Delete()
	if err != nil {
		return fmt.Errorf("failed to create delete operations: %w", err)
	}

	// Execute the transaction
	results, err := c.Transact(ctx, ops...)
	if err != nil {
		return fmt.Errorf("failed to delete logical router: %w", err)
	}

	// Check results
	for _, result := range results {
		if result.Error != "" {
			return fmt.Errorf("transaction error: %s", result.Error)
		}
	}

	return nil
}

// Helper function to convert OVN model to our model
func convertLogicalRouter(ovnLR *nbdb.LogicalRouter) *models.LogicalRouter {
	lr := &models.LogicalRouter{
		UUID:         ovnLR.UUID,
		Name:         ovnLR.Name,
		Ports:        ovnLR.Ports,
		Policies:     ovnLR.Policies,
		NAT:          []models.NAT{}, // Will need to fetch NAT rules separately
		LoadBalancer: ovnLR.LoadBalancer,
		Options:      ovnLR.Options,
		ExternalIDs:  ovnLR.ExternalIDs,
		CreatedAt:    time.Now(), // Default
		UpdatedAt:    time.Now(), // Default
	}

	// Convert static routes
	lr.StaticRoutes = make([]models.StaticRoute, 0)
	// Note: In a real implementation, we would need to fetch the actual static route objects
	// For now, we'll leave it empty as it requires additional queries

	// Try to parse timestamps from external_ids if available
	if created, ok := ovnLR.ExternalIDs["created_at"]; ok {
		if t, err := time.Parse(time.RFC3339, created); err == nil {
			lr.CreatedAt = t
		}
	}

	if updated, ok := ovnLR.ExternalIDs["updated_at"]; ok {
		if t, err := time.Parse(time.RFC3339, updated); err == nil {
			lr.UpdatedAt = t
		}
	}

	return lr
}