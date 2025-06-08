package api

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/stretchr/testify/mock"
	"github.com/lspecian/ovncp/internal/auth"
	"github.com/lspecian/ovncp/internal/service"
)

// Benchmark API handlers
func BenchmarkHandler_ListLogicalSwitches(b *testing.B) {
	mockSvc := new(mockService)
	router := setupBenchmarkRouter(mockSvc)
	
	// Prepare test data
	switches := make([]service.LogicalSwitch, 100)
	for i := range switches {
		switches[i] = service.LogicalSwitch{
			UUID:        uuid.New().String(),
			Name:        "switch-" + string(rune(i)),
			Description: "Benchmark switch",
		}
	}
	
	mockSvc.On("ListLogicalSwitches", "test-user-id").Return(switches, nil)
	
	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			req, _ := http.NewRequest("GET", "/api/v1/switches", nil)
			w := httptest.NewRecorder()
			router.ServeHTTP(w, req)
			
			if w.Code != http.StatusOK {
				b.Fatalf("unexpected status code: %d", w.Code)
			}
		}
	})
}

func BenchmarkHandler_CreateLogicalSwitch(b *testing.B) {
	mockSvc := new(mockService)
	router := setupBenchmarkRouter(mockSvc)
	
	mockSvc.On("CreateLogicalSwitch", "test-user-id", mock.Anything).
		Return(&service.LogicalSwitch{
			UUID:        uuid.New().String(),
			Name:        "new-switch",
			Description: "Created switch",
		}, nil)
	
	reqBody := service.CreateLogicalSwitchRequest{
		Name:        "bench-switch",
		Description: "Benchmark switch",
	}
	body, _ := json.Marshal(reqBody)
	
	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			req, _ := http.NewRequest("POST", "/api/v1/switches", bytes.NewBuffer(body))
			req.Header.Set("Content-Type", "application/json")
			w := httptest.NewRecorder()
			router.ServeHTTP(w, req)
			
			if w.Code != http.StatusCreated {
				b.Fatalf("unexpected status code: %d", w.Code)
			}
		}
	})
}

func BenchmarkHandler_GetNetworkTopology(b *testing.B) {
	mockSvc := new(mockService)
	router := setupBenchmarkRouter(mockSvc)
	
	// Prepare topology data
	topology := &service.NetworkTopology{
		Switches: make([]service.LogicalSwitch, 50),
		Routers:  make([]service.LogicalRouter, 10),
		Ports:    make([]service.LogicalPort, 100),
	}
	
	for i := range topology.Switches {
		topology.Switches[i] = service.LogicalSwitch{
			UUID: uuid.New().String(),
			Name: "switch-" + string(rune(i)),
		}
	}
	
	for i := range topology.Routers {
		topology.Routers[i] = service.LogicalRouter{
			UUID: uuid.New().String(),
			Name: "router-" + string(rune(i)),
		}
	}
	
	for i := range topology.Ports {
		topology.Ports[i] = service.LogicalPort{
			UUID: uuid.New().String(),
			Name: "port-" + string(rune(i)),
		}
	}
	
	mockSvc.On("GetNetworkTopology", "test-user-id").Return(topology, nil)
	
	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			req, _ := http.NewRequest("GET", "/api/v1/topology", nil)
			w := httptest.NewRecorder()
			router.ServeHTTP(w, req)
			
			if w.Code != http.StatusOK {
				b.Fatalf("unexpected status code: %d", w.Code)
			}
		}
	})
}

// Benchmark JSON serialization/deserialization
func BenchmarkJSON_Serialization(b *testing.B) {
	topology := &service.NetworkTopology{
		Switches: make([]service.LogicalSwitch, 50),
		Routers:  make([]service.LogicalRouter, 10),
		Ports:    make([]service.LogicalPort, 100),
	}
	
	for i := range topology.Switches {
		topology.Switches[i] = service.LogicalSwitch{
			UUID:        uuid.New().String(),
			Name:        "switch-" + string(rune(i)),
			Description: "Test switch with a longer description for more realistic payload size",
		}
	}
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		data, err := json.Marshal(topology)
		if err != nil {
			b.Fatal(err)
		}
		_ = data
	}
}

func BenchmarkJSON_Deserialization(b *testing.B) {
	topology := &service.NetworkTopology{
		Switches: make([]service.LogicalSwitch, 50),
		Routers:  make([]service.LogicalRouter, 10),
		Ports:    make([]service.LogicalPort, 100),
	}
	
	for i := range topology.Switches {
		topology.Switches[i] = service.LogicalSwitch{
			UUID:        uuid.New().String(),
			Name:        "switch-" + string(rune(i)),
			Description: "Test switch with a longer description for more realistic payload size",
		}
	}
	
	data, _ := json.Marshal(topology)
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		var result service.NetworkTopology
		err := json.Unmarshal(data, &result)
		if err != nil {
			b.Fatal(err)
		}
	}
}

// Benchmark middleware
func BenchmarkMiddleware_Auth(b *testing.B) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	
	// Add auth middleware
	router.Use(func(c *gin.Context) {
		// Simulate auth check
		c.Set("user", &auth.User{
			ID:    "test-user-id",
			Email: "test@example.com",
			Roles: []string{"admin"},
		})
		c.Next()
	})
	
	router.GET("/test", func(c *gin.Context) {
		c.JSON(200, gin.H{"status": "ok"})
	})
	
	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			req, _ := http.NewRequest("GET", "/test", nil)
			w := httptest.NewRecorder()
			router.ServeHTTP(w, req)
			
			if w.Code != http.StatusOK {
				b.Fatalf("unexpected status code: %d", w.Code)
			}
		}
	})
}

// Helper function for benchmarks
func setupBenchmarkRouter(mockSvc *mockService) *gin.Engine {
	gin.SetMode(gin.ReleaseMode)
	r := gin.New()
	
	// Add minimal middleware for benchmarks
	r.Use(func(c *gin.Context) {
		c.Set("user", &auth.User{
			ID:    "test-user-id",
			Email: "test@example.com",
			Roles: []string{"admin"},
		})
		c.Next()
	})
	
	h := &Handler{service: mockSvc}
	api := r.Group("/api/v1")
	
	// Register routes
	api.GET("/switches", h.ListLogicalSwitches)
	api.POST("/switches", h.CreateLogicalSwitch)
	api.GET("/topology", h.GetNetworkTopology)
	
	return r
}