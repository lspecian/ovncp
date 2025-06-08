package ovn

import (
	"context"
	"fmt"
	"log"
	"sync"
	"time"

	"github.com/ovn-org/libovsdb/client"
	"github.com/ovn-org/libovsdb/model"
	"github.com/ovn-org/libovsdb/ovsdb"
	
	"github.com/lspecian/ovncp/internal/config"
	"github.com/lspecian/ovncp/pkg/ovn/nbdb"
)

type Client struct {
	config     *config.OVNConfig
	mu         sync.RWMutex
	nbClient   client.Client
	connected  bool
	closed     bool
	lastPing   time.Time
}

// DatabaseModel returns the OVN Northbound database model
func DatabaseModel() model.ClientDBModel {
	dbModel, _ := model.NewClientDBModel("OVN_Northbound", map[string]model.Model{
		"Logical_Switch":              &nbdb.LogicalSwitch{},
		"Logical_Switch_Port":         &nbdb.LogicalSwitchPort{},
		"Logical_Router":              &nbdb.LogicalRouter{},
		"Logical_Router_Port":         &nbdb.LogicalRouterPort{},
		"Logical_Router_Static_Route": &nbdb.LogicalRouterStaticRoute{},
		"ACL":                         &nbdb.ACL{},
		"Address_Set":                 &nbdb.AddressSet{},
		"Port_Group":                  &nbdb.PortGroup{},
		"Load_Balancer":               &nbdb.LoadBalancer{},
		"Load_Balancer_Group":         &nbdb.LoadBalancerGroup{},
		"NAT":                         &nbdb.NAT{},
		"DHCP_Options":                &nbdb.DHCPOptions{},
		"QoS":                         &nbdb.QoS{},
		"Meter":                       &nbdb.Meter{},
		"Meter_Band":                  &nbdb.MeterBand{},
		"DNS":                         &nbdb.DNS{},
		"Connection":                  &nbdb.Connection{},
		"SSL":                         &nbdb.SSL{},
		"NB_Global":                   &nbdb.NBGlobal{},
	})
	return dbModel
}

func NewClient(cfg *config.OVNConfig) (*Client, error) {
	dbModel := DatabaseModel()

	// Create the OVSDB client
	ovnClient, err := client.NewOVSDBClient(dbModel)
	if err != nil {
		return nil, fmt.Errorf("failed to create OVSDB client: %w", err)
	}

	c := &Client{
		config:   cfg,
		nbClient: ovnClient,
	}

	return c, nil
}

func (c *Client) Connect(ctx context.Context) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.connected {
		return nil
	}

	// Connect to the database
	if err := c.nbClient.Connect(ctx); err != nil {
		return fmt.Errorf("failed to connect to OVN northbound DB: %w", err)
	}

	// Monitor all tables we're interested in
	monitor := c.nbClient.NewMonitor(
		client.WithTable(&nbdb.LogicalSwitch{}),
		client.WithTable(&nbdb.LogicalSwitchPort{}),
		client.WithTable(&nbdb.LogicalRouter{}),
		client.WithTable(&nbdb.LogicalRouterPort{}),
		client.WithTable(&nbdb.ACL{}),
		client.WithTable(&nbdb.LoadBalancer{}),
		client.WithTable(&nbdb.NAT{}),
		client.WithTable(&nbdb.PortGroup{}),
		client.WithTable(&nbdb.AddressSet{}),
	)
	
	_, err := c.nbClient.Monitor(ctx, monitor)
	if err != nil {
		return fmt.Errorf("failed to start monitoring: %w", err)
	}

	c.connected = true
	log.Println("Successfully connected to OVN northbound database")
	
	return nil
}

func (c *Client) Close() error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if !c.connected || c.closed {
		return nil
	}

	c.nbClient.Close()
	c.connected = false
	c.closed = true
	
	return nil
}

func (c *Client) IsConnected() bool {
	c.mu.RLock()
	defer c.mu.RUnlock()

	return c.connected && !c.closed
}

// IsClosed returns true if the client has been closed
func (c *Client) IsClosed() bool {
	c.mu.RLock()
	defer c.mu.RUnlock()

	return c.closed
}

// Ping checks if the connection is alive
func (c *Client) Ping(ctx context.Context) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if !c.connected || c.closed {
		return fmt.Errorf("client not connected")
	}

	// Use a simple list operation to check connectivity
	// List NB_Global which should always exist and have minimal data
	nbGlobal := &nbdb.NBGlobal{}
	err := c.nbClient.List(ctx, nbGlobal)
	if err != nil {
		return fmt.Errorf("ping failed: %w", err)
	}

	c.lastPing = time.Now()
	return nil
}

func (c *Client) ExecuteWithRetry(ctx context.Context, fn func() error) error {
	var lastErr error
	for i := 0; i < c.config.MaxRetries; i++ {
		if err := fn(); err != nil {
			lastErr = err
			if i < c.config.MaxRetries-1 {
				time.Sleep(time.Duration(i+1) * time.Second)
			}
			continue
		}
		return nil
	}
	return fmt.Errorf("operation failed after %d retries: %w", c.config.MaxRetries, lastErr)
}

// Transact executes a transaction with the given operations
func (c *Client) Transact(ctx context.Context, ops ...ovsdb.Operation) ([]ovsdb.OperationResult, error) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	if !c.connected {
		return nil, fmt.Errorf("client not connected")
	}

	return c.nbClient.Transact(ctx, ops...)
}

// GetClient returns the underlying OVSDB client
func (c *Client) GetClient() client.Client {
	c.mu.RLock()
	defer c.mu.RUnlock()

	return c.nbClient
}