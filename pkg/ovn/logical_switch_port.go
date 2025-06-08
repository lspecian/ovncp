package ovn

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/lspecian/ovncp/internal/models"
	"github.com/lspecian/ovncp/pkg/ovn/nbdb"
	"github.com/ovn-org/libovsdb/ovsdb"
)

// parseTime parses a time string from external_ids
func parseTime(timeStr string) time.Time {
	if timeStr == "" {
		return time.Time{}
	}
	t, err := time.Parse(time.RFC3339, timeStr)
	if err != nil {
		return time.Time{}
	}
	return t
}

// ListLogicalSwitchPorts returns all logical switch ports for a given switch
func (c *Client) ListLogicalSwitchPorts(ctx context.Context, switchID string) ([]*models.LogicalSwitchPort, error) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	if !c.connected {
		return nil, fmt.Errorf("client not connected")
	}

	// First get the switch to ensure it exists
	sw := &nbdb.LogicalSwitch{UUID: switchID}
	err := c.nbClient.Get(ctx, sw)
	if err != nil {
		return nil, fmt.Errorf("failed to get logical switch %s: %w", switchID, err)
	}

	// Get all ports for this switch
	portList := []nbdb.LogicalSwitchPort{}
	err = c.nbClient.WhereCache(func(port *nbdb.LogicalSwitchPort) bool {
		// Check if this port belongs to the switch
		for _, portUUID := range sw.Ports {
			if port.UUID == portUUID {
				return true
			}
		}
		return false
	}).List(ctx, &portList)
	if err != nil {
		return nil, fmt.Errorf("failed to list logical switch ports: %w", err)
	}

	ports := make([]*models.LogicalSwitchPort, len(portList))
	for i, port := range portList {
		ports[i] = c.nbdbPortToModel(&port)
	}

	return ports, nil
}

// GetLogicalSwitchPort returns a specific logical switch port by ID
func (c *Client) GetLogicalSwitchPort(ctx context.Context, id string) (*models.LogicalSwitchPort, error) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	if !c.connected {
		return nil, fmt.Errorf("client not connected")
	}

	port := &nbdb.LogicalSwitchPort{UUID: id}
	err := c.nbClient.Get(ctx, port)
	if err != nil {
		return nil, fmt.Errorf("logical switch port %s not found", id)
	}

	return c.nbdbPortToModel(port), nil
}

// CreateLogicalSwitchPort creates a new logical switch port
func (c *Client) CreateLogicalSwitchPort(ctx context.Context, switchID string, port *models.LogicalSwitchPort) (*models.LogicalSwitchPort, error) {
	c.mu.Lock()
	defer c.mu.Unlock()

	if !c.connected {
		return nil, fmt.Errorf("client not connected")
	}

	// First get the switch to ensure it exists
	sw := &nbdb.LogicalSwitch{UUID: switchID}
	err := c.nbClient.Get(ctx, sw)
	if err != nil {
		return nil, fmt.Errorf("failed to get logical switch %s: %w", switchID, err)
	}

	// Check if port with same name already exists
	existingPorts := []nbdb.LogicalSwitchPort{}
	err = c.nbClient.WhereCache(func(p *nbdb.LogicalSwitchPort) bool {
		return p.Name == port.Name
	}).List(ctx, &existingPorts)
	if err != nil {
		return nil, fmt.Errorf("failed to check existing ports: %w", err)
	}

	if len(existingPorts) > 0 {
		return nil, fmt.Errorf("port %s already exists", port.Name)
	}

	// Create the port
	portUUID := uuid.New().String()
	now := time.Now().Format(time.RFC3339)
	
	nbdbPort := &nbdb.LogicalSwitchPort{
		UUID:         portUUID,
		Name:         port.Name,
		Addresses:    port.Addresses,
		PortSecurity: port.PortSecurity,
		Type:         port.Type,
		Options:      port.Options,
		ExternalIDs: map[string]string{
			"created_at": now,
			"updated_at": now,
		},
	}

	// Set optional fields
	if port.Enabled != nil {
		nbdbPort.Enabled = port.Enabled
	}
	if port.Tag > 0 {
		nbdbPort.Tag = &port.Tag
	}
	if port.ParentName != "" {
		nbdbPort.ParentName = &port.ParentName
	}

	// Copy additional external IDs
	for k, v := range port.ExternalIDs {
		if k != "created_at" && k != "updated_at" {
			nbdbPort.ExternalIDs[k] = v
		}
	}

	// Start transaction
	ops := []ovsdb.Operation{}
	
	// Create the port
	createOp, err := c.nbClient.Create(nbdbPort)
	if err != nil {
		return nil, fmt.Errorf("failed to create port operation: %w", err)
	}
	ops = append(ops, createOp...)

	// Update the switch to include the new port
	sw.Ports = append(sw.Ports, portUUID)
	updateOp, err := c.nbClient.Where(sw).Update(sw, &sw.Ports)
	if err != nil {
		return nil, fmt.Errorf("failed to create switch update operation: %w", err)
	}
	ops = append(ops, updateOp...)

	// Execute transaction
	result, err := c.nbClient.Transact(ctx, ops...)
	if err != nil {
		return nil, fmt.Errorf("failed to execute transaction: %w", err)
	}

	if len(result) > 0 && result[0].Error != "" {
		return nil, fmt.Errorf("transaction failed: %s", result[0].Error)
	}

	// Set the UUID in the model
	port.UUID = portUUID
	port.SwitchID = switchID
	port.CreatedAt = parseTime(now)
	port.UpdatedAt = parseTime(now)

	return port, nil
}

// UpdateLogicalSwitchPort updates an existing logical switch port
func (c *Client) UpdateLogicalSwitchPort(ctx context.Context, id string, port *models.LogicalSwitchPort) (*models.LogicalSwitchPort, error) {
	c.mu.Lock()
	defer c.mu.Unlock()

	if !c.connected {
		return nil, fmt.Errorf("client not connected")
	}

	// Get existing port
	existing := &nbdb.LogicalSwitchPort{UUID: id}
	err := c.nbClient.Get(ctx, existing)
	if err != nil {
		return nil, fmt.Errorf("logical switch port %s not found", id)
	}

	// Update fields if provided
	if port.Name != "" && port.Name != existing.Name {
		existing.Name = port.Name
	}
	if len(port.Addresses) > 0 {
		existing.Addresses = port.Addresses
	}
	if len(port.PortSecurity) > 0 {
		existing.PortSecurity = port.PortSecurity
	}
	if port.Type != "" {
		existing.Type = port.Type
	}
	if port.Options != nil {
		existing.Options = port.Options
	}
	if port.Enabled != nil {
		existing.Enabled = port.Enabled
	}
	if port.Tag > 0 {
		existing.Tag = &port.Tag
	}

	// Update timestamp
	if existing.ExternalIDs == nil {
		existing.ExternalIDs = make(map[string]string)
	}
	existing.ExternalIDs["updated_at"] = time.Now().Format(time.RFC3339)

	// Copy additional external IDs
	for k, v := range port.ExternalIDs {
		if k != "created_at" && k != "updated_at" {
			existing.ExternalIDs[k] = v
		}
	}

	// Update the port
	ops, err := c.nbClient.Where(existing).Update(existing)
	if err != nil {
		return nil, fmt.Errorf("failed to create update operation: %w", err)
	}

	result, err := c.nbClient.Transact(ctx, ops...)
	if err != nil {
		return nil, fmt.Errorf("failed to update port: %w", err)
	}

	if len(result) > 0 && result[0].Error != "" {
		return nil, fmt.Errorf("update failed: %s", result[0].Error)
	}

	return c.nbdbPortToModel(existing), nil
}

// DeleteLogicalSwitchPort deletes a logical switch port
func (c *Client) DeleteLogicalSwitchPort(ctx context.Context, id string) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if !c.connected {
		return fmt.Errorf("client not connected")
	}

	// Get the port to ensure it exists
	port := &nbdb.LogicalSwitchPort{UUID: id}
	err := c.nbClient.Get(ctx, port)
	if err != nil {
		return fmt.Errorf("logical switch port %s not found", id)
	}

	// Find the switch that contains this port
	switches := []nbdb.LogicalSwitch{}
	err = c.nbClient.WhereCache(func(sw *nbdb.LogicalSwitch) bool {
		for _, portUUID := range sw.Ports {
			if portUUID == id {
				return true
			}
		}
		return false
	}).List(ctx, &switches)
	if err != nil {
		return fmt.Errorf("failed to find switch for port: %w", err)
	}

	if len(switches) == 0 {
		return fmt.Errorf("port %s is not attached to any switch", id)
	}

	sw := &switches[0]

	// Start transaction
	ops := []ovsdb.Operation{}

	// Remove port from switch
	newPorts := []string{}
	for _, portUUID := range sw.Ports {
		if portUUID != id {
			newPorts = append(newPorts, portUUID)
		}
	}
	sw.Ports = newPorts

	updateOp, err := c.nbClient.Where(&nbdb.LogicalSwitch{UUID: sw.UUID}).Update(sw, &sw.Ports)
	if err != nil {
		return fmt.Errorf("failed to create switch update operation: %w", err)
	}
	ops = append(ops, updateOp...)

	// Delete the port
	deleteOp, err := c.nbClient.Where(port).Delete()
	if err != nil {
		return fmt.Errorf("failed to create delete operation: %w", err)
	}
	ops = append(ops, deleteOp...)

	// Execute transaction
	result, err := c.nbClient.Transact(ctx, ops...)
	if err != nil {
		return fmt.Errorf("failed to delete port: %w", err)
	}

	if len(result) > 0 && result[0].Error != "" {
		return fmt.Errorf("delete failed: %s", result[0].Error)
	}

	return nil
}

// nbdbPortToModel converts an nbdb.LogicalSwitchPort to a models.LogicalSwitchPort
func (c *Client) nbdbPortToModel(port *nbdb.LogicalSwitchPort) *models.LogicalSwitchPort {
	m := &models.LogicalSwitchPort{
		UUID:         port.UUID,
		Name:         port.Name,
		Addresses:    port.Addresses,
		PortSecurity: port.PortSecurity,
		Type:         port.Type,
		Options:      port.Options,
		ExternalIDs:  port.ExternalIDs,
	}

	// Set optional fields
	if port.Enabled != nil {
		m.Enabled = port.Enabled
	}
	if port.Tag != nil {
		m.Tag = *port.Tag
	}
	if port.ParentName != nil {
		m.ParentName = *port.ParentName
	}
	if port.Up != nil {
		m.Up = port.Up
	}

	// Parse timestamps from external IDs
	if created, ok := port.ExternalIDs["created_at"]; ok {
		m.CreatedAt = parseTime(created)
	}
	if updated, ok := port.ExternalIDs["updated_at"]; ok {
		m.UpdatedAt = parseTime(updated)
	}

	return m
}