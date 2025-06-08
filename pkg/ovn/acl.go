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

// ListACLs returns all ACLs for a given switch
func (c *Client) ListACLs(ctx context.Context, switchID string) ([]*models.ACL, error) {
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

	// Get all ACLs for this switch
	aclList := []nbdb.ACL{}
	err = c.nbClient.WhereCache(func(acl *nbdb.ACL) bool {
		// Check if this ACL belongs to the switch
		for _, aclUUID := range sw.ACLs {
			if acl.UUID == aclUUID {
				return true
			}
		}
		return false
	}).List(ctx, &aclList)
	if err != nil {
		return nil, fmt.Errorf("failed to list ACLs: %w", err)
	}

	acls := make([]*models.ACL, len(aclList))
	for i, acl := range aclList {
		acls[i] = c.nbdbACLToModel(&acl)
	}

	return acls, nil
}

// GetACL returns a specific ACL by ID
func (c *Client) GetACL(ctx context.Context, id string) (*models.ACL, error) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	if !c.connected {
		return nil, fmt.Errorf("client not connected")
	}

	acl := &nbdb.ACL{UUID: id}
	err := c.nbClient.Get(ctx, acl)
	if err != nil {
		return nil, fmt.Errorf("ACL %s not found", id)
	}

	return c.nbdbACLToModel(acl), nil
}

// CreateACL creates a new ACL
func (c *Client) CreateACL(ctx context.Context, switchID string, acl *models.ACL) (*models.ACL, error) {
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

	// Validate ACL fields
	if err := validateACL(acl); err != nil {
		return nil, err
	}

	// Create the ACL
	aclUUID := uuid.New().String()
	now := time.Now().Format(time.RFC3339)
	
	nbdbACL := &nbdb.ACL{
		UUID:      aclUUID,
		Action:    nbdb.ACLAction(acl.Action),
		Direction: nbdb.ACLDirection(acl.Direction),
		Match:     acl.Match,
		Priority:  acl.Priority,
		Log:       acl.Log,
		ExternalIDs: map[string]string{
			"created_at": now,
			"updated_at": now,
		},
	}

	// Set optional fields
	if acl.Name != "" {
		nbdbACL.Name = &acl.Name
	}
	if acl.Severity != "" {
		severity := nbdb.ACLSeverity(acl.Severity)
		nbdbACL.Severity = &severity
	}

	// Copy additional external IDs
	for k, v := range acl.ExternalIDs {
		if k != "created_at" && k != "updated_at" {
			nbdbACL.ExternalIDs[k] = v
		}
	}

	// Start transaction
	ops := []ovsdb.Operation{}
	
	// Create the ACL
	createOp, err := c.nbClient.Create(nbdbACL)
	if err != nil {
		return nil, fmt.Errorf("failed to create ACL operation: %w", err)
	}
	ops = append(ops, createOp...)

	// Update the switch to include the new ACL
	sw.ACLs = append(sw.ACLs, aclUUID)
	updateOp, err := c.nbClient.Where(sw).Update(sw, &sw.ACLs)
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
	acl.UUID = aclUUID
	acl.CreatedAt = parseTime(now)
	acl.UpdatedAt = parseTime(now)

	return acl, nil
}

// UpdateACL updates an existing ACL
func (c *Client) UpdateACL(ctx context.Context, id string, acl *models.ACL) (*models.ACL, error) {
	c.mu.Lock()
	defer c.mu.Unlock()

	if !c.connected {
		return nil, fmt.Errorf("client not connected")
	}

	// Get existing ACL
	existing := &nbdb.ACL{UUID: id}
	err := c.nbClient.Get(ctx, existing)
	if err != nil {
		return nil, fmt.Errorf("ACL %s not found", id)
	}

	// Update fields if provided
	if acl.Action != "" {
		existing.Action = nbdb.ACLAction(acl.Action)
	}
	if acl.Direction != "" {
		existing.Direction = nbdb.ACLDirection(acl.Direction)
	}
	if acl.Match != "" {
		existing.Match = acl.Match
	}
	if acl.Priority > 0 {
		existing.Priority = acl.Priority
	}
	existing.Log = acl.Log

	if acl.Name != "" {
		existing.Name = &acl.Name
	}
	if acl.Severity != "" {
		severity := nbdb.ACLSeverity(acl.Severity)
		existing.Severity = &severity
	}

	// Update timestamp
	if existing.ExternalIDs == nil {
		existing.ExternalIDs = make(map[string]string)
	}
	existing.ExternalIDs["updated_at"] = time.Now().Format(time.RFC3339)

	// Copy additional external IDs
	for k, v := range acl.ExternalIDs {
		if k != "created_at" && k != "updated_at" {
			existing.ExternalIDs[k] = v
		}
	}

	// Update the ACL
	ops, err := c.nbClient.Where(existing).Update(existing)
	if err != nil {
		return nil, fmt.Errorf("failed to create update operation: %w", err)
	}

	result, err := c.nbClient.Transact(ctx, ops...)
	if err != nil {
		return nil, fmt.Errorf("failed to update ACL: %w", err)
	}

	if len(result) > 0 && result[0].Error != "" {
		return nil, fmt.Errorf("update failed: %s", result[0].Error)
	}

	return c.nbdbACLToModel(existing), nil
}

// DeleteACL deletes an ACL
func (c *Client) DeleteACL(ctx context.Context, id string) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if !c.connected {
		return fmt.Errorf("client not connected")
	}

	// Get the ACL to ensure it exists
	acl := &nbdb.ACL{UUID: id}
	err := c.nbClient.Get(ctx, acl)
	if err != nil {
		return fmt.Errorf("ACL %s not found", id)
	}

	// Find the switch that contains this ACL
	switches := []nbdb.LogicalSwitch{}
	err = c.nbClient.WhereCache(func(sw *nbdb.LogicalSwitch) bool {
		for _, aclUUID := range sw.ACLs {
			if aclUUID == id {
				return true
			}
		}
		return false
	}).List(ctx, &switches)
	if err != nil {
		return fmt.Errorf("failed to find switch for ACL: %w", err)
	}

	if len(switches) == 0 {
		return fmt.Errorf("ACL %s is not attached to any switch", id)
	}

	sw := &switches[0]

	// Start transaction
	ops := []ovsdb.Operation{}

	// Remove ACL from switch
	newACLs := []string{}
	for _, aclUUID := range sw.ACLs {
		if aclUUID != id {
			newACLs = append(newACLs, aclUUID)
		}
	}
	sw.ACLs = newACLs

	updateOp, err := c.nbClient.Where(&nbdb.LogicalSwitch{UUID: sw.UUID}).Update(sw, &sw.ACLs)
	if err != nil {
		return fmt.Errorf("failed to create switch update operation: %w", err)
	}
	ops = append(ops, updateOp...)

	// Delete the ACL
	deleteOp, err := c.nbClient.Where(acl).Delete()
	if err != nil {
		return fmt.Errorf("failed to create delete operation: %w", err)
	}
	ops = append(ops, deleteOp...)

	// Execute transaction
	result, err := c.nbClient.Transact(ctx, ops...)
	if err != nil {
		return fmt.Errorf("failed to delete ACL: %w", err)
	}

	if len(result) > 0 && result[0].Error != "" {
		return fmt.Errorf("delete failed: %s", result[0].Error)
	}

	return nil
}

// nbdbACLToModel converts an nbdb.ACL to a models.ACL
func (c *Client) nbdbACLToModel(acl *nbdb.ACL) *models.ACL {
	m := &models.ACL{
		UUID:        acl.UUID,
		Priority:    acl.Priority,
		Direction:   string(acl.Direction),
		Match:       acl.Match,
		Action:      string(acl.Action),
		Log:         acl.Log,
		ExternalIDs: acl.ExternalIDs,
	}

	// Set optional fields
	if acl.Name != nil {
		m.Name = *acl.Name
	}
	if acl.Severity != nil {
		m.Severity = string(*acl.Severity)
	}
	// Note: Alert field doesn't exist in nbdb.ACL, we'll set it based on severity
	m.Alert = acl.Severity != nil && (*acl.Severity == nbdb.ACLSeverityAlert || *acl.Severity == nbdb.ACLSeverityWarning)

	// Parse timestamps from external IDs
	if created, ok := acl.ExternalIDs["created_at"]; ok {
		m.CreatedAt = parseTime(created)
	}
	if updated, ok := acl.ExternalIDs["updated_at"]; ok {
		m.UpdatedAt = parseTime(updated)
	}

	return m
}

// validateACL validates ACL fields
func validateACL(acl *models.ACL) error {
	// Validate action
	validActions := []string{"allow", "allow-related", "allow-stateless", "drop", "reject", "pass"}
	isValidAction := false
	for _, valid := range validActions {
		if acl.Action == valid {
			isValidAction = true
			break
		}
	}
	if !isValidAction {
		return fmt.Errorf("invalid action: %s", acl.Action)
	}

	// Validate direction
	if acl.Direction != "from-lport" && acl.Direction != "to-lport" {
		return fmt.Errorf("invalid direction: %s", acl.Direction)
	}

	// Validate priority
	if acl.Priority < 0 || acl.Priority > 65535 {
		return fmt.Errorf("priority must be between 0 and 65535")
	}

	// Validate match expression
	if acl.Match == "" {
		return fmt.Errorf("match expression is required")
	}

	// Validate severity if provided
	if acl.Severity != "" {
		validSeverities := []string{"alert", "warning", "notice", "info", "debug"}
		isValidSeverity := false
		for _, valid := range validSeverities {
			if acl.Severity == valid {
				isValidSeverity = true
				break
			}
		}
		if !isValidSeverity {
			return fmt.Errorf("invalid severity: %s", acl.Severity)
		}
	}

	return nil
}