//go:build integration
// +build integration

package integration

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
	"github.com/lspecian/ovncp/internal/api"
	"github.com/lspecian/ovncp/internal/auth"
	"github.com/lspecian/ovncp/internal/config"
	"github.com/lspecian/ovncp/internal/db"
	"github.com/lspecian/ovncp/pkg/ovn"
	"github.com/lspecian/ovncp/internal/models"
	"github.com/lspecian/ovncp/internal/services"
)

func setupTestServer(t *testing.T) *httptest.Server {
	// Setup configuration
	cfg := &config.Config{
		API: config.APIConfig{
			Port: "0", // Random port
			Host: "localhost",
		},
		Database: config.DatabaseConfig{
			Host:     getEnv("DB_HOST", "localhost"),
			Port:     "5432",
			Name:     "ovncp_test",
			User:     getEnv("DB_USER", "ovncp_test"),
			Password: getEnv("DB_PASSWORD", "test_password"),
			SSLMode:  "disable",
		},
		OVN: config.OVNConfig{
			NorthboundDB: ovnNBAddr,
			SouthboundDB: ovnSBAddr,
			Timeout:      30 * time.Second,
		},
		Auth: config.AuthConfig{
			Enabled: false, // Disable auth for integration tests
		},
	}
	
	// Initialize database
	database, err := db.New(&cfg.Database)
	require.NoError(t, err)
	
	// Run migrations
	err = database.Migrate()
	require.NoError(t, err)
	
	// Initialize OVN client
	ovnClient, err := ovn.NewClient(&cfg.OVN)
	require.NoError(t, err)
	
	ctx := context.Background()
	err = ovnClient.Connect(ctx)
	require.NoError(t, err)
	
	// Initialize service
	// svc := services.NewOVNService(ovnClient)
	
	// Setup router
	gin.SetMode(gin.TestMode)
	router := gin.New()
	// TODO: Setup API routes properly
	// Integration tests need to be updated to match new architecture
	_ = ovnClient // temporary: suppress unused variable warning
	
	// Create test server
	server := httptest.NewServer(router)
	
	t.Cleanup(func() {
		server.Close()
		// ovnClient.Disconnect() // Method doesn't exist
		database.Close()
	})
	
	return server
}

// API Integration Tests
func TestAPI_LogicalSwitch_FullWorkflow(t *testing.T) {
	server := setupTestServer(t)
	client := &http.Client{Timeout: 10 * time.Second}
	
	var switchID string
	switchName := "api-test-switch-" + uuid.New().String()[:8]
	
	// Create switch
	t.Run("Create", func(t *testing.T) {
		reqBody := map[string]interface{}{
			"name":        switchName,
			"description": "API integration test switch",
		}
		
		body, _ := json.Marshal(reqBody)
		req, _ := http.NewRequest("POST", server.URL+"/api/v1/switches", bytes.NewBuffer(body))
		req.Header.Set("Content-Type", "application/json")
		
		resp, err := client.Do(req)
		require.NoError(t, err)
		defer resp.Body.Close()
		
		assert.Equal(t, http.StatusCreated, resp.StatusCode)
		
		var result map[string]interface{}
		err = json.NewDecoder(resp.Body).Decode(&result)
		require.NoError(t, err)
		
		switchID = result["uuid"].(string)
		assert.Equal(t, switchName, result["name"])
	})
	
	// List switches
	t.Run("List", func(t *testing.T) {
		req, _ := http.NewRequest("GET", server.URL+"/api/v1/switches", nil)
		
		resp, err := client.Do(req)
		require.NoError(t, err)
		defer resp.Body.Close()
		
		assert.Equal(t, http.StatusOK, resp.StatusCode)
		
		var switches []map[string]interface{}
		err = json.NewDecoder(resp.Body).Decode(&switches)
		require.NoError(t, err)
		
		found := false
		for _, s := range switches {
			if s["name"] == switchName {
				found = true
				break
			}
		}
		assert.True(t, found)
	})
	
	// Get specific switch
	t.Run("Get", func(t *testing.T) {
		req, _ := http.NewRequest("GET", server.URL+"/api/v1/switches/"+switchID, nil)
		
		resp, err := client.Do(req)
		require.NoError(t, err)
		defer resp.Body.Close()
		
		assert.Equal(t, http.StatusOK, resp.StatusCode)
		
		var result map[string]interface{}
		err = json.NewDecoder(resp.Body).Decode(&result)
		require.NoError(t, err)
		
		assert.Equal(t, switchName, result["name"])
	})
	
	// Update switch
	t.Run("Update", func(t *testing.T) {
		reqBody := map[string]interface{}{
			"description": "Updated description",
		}
		
		body, _ := json.Marshal(reqBody)
		req, _ := http.NewRequest("PUT", server.URL+"/api/v1/switches/"+switchID, bytes.NewBuffer(body))
		req.Header.Set("Content-Type", "application/json")
		
		resp, err := client.Do(req)
		require.NoError(t, err)
		defer resp.Body.Close()
		
		assert.Equal(t, http.StatusOK, resp.StatusCode)
		
		var result map[string]interface{}
		err = json.NewDecoder(resp.Body).Decode(&result)
		require.NoError(t, err)
		
		assert.Equal(t, "Updated description", result["description"])
	})
	
	// Delete switch
	t.Run("Delete", func(t *testing.T) {
		req, _ := http.NewRequest("DELETE", server.URL+"/api/v1/switches/"+switchID, nil)
		
		resp, err := client.Do(req)
		require.NoError(t, err)
		defer resp.Body.Close()
		
		assert.Equal(t, http.StatusNoContent, resp.StatusCode)
		
		// Verify deletion
		req, _ = http.NewRequest("GET", server.URL+"/api/v1/switches/"+switchID, nil)
		resp, err = client.Do(req)
		require.NoError(t, err)
		defer resp.Body.Close()
		
		assert.Equal(t, http.StatusNotFound, resp.StatusCode)
	})
}

func TestAPI_NetworkTopology(t *testing.T) {
	server := setupTestServer(t)
	client := &http.Client{Timeout: 10 * time.Second}
	
	// Create test resources
	switchName := "topo-switch-" + uuid.New().String()[:8]
	routerName := "topo-router-" + uuid.New().String()[:8]
	
	// Create switch
	switchReq := map[string]interface{}{
		"name":        switchName,
		"description": "Topology test switch",
	}
	body, _ := json.Marshal(switchReq)
	req, _ := http.NewRequest("POST", server.URL+"/api/v1/switches", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	resp, err := client.Do(req)
	require.NoError(t, err)
	resp.Body.Close()
	
	// Create router
	routerReq := map[string]interface{}{
		"name":    routerName,
		"enabled": true,
	}
	body, _ = json.Marshal(routerReq)
	req, _ = http.NewRequest("POST", server.URL+"/api/v1/routers", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	resp, err = client.Do(req)
	require.NoError(t, err)
	resp.Body.Close()
	
	// Get topology
	t.Run("GetTopology", func(t *testing.T) {
		req, _ := http.NewRequest("GET", server.URL+"/api/v1/topology", nil)
		
		resp, err := client.Do(req)
		require.NoError(t, err)
		defer resp.Body.Close()
		
		assert.Equal(t, http.StatusOK, resp.StatusCode)
		
		var topology map[string]interface{}
		err = json.NewDecoder(resp.Body).Decode(&topology)
		require.NoError(t, err)
		
		assert.Contains(t, topology, "switches")
		assert.Contains(t, topology, "routers")
		assert.Contains(t, topology, "ports")
		
		// Check if our created resources are in topology
		switches := topology["switches"].([]interface{})
		routers := topology["routers"].([]interface{})
		
		foundSwitch := false
		for _, s := range switches {
			sw := s.(map[string]interface{})
			if sw["name"] == switchName {
				foundSwitch = true
				break
			}
		}
		assert.True(t, foundSwitch)
		
		foundRouter := false
		for _, r := range routers {
			rt := r.(map[string]interface{})
			if rt["name"] == routerName {
				foundRouter = true
				break
			}
		}
		assert.True(t, foundRouter)
	})
}

func TestAPI_Transaction(t *testing.T) {
	server := setupTestServer(t)
	client := &http.Client{Timeout: 10 * time.Second}
	
	t.Run("BatchCreate", func(t *testing.T) {
		reqBody := map[string]interface{}{
			"operations": []map[string]interface{}{
				{
					"type":   "create_switch",
					"params": map[string]string{
						"name":        "batch-switch-1",
						"description": "Batch created switch 1",
					},
				},
				{
					"type":   "create_switch",
					"params": map[string]string{
						"name":        "batch-switch-2",
						"description": "Batch created switch 2",
					},
				},
				{
					"type":   "create_router",
					"params": map[string]interface{}{
						"name":    "batch-router-1",
						"enabled": true,
					},
				},
			},
		}
		
		body, _ := json.Marshal(reqBody)
		req, _ := http.NewRequest("POST", server.URL+"/api/v1/transaction", bytes.NewBuffer(body))
		req.Header.Set("Content-Type", "application/json")
		
		resp, err := client.Do(req)
		require.NoError(t, err)
		defer resp.Body.Close()
		
		assert.Equal(t, http.StatusOK, resp.StatusCode)
		
		var result map[string]interface{}
		err = json.NewDecoder(resp.Body).Decode(&result)
		require.NoError(t, err)
		
		assert.Equal(t, "success", result["status"])
		
		// Verify resources were created
		req, _ = http.NewRequest("GET", server.URL+"/api/v1/switches", nil)
		resp, err = client.Do(req)
		require.NoError(t, err)
		defer resp.Body.Close()
		
		var switches []map[string]interface{}
		err = json.NewDecoder(resp.Body).Decode(&switches)
		require.NoError(t, err)
		
		foundCount := 0
		for _, s := range switches {
			name := s["name"].(string)
			if name == "batch-switch-1" || name == "batch-switch-2" {
				foundCount++
			}
		}
		assert.Equal(t, 2, foundCount)
	})
}

func TestAPI_ErrorHandling(t *testing.T) {
	server := setupTestServer(t)
	client := &http.Client{Timeout: 10 * time.Second}
	
	t.Run("InvalidJSON", func(t *testing.T) {
		req, _ := http.NewRequest("POST", server.URL+"/api/v1/switches", bytes.NewBufferString("invalid json"))
		req.Header.Set("Content-Type", "application/json")
		
		resp, err := client.Do(req)
		require.NoError(t, err)
		defer resp.Body.Close()
		
		assert.Equal(t, http.StatusBadRequest, resp.StatusCode)
	})
	
	t.Run("MissingRequiredField", func(t *testing.T) {
		reqBody := map[string]interface{}{
			"description": "Missing name field",
		}
		
		body, _ := json.Marshal(reqBody)
		req, _ := http.NewRequest("POST", server.URL+"/api/v1/switches", bytes.NewBuffer(body))
		req.Header.Set("Content-Type", "application/json")
		
		resp, err := client.Do(req)
		require.NoError(t, err)
		defer resp.Body.Close()
		
		assert.Equal(t, http.StatusBadRequest, resp.StatusCode)
	})
	
	t.Run("NotFound", func(t *testing.T) {
		req, _ := http.NewRequest("GET", server.URL+"/api/v1/switches/"+uuid.New().String(), nil)
		
		resp, err := client.Do(req)
		require.NoError(t, err)
		defer resp.Body.Close()
		
		assert.Equal(t, http.StatusNotFound, resp.StatusCode)
	})
	
	t.Run("DuplicateName", func(t *testing.T) {
		switchName := "duplicate-test-" + uuid.New().String()[:8]
		
		// Create first switch
		reqBody := map[string]interface{}{
			"name": switchName,
		}
		body, _ := json.Marshal(reqBody)
		req, _ := http.NewRequest("POST", server.URL+"/api/v1/switches", bytes.NewBuffer(body))
		req.Header.Set("Content-Type", "application/json")
		
		resp, err := client.Do(req)
		require.NoError(t, err)
		resp.Body.Close()
		assert.Equal(t, http.StatusCreated, resp.StatusCode)
		
		// Try to create duplicate
		req, _ = http.NewRequest("POST", server.URL+"/api/v1/switches", bytes.NewBuffer(body))
		req.Header.Set("Content-Type", "application/json")
		
		resp, err = client.Do(req)
		require.NoError(t, err)
		defer resp.Body.Close()
		
		assert.Equal(t, http.StatusConflict, resp.StatusCode)
	})
}

func TestAPI_HealthCheck(t *testing.T) {
	server := setupTestServer(t)
	client := &http.Client{Timeout: 10 * time.Second}
	
	req, _ := http.NewRequest("GET", server.URL+"/health", nil)
	
	resp, err := client.Do(req)
	require.NoError(t, err)
	defer resp.Body.Close()
	
	assert.Equal(t, http.StatusOK, resp.StatusCode)
	
	var health map[string]interface{}
	err = json.NewDecoder(resp.Body).Decode(&health)
	require.NoError(t, err)
	
	assert.Equal(t, "healthy", health["status"])
	assert.Contains(t, health, "services")
	
	services := health["services"].(map[string]interface{})
	assert.Equal(t, "connected", services["ovn"])
	assert.Equal(t, "connected", services["database"])
}

// Concurrent API Tests
func TestAPI_ConcurrentOperations(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping concurrent test in short mode")
	}
	
	server := setupTestServer(t)
	client := &http.Client{Timeout: 10 * time.Second}
	
	// Create 10 switches concurrently
	numGoroutines := 10
	done := make(chan bool, numGoroutines)
	errors := make(chan error, numGoroutines)
	
	for i := 0; i < numGoroutines; i++ {
		go func(id int) {
			switchName := fmt.Sprintf("concurrent-switch-%d", id)
			reqBody := map[string]interface{}{
				"name":        switchName,
				"description": fmt.Sprintf("Concurrent test switch %d", id),
			}
			
			body, _ := json.Marshal(reqBody)
			req, _ := http.NewRequest("POST", server.URL+"/api/v1/switches", bytes.NewBuffer(body))
			req.Header.Set("Content-Type", "application/json")
			
			resp, err := client.Do(req)
			if err != nil {
				errors <- err
				done <- false
				return
			}
			defer resp.Body.Close()
			
			if resp.StatusCode != http.StatusCreated {
				errors <- fmt.Errorf("unexpected status code: %d", resp.StatusCode)
				done <- false
				return
			}
			
			done <- true
		}(i)
	}
	
	// Wait for all goroutines
	successCount := 0
	for i := 0; i < numGoroutines; i++ {
		if <-done {
			successCount++
		}
	}
	
	close(errors)
	for err := range errors {
		t.Errorf("Concurrent operation error: %v", err)
	}
	
	assert.Equal(t, numGoroutines, successCount)
	
	// Verify all switches were created
	req, _ := http.NewRequest("GET", server.URL+"/api/v1/switches", nil)
	resp, err := client.Do(req)
	require.NoError(t, err)
	defer resp.Body.Close()
	
	var switches []map[string]interface{}
	err = json.NewDecoder(resp.Body).Decode(&switches)
	require.NoError(t, err)
	
	concurrentCount := 0
	for _, s := range switches {
		name := s["name"].(string)
		if strings.HasPrefix(name, "concurrent-switch-") {
			concurrentCount++
		}
	}
	assert.Equal(t, numGoroutines, concurrentCount)
}