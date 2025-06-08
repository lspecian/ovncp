# OVN Go Libraries Analysis and Integration Guide

## Current State Analysis

Based on the codebase analysis, the OVNCP project has a skeleton structure for OVN integration but lacks actual implementation. The current code has:

1. **Basic structure in place:**
   - `/pkg/ovn/client.go` - Client wrapper with connection management placeholders
   - `/internal/services/ovn_service.go` - Service layer with CRUD operations for logical switches, routers, ports, and ACLs
   - All methods have TODO comments indicating missing implementation

2. **No OVN/OVSDB libraries currently imported** in `go.mod`

## Recommended Go Libraries for OVN Integration

### 1. **libovsdb by ovn-org** (RECOMMENDED)
- **Repository:** https://github.com/ovn-org/libovsdb
- **Import:** `github.com/ovn-org/libovsdb`

**Advantages:**
- Actively maintained by the OVN organization
- Model-based API using Go structs with tags
- Built-in code generation from OVSDB schemas
- Event notification system for database changes
- Efficient caching mechanism
- Used by major projects like ovn-kubernetes

**Installation:**
```bash
go get github.com/ovn-org/libovsdb
```

### 2. **go-ovn by eBay**
- **Repository:** https://github.com/eBay/go-ovn
- **Import:** `github.com/ebay/go-ovn`

**Advantages:**
- High-level abstraction specifically for OVN
- Pre-built methods for common OVN operations
- Simpler API for basic use cases

**Disadvantages:**
- Less flexible than libovsdb
- Uses a forked version of libovsdb
- May not support latest OVN features as quickly

### 3. **libovsdb by ovn-kubernetes**
- **Repository:** https://github.com/ovn-kubernetes/libovsdb
- **Import:** `github.com/ovn-kubernetes/libovsdb`

**Advantages:**
- Tailored for Kubernetes integration
- Similar features to ovn-org version

**Use case:** Only if building Kubernetes-specific integrations

## Recommended Approach: Using libovsdb

### Step 1: Install Dependencies
```bash
go get github.com/ovn-org/libovsdb
go get github.com/ovn-org/libovsdb/client
go get github.com/ovn-org/libovsdb/model
go get github.com/ovn-org/libovsdb/ovsdb
```

### Step 2: Generate Models from OVN Schema
```bash
# Install the model generator
go install github.com/ovn-org/libovsdb/cmd/modelgen@latest

# Generate models from OVN Northbound schema
modelgen -p nbdb -o ./internal/models/nbdb /usr/share/ovn/ovn-nb.ovsschema
```

### Step 3: Implementation Example

Here's how to implement the OVN client using libovsdb:

```go
package ovn

import (
    "context"
    "fmt"
    "time"

    "github.com/ovn-org/libovsdb/client"
    "github.com/ovn-org/libovsdb/model"
    "github.com/ovn-org/libovsdb/ovsdb"
    
    "github.com/lspecian/ovncp/internal/config"
    "github.com/lspecian/ovncp/internal/models/nbdb"
)

type Client struct {
    config   *config.OVNConfig
    nbClient client.Client
    sbClient client.Client
}

func NewClient(cfg *config.OVNConfig) (*Client, error) {
    return &Client{
        config: cfg,
    }, nil
}

func (c *Client) Connect(ctx context.Context) error {
    // Create database model for Northbound DB
    dbModel, err := nbdb.FullDatabaseModel()
    if err != nil {
        return fmt.Errorf("creating database model: %w", err)
    }

    // Connect to Northbound DB
    nbClient, err := client.NewOVSDBClient(
        dbModel,
        client.WithEndpoint(c.config.NorthboundDB),
        client.WithTimeout(c.config.Timeout),
    )
    if err != nil {
        return fmt.Errorf("creating northbound client: %w", err)
    }

    if err := nbClient.Connect(ctx); err != nil {
        return fmt.Errorf("connecting to northbound DB: %w", err)
    }

    // Monitor all tables for cache
    if err := nbClient.MonitorAll(ctx); err != nil {
        return fmt.Errorf("monitoring northbound DB: %w", err)
    }

    c.nbClient = nbClient
    return nil
}

// Example: Create Logical Switch
func (c *Client) CreateLogicalSwitch(ctx context.Context, name string) (*nbdb.LogicalSwitch, error) {
    ls := &nbdb.LogicalSwitch{
        Name: name,
    }

    ops, err := c.nbClient.Create(ls)
    if err != nil {
        return nil, fmt.Errorf("creating logical switch: %w", err)
    }

    if err := c.nbClient.Transact(ctx, ops...); err != nil {
        return nil, fmt.Errorf("transacting create: %w", err)
    }

    return ls, nil
}

// Example: List Logical Switches
func (c *Client) ListLogicalSwitches(ctx context.Context) ([]*nbdb.LogicalSwitch, error) {
    lsList := []*nbdb.LogicalSwitch{}
    err := c.nbClient.List(ctx, &lsList)
    return lsList, err
}

// Example: Create ACL
func (c *Client) CreateACL(ctx context.Context, switchName string, acl *nbdb.ACL) error {
    // First, get the logical switch
    ls := &nbdb.LogicalSwitch{Name: switchName}
    err := c.nbClient.Get(ctx, ls)
    if err != nil {
        return fmt.Errorf("getting logical switch: %w", err)
    }

    // Create the ACL
    aclOps, err := c.nbClient.Create(acl)
    if err != nil {
        return fmt.Errorf("creating ACL: %w", err)
    }

    // Update the logical switch to include the ACL
    ls.ACLs = append(ls.ACLs, acl.UUID)
    updateOps, err := c.nbClient.Where(ls).Update(ls)
    if err != nil {
        return fmt.Errorf("updating logical switch: %w", err)
    }

    // Combine operations in a single transaction
    ops := append(aclOps, updateOps...)
    return c.nbClient.Transact(ctx, ops...)
}
```

### Step 4: Service Layer Integration

Update the service layer to use the new client implementation:

```go
func (s *OVNService) ListLogicalSwitches(ctx context.Context) ([]*models.LogicalSwitch, error) {
    switches, err := s.client.ListLogicalSwitches(ctx)
    if err != nil {
        return nil, err
    }

    // Convert from nbdb.LogicalSwitch to models.LogicalSwitch
    result := make([]*models.LogicalSwitch, len(switches))
    for i, sw := range switches {
        result[i] = &models.LogicalSwitch{
            UUID: sw.UUID,
            Name: sw.Name,
            // Map other fields as needed
        }
    }
    return result, nil
}
```

## Best Practices

1. **Use Model Generation**: Always generate models from the OVN schema to ensure type safety and compatibility
2. **Transaction Batching**: Group multiple operations in a single transaction for better performance
3. **Cache Usage**: Leverage libovsdb's built-in cache for read operations
4. **Error Handling**: Implement proper retry logic for transient failures
5. **Connection Management**: Implement health checks and reconnection logic
6. **Event Monitoring**: Use event handlers for real-time updates

## Example Features to Implement

1. **Logical Switch Management**
   - Create/Delete logical switches
   - Add/Remove ports
   - Configure DHCP options

2. **Logical Router Management**
   - Create/Delete logical routers
   - Configure routing policies
   - Set up NAT rules

3. **ACL Management**
   - Create security policies
   - Configure port-level ACLs
   - Implement address sets

4. **Load Balancer Configuration**
   - Create load balancer entries
   - Configure health checks
   - Set up VIPs

5. **QoS Rules**
   - Configure bandwidth limits
   - Set DSCP marking
   - Implement traffic shaping

## Migration Path

1. Add libovsdb to go.mod
2. Generate models from OVN schema
3. Implement the Client wrapper with actual OVSDB operations
4. Update service layer to use the new client
5. Add integration tests with a test OVN instance
6. Implement connection pooling and retry logic
7. Add monitoring and metrics

This approach provides a solid foundation for building a production-ready OVN control plane in Go.