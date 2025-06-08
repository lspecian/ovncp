package service

import (
	"errors"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

// Mock OVN client
type mockOVNClient struct {
	mock.Mock
}

func (m *mockOVNClient) ListLogicalSwitches() ([]LogicalSwitch, error) {
	args := m.Called()
	return args.Get(0).([]LogicalSwitch), args.Error(1)
}

func (m *mockOVNClient) GetLogicalSwitch(id string) (*LogicalSwitch, error) {
	args := m.Called(id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*LogicalSwitch), args.Error(1)
}

func (m *mockOVNClient) CreateLogicalSwitch(name, description string) (*LogicalSwitch, error) {
	args := m.Called(name, description)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*LogicalSwitch), args.Error(1)
}

func (m *mockOVNClient) UpdateLogicalSwitch(id string, updates map[string]interface{}) (*LogicalSwitch, error) {
	args := m.Called(id, updates)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*LogicalSwitch), args.Error(1)
}

func (m *mockOVNClient) DeleteLogicalSwitch(id string) error {
	args := m.Called(id)
	return args.Error(0)
}

func (m *mockOVNClient) ListLogicalRouters() ([]LogicalRouter, error) {
	args := m.Called()
	return args.Get(0).([]LogicalRouter), args.Error(1)
}

func (m *mockOVNClient) GetLogicalRouter(id string) (*LogicalRouter, error) {
	args := m.Called(id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*LogicalRouter), args.Error(1)
}

func (m *mockOVNClient) CreateLogicalRouter(name string, enabled bool) (*LogicalRouter, error) {
	args := m.Called(name, enabled)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*LogicalRouter), args.Error(1)
}

func (m *mockOVNClient) UpdateLogicalRouter(id string, updates map[string]interface{}) (*LogicalRouter, error) {
	args := m.Called(id, updates)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*LogicalRouter), args.Error(1)
}

func (m *mockOVNClient) DeleteLogicalRouter(id string) error {
	args := m.Called(id)
	return args.Error(0)
}

func (m *mockOVNClient) ListLogicalPorts() ([]LogicalPort, error) {
	args := m.Called()
	return args.Get(0).([]LogicalPort), args.Error(1)
}

func (m *mockOVNClient) GetLogicalPort(id string) (*LogicalPort, error) {
	args := m.Called(id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*LogicalPort), args.Error(1)
}

func (m *mockOVNClient) CreateLogicalPort(req CreateLogicalPortRequest) (*LogicalPort, error) {
	args := m.Called(req)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*LogicalPort), args.Error(1)
}

func (m *mockOVNClient) CreateTransaction() Transaction {
	args := m.Called()
	return args.Get(0).(Transaction)
}

// Test Service
func TestService_ListLogicalSwitches(t *testing.T) {
	mockClient := new(mockOVNClient)
	svc := &Service{ovnClient: mockClient}
	
	switches := []LogicalSwitch{
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
	
	mockClient.On("ListLogicalSwitches").Return(switches, nil)
	
	result, err := svc.ListLogicalSwitches("user1")
	assert.NoError(t, err)
	assert.Equal(t, len(switches), len(result))
	
	mockClient.AssertExpectations(t)
}

func TestService_GetLogicalSwitch(t *testing.T) {
	mockClient := new(mockOVNClient)
	svc := &Service{ovnClient: mockClient}
	
	switchID := uuid.New().String()
	ls := &LogicalSwitch{
		UUID:        switchID,
		Name:        "switch1",
		Description: "Test switch",
	}
	
	mockClient.On("GetLogicalSwitch", switchID).Return(ls, nil)
	
	result, err := svc.GetLogicalSwitch("user1", switchID)
	assert.NoError(t, err)
	assert.Equal(t, ls.UUID, result.UUID)
	assert.Equal(t, ls.Name, result.Name)
	
	mockClient.AssertExpectations(t)
}

func TestService_GetLogicalSwitch_NotFound(t *testing.T) {
	mockClient := new(mockOVNClient)
	svc := &Service{ovnClient: mockClient}
	
	switchID := uuid.New().String()
	mockClient.On("GetLogicalSwitch", switchID).Return(nil, errors.New("not found"))
	
	result, err := svc.GetLogicalSwitch("user1", switchID)
	assert.Error(t, err)
	assert.Nil(t, result)
	
	mockClient.AssertExpectations(t)
}

func TestService_CreateLogicalSwitch(t *testing.T) {
	mockClient := new(mockOVNClient)
	svc := &Service{ovnClient: mockClient}
	
	req := CreateLogicalSwitchRequest{
		Name:        "new-switch",
		Description: "New test switch",
	}
	
	created := &LogicalSwitch{
		UUID:        uuid.New().String(),
		Name:        req.Name,
		Description: req.Description,
	}
	
	mockClient.On("CreateLogicalSwitch", req.Name, req.Description).Return(created, nil)
	
	result, err := svc.CreateLogicalSwitch("user1", req)
	assert.NoError(t, err)
	assert.Equal(t, created.Name, result.Name)
	
	mockClient.AssertExpectations(t)
}

func TestService_CreateLogicalSwitch_DuplicateName(t *testing.T) {
	mockClient := new(mockOVNClient)
	svc := &Service{ovnClient: mockClient}
	
	req := CreateLogicalSwitchRequest{
		Name:        "duplicate-switch",
		Description: "Duplicate test switch",
	}
	
	mockClient.On("CreateLogicalSwitch", req.Name, req.Description).
		Return(nil, errors.New("duplicate name"))
	
	result, err := svc.CreateLogicalSwitch("user1", req)
	assert.Error(t, err)
	assert.Nil(t, result)
	
	mockClient.AssertExpectations(t)
}

func TestService_UpdateLogicalSwitch(t *testing.T) {
	mockClient := new(mockOVNClient)
	svc := &Service{ovnClient: mockClient}
	
	switchID := uuid.New().String()
	req := UpdateLogicalSwitchRequest{
		Name:        stringPtr("updated-switch"),
		Description: stringPtr("Updated description"),
	}
	
	updates := map[string]interface{}{}
	if req.Name != nil {
		updates["name"] = *req.Name
	}
	if req.Description != nil {
		updates["description"] = *req.Description
	}
	
	updated := &LogicalSwitch{
		UUID:        switchID,
		Name:        *req.Name,
		Description: *req.Description,
	}
	
	mockClient.On("UpdateLogicalSwitch", switchID, updates).Return(updated, nil)
	
	result, err := svc.UpdateLogicalSwitch("user1", switchID, req)
	assert.NoError(t, err)
	assert.Equal(t, updated.Name, result.Name)
	
	mockClient.AssertExpectations(t)
}

func TestService_DeleteLogicalSwitch(t *testing.T) {
	mockClient := new(mockOVNClient)
	svc := &Service{ovnClient: mockClient}
	
	switchID := uuid.New().String()
	mockClient.On("DeleteLogicalSwitch", switchID).Return(nil)
	
	err := svc.DeleteLogicalSwitch("user1", switchID)
	assert.NoError(t, err)
	
	mockClient.AssertExpectations(t)
}

// Router Tests
func TestService_ListLogicalRouters(t *testing.T) {
	mockClient := new(mockOVNClient)
	svc := &Service{ovnClient: mockClient}
	
	routers := []LogicalRouter{
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
	
	mockClient.On("ListLogicalRouters").Return(routers, nil)
	
	result, err := svc.ListLogicalRouters("user1")
	assert.NoError(t, err)
	assert.Equal(t, len(routers), len(result))
	
	mockClient.AssertExpectations(t)
}

func TestService_CreateLogicalRouter(t *testing.T) {
	mockClient := new(mockOVNClient)
	svc := &Service{ovnClient: mockClient}
	
	req := CreateLogicalRouterRequest{
		Name:    "new-router",
		Enabled: true,
	}
	
	created := &LogicalRouter{
		UUID:    uuid.New().String(),
		Name:    req.Name,
		Enabled: req.Enabled,
	}
	
	mockClient.On("CreateLogicalRouter", req.Name, req.Enabled).Return(created, nil)
	
	result, err := svc.CreateLogicalRouter("user1", req)
	assert.NoError(t, err)
	assert.Equal(t, created.Name, result.Name)
	
	mockClient.AssertExpectations(t)
}

// Network Topology Tests
func TestService_GetNetworkTopology(t *testing.T) {
	mockClient := new(mockOVNClient)
	svc := &Service{ovnClient: mockClient}
	
	switches := []LogicalSwitch{
		{
			UUID: uuid.New().String(),
			Name: "switch1",
		},
	}
	
	routers := []LogicalRouter{
		{
			UUID: uuid.New().String(),
			Name: "router1",
		},
	}
	
	ports := []LogicalPort{
		{
			UUID: uuid.New().String(),
			Name: "port1",
		},
	}
	
	mockClient.On("ListLogicalSwitches").Return(switches, nil)
	mockClient.On("ListLogicalRouters").Return(routers, nil)
	mockClient.On("ListLogicalPorts").Return(ports, nil)
	
	result, err := svc.GetNetworkTopology("user1")
	assert.NoError(t, err)
	assert.Equal(t, len(switches), len(result.Switches))
	assert.Equal(t, len(routers), len(result.Routers))
	assert.Equal(t, len(ports), len(result.Ports))
	
	mockClient.AssertExpectations(t)
}

func TestService_GetNetworkTopology_PartialFailure(t *testing.T) {
	mockClient := new(mockOVNClient)
	svc := &Service{ovnClient: mockClient}
	
	switches := []LogicalSwitch{
		{
			UUID: uuid.New().String(),
			Name: "switch1",
		},
	}
	
	mockClient.On("ListLogicalSwitches").Return(switches, nil)
	mockClient.On("ListLogicalRouters").Return([]LogicalRouter{}, errors.New("router error"))
	mockClient.On("ListLogicalPorts").Return([]LogicalPort{}, nil)
	
	result, err := svc.GetNetworkTopology("user1")
	assert.NoError(t, err) // Should still return partial data
	assert.Equal(t, len(switches), len(result.Switches))
	assert.Equal(t, 0, len(result.Routers))
	assert.Equal(t, 0, len(result.Ports))
	
	mockClient.AssertExpectations(t)
}

// Helper functions
func stringPtr(s string) *string {
	return &s
}

func boolPtr(b bool) *bool {
	return &b
}