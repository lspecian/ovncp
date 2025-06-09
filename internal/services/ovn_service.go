package services

import (
	"context"
	"fmt"

	"github.com/lspecian/ovncp/internal/models"
	"github.com/lspecian/ovncp/pkg/ovn"
)

type OVNService struct {
	client *ovn.Client
}

func NewOVNService(client *ovn.Client) *OVNService {
	return &OVNService{
		client: client,
	}
}

// GetOVNClient returns the underlying OVN client
func (s *OVNService) GetOVNClient() *ovn.Client {
	return s.client
}

func (s *OVNService) ListLogicalSwitches(ctx context.Context) ([]*models.LogicalSwitch, error) {
	return s.client.ListLogicalSwitches(ctx)
}

func (s *OVNService) GetLogicalSwitch(ctx context.Context, id string) (*models.LogicalSwitch, error) {
	return s.client.GetLogicalSwitch(ctx, id)
}

func (s *OVNService) CreateLogicalSwitch(ctx context.Context, ls *models.LogicalSwitch) (*models.LogicalSwitch, error) {
	// Validate input
	if ls.Name == "" {
		return nil, fmt.Errorf("logical switch name is required")
	}

	return s.client.CreateLogicalSwitch(ctx, ls)
}

func (s *OVNService) UpdateLogicalSwitch(ctx context.Context, id string, ls *models.LogicalSwitch) (*models.LogicalSwitch, error) {
	// Validate input
	if id == "" {
		return nil, fmt.Errorf("logical switch ID is required")
	}

	return s.client.UpdateLogicalSwitch(ctx, id, ls)
}

func (s *OVNService) DeleteLogicalSwitch(ctx context.Context, id string) error {
	// Validate input
	if id == "" {
		return fmt.Errorf("logical switch ID is required")
	}

	return s.client.DeleteLogicalSwitch(ctx, id)
}

func (s *OVNService) ListLogicalRouters(ctx context.Context) ([]*models.LogicalRouter, error) {
	return s.client.ListLogicalRouters(ctx)
}

func (s *OVNService) GetLogicalRouter(ctx context.Context, id string) (*models.LogicalRouter, error) {
	return s.client.GetLogicalRouter(ctx, id)
}

func (s *OVNService) CreateLogicalRouter(ctx context.Context, lr *models.LogicalRouter) (*models.LogicalRouter, error) {
	// Validate input
	if lr.Name == "" {
		return nil, fmt.Errorf("logical router name is required")
	}

	return s.client.CreateLogicalRouter(ctx, lr)
}

func (s *OVNService) UpdateLogicalRouter(ctx context.Context, id string, lr *models.LogicalRouter) (*models.LogicalRouter, error) {
	// Validate input
	if id == "" {
		return nil, fmt.Errorf("logical router ID is required")
	}

	return s.client.UpdateLogicalRouter(ctx, id, lr)
}

func (s *OVNService) DeleteLogicalRouter(ctx context.Context, id string) error {
	// Validate input
	if id == "" {
		return fmt.Errorf("logical router ID is required")
	}

	return s.client.DeleteLogicalRouter(ctx, id)
}

func (s *OVNService) ListPorts(ctx context.Context, switchID string) ([]*models.LogicalSwitchPort, error) {
	// Validate input
	if switchID == "" {
		return nil, fmt.Errorf("switch ID is required")
	}

	return s.client.ListLogicalSwitchPorts(ctx, switchID)
}

func (s *OVNService) GetPort(ctx context.Context, id string) (*models.LogicalSwitchPort, error) {
	// Validate input
	if id == "" {
		return nil, fmt.Errorf("port ID is required")
	}

	return s.client.GetLogicalSwitchPort(ctx, id)
}

func (s *OVNService) CreatePort(ctx context.Context, switchID string, port *models.LogicalSwitchPort) (*models.LogicalSwitchPort, error) {
	// Validate input
	if switchID == "" {
		return nil, fmt.Errorf("switch ID is required")
	}
	if port.Name == "" {
		return nil, fmt.Errorf("port name is required")
	}

	return s.client.CreateLogicalSwitchPort(ctx, switchID, port)
}

func (s *OVNService) UpdatePort(ctx context.Context, id string, port *models.LogicalSwitchPort) (*models.LogicalSwitchPort, error) {
	// Validate input
	if id == "" {
		return nil, fmt.Errorf("port ID is required")
	}

	return s.client.UpdateLogicalSwitchPort(ctx, id, port)
}

func (s *OVNService) DeletePort(ctx context.Context, id string) error {
	// Validate input
	if id == "" {
		return fmt.Errorf("port ID is required")
	}

	return s.client.DeleteLogicalSwitchPort(ctx, id)
}

func (s *OVNService) ListACLs(ctx context.Context, switchID string) ([]*models.ACL, error) {
	// Validate input
	if switchID == "" {
		return nil, fmt.Errorf("switch ID is required")
	}

	return s.client.ListACLs(ctx, switchID)
}

func (s *OVNService) GetACL(ctx context.Context, id string) (*models.ACL, error) {
	// Validate input
	if id == "" {
		return nil, fmt.Errorf("ACL ID is required")
	}

	return s.client.GetACL(ctx, id)
}

func (s *OVNService) CreateACL(ctx context.Context, switchID string, acl *models.ACL) (*models.ACL, error) {
	// Validate input
	if switchID == "" {
		return nil, fmt.Errorf("switch ID is required")
	}
	if acl.Match == "" {
		return nil, fmt.Errorf("ACL match expression is required")
	}
	if acl.Action == "" {
		return nil, fmt.Errorf("ACL action is required")
	}
	if acl.Direction == "" {
		return nil, fmt.Errorf("ACL direction is required")
	}

	return s.client.CreateACL(ctx, switchID, acl)
}

func (s *OVNService) UpdateACL(ctx context.Context, id string, acl *models.ACL) (*models.ACL, error) {
	// Validate input
	if id == "" {
		return nil, fmt.Errorf("ACL ID is required")
	}

	return s.client.UpdateACL(ctx, id, acl)
}

func (s *OVNService) DeleteACL(ctx context.Context, id string) error {
	// Validate input
	if id == "" {
		return fmt.Errorf("ACL ID is required")
	}

	return s.client.DeleteACL(ctx, id)
}

