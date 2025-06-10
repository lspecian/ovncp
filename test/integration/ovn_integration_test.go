//go:build integration
// +build integration

package integration

import (
	"context"
	"fmt"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/lspecian/ovncp/internal/config"
	"github.com/lspecian/ovncp/internal/models"
	"github.com/lspecian/ovncp/pkg/ovn"
)

// Test configuration from environment
var (
	ovnNBAddr = getEnv("OVN_NB_ADDR", "tcp:localhost:6641")
	ovnSBAddr = getEnv("OVN_SB_ADDR", "tcp:localhost:6642")
)

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

func setupOVNClient(t *testing.T) *ovn.Client {
	cfg := &config.OVNConfig{
		NorthboundDB: ovnNBAddr,
		SouthboundDB: ovnSBAddr,
		Timeout:      30 * time.Second,
	}
	
	client, err := ovn.NewClient(cfg)
	require.NoError(t, err)
	
	ctx := context.Background()
	err = client.Connect(ctx)
	require.NoError(t, err)
	
	t.Cleanup(func() {
		// client.Disconnect() // Method doesn't exist
	})
	
	return client
}

// Logical Switch Integration Tests
func TestOVN_LogicalSwitch_CRUD(t *testing.T) {
	client := setupOVNClient(t)
	ctx := context.Background()
	
	// Create a unique switch name
	switchName := "test-switch-" + uuid.New().String()[:8]
	switchDesc := "Integration test switch"
	
	// Test Create
	t.Run("Create", func(t *testing.T) {
		ls, err := client.CreateLogicalSwitch(switchName, switchDesc)
		assert.NoError(t, err)
		assert.NotEmpty(t, ls.UUID)
		assert.Equal(t, switchName, ls.Name)
		assert.Equal(t, switchDesc, ls.Description)
	})
	
	// Test List
	t.Run("List", func(t *testing.T) {
		switches, err := client.ListLogicalSwitches()
		assert.NoError(t, err)
		
		found := false
		for _, s := range switches {
			if s.Name == switchName {
				found = true
				break
			}
		}
		assert.True(t, found, "Created switch should be in list")
	})
	
	// Test Get
	var switchUUID string
	t.Run("Get", func(t *testing.T) {
		switches, err := client.ListLogicalSwitches()
		require.NoError(t, err)
		
		for _, s := range switches {
			if s.Name == switchName {
				switchUUID = s.UUID
				break
			}
		}
		require.NotEmpty(t, switchUUID)
		
		ls, err := client.GetLogicalSwitch(switchUUID)
		assert.NoError(t, err)
		assert.Equal(t, switchName, ls.Name)
		assert.Equal(t, switchDesc, ls.Description)
	})
	
	// Test Update
	t.Run("Update", func(t *testing.T) {
		newDesc := "Updated description"
		updatedLS := &models.LogicalSwitch{
			Description: newDesc,
		}
		
		ls, err := client.UpdateLogicalSwitch(ctx, switchUUID, updatedLS)
		assert.NoError(t, err)
		assert.Equal(t, newDesc, ls.Description)
		
		// Verify update persisted
		ls, err = client.GetLogicalSwitch(switchUUID)
		assert.NoError(t, err)
		assert.Equal(t, newDesc, ls.Description)
	})
	
	// Test Delete
	t.Run("Delete", func(t *testing.T) {
		err := client.DeleteLogicalSwitch(switchUUID)
		assert.NoError(t, err)
		
		// Verify deletion
		_, err = client.GetLogicalSwitch(ctx, switchUUID)
		assert.Error(t, err)
	})
}

// Logical Router Integration Tests
func TestOVN_LogicalRouter_CRUD(t *testing.T) {
	client := setupOVNClient(t)
	
	routerName := "test-router-" + uuid.New().String()[:8]
	
	// Test Create
	var routerUUID string
	t.Run("Create", func(t *testing.T) {
		lr, err := client.CreateLogicalRouter(routerName, true)
		assert.NoError(t, err)
		assert.NotEmpty(t, lr.UUID)
		assert.Equal(t, routerName, lr.Name)
		assert.True(t, lr.Enabled)
		routerUUID = lr.UUID
	})
	
	// Test Get
	t.Run("Get", func(t *testing.T) {
		lr, err := client.GetLogicalRouter(routerUUID)
		assert.NoError(t, err)
		assert.Equal(t, routerName, lr.Name)
		assert.True(t, lr.Enabled)
	})
	
	// Test Update
	t.Run("Update", func(t *testing.T) {
		updates := map[string]interface{}{
			"enabled": false,
		}
		
		lr, err := client.UpdateLogicalRouter(routerUUID, updates)
		assert.NoError(t, err)
		assert.False(t, lr.Enabled)
	})
	
	// Test Delete
	t.Run("Delete", func(t *testing.T) {
		err := client.DeleteLogicalRouter(routerUUID)
		assert.NoError(t, err)
	})
}

// Logical Port Integration Tests
func TestOVN_LogicalPort_WithSwitch(t *testing.T) {
	client := setupOVNClient(t)
	
	// Create a switch first
	switchName := "port-test-switch-" + uuid.New().String()[:8]
	ls, err := client.CreateLogicalSwitch(switchName, "Switch for port testing")
	require.NoError(t, err)
	
	defer func() {
		_ = client.DeleteLogicalSwitch(ls.UUID)
	}()
	
	// Create port
	portName := "test-port-" + uuid.New().String()[:8]
	t.Run("CreatePort", func(t *testing.T) {
		req := ovn.CreateLogicalPortRequest{
			Name:      portName,
			Switch:    ls.UUID,
			Addresses: []string{"00:00:00:00:00:01 192.168.1.10"},
			Type:      "",
		}
		
		port, err := client.CreateLogicalPort(req)
		assert.NoError(t, err)
		assert.Equal(t, portName, port.Name)
		assert.Equal(t, ls.UUID, port.Switch)
		assert.Contains(t, port.Addresses, "00:00:00:00:00:01 192.168.1.10")
	})
	
	// List ports
	t.Run("ListPorts", func(t *testing.T) {
		ports, err := client.ListLogicalPorts()
		assert.NoError(t, err)
		
		found := false
		for _, p := range ports {
			if p.Name == portName {
				found = true
				assert.Equal(t, ls.UUID, p.Switch)
				break
			}
		}
		assert.True(t, found)
	})
}

// Transaction Integration Tests
func TestOVN_Transaction(t *testing.T) {
	client := setupOVNClient(t)
	
	// Create multiple resources in a single transaction
	t.Run("CreateMultiple", func(t *testing.T) {
		tx := client.CreateTransaction()
		
		switch1 := "tx-switch-1-" + uuid.New().String()[:8]
		switch2 := "tx-switch-2-" + uuid.New().String()[:8]
		router1 := "tx-router-1-" + uuid.New().String()[:8]
		
		tx.AddCreateSwitch(switch1, "Transaction switch 1")
		tx.AddCreateSwitch(switch2, "Transaction switch 2")
		tx.AddCreateRouter(router1, true)
		
		err := tx.Commit()
		assert.NoError(t, err)
		
		// Verify all resources were created
		switches, err := client.ListLogicalSwitches()
		assert.NoError(t, err)
		
		foundSwitch1 := false
		foundSwitch2 := false
		for _, s := range switches {
			if s.Name == switch1 {
				foundSwitch1 = true
			}
			if s.Name == switch2 {
				foundSwitch2 = true
			}
		}
		assert.True(t, foundSwitch1)
		assert.True(t, foundSwitch2)
		
		routers, err := client.ListLogicalRouters()
		assert.NoError(t, err)
		
		foundRouter := false
		for _, r := range routers {
			if r.Name == router1 {
				foundRouter = true
				break
			}
		}
		assert.True(t, foundRouter)
		
		// Cleanup
		for _, s := range switches {
			if s.Name == switch1 || s.Name == switch2 {
				_ = client.DeleteLogicalSwitch(s.UUID)
			}
		}
		for _, r := range routers {
			if r.Name == router1 {
				_ = client.DeleteLogicalRouter(r.UUID)
			}
		}
	})
	
	// Test transaction rollback on error
	t.Run("Rollback", func(t *testing.T) {
		tx := client.CreateTransaction()
		
		validSwitch := "tx-valid-" + uuid.New().String()[:8]
		tx.AddCreateSwitch(validSwitch, "Valid switch")
		
		// Add an operation that will fail (duplicate name)
		tx.AddCreateSwitch(validSwitch, "Duplicate name should fail")
		
		err := tx.Commit()
		assert.Error(t, err)
		
		// Verify no resources were created
		switches, err := client.ListLogicalSwitches()
		assert.NoError(t, err)
		
		found := false
		for _, s := range switches {
			if s.Name == validSwitch {
				found = true
				break
			}
		}
		assert.False(t, found, "Transaction should have rolled back")
	})
}

// ACL Integration Tests
func TestOVN_ACL_Operations(t *testing.T) {
	client := setupOVNClient(t)
	
	// Create a switch for ACL testing
	switchName := "acl-test-switch-" + uuid.New().String()[:8]
	ls, err := client.CreateLogicalSwitch(switchName, "Switch for ACL testing")
	require.NoError(t, err)
	
	defer func() {
		_ = client.DeleteLogicalSwitch(ls.UUID)
	}()
	
	t.Run("CreateACL", func(t *testing.T) {
		req := ovn.CreateACLRequest{
			Switch:    ls.UUID,
			Direction: "from-lport",
			Priority:  1000,
			Match:     "ip4.src == 192.168.1.0/24",
			Action:    "allow",
		}
		
		acl, err := client.CreateACL(req)
		assert.NoError(t, err)
		assert.Equal(t, req.Direction, acl.Direction)
		assert.Equal(t, req.Priority, acl.Priority)
		assert.Equal(t, req.Match, acl.Match)
		assert.Equal(t, req.Action, acl.Action)
		
		// Cleanup
		_ = client.DeleteACL(acl.UUID)
	})
}

// Load Balancer Integration Tests
func TestOVN_LoadBalancer_Operations(t *testing.T) {
	client := setupOVNClient(t)
	
	lbName := "test-lb-" + uuid.New().String()[:8]
	
	t.Run("CreateLoadBalancer", func(t *testing.T) {
		req := ovn.CreateLoadBalancerRequest{
			Name:     lbName,
			Protocol: "tcp",
			VIPs: map[string]string{
				"192.168.1.100:80": "192.168.1.10:8080,192.168.1.11:8080",
			},
		}
		
		lb, err := client.CreateLoadBalancer(req)
		assert.NoError(t, err)
		assert.Equal(t, lbName, lb.Name)
		assert.Equal(t, "tcp", lb.Protocol)
		assert.NotEmpty(t, lb.VIPs)
		
		// Cleanup
		_ = client.DeleteLoadBalancer(lb.UUID)
	})
}

// Performance Tests
func TestOVN_Performance_BulkOperations(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping performance test in short mode")
	}
	
	client := setupOVNClient(t)
	
	t.Run("BulkCreate", func(t *testing.T) {
		start := time.Now()
		
		// Create 100 switches
		tx := client.CreateTransaction()
		for i := 0; i < 100; i++ {
			name := fmt.Sprintf("perf-switch-%d", i)
			tx.AddCreateSwitch(name, "Performance test switch")
		}
		
		err := tx.Commit()
		assert.NoError(t, err)
		
		duration := time.Since(start)
		t.Logf("Created 100 switches in %v", duration)
		assert.Less(t, duration, 10*time.Second, "Bulk create should complete within 10 seconds")
		
		// Cleanup
		switches, _ := client.ListLogicalSwitches()
		for _, s := range switches {
			if strings.HasPrefix(s.Name, "perf-switch-") {
				_ = client.DeleteLogicalSwitch(s.UUID)
			}
		}
	})
}