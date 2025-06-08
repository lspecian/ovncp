package ovn

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"

	"github.com/lspecian/ovncp/internal/models"
	"github.com/lspecian/ovncp/pkg/ovn/nbdb"
)

// ListLogicalSwitches returns all logical switches
func (c *Client) ListLogicalSwitches(ctx context.Context) ([]*models.LogicalSwitch, error) {
	if !c.IsConnected() {
		return nil, fmt.Errorf("client not connected")
	}

	// Get all logical switches from database
	lsList := []nbdb.LogicalSwitch{}
	err := c.nbClient.List(ctx, &lsList)
	if err != nil {
		return nil, fmt.Errorf("failed to list logical switches: %w", err)
	}

	// Convert to our model
	result := make([]*models.LogicalSwitch, 0, len(lsList))
	for i := range lsList {
		result = append(result, convertLogicalSwitch(&lsList[i]))
	}

	return result, nil
}

// GetLogicalSwitch returns a specific logical switch by UUID or name
func (c *Client) GetLogicalSwitch(ctx context.Context, id string) (*models.LogicalSwitch, error) {
	if !c.IsConnected() {
		return nil, fmt.Errorf("client not connected")
	}

	// First, list all switches and find by UUID or name
	lsList := []nbdb.LogicalSwitch{}
	err := c.nbClient.List(ctx, &lsList)
	if err != nil {
		return nil, fmt.Errorf("failed to list logical switches: %w", err)
	}

	// Look for match by UUID or name
	for i := range lsList {
		if lsList[i].UUID == id || lsList[i].Name == id {
			return convertLogicalSwitch(&lsList[i]), nil
		}
	}

	return nil, fmt.Errorf("logical switch %s not found", id)
}

// CreateLogicalSwitch creates a new logical switch
func (c *Client) CreateLogicalSwitch(ctx context.Context, ls *models.LogicalSwitch) (*models.LogicalSwitch, error) {
	if !c.IsConnected() {
		return nil, fmt.Errorf("client not connected")
	}

	// Generate UUID if not provided
	if ls.UUID == "" {
		ls.UUID = uuid.New().String()
	}

	// Add timestamps to external_ids
	now := time.Now()
	if ls.ExternalIDs == nil {
		ls.ExternalIDs = make(map[string]string)
	}
	ls.ExternalIDs["created_at"] = now.Format(time.RFC3339)
	ls.ExternalIDs["updated_at"] = now.Format(time.RFC3339)

	// Create the OVN logical switch
	ovnLS := &nbdb.LogicalSwitch{
		UUID:        ls.UUID,
		Name:        ls.Name,
		OtherConfig: ls.OtherConfig,
		ExternalIDs: ls.ExternalIDs,
	}

	// Create the transaction
	ops, err := c.nbClient.Create(ovnLS)
	if err != nil {
		return nil, fmt.Errorf("failed to create logical switch operations: %w", err)
	}

	// Execute the transaction
	results, err := c.Transact(ctx, ops...)
	if err != nil {
		return nil, fmt.Errorf("failed to create logical switch: %w", err)
	}

	// Check results
	for _, result := range results {
		if result.Error != "" {
			return nil, fmt.Errorf("transaction error: %s", result.Error)
		}
	}

	// Set timestamps
	ls.CreatedAt = now
	ls.UpdatedAt = now

	return ls, nil
}

// UpdateLogicalSwitch updates an existing logical switch
func (c *Client) UpdateLogicalSwitch(ctx context.Context, id string, updates *models.LogicalSwitch) (*models.LogicalSwitch, error) {
	if !c.IsConnected() {
		return nil, fmt.Errorf("client not connected")
	}

	// Get the existing switch
	existing, err := c.GetLogicalSwitch(ctx, id)
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

	// Create the OVN logical switch with updates
	ovnLS := &nbdb.LogicalSwitch{
		UUID: existing.UUID,
		Name: existing.Name,
		OtherConfig: existing.OtherConfig,
		ExternalIDs: existing.ExternalIDs,
	}

	// Apply updates
	if updates.Name != "" {
		ovnLS.Name = updates.Name
	}
	if updates.OtherConfig != nil {
		ovnLS.OtherConfig = updates.OtherConfig
	}
	if updates.ExternalIDs != nil {
		ovnLS.ExternalIDs = updates.ExternalIDs
	}

	// Create the transaction
	ops, err := c.nbClient.Where(ovnLS).Update(ovnLS, &ovnLS.Name, &ovnLS.OtherConfig, &ovnLS.ExternalIDs)
	if err != nil {
		return nil, fmt.Errorf("failed to create update operations: %w", err)
	}

	// Execute the transaction
	results, err := c.Transact(ctx, ops...)
	if err != nil {
		return nil, fmt.Errorf("failed to update logical switch: %w", err)
	}

	// Check results
	for _, result := range results {
		if result.Error != "" {
			return nil, fmt.Errorf("transaction error: %s", result.Error)
		}
	}

	// Get the updated switch
	return c.GetLogicalSwitch(ctx, existing.UUID)
}

// DeleteLogicalSwitch deletes a logical switch
func (c *Client) DeleteLogicalSwitch(ctx context.Context, id string) error {
	if !c.IsConnected() {
		return fmt.Errorf("client not connected")
	}

	// Get the existing switch
	existing, err := c.GetLogicalSwitch(ctx, id)
	if err != nil {
		return err
	}

	// Create the OVN logical switch for deletion
	ovnLS := &nbdb.LogicalSwitch{
		UUID: existing.UUID,
	}

	// Create the transaction
	ops, err := c.nbClient.Where(ovnLS).Delete()
	if err != nil {
		return fmt.Errorf("failed to create delete operations: %w", err)
	}

	// Execute the transaction
	results, err := c.Transact(ctx, ops...)
	if err != nil {
		return fmt.Errorf("failed to delete logical switch: %w", err)
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
func convertLogicalSwitch(ovnLS *nbdb.LogicalSwitch) *models.LogicalSwitch {
	ls := &models.LogicalSwitch{
		UUID:        ovnLS.UUID,
		Name:        ovnLS.Name,
		Ports:       ovnLS.Ports,
		ACLs:        ovnLS.ACLs,
		QoSRules:    ovnLS.QOSRules,
		LoadBalancer: ovnLS.LoadBalancer,
		DNSRecords:  ovnLS.DNSRecords,
		OtherConfig: ovnLS.OtherConfig,
		ExternalIDs: ovnLS.ExternalIDs,
		CreatedAt:   time.Now(), // Default
		UpdatedAt:   time.Now(), // Default
	}

	// Try to parse timestamps from external_ids if available
	if created, ok := ovnLS.ExternalIDs["created_at"]; ok {
		if t, err := time.Parse(time.RFC3339, created); err == nil {
			ls.CreatedAt = t
		}
	}

	if updated, ok := ovnLS.ExternalIDs["updated_at"]; ok {
		if t, err := time.Parse(time.RFC3339, updated); err == nil {
			ls.UpdatedAt = t
		}
	}

	return ls
}