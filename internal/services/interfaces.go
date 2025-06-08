package services

import (
	"context"

	"github.com/lspecian/ovncp/internal/models"
)

// OVNServiceInterface defines the interface for OVN operations
type OVNServiceInterface interface {
	// Logical Switch operations
	ListLogicalSwitches(ctx context.Context) ([]*models.LogicalSwitch, error)
	GetLogicalSwitch(ctx context.Context, id string) (*models.LogicalSwitch, error)
	CreateLogicalSwitch(ctx context.Context, ls *models.LogicalSwitch) (*models.LogicalSwitch, error)
	UpdateLogicalSwitch(ctx context.Context, id string, ls *models.LogicalSwitch) (*models.LogicalSwitch, error)
	DeleteLogicalSwitch(ctx context.Context, id string) error

	// Logical Router operations
	ListLogicalRouters(ctx context.Context) ([]*models.LogicalRouter, error)
	GetLogicalRouter(ctx context.Context, id string) (*models.LogicalRouter, error)
	CreateLogicalRouter(ctx context.Context, lr *models.LogicalRouter) (*models.LogicalRouter, error)
	UpdateLogicalRouter(ctx context.Context, id string, lr *models.LogicalRouter) (*models.LogicalRouter, error)
	DeleteLogicalRouter(ctx context.Context, id string) error

	// Port operations
	ListPorts(ctx context.Context, switchID string) ([]*models.LogicalSwitchPort, error)
	GetPort(ctx context.Context, id string) (*models.LogicalSwitchPort, error)
	CreatePort(ctx context.Context, switchID string, port *models.LogicalSwitchPort) (*models.LogicalSwitchPort, error)
	UpdatePort(ctx context.Context, id string, port *models.LogicalSwitchPort) (*models.LogicalSwitchPort, error)
	DeletePort(ctx context.Context, id string) error

	// ACL operations
	ListACLs(ctx context.Context, switchID string) ([]*models.ACL, error)
	GetACL(ctx context.Context, id string) (*models.ACL, error)
	CreateACL(ctx context.Context, switchID string, acl *models.ACL) (*models.ACL, error)
	UpdateACL(ctx context.Context, id string, acl *models.ACL) (*models.ACL, error)
	DeleteACL(ctx context.Context, id string) error
	
	// Transaction operations
	ExecuteTransaction(ctx context.Context, ops []TransactionOp) error
	
	// Topology operations
	GetTopology(ctx context.Context) (*Topology, error)
}

// Ensure OVNService implements OVNServiceInterface
var _ OVNServiceInterface = (*OVNService)(nil)