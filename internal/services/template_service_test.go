package services

import (
	"context"
	"testing"

	"github.com/lspecian/ovncp/internal/models"
	"github.com/lspecian/ovncp/internal/templates"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"go.uber.org/zap"
)

// MockOVNService for testing
type MockOVNService struct {
	mock.Mock
}

func (m *MockOVNService) ListLogicalSwitches(ctx context.Context) ([]*models.LogicalSwitch, error) {
	args := m.Called(ctx)
	return args.Get(0).([]*models.LogicalSwitch), args.Error(1)
}

func (m *MockOVNService) GetLogicalSwitch(ctx context.Context, id string) (*models.LogicalSwitch, error) {
	args := m.Called(ctx, id)
	return args.Get(0).(*models.LogicalSwitch), args.Error(1)
}

func (m *MockOVNService) CreateLogicalSwitch(ctx context.Context, ls *models.LogicalSwitch) (*models.LogicalSwitch, error) {
	args := m.Called(ctx, ls)
	return args.Get(0).(*models.LogicalSwitch), args.Error(1)
}

func (m *MockOVNService) UpdateLogicalSwitch(ctx context.Context, id string, ls *models.LogicalSwitch) (*models.LogicalSwitch, error) {
	args := m.Called(ctx, id, ls)
	return args.Get(0).(*models.LogicalSwitch), args.Error(1)
}

func (m *MockOVNService) DeleteLogicalSwitch(ctx context.Context, id string) error {
	args := m.Called(ctx, id)
	return args.Error(0)
}

func (m *MockOVNService) ListLogicalRouters(ctx context.Context) ([]*models.LogicalRouter, error) {
	args := m.Called(ctx)
	return args.Get(0).([]*models.LogicalRouter), args.Error(1)
}

func (m *MockOVNService) GetLogicalRouter(ctx context.Context, id string) (*models.LogicalRouter, error) {
	args := m.Called(ctx, id)
	return args.Get(0).(*models.LogicalRouter), args.Error(1)
}

func (m *MockOVNService) CreateLogicalRouter(ctx context.Context, lr *models.LogicalRouter) (*models.LogicalRouter, error) {
	args := m.Called(ctx, lr)
	return args.Get(0).(*models.LogicalRouter), args.Error(1)
}

func (m *MockOVNService) UpdateLogicalRouter(ctx context.Context, id string, lr *models.LogicalRouter) (*models.LogicalRouter, error) {
	args := m.Called(ctx, id, lr)
	return args.Get(0).(*models.LogicalRouter), args.Error(1)
}

func (m *MockOVNService) DeleteLogicalRouter(ctx context.Context, id string) error {
	args := m.Called(ctx, id)
	return args.Error(0)
}

func (m *MockOVNService) ListPorts(ctx context.Context, switchID string) ([]*models.LogicalSwitchPort, error) {
	args := m.Called(ctx, switchID)
	return args.Get(0).([]*models.LogicalSwitchPort), args.Error(1)
}

func (m *MockOVNService) GetPort(ctx context.Context, id string) (*models.LogicalSwitchPort, error) {
	args := m.Called(ctx, id)
	return args.Get(0).(*models.LogicalSwitchPort), args.Error(1)
}

func (m *MockOVNService) CreatePort(ctx context.Context, switchID string, port *models.LogicalSwitchPort) (*models.LogicalSwitchPort, error) {
	args := m.Called(ctx, switchID, port)
	return args.Get(0).(*models.LogicalSwitchPort), args.Error(1)
}

func (m *MockOVNService) UpdatePort(ctx context.Context, id string, port *models.LogicalSwitchPort) (*models.LogicalSwitchPort, error) {
	args := m.Called(ctx, id, port)
	return args.Get(0).(*models.LogicalSwitchPort), args.Error(1)
}

func (m *MockOVNService) DeletePort(ctx context.Context, id string) error {
	args := m.Called(ctx, id)
	return args.Error(0)
}

func (m *MockOVNService) ListACLs(ctx context.Context, switchID string) ([]*models.ACL, error) {
	args := m.Called(ctx, switchID)
	return args.Get(0).([]*models.ACL), args.Error(1)
}

func (m *MockOVNService) GetACL(ctx context.Context, id string) (*models.ACL, error) {
	args := m.Called(ctx, id)
	return args.Get(0).(*models.ACL), args.Error(1)
}

func (m *MockOVNService) CreateACL(ctx context.Context, switchID string, acl *models.ACL) (*models.ACL, error) {
	args := m.Called(ctx, switchID, acl)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.ACL), args.Error(1)
}

func (m *MockOVNService) UpdateACL(ctx context.Context, id string, acl *models.ACL) (*models.ACL, error) {
	args := m.Called(ctx, id, acl)
	return args.Get(0).(*models.ACL), args.Error(1)
}

func (m *MockOVNService) DeleteACL(ctx context.Context, id string) error {
	args := m.Called(ctx, id)
	return args.Error(0)
}

func (m *MockOVNService) ExecuteTransaction(ctx context.Context, ops []TransactionOp) error {
	args := m.Called(ctx, ops)
	return args.Error(0)
}

func (m *MockOVNService) GetTopology(ctx context.Context) (*Topology, error) {
	args := m.Called(ctx)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*Topology), args.Error(1)
}

func TestTemplateService_ListTemplates(t *testing.T) {
	mockOVN := new(MockOVNService)
	logger := zap.NewNop()
	service := NewTemplateService(mockOVN, logger)

	templates := service.ListTemplates()
	assert.NotEmpty(t, templates)
	assert.Greater(t, len(templates), 5) // We have at least 6 built-in templates
}

func TestTemplateService_GetTemplate(t *testing.T) {
	mockOVN := new(MockOVNService)
	logger := zap.NewNop()
	service := NewTemplateService(mockOVN, logger)

	// Test valid template
	template, err := service.GetTemplate("web-server")
	assert.NoError(t, err)
	assert.NotNil(t, template)
	assert.Equal(t, "web-server", template.ID)
	assert.Equal(t, "Web Server", template.Name)

	// Test invalid template
	_, err = service.GetTemplate("non-existent")
	assert.Error(t, err)
}

func TestTemplateService_ValidateTemplate(t *testing.T) {
	mockOVN := new(MockOVNService)
	logger := zap.NewNop()
	service := NewTemplateService(mockOVN, logger)

	tests := []struct {
		name        string
		templateID  string
		variables   map[string]interface{}
		expectValid bool
		expectError bool
	}{
		{
			name:       "Valid web server template",
			templateID: "web-server",
			variables: map[string]interface{}{
				"server_ip": "10.0.1.10",
			},
			expectValid: true,
			expectError: false,
		},
		{
			name:       "Missing required variable",
			templateID: "web-server",
			variables:  map[string]interface{}{},
			expectValid: false,
			expectError: false,
		},
		{
			name:       "Invalid IP address",
			templateID: "web-server",
			variables: map[string]interface{}{
				"server_ip": "invalid-ip",
			},
			expectValid: false,
			expectError: false,
		},
		{
			name:       "Invalid template ID",
			templateID: "non-existent",
			variables:  map[string]interface{}{},
			expectValid: false,
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := service.ValidateTemplate(tt.templateID, tt.variables)
			
			if tt.expectError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, result)
				assert.Equal(t, tt.expectValid, result.Valid)
			}
		})
	}
}

func TestTemplateService_InstantiateTemplate(t *testing.T) {
	mockOVN := new(MockOVNService)
	logger := zap.NewNop()
	service := NewTemplateService(mockOVN, logger)

	ctx := context.Background()
	
	// Mock ACL creation
	mockOVN.On("CreateACL", ctx, "test-switch", mock.Anything).Return(&models.ACL{
		UUID: "test-uuid",
	}, nil)

	variables := map[string]interface{}{
		"server_ip":       "10.0.1.10",
		"allowed_sources": "192.168.0.0/16",
		"enable_ssh":      true,
		"ssh_sources":     "10.0.100.0/24",
	}

	instance, err := service.InstantiateTemplate(ctx, "web-server", variables, "test-switch")
	assert.NoError(t, err)
	assert.NotNil(t, instance)
	assert.Equal(t, "web-server", instance.TemplateID)
	assert.NotEmpty(t, instance.Rules)

	// Verify ACL creation was called
	mockOVN.AssertExpectations(t)
}

func TestTemplateService_ProcessTemplate(t *testing.T) {
	mockOVN := new(MockOVNService)
	logger := zap.NewNop()
	service := NewTemplateService(mockOVN, logger)

	tests := []struct {
		name      string
		template  string
		variables map[string]interface{}
		expected  string
	}{
		{
			name:     "Simple substitution",
			template: "ip4.dst == {{server_ip}}",
			variables: map[string]interface{}{
				"server_ip": "10.0.1.10",
			},
			expected: "ip4.dst == 10.0.1.10",
		},
		{
			name:     "Conditional logic",
			template: "{{if enable_ssh}}tcp.dst == 22{{else}}0{{end}}",
			variables: map[string]interface{}{
				"enable_ssh": true,
			},
			expected: "tcp.dst == 22",
		},
		{
			name:     "List formatting",
			template: "ip4.src == {{{allowed_ips}}}",
			variables: map[string]interface{}{
				"allowed_ips": "10.0.1.1,10.0.1.2,10.0.1.3",
			},
			expected: "ip4.src == {10.0.1.1, 10.0.1.2, 10.0.1.3}",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := service.processTemplate(tt.template, tt.variables)
			assert.NoError(t, err)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestTemplateService_ValidateVariable(t *testing.T) {
	mockOVN := new(MockOVNService)
	logger := zap.NewNop()
	service := NewTemplateService(mockOVN, logger)

	tests := []struct {
		name      string
		varDef    templates.TemplateVariable
		value     interface{}
		expectErr bool
	}{
		{
			name: "Valid IPv4",
			varDef: templates.TemplateVariable{
				Type: "ipv4",
			},
			value:     "192.168.1.1",
			expectErr: false,
		},
		{
			name: "Invalid IPv4",
			varDef: templates.TemplateVariable{
				Type: "ipv4",
			},
			value:     "256.1.1.1",
			expectErr: true,
		},
		{
			name: "Valid port",
			varDef: templates.TemplateVariable{
				Type: "port",
			},
			value:     8080,
			expectErr: false,
		},
		{
			name: "Invalid port",
			varDef: templates.TemplateVariable{
				Type: "port",
			},
			value:     70000,
			expectErr: true,
		},
		{
			name: "Valid CIDR",
			varDef: templates.TemplateVariable{
				Type: "cidr",
			},
			value:     "10.0.0.0/24",
			expectErr: false,
		},
		{
			name: "Invalid CIDR",
			varDef: templates.TemplateVariable{
				Type: "cidr",
			},
			value:     "10.0.0.0/33",
			expectErr: true,
		},
		{
			name: "Valid MAC",
			varDef: templates.TemplateVariable{
				Type: "mac",
			},
			value:     "00:00:00:00:00:01",
			expectErr: false,
		},
		{
			name: "Invalid MAC",
			varDef: templates.TemplateVariable{
				Type: "mac",
			},
			value:     "00:00:00:00:00:GG",
			expectErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := service.validateVariable(tt.varDef, tt.value)
			if tt.expectErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}