//go:build e2e
// +build e2e

package e2e

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/playwright-community/playwright-go"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var (
	baseURL    = getEnv("E2E_BASE_URL", "http://localhost:3000")
	apiURL     = getEnv("E2E_API_URL", "http://localhost:8080")
	headless   = getEnv("E2E_HEADLESS", "true") == "true"
	slowMo     = 100.0 // milliseconds between actions for visibility
)

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

// Test complete network creation workflow
func TestE2E_CreateNetworkTopology(t *testing.T) {
	// Initialize Playwright
	pw, err := playwright.Run()
	require.NoError(t, err)
	defer pw.Stop()
	
	// Launch browser
	browser, err := pw.Chromium.Launch(playwright.BrowserTypeLaunchOptions{
		Headless: &headless,
		SlowMo:   &slowMo,
	})
	require.NoError(t, err)
	defer browser.Close()
	
	// Create context and page
	context, err := browser.NewContext()
	require.NoError(t, err)
	defer context.Close()
	
	page, err := context.NewPage()
	require.NoError(t, err)
	
	// Test data
	testID := uuid.New().String()[:8]
	switchName1 := fmt.Sprintf("e2e-switch-1-%s", testID)
	switchName2 := fmt.Sprintf("e2e-switch-2-%s", testID)
	routerName := fmt.Sprintf("e2e-router-%s", testID)
	
	// Navigate to application
	t.Run("NavigateToApp", func(t *testing.T) {
		_, err := page.Goto(baseURL)
		require.NoError(t, err)
		
		// Wait for app to load
		_, err = page.WaitForSelector("[data-testid='dashboard']", playwright.PageWaitForSelectorOptions{
			Timeout: playwright.Float(10000),
		})
		require.NoError(t, err)
	})
	
	// Create first switch
	t.Run("CreateFirstSwitch", func(t *testing.T) {
		// Navigate to switches page
		err := page.Click("[data-testid='nav-switches']")
		require.NoError(t, err)
		
		// Click create button
		err = page.Click("[data-testid='create-switch-btn']")
		require.NoError(t, err)
		
		// Fill form
		err = page.Fill("[data-testid='switch-name-input']", switchName1)
		require.NoError(t, err)
		
		err = page.Fill("[data-testid='switch-description-input']", "E2E test switch 1")
		require.NoError(t, err)
		
		// Submit form
		err = page.Click("[data-testid='submit-btn']")
		require.NoError(t, err)
		
		// Wait for success notification
		_, err = page.WaitForSelector("[data-testid='success-notification']", playwright.PageWaitForSelectorOptions{
			Timeout: playwright.Float(5000),
		})
		require.NoError(t, err)
		
		// Verify switch appears in list
		_, err = page.WaitForSelector(fmt.Sprintf("text=%s", switchName1), playwright.PageWaitForSelectorOptions{
			Timeout: playwright.Float(5000),
		})
		require.NoError(t, err)
	})
	
	// Create second switch
	t.Run("CreateSecondSwitch", func(t *testing.T) {
		err := page.Click("[data-testid='create-switch-btn']")
		require.NoError(t, err)
		
		err = page.Fill("[data-testid='switch-name-input']", switchName2)
		require.NoError(t, err)
		
		err = page.Fill("[data-testid='switch-description-input']", "E2E test switch 2")
		require.NoError(t, err)
		
		err = page.Click("[data-testid='submit-btn']")
		require.NoError(t, err)
		
		_, err = page.WaitForSelector("[data-testid='success-notification']", playwright.PageWaitForSelectorOptions{
			Timeout: playwright.Float(5000),
		})
		require.NoError(t, err)
	})
	
	// Create router
	t.Run("CreateRouter", func(t *testing.T) {
		// Navigate to routers page
		err := page.Click("[data-testid='nav-routers']")
		require.NoError(t, err)
		
		// Click create button
		err = page.Click("[data-testid='create-router-btn']")
		require.NoError(t, err)
		
		// Fill form
		err = page.Fill("[data-testid='router-name-input']", routerName)
		require.NoError(t, err)
		
		// Enable router
		err = page.Check("[data-testid='router-enabled-checkbox']")
		require.NoError(t, err)
		
		// Submit form
		err = page.Click("[data-testid='submit-btn']")
		require.NoError(t, err)
		
		// Wait for success
		_, err = page.WaitForSelector("[data-testid='success-notification']", playwright.PageWaitForSelectorOptions{
			Timeout: playwright.Float(5000),
		})
		require.NoError(t, err)
	})
	
	// View network topology
	t.Run("ViewTopology", func(t *testing.T) {
		// Navigate to topology view
		err := page.Click("[data-testid='nav-topology']")
		require.NoError(t, err)
		
		// Wait for topology to load
		_, err = page.WaitForSelector("[data-testid='network-topology']", playwright.PageWaitForSelectorOptions{
			Timeout: playwright.Float(10000),
		})
		require.NoError(t, err)
		
		// Take screenshot for debugging
		screenshotPath := fmt.Sprintf("topology-%s.png", testID)
		_, err = page.Screenshot(playwright.PageScreenshotOptions{
			Path: &screenshotPath,
		})
		assert.NoError(t, err)
		
		// Verify all components are visible
		_, err = page.WaitForSelector(fmt.Sprintf("[data-testid='node-switch-%s']", switchName1), playwright.PageWaitForSelectorOptions{
			Timeout: playwright.Float(5000),
		})
		require.NoError(t, err)
		
		_, err = page.WaitForSelector(fmt.Sprintf("[data-testid='node-switch-%s']", switchName2), playwright.PageWaitForSelectorOptions{
			Timeout: playwright.Float(5000),
		})
		require.NoError(t, err)
		
		_, err = page.WaitForSelector(fmt.Sprintf("[data-testid='node-router-%s']", routerName), playwright.PageWaitForSelectorOptions{
			Timeout: playwright.Float(5000),
		})
		require.NoError(t, err)
	})
	
	// Connect router to switches
	t.Run("ConnectComponents", func(t *testing.T) {
		// Click on router node
		err := page.Click(fmt.Sprintf("[data-testid='node-router-%s']", routerName))
		require.NoError(t, err)
		
		// Click connect button
		err = page.Click("[data-testid='connect-btn']")
		require.NoError(t, err)
		
		// Select first switch
		err = page.Click(fmt.Sprintf("[data-testid='connect-to-switch-%s']", switchName1))
		require.NoError(t, err)
		
		// Fill port details
		err = page.Fill("[data-testid='port-name-input']", fmt.Sprintf("port-%s-to-%s", routerName, switchName1))
		require.NoError(t, err)
		
		err = page.Fill("[data-testid='port-ip-input']", "192.168.1.1/24")
		require.NoError(t, err)
		
		// Create connection
		err = page.Click("[data-testid='create-connection-btn']")
		require.NoError(t, err)
		
		// Wait for connection to appear
		_, err = page.WaitForSelector("[data-testid='connection-line']", playwright.PageWaitForSelectorOptions{
			Timeout: playwright.Float(5000),
		})
		require.NoError(t, err)
	})
	
	// Verify via API
	t.Run("VerifyViaAPI", func(t *testing.T) {
		client := &http.Client{Timeout: 10 * time.Second}
		
		// Get topology from API
		resp, err := client.Get(apiURL + "/api/v1/topology")
		require.NoError(t, err)
		defer resp.Body.Close()
		
		assert.Equal(t, http.StatusOK, resp.StatusCode)
		
		var topology map[string]interface{}
		err = json.NewDecoder(resp.Body).Decode(&topology)
		require.NoError(t, err)
		
		// Verify switches exist
		switches := topology["switches"].([]interface{})
		switchCount := 0
		for _, s := range switches {
			sw := s.(map[string]interface{})
			name := sw["name"].(string)
			if name == switchName1 || name == switchName2 {
				switchCount++
			}
		}
		assert.GreaterOrEqual(t, switchCount, 2)
		
		// Verify router exists
		routers := topology["routers"].([]interface{})
		routerFound := false
		for _, r := range routers {
			rt := r.(map[string]interface{})
			if rt["name"] == routerName {
				routerFound = true
				break
			}
		}
		assert.True(t, routerFound)
	})
	
	// Cleanup
	t.Run("Cleanup", func(t *testing.T) {
		// Delete resources via API
		client := &http.Client{Timeout: 10 * time.Second}
		
		// Get all switches and delete test ones
		resp, err := client.Get(apiURL + "/api/v1/switches")
		require.NoError(t, err)
		
		var switches []map[string]interface{}
		err = json.NewDecoder(resp.Body).Decode(&switches)
		require.NoError(t, err)
		resp.Body.Close()
		
		for _, s := range switches {
			name := s["name"].(string)
			if name == switchName1 || name == switchName2 {
				req, _ := http.NewRequest("DELETE", apiURL+"/api/v1/switches/"+s["uuid"].(string), nil)
				resp, err := client.Do(req)
				assert.NoError(t, err)
				resp.Body.Close()
			}
		}
		
		// Get all routers and delete test ones
		resp, err = client.Get(apiURL + "/api/v1/routers")
		require.NoError(t, err)
		
		var routers []map[string]interface{}
		err = json.NewDecoder(resp.Body).Decode(&routers)
		require.NoError(t, err)
		resp.Body.Close()
		
		for _, r := range routers {
			if r["name"] == routerName {
				req, _ := http.NewRequest("DELETE", apiURL+"/api/v1/routers/"+r["uuid"].(string), nil)
				resp, err := client.Do(req)
				assert.NoError(t, err)
				resp.Body.Close()
			}
		}
	})
}

// Test authentication flow
func TestE2E_AuthenticationFlow(t *testing.T) {
	if getEnv("AUTH_ENABLED", "false") != "true" {
		t.Skip("Skipping auth test - authentication not enabled")
	}
	
	pw, err := playwright.Run()
	require.NoError(t, err)
	defer pw.Stop()
	
	browser, err := pw.Chromium.Launch(playwright.BrowserTypeLaunchOptions{
		Headless: &headless,
		SlowMo:   &slowMo,
	})
	require.NoError(t, err)
	defer browser.Close()
	
	context, err := browser.NewContext()
	require.NoError(t, err)
	defer context.Close()
	
	page, err := context.NewPage()
	require.NoError(t, err)
	
	t.Run("RedirectToLogin", func(t *testing.T) {
		_, err := page.Goto(baseURL)
		require.NoError(t, err)
		
		// Should redirect to login
		err = page.WaitForURL("**/login**", playwright.PageWaitForURLOptions{
			Timeout: playwright.Float(5000),
		})
		require.NoError(t, err)
	})
	
	t.Run("LoginWithOAuth", func(t *testing.T) {
		// Click OAuth login button
		err := page.Click("[data-testid='oauth-login-btn']")
		require.NoError(t, err)
		
		// Handle OAuth provider login (mock for testing)
		// In real test, would interact with actual OAuth provider
		
		// Wait for redirect back to app
		err = page.WaitForURL(baseURL+"/**", playwright.PageWaitForURLOptions{
			Timeout: playwright.Float(10000),
		})
		require.NoError(t, err)
		
		// Verify logged in
		_, err = page.WaitForSelector("[data-testid='user-menu']", playwright.PageWaitForSelectorOptions{
			Timeout: playwright.Float(5000),
		})
		require.NoError(t, err)
	})
}

// Test error handling and recovery
func TestE2E_ErrorHandling(t *testing.T) {
	pw, err := playwright.Run()
	require.NoError(t, err)
	defer pw.Stop()
	
	browser, err := pw.Chromium.Launch(playwright.BrowserTypeLaunchOptions{
		Headless: &headless,
		SlowMo:   &slowMo,
	})
	require.NoError(t, err)
	defer browser.Close()
	
	context, err := browser.NewContext()
	require.NoError(t, err)
	defer context.Close()
	
	page, err := context.NewPage()
	require.NoError(t, err)
	
	// Navigate to app
	_, err = page.Goto(baseURL)
	require.NoError(t, err)
	
	t.Run("HandleDuplicateName", func(t *testing.T) {
		// Navigate to switches
		err := page.Click("[data-testid='nav-switches']")
		require.NoError(t, err)
		
		// Create first switch
		err = page.Click("[data-testid='create-switch-btn']")
		require.NoError(t, err)
		
		dupName := "dup-test-" + uuid.New().String()[:8]
		err = page.Fill("[data-testid='switch-name-input']", dupName)
		require.NoError(t, err)
		
		err = page.Click("[data-testid='submit-btn']")
		require.NoError(t, err)
		
		// Wait for success
		_, err = page.WaitForSelector("[data-testid='success-notification']", playwright.PageWaitForSelectorOptions{
			Timeout: playwright.Float(5000),
		})
		require.NoError(t, err)
		
		// Try to create duplicate
		err = page.Click("[data-testid='create-switch-btn']")
		require.NoError(t, err)
		
		err = page.Fill("[data-testid='switch-name-input']", dupName)
		require.NoError(t, err)
		
		err = page.Click("[data-testid='submit-btn']")
		require.NoError(t, err)
		
		// Should show error
		_, err = page.WaitForSelector("[data-testid='error-notification']", playwright.PageWaitForSelectorOptions{
			Timeout: playwright.Float(5000),
		})
		require.NoError(t, err)
		
		// Verify error message
		errorText, err := page.TextContent("[data-testid='error-notification']")
		require.NoError(t, err)
		assert.Contains(t, errorText, "already exists")
	})
	
	t.Run("HandleNetworkError", func(t *testing.T) {
		// Simulate network offline
		err := context.SetOffline(true)
		require.NoError(t, err)
		
		// Try to create switch
		err = page.Click("[data-testid='create-switch-btn']")
		require.NoError(t, err)
		
		err = page.Fill("[data-testid='switch-name-input']", "offline-test")
		require.NoError(t, err)
		
		err = page.Click("[data-testid='submit-btn']")
		require.NoError(t, err)
		
		// Should show network error
		_, err = page.WaitForSelector("[data-testid='network-error']", playwright.PageWaitForSelectorOptions{
			Timeout: playwright.Float(5000),
		})
		require.NoError(t, err)
		
		// Restore network
		err = context.SetOffline(false)
		require.NoError(t, err)
		
		// Retry button should work
		err = page.Click("[data-testid='retry-btn']")
		require.NoError(t, err)
		
		// Should succeed now
		_, err = page.WaitForSelector("[data-testid='success-notification']", playwright.PageWaitForSelectorOptions{
			Timeout: playwright.Float(5000),
		})
		require.NoError(t, err)
	})
}

// Test responsive design
func TestE2E_ResponsiveDesign(t *testing.T) {
	pw, err := playwright.Run()
	require.NoError(t, err)
	defer pw.Stop()
	
	// Test different viewport sizes
	viewports := []struct {
		name   string
		width  int
		height int
		mobile bool
	}{
		{"Desktop", 1920, 1080, false},
		{"Tablet", 768, 1024, false},
		{"Mobile", 375, 667, true},
	}
	
	for _, vp := range viewports {
		t.Run(vp.name, func(t *testing.T) {
			browser, err := pw.Chromium.Launch(playwright.BrowserTypeLaunchOptions{
				Headless: &headless,
			})
			require.NoError(t, err)
			defer browser.Close()
			
			context, err := browser.NewContext(playwright.BrowserNewContextOptions{
				Viewport: &playwright.Size{
					Width:  vp.width,
					Height: vp.height,
				},
				IsMobile: &vp.mobile,
			})
			require.NoError(t, err)
			defer context.Close()
			
			page, err := context.NewPage()
			require.NoError(t, err)
			
			_, err = page.Goto(baseURL)
			require.NoError(t, err)
			
			// Check mobile menu
			if vp.mobile {
				// Mobile menu should be visible
				_, err = page.WaitForSelector("[data-testid='mobile-menu-btn']", playwright.PageWaitForSelectorOptions{
					Timeout: playwright.Float(5000),
				})
				require.NoError(t, err)
				
				// Click to open menu
				err = page.Click("[data-testid='mobile-menu-btn']")
				require.NoError(t, err)
				
				// Navigation should be in drawer
				_, err = page.WaitForSelector("[data-testid='mobile-nav-drawer']", playwright.PageWaitForSelectorOptions{
					Timeout: playwright.Float(5000),
				})
				require.NoError(t, err)
			} else {
				// Desktop navigation should be visible
				_, err = page.WaitForSelector("[data-testid='desktop-nav']", playwright.PageWaitForSelectorOptions{
					Timeout: playwright.Float(5000),
				})
				require.NoError(t, err)
			}
			
			// Take screenshot
			screenshotPath := fmt.Sprintf("responsive-%s.png", vp.name)
			_, err = page.Screenshot(playwright.PageScreenshotOptions{
				Path:     &screenshotPath,
				FullPage: playwright.Bool(true),
			})
			assert.NoError(t, err)
		})
	}
}

// Helper function to clean up test resources
func cleanupTestResources(t *testing.T, prefix string) {
	client := &http.Client{Timeout: 10 * time.Second}
	
	// Delete switches
	resp, err := client.Get(apiURL + "/api/v1/switches")
	if err == nil {
		var switches []map[string]interface{}
		json.NewDecoder(resp.Body).Decode(&switches)
		resp.Body.Close()
		
		for _, s := range switches {
			name := s["name"].(string)
			if len(name) > len(prefix) && name[:len(prefix)] == prefix {
				req, _ := http.NewRequest("DELETE", apiURL+"/api/v1/switches/"+s["uuid"].(string), nil)
				client.Do(req)
			}
		}
	}
	
	// Delete routers
	resp, err = client.Get(apiURL + "/api/v1/routers")
	if err == nil {
		var routers []map[string]interface{}
		json.NewDecoder(resp.Body).Decode(&routers)
		resp.Body.Close()
		
		for _, r := range routers {
			name := r["name"].(string)
			if len(name) > len(prefix) && name[:len(prefix)] == prefix {
				req, _ := http.NewRequest("DELETE", apiURL+"/api/v1/routers/"+r["uuid"].(string), nil)
				client.Do(req)
			}
		}
	}
}