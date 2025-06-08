package api

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/lspecian/ovncp/internal/auth"
	"github.com/lspecian/ovncp/internal/service"
)

// Mock service
type mockService struct {
	mock.Mock
}

func (m *mockService) ListLogicalSwitches(userID string) ([]service.LogicalSwitch, error) {
	args := m.Called(userID)
	return args.Get(0).([]service.LogicalSwitch), args.Error(1)
}

func (m *mockService) GetLogicalSwitch(userID, id string) (*service.LogicalSwitch, error) {
	args := m.Called(userID, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*service.LogicalSwitch), args.Error(1)
}

func (m *mockService) CreateLogicalSwitch(userID string, req service.CreateLogicalSwitchRequest) (*service.LogicalSwitch, error) {
	args := m.Called(userID, req)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*service.LogicalSwitch), args.Error(1)
}

func (m *mockService) UpdateLogicalSwitch(userID, id string, req service.UpdateLogicalSwitchRequest) (*service.LogicalSwitch, error) {
	args := m.Called(userID, id, req)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*service.LogicalSwitch), args.Error(1)
}

func (m *mockService) DeleteLogicalSwitch(userID, id string) error {
	args := m.Called(userID, id)
	return args.Error(0)
}

func (m *mockService) ListLogicalRouters(userID string) ([]service.LogicalRouter, error) {
	args := m.Called(userID)
	return args.Get(0).([]service.LogicalRouter), args.Error(1)
}

func (m *mockService) GetLogicalRouter(userID, id string) (*service.LogicalRouter, error) {
	args := m.Called(userID, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*service.LogicalRouter), args.Error(1)
}

func (m *mockService) CreateLogicalRouter(userID string, req service.CreateLogicalRouterRequest) (*service.LogicalRouter, error) {
	args := m.Called(userID, req)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*service.LogicalRouter), args.Error(1)
}

func (m *mockService) UpdateLogicalRouter(userID, id string, req service.UpdateLogicalRouterRequest) (*service.LogicalRouter, error) {
	args := m.Called(userID, id, req)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*service.LogicalRouter), args.Error(1)
}

func (m *mockService) DeleteLogicalRouter(userID, id string) error {
	args := m.Called(userID, id)
	return args.Error(0)
}

func (m *mockService) GetNetworkTopology(userID string) (*service.NetworkTopology, error) {
	args := m.Called(userID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*service.NetworkTopology), args.Error(1)
}

// Test helpers
func setupTestRouter(mockSvc *mockService) *gin.Engine {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	
	// Add auth middleware that sets test user
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
	api.GET("/switches/:id", h.GetLogicalSwitch)
	api.POST("/switches", h.CreateLogicalSwitch)
	api.PUT("/switches/:id", h.UpdateLogicalSwitch)
	api.DELETE("/switches/:id", h.DeleteLogicalSwitch)
	
	api.GET("/routers", h.ListLogicalRouters)
	api.GET("/routers/:id", h.GetLogicalRouter)
	api.POST("/routers", h.CreateLogicalRouter)
	api.PUT("/routers/:id", h.UpdateLogicalRouter)
	api.DELETE("/routers/:id", h.DeleteLogicalRouter)
	
	api.GET("/topology", h.GetNetworkTopology)
	
	return r
}

// Logical Switch Tests
func TestListLogicalSwitches(t *testing.T) {
	mockSvc := new(mockService)
	router := setupTestRouter(mockSvc)
	
	switches := []service.LogicalSwitch{
		{
			UUID:        uuid.New().String(),
			Name:        "switch1",
			Description: "Test switch 1",
		},
		{
			UUID:        uuid.New().String(),
			Name:        "switch2",
			Description: "Test switch 2",
		},
	}
	
	mockSvc.On("ListLogicalSwitches", "test-user-id").Return(switches, nil)
	
	req, _ := http.NewRequest("GET", "/api/v1/switches", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)
	
	assert.Equal(t, http.StatusOK, w.Code)
	
	var response []service.LogicalSwitch
	err := json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(t, err)
	assert.Equal(t, len(switches), len(response))
	assert.Equal(t, switches[0].Name, response[0].Name)
	
	mockSvc.AssertExpectations(t)
}

func TestGetLogicalSwitch(t *testing.T) {
	mockSvc := new(mockService)
	router := setupTestRouter(mockSvc)
	
	switchID := uuid.New().String()
	ls := &service.LogicalSwitch{
		UUID:        switchID,
		Name:        "switch1",
		Description: "Test switch",
	}
	
	mockSvc.On("GetLogicalSwitch", "test-user-id", switchID).Return(ls, nil)
	
	req, _ := http.NewRequest("GET", "/api/v1/switches/"+switchID, nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)
	
	assert.Equal(t, http.StatusOK, w.Code)
	
	var response service.LogicalSwitch
	err := json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(t, err)
	assert.Equal(t, ls.UUID, response.UUID)
	assert.Equal(t, ls.Name, response.Name)
	
	mockSvc.AssertExpectations(t)
}

func TestGetLogicalSwitch_NotFound(t *testing.T) {
	mockSvc := new(mockService)
	router := setupTestRouter(mockSvc)
	
	switchID := uuid.New().String()
	mockSvc.On("GetLogicalSwitch", "test-user-id", switchID).Return(nil, service.ErrNotFound)
	
	req, _ := http.NewRequest("GET", "/api/v1/switches/"+switchID, nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)
	
	assert.Equal(t, http.StatusNotFound, w.Code)
	mockSvc.AssertExpectations(t)
}

func TestCreateLogicalSwitch(t *testing.T) {
	mockSvc := new(mockService)
	router := setupTestRouter(mockSvc)
	
	createReq := service.CreateLogicalSwitchRequest{
		Name:        "new-switch",
		Description: "New test switch",
	}
	
	createdSwitch := &service.LogicalSwitch{
		UUID:        uuid.New().String(),
		Name:        createReq.Name,
		Description: createReq.Description,
	}
	
	mockSvc.On("CreateLogicalSwitch", "test-user-id", createReq).Return(createdSwitch, nil)
	
	body, _ := json.Marshal(createReq)
	req, _ := http.NewRequest("POST", "/api/v1/switches", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)
	
	assert.Equal(t, http.StatusCreated, w.Code)
	
	var response service.LogicalSwitch
	err := json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(t, err)
	assert.Equal(t, createdSwitch.Name, response.Name)
	
	mockSvc.AssertExpectations(t)
}

func TestCreateLogicalSwitch_InvalidRequest(t *testing.T) {
	mockSvc := new(mockService)
	router := setupTestRouter(mockSvc)
	
	// Missing required name field
	createReq := map[string]string{
		"description": "Missing name",
	}
	
	body, _ := json.Marshal(createReq)
	req, _ := http.NewRequest("POST", "/api/v1/switches", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)
	
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestUpdateLogicalSwitch(t *testing.T) {
	mockSvc := new(mockService)
	router := setupTestRouter(mockSvc)
	
	switchID := uuid.New().String()
	updateReq := service.UpdateLogicalSwitchRequest{
		Name:        stringPtr("updated-switch"),
		Description: stringPtr("Updated description"),
	}
	
	updatedSwitch := &service.LogicalSwitch{
		UUID:        switchID,
		Name:        *updateReq.Name,
		Description: *updateReq.Description,
	}
	
	mockSvc.On("UpdateLogicalSwitch", "test-user-id", switchID, updateReq).Return(updatedSwitch, nil)
	
	body, _ := json.Marshal(updateReq)
	req, _ := http.NewRequest("PUT", "/api/v1/switches/"+switchID, bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)
	
	assert.Equal(t, http.StatusOK, w.Code)
	
	var response service.LogicalSwitch
	err := json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(t, err)
	assert.Equal(t, updatedSwitch.Name, response.Name)
	
	mockSvc.AssertExpectations(t)
}

func TestDeleteLogicalSwitch(t *testing.T) {
	mockSvc := new(mockService)
	router := setupTestRouter(mockSvc)
	
	switchID := uuid.New().String()
	mockSvc.On("DeleteLogicalSwitch", "test-user-id", switchID).Return(nil)
	
	req, _ := http.NewRequest("DELETE", "/api/v1/switches/"+switchID, nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)
	
	assert.Equal(t, http.StatusNoContent, w.Code)
	mockSvc.AssertExpectations(t)
}

// Logical Router Tests
func TestListLogicalRouters(t *testing.T) {
	mockSvc := new(mockService)
	router := setupTestRouter(mockSvc)
	
	routers := []service.LogicalRouter{
		{
			UUID:    uuid.New().String(),
			Name:    "router1",
			Enabled: true,
		},
		{
			UUID:    uuid.New().String(),
			Name:    "router2",
			Enabled: false,
		},
	}
	
	mockSvc.On("ListLogicalRouters", "test-user-id").Return(routers, nil)
	
	req, _ := http.NewRequest("GET", "/api/v1/routers", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)
	
	assert.Equal(t, http.StatusOK, w.Code)
	
	var response []service.LogicalRouter
	err := json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(t, err)
	assert.Equal(t, len(routers), len(response))
	
	mockSvc.AssertExpectations(t)
}

func TestCreateLogicalRouter(t *testing.T) {
	mockSvc := new(mockService)
	router := setupTestRouter(mockSvc)
	
	createReq := service.CreateLogicalRouterRequest{
		Name:    "new-router",
		Enabled: true,
	}
	
	createdRouter := &service.LogicalRouter{
		UUID:    uuid.New().String(),
		Name:    createReq.Name,
		Enabled: createReq.Enabled,
	}
	
	mockSvc.On("CreateLogicalRouter", "test-user-id", createReq).Return(createdRouter, nil)
	
	body, _ := json.Marshal(createReq)
	req, _ := http.NewRequest("POST", "/api/v1/routers", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)
	
	assert.Equal(t, http.StatusCreated, w.Code)
	
	var response service.LogicalRouter
	err := json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(t, err)
	assert.Equal(t, createdRouter.Name, response.Name)
	
	mockSvc.AssertExpectations(t)
}

// Network Topology Tests
func TestGetNetworkTopology(t *testing.T) {
	mockSvc := new(mockService)
	router := setupTestRouter(mockSvc)
	
	topology := &service.NetworkTopology{
		Switches: []service.LogicalSwitch{
			{
				UUID: uuid.New().String(),
				Name: "switch1",
			},
		},
		Routers: []service.LogicalRouter{
			{
				UUID: uuid.New().String(),
				Name: "router1",
			},
		},
		Ports: []service.LogicalPort{
			{
				UUID: uuid.New().String(),
				Name: "port1",
			},
		},
	}
	
	mockSvc.On("GetNetworkTopology", "test-user-id").Return(topology, nil)
	
	req, _ := http.NewRequest("GET", "/api/v1/topology", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)
	
	assert.Equal(t, http.StatusOK, w.Code)
	
	var response service.NetworkTopology
	err := json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(t, err)
	assert.Equal(t, len(topology.Switches), len(response.Switches))
	assert.Equal(t, len(topology.Routers), len(response.Routers))
	
	mockSvc.AssertExpectations(t)
}

// Helper functions
func stringPtr(s string) *string {
	return &s
}

func boolPtr(b bool) *bool {
	return &b
}