//go:build performance
// +build performance

package performance

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var (
	apiURL = getEnv("PERF_API_URL", "http://localhost:8080")
)

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

// Load test configuration
type LoadTestConfig struct {
	Name            string
	Duration        time.Duration
	Concurrency     int
	RequestsPerSec  int
	Endpoint        string
	Method          string
	Body            interface{}
}

// Load test metrics
type LoadTestMetrics struct {
	TotalRequests   int64
	SuccessRequests int64
	FailedRequests  int64
	TotalDuration   time.Duration
	MinLatency      time.Duration
	MaxLatency      time.Duration
	AvgLatency      time.Duration
	P50Latency      time.Duration
	P95Latency      time.Duration
	P99Latency      time.Duration
	RequestsPerSec  float64
}

// Test high load on list operations
func TestPerformance_HighLoadList(t *testing.T) {
	cfg := LoadTestConfig{
		Name:           "High Load List Switches",
		Duration:       30 * time.Second,
		Concurrency:    100,
		RequestsPerSec: 1000,
		Endpoint:       "/api/v1/switches",
		Method:         "GET",
	}
	
	metrics := runLoadTest(t, cfg)
	
	// Assertions
	assert.Greater(t, metrics.SuccessRequests, int64(0))
	assert.Less(t, metrics.FailedRequests, metrics.TotalRequests/10) // Less than 10% failure
	assert.Less(t, metrics.P95Latency, 100*time.Millisecond)         // P95 under 100ms
	assert.Greater(t, metrics.RequestsPerSec, float64(500))          // At least 500 RPS
	
	t.Logf("Load Test Results for %s:", cfg.Name)
	t.Logf("  Total Requests: %d", metrics.TotalRequests)
	t.Logf("  Success Rate: %.2f%%", float64(metrics.SuccessRequests)/float64(metrics.TotalRequests)*100)
	t.Logf("  Requests/sec: %.2f", metrics.RequestsPerSec)
	t.Logf("  P50 Latency: %v", metrics.P50Latency)
	t.Logf("  P95 Latency: %v", metrics.P95Latency)
	t.Logf("  P99 Latency: %v", metrics.P99Latency)
}

// Test sustained load on create operations
func TestPerformance_SustainedCreateLoad(t *testing.T) {
	cfg := LoadTestConfig{
		Name:           "Sustained Create Load",
		Duration:       60 * time.Second,
		Concurrency:    50,
		RequestsPerSec: 100,
		Endpoint:       "/api/v1/switches",
		Method:         "POST",
		Body: map[string]interface{}{
			"name":        "", // Will be generated per request
			"description": "Performance test switch",
		},
	}
	
	metrics := runLoadTest(t, cfg)
	
	// Assertions for write operations
	assert.Greater(t, metrics.SuccessRequests, int64(0))
	assert.Less(t, metrics.FailedRequests, metrics.TotalRequests/5) // Less than 20% failure for writes
	assert.Less(t, metrics.P95Latency, 200*time.Millisecond)        // P95 under 200ms for writes
	assert.Greater(t, metrics.RequestsPerSec, float64(50))          // At least 50 RPS for writes
}

// Test burst traffic
func TestPerformance_BurstTraffic(t *testing.T) {
	// Prepare some test data first
	createTestSwitches(t, 10)
	
	// Burst configuration
	burstSize := 1000
	burstDuration := 5 * time.Second
	
	var wg sync.WaitGroup
	var successCount int64
	var failCount int64
	latencies := make([]time.Duration, 0, burstSize)
	latenciesMu := &sync.Mutex{}
	
	start := time.Now()
	
	// Send burst of requests
	for i := 0; i < burstSize; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			
			reqStart := time.Now()
			req, _ := http.NewRequest("GET", apiURL+"/api/v1/topology", nil)
			
			client := &http.Client{Timeout: 10 * time.Second}
			resp, err := client.Do(req)
			latency := time.Since(reqStart)
			
			latenciesMu.Lock()
			latencies = append(latencies, latency)
			latenciesMu.Unlock()
			
			if err != nil || resp.StatusCode != http.StatusOK {
				atomic.AddInt64(&failCount, 1)
			} else {
				atomic.AddInt64(&successCount, 1)
			}
			
			if resp != nil {
				resp.Body.Close()
			}
		}()
		
		// Small delay to prevent overwhelming the system instantly
		if i%100 == 0 {
			time.Sleep(10 * time.Millisecond)
		}
	}
	
	wg.Wait()
	totalDuration := time.Since(start)
	
	// Calculate metrics
	successRate := float64(successCount) / float64(burstSize) * 100
	avgLatency := calculateAverage(latencies)
	p95Latency := calculatePercentile(latencies, 95)
	
	t.Logf("Burst Test Results:")
	t.Logf("  Burst Size: %d requests", burstSize)
	t.Logf("  Duration: %v", totalDuration)
	t.Logf("  Success Rate: %.2f%%", successRate)
	t.Logf("  Average Latency: %v", avgLatency)
	t.Logf("  P95 Latency: %v", p95Latency)
	
	// Assertions
	assert.Greater(t, successRate, float64(90)) // At least 90% success
	assert.Less(t, p95Latency, 500*time.Millisecond) // P95 under 500ms during burst
}

// Test concurrent operations (mixed read/write)
func TestPerformance_ConcurrentMixedOperations(t *testing.T) {
	duration := 30 * time.Second
	concurrency := 100
	
	var wg sync.WaitGroup
	stopCh := make(chan struct{})
	
	// Metrics
	var totalOps int64
	var successOps int64
	var readOps int64
	var writeOps int64
	
	// Start workers
	for i := 0; i < concurrency; i++ {
		wg.Add(1)
		go func(workerID int) {
			defer wg.Done()
			
			client := &http.Client{Timeout: 5 * time.Second}
			
			for {
				select {
				case <-stopCh:
					return
				default:
					// Randomly choose operation (70% read, 30% write)
					if workerID%10 < 7 {
						// Read operation
						atomic.AddInt64(&readOps, 1)
						req, _ := http.NewRequest("GET", apiURL+"/api/v1/switches", nil)
						resp, err := client.Do(req)
						if err == nil && resp.StatusCode == http.StatusOK {
							atomic.AddInt64(&successOps, 1)
						}
						if resp != nil {
							resp.Body.Close()
						}
					} else {
						// Write operation
						atomic.AddInt64(&writeOps, 1)
						body := map[string]interface{}{
							"name":        fmt.Sprintf("perf-switch-%s", uuid.New().String()[:8]),
							"description": "Mixed operation test",
						}
						data, _ := json.Marshal(body)
						req, _ := http.NewRequest("POST", apiURL+"/api/v1/switches", bytes.NewBuffer(data))
						req.Header.Set("Content-Type", "application/json")
						
						resp, err := client.Do(req)
						if err == nil && resp.StatusCode == http.StatusCreated {
							atomic.AddInt64(&successOps, 1)
						}
						if resp != nil {
							resp.Body.Close()
						}
					}
					atomic.AddInt64(&totalOps, 1)
					
					// Small delay between operations
					time.Sleep(10 * time.Millisecond)
				}
			}
		}(i)
	}
	
	// Run for specified duration
	time.Sleep(duration)
	close(stopCh)
	wg.Wait()
	
	// Calculate metrics
	successRate := float64(successOps) / float64(totalOps) * 100
	opsPerSec := float64(totalOps) / duration.Seconds()
	
	t.Logf("Concurrent Mixed Operations Results:")
	t.Logf("  Duration: %v", duration)
	t.Logf("  Concurrency: %d", concurrency)
	t.Logf("  Total Operations: %d", totalOps)
	t.Logf("  Read Operations: %d", readOps)
	t.Logf("  Write Operations: %d", writeOps)
	t.Logf("  Success Rate: %.2f%%", successRate)
	t.Logf("  Operations/sec: %.2f", opsPerSec)
	
	// Assertions
	assert.Greater(t, successRate, float64(95)) // At least 95% success
	assert.Greater(t, opsPerSec, float64(100))  // At least 100 ops/sec
}

// Helper function to run load test
func runLoadTest(t *testing.T, cfg LoadTestConfig) *LoadTestMetrics {
	var wg sync.WaitGroup
	stopCh := make(chan struct{})
	
	// Metrics collection
	var totalRequests int64
	var successRequests int64
	var failedRequests int64
	latencies := make([]time.Duration, 0, int(cfg.Duration.Seconds())*cfg.RequestsPerSec)
	latenciesMu := &sync.Mutex{}
	
	// Rate limiter
	ticker := time.NewTicker(time.Second / time.Duration(cfg.RequestsPerSec))
	defer ticker.Stop()
	
	start := time.Now()
	
	// Start workers
	for i := 0; i < cfg.Concurrency; i++ {
		wg.Add(1)
		go func(workerID int) {
			defer wg.Done()
			
			client := &http.Client{Timeout: 5 * time.Second}
			
			for {
				select {
				case <-stopCh:
					return
				case <-ticker.C:
					atomic.AddInt64(&totalRequests, 1)
					
					// Prepare request
					var req *http.Request
					if cfg.Method == "POST" && cfg.Body != nil {
						body := cfg.Body.(map[string]interface{})
						body["name"] = fmt.Sprintf("load-test-%s", uuid.New().String()[:8])
						data, _ := json.Marshal(body)
						req, _ = http.NewRequest(cfg.Method, apiURL+cfg.Endpoint, bytes.NewBuffer(data))
						req.Header.Set("Content-Type", "application/json")
					} else {
						req, _ = http.NewRequest(cfg.Method, apiURL+cfg.Endpoint, nil)
					}
					
					// Execute request
					reqStart := time.Now()
					resp, err := client.Do(req)
					latency := time.Since(reqStart)
					
					// Record metrics
					latenciesMu.Lock()
					latencies = append(latencies, latency)
					latenciesMu.Unlock()
					
					if err != nil || (cfg.Method == "GET" && resp.StatusCode != http.StatusOK) ||
						(cfg.Method == "POST" && resp.StatusCode != http.StatusCreated) {
						atomic.AddInt64(&failedRequests, 1)
					} else {
						atomic.AddInt64(&successRequests, 1)
					}
					
					if resp != nil {
						resp.Body.Close()
					}
				}
			}
		}(i)
	}
	
	// Run for specified duration
	time.Sleep(cfg.Duration)
	close(stopCh)
	wg.Wait()
	
	totalDuration := time.Since(start)
	
	// Calculate metrics
	metrics := &LoadTestMetrics{
		TotalRequests:   totalRequests,
		SuccessRequests: successRequests,
		FailedRequests:  failedRequests,
		TotalDuration:   totalDuration,
		RequestsPerSec:  float64(totalRequests) / totalDuration.Seconds(),
	}
	
	if len(latencies) > 0 {
		metrics.MinLatency = findMin(latencies)
		metrics.MaxLatency = findMax(latencies)
		metrics.AvgLatency = calculateAverage(latencies)
		metrics.P50Latency = calculatePercentile(latencies, 50)
		metrics.P95Latency = calculatePercentile(latencies, 95)
		metrics.P99Latency = calculatePercentile(latencies, 99)
	}
	
	return metrics
}

// Helper functions for metrics calculation
func calculateAverage(latencies []time.Duration) time.Duration {
	if len(latencies) == 0 {
		return 0
	}
	
	var sum time.Duration
	for _, l := range latencies {
		sum += l
	}
	return sum / time.Duration(len(latencies))
}

func calculatePercentile(latencies []time.Duration, percentile float64) time.Duration {
	if len(latencies) == 0 {
		return 0
	}
	
	// Sort latencies
	sorted := make([]time.Duration, len(latencies))
	copy(sorted, latencies)
	for i := 0; i < len(sorted); i++ {
		for j := i + 1; j < len(sorted); j++ {
			if sorted[i] > sorted[j] {
				sorted[i], sorted[j] = sorted[j], sorted[i]
			}
		}
	}
	
	index := int(float64(len(sorted)-1) * percentile / 100)
	return sorted[index]
}

func findMin(latencies []time.Duration) time.Duration {
	if len(latencies) == 0 {
		return 0
	}
	
	min := latencies[0]
	for _, l := range latencies[1:] {
		if l < min {
			min = l
		}
	}
	return min
}

func findMax(latencies []time.Duration) time.Duration {
	if len(latencies) == 0 {
		return 0
	}
	
	max := latencies[0]
	for _, l := range latencies[1:] {
		if l > max {
			max = l
		}
	}
	return max
}

// Helper to create test data
func createTestSwitches(t *testing.T, count int) {
	client := &http.Client{Timeout: 5 * time.Second}
	
	for i := 0; i < count; i++ {
		body := map[string]interface{}{
			"name":        fmt.Sprintf("test-switch-%d", i),
			"description": "Test switch for performance testing",
		}
		data, _ := json.Marshal(body)
		
		req, _ := http.NewRequest("POST", apiURL+"/api/v1/switches", bytes.NewBuffer(data))
		req.Header.Set("Content-Type", "application/json")
		
		resp, err := client.Do(req)
		require.NoError(t, err)
		require.Equal(t, http.StatusCreated, resp.StatusCode)
		resp.Body.Close()
	}
}