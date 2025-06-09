package backup

import (
	"context"
	"testing"

	"github.com/lspecian/ovncp/internal/models"
	"github.com/lspecian/ovncp/internal/services"
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
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*models.LogicalSwitch), args.Error(1)
}

func (m *MockOVNService) GetLogicalSwitch(ctx context.Context, id string) (*models.LogicalSwitch, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.LogicalSwitch), args.Error(1)
}

func (m *MockOVNService) CreateLogicalSwitch(ctx context.Context, ls *models.LogicalSwitch) (*models.LogicalSwitch, error) {
	args := m.Called(ctx, ls)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.LogicalSwitch), args.Error(1)
}

func (m *MockOVNService) UpdateLogicalSwitch(ctx context.Context, id string, ls *models.LogicalSwitch) (*models.LogicalSwitch, error) {
	args := m.Called(ctx, id, ls)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.LogicalSwitch), args.Error(1)
}

func (m *MockOVNService) DeleteLogicalSwitch(ctx context.Context, id string) error {
	args := m.Called(ctx, id)
	return args.Error(0)
}

func (m *MockOVNService) ListLogicalRouters(ctx context.Context) ([]*models.LogicalRouter, error) {
	args := m.Called(ctx)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*models.LogicalRouter), args.Error(1)
}

func (m *MockOVNService) GetLogicalRouter(ctx context.Context, id string) (*models.LogicalRouter, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.LogicalRouter), args.Error(1)
}

func (m *MockOVNService) CreateLogicalRouter(ctx context.Context, lr *models.LogicalRouter) (*models.LogicalRouter, error) {
	args := m.Called(ctx, lr)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.LogicalRouter), args.Error(1)
}

func (m *MockOVNService) UpdateLogicalRouter(ctx context.Context, id string, lr *models.LogicalRouter) (*models.LogicalRouter, error) {
	args := m.Called(ctx, id, lr)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.LogicalRouter), args.Error(1)
}

func (m *MockOVNService) DeleteLogicalRouter(ctx context.Context, id string) error {
	args := m.Called(ctx, id)
	return args.Error(0)
}

func (m *MockOVNService) ListPorts(ctx context.Context, switchID string) ([]*models.LogicalSwitchPort, error) {
	args := m.Called(ctx, switchID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*models.LogicalSwitchPort), args.Error(1)
}

func (m *MockOVNService) GetPort(ctx context.Context, id string) (*models.LogicalSwitchPort, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.LogicalSwitchPort), args.Error(1)
}

func (m *MockOVNService) CreatePort(ctx context.Context, switchID string, port *models.LogicalSwitchPort) (*models.LogicalSwitchPort, error) {
	args := m.Called(ctx, switchID, port)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.LogicalSwitchPort), args.Error(1)
}

func (m *MockOVNService) UpdatePort(ctx context.Context, id string, port *models.LogicalSwitchPort) (*models.LogicalSwitchPort, error) {
	args := m.Called(ctx, id, port)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.LogicalSwitchPort), args.Error(1)
}

func (m *MockOVNService) DeletePort(ctx context.Context, id string) error {
	args := m.Called(ctx, id)
	return args.Error(0)
}

func (m *MockOVNService) ListACLs(ctx context.Context, switchID string) ([]*models.ACL, error) {
	args := m.Called(ctx, switchID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*models.ACL), args.Error(1)
}

func (m *MockOVNService) GetACL(ctx context.Context, id string) (*models.ACL, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
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
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.ACL), args.Error(1)
}

func (m *MockOVNService) DeleteACL(ctx context.Context, id string) error {
	args := m.Called(ctx, id)
	return args.Error(0)
}

func (m *MockOVNService) ExecuteTransaction(ctx context.Context, ops []services.TransactionOp) error {
	args := m.Called(ctx, ops)
	return args.Error(0)
}

func (m *MockOVNService) GetTopology(ctx context.Context) (*services.Topology, error) {
	args := m.Called(ctx)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*services.Topology), args.Error(1)
}

// MockBackupStorage for testing
type MockBackupStorage struct {
	mock.Mock
	backups map[string]*BackupData
}

func NewMockBackupStorage() *MockBackupStorage {
	return &MockBackupStorage{
		backups: make(map[string]*BackupData),
	}
}

func (m *MockBackupStorage) Store(backup *BackupData, options *BackupOptions) (string, error) {
	args := m.Called(backup, options)
	if args.Error(1) == nil {
		m.backups[backup.Metadata.ID] = backup
	}
	return args.String(0), args.Error(1)
}

func (m *MockBackupStorage) Retrieve(backupID string) (*BackupData, error) {
	args := m.Called(backupID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*BackupData), args.Error(1)
}

func (m *MockBackupStorage) List() ([]*BackupMetadata, error) {
	args := m.Called()
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*BackupMetadata), args.Error(1)
}

func (m *MockBackupStorage) Delete(backupID string) error {
	args := m.Called(backupID)
	delete(m.backups, backupID)
	return args.Error(0)
}

func (m *MockBackupStorage) Exists(backupID string) (bool, error) {
	args := m.Called(backupID)
	return args.Bool(0), args.Error(1)
}

// Tests

func TestBackupService_CreateFullBackup(t *testing.T) {
	ctx := context.Background()
	
	// Setup mocks
	mockOVN := new(MockOVNService)
	mockStorage := NewMockBackupStorage()
	logger := zap.NewNop()
	
	service := NewBackupService(mockOVN, mockStorage, logger)
	
	// Mock data
	switches := []*models.LogicalSwitch{
		{UUID: "sw1", Name: "switch1"},
		{UUID: "sw2", Name: "switch2"},
	}
	
	routers := []*models.LogicalRouter{
		{UUID: "r1", Name: "router1"},
	}
	
	ports := []*models.LogicalSwitchPort{
		{UUID: "p1", Name: "port1"},
		{UUID: "p2", Name: "port2"},
	}
	
	acls := []*models.ACL{
		{UUID: "acl1", Name: "allow-http"},
	}
	
	// Setup expectations
	mockOVN.On("ListLogicalSwitches", ctx).Return(switches, nil)
	mockOVN.On("ListLogicalRouters", ctx).Return(routers, nil)
	mockOVN.On("ListPorts", ctx, "sw1").Return(ports, nil)
	mockOVN.On("ListPorts", ctx, "sw2").Return([]*models.LogicalSwitchPort{}, nil)
	mockOVN.On("ListACLs", ctx, "sw1").Return(acls, nil)
	mockOVN.On("ListACLs", ctx, "sw2").Return([]*models.ACL{}, nil)
	
	mockStorage.On("Store", mock.Anything, mock.Anything).Return("backup-id", nil)
	
	// Create backup
	options := &BackupOptions{
		Name:   "Test Backup",
		Type:   BackupTypeFull,
		Format: BackupFormatJSON,
	}
	
	metadata, err := service.CreateBackup(ctx, options)
	
	// Assertions
	assert.NoError(t, err)
	assert.NotNil(t, metadata)
	assert.Equal(t, "Test Backup", metadata.Name)
	assert.Equal(t, BackupTypeFull, metadata.Type)
	
	// Verify mocks
	mockOVN.AssertExpectations(t)
	mockStorage.AssertExpectations(t)
}

func TestBackupService_CreateSelectiveBackup(t *testing.T) {
	ctx := context.Background()
	
	// Setup mocks
	mockOVN := new(MockOVNService)
	mockStorage := NewMockBackupStorage()
	logger := zap.NewNop()
	
	service := NewBackupService(mockOVN, mockStorage, logger)
	
	// Mock data
	switch1 := &models.LogicalSwitch{UUID: "sw1", Name: "switch1"}
	
	// Setup expectations
	mockOVN.On("GetLogicalSwitch", ctx, "sw1").Return(switch1, nil)
	mockStorage.On("Store", mock.Anything, mock.Anything).Return("backup-id", nil)
	
	// Create selective backup
	options := &BackupOptions{
		Name:   "Selective Backup",
		Type:   BackupTypeSelective,
		Format: BackupFormatJSON,
		ResourceFilter: &ResourceFilter{
			Switches:    []string{"sw1"},
			IncludeACLs: false,
		},
	}
	
	metadata, err := service.CreateBackup(ctx, options)
	
	// Assertions
	assert.NoError(t, err)
	assert.NotNil(t, metadata)
	assert.Equal(t, "Selective Backup", metadata.Name)
	assert.Equal(t, BackupTypeSelective, metadata.Type)
	
	// Verify mocks
	mockOVN.AssertExpectations(t)
	mockStorage.AssertExpectations(t)
}

func TestBackupService_RestoreBackup(t *testing.T) {
	ctx := context.Background()
	
	// Setup mocks
	mockOVN := new(MockOVNService)
	mockStorage := NewMockBackupStorage()
	logger := zap.NewNop()
	
	service := NewBackupService(mockOVN, mockStorage, logger)
	
	// Mock backup data
	backupData := &BackupData{
		Metadata: BackupMetadata{
			ID:   "backup-123",
			Name: "Test Backup",
		},
		LogicalSwitches: []*models.LogicalSwitch{
			{UUID: "sw1", Name: "switch1"},
		},
		LogicalRouters: []*models.LogicalRouter{
			{UUID: "r1", Name: "router1"},
		},
		LogicalPorts: []*LogicalPortWithSwitch{
			{
				LogicalSwitchPort: &models.LogicalSwitchPort{
					UUID: "p1",
					Name: "port1",
				},
				SwitchID: "sw1",
			},
		},
		ACLs: []*ACLWithSwitch{
			{
				ACL: &models.ACL{
					UUID: "acl1",
					Name: "allow-http",
				},
				SwitchID: "sw1",
			},
		},
	}
	
	// Setup expectations
	mockStorage.On("Retrieve", "backup-123").Return(backupData, nil)
	
	// Existing resources (for conflict detection)
	mockOVN.On("GetLogicalSwitch", ctx, "switch1").Return(nil, nil) // Not exists
	mockOVN.On("GetLogicalRouter", ctx, "router1").Return(nil, nil) // Not exists
	
	// Create resources
	mockOVN.On("CreateLogicalSwitch", ctx, mock.Anything).Return(&models.LogicalSwitch{}, nil)
	mockOVN.On("CreateLogicalRouter", ctx, mock.Anything).Return(&models.LogicalRouter{}, nil)
	mockOVN.On("CreatePort", ctx, "sw1", mock.Anything).Return(&models.LogicalSwitchPort{}, nil)
	mockOVN.On("CreateACL", ctx, "sw1", mock.Anything).Return(&models.ACL{}, nil)
	
	// Restore backup
	options := &RestoreOptions{
		DryRun:         false,
		ConflictPolicy: ConflictPolicySkip,
	}
	
	result, err := service.RestoreBackup(ctx, "backup-123", options)
	
	// Assertions
	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.True(t, result.Success)
	assert.Equal(t, 4, result.RestoredCount) // 1 switch + 1 router + 1 port + 1 ACL
	assert.Equal(t, 0, result.SkippedCount)
	assert.Equal(t, 0, result.ErrorCount)
	
	// Verify mocks
	mockOVN.AssertExpectations(t)
	mockStorage.AssertExpectations(t)
}

func TestBackupService_RestoreWithConflicts(t *testing.T) {
	ctx := context.Background()
	
	// Setup mocks
	mockOVN := new(MockOVNService)
	mockStorage := NewMockBackupStorage()
	logger := zap.NewNop()
	
	service := NewBackupService(mockOVN, mockStorage, logger)
	
	// Mock backup data
	backupData := &BackupData{
		Metadata: BackupMetadata{
			ID:   "backup-123",
			Name: "Test Backup",
		},
		LogicalSwitches: []*models.LogicalSwitch{
			{UUID: "sw1", Name: "switch1"},
		},
	}
	
	// Setup expectations
	mockStorage.On("Retrieve", "backup-123").Return(backupData, nil)
	
	// Existing resource (conflict)
	existingSwitch := &models.LogicalSwitch{UUID: "existing-sw1", Name: "switch1"}
	mockOVN.On("GetLogicalSwitch", ctx, "switch1").Return(existingSwitch, nil)
	
	// Test skip policy
	options := &RestoreOptions{
		DryRun:         false,
		ConflictPolicy: ConflictPolicySkip,
	}
	
	result, err := service.RestoreBackup(ctx, "backup-123", options)
	
	// Assertions
	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.True(t, result.Success)
	assert.Equal(t, 0, result.RestoredCount)
	assert.Equal(t, 1, result.SkippedCount)
	
	// Verify mocks
	mockOVN.AssertExpectations(t)
	mockStorage.AssertExpectations(t)
}

func TestBackupService_DryRunRestore(t *testing.T) {
	ctx := context.Background()
	
	// Setup mocks
	mockOVN := new(MockOVNService)
	mockStorage := NewMockBackupStorage()
	logger := zap.NewNop()
	
	service := NewBackupService(mockOVN, mockStorage, logger)
	
	// Mock backup data
	backupData := &BackupData{
		Metadata: BackupMetadata{
			ID:   "backup-123",
			Name: "Test Backup",
		},
		LogicalSwitches: []*models.LogicalSwitch{
			{UUID: "sw1", Name: "switch1"},
			{UUID: "sw2", Name: "switch2"},
		},
	}
	
	// Setup expectations
	mockStorage.On("Retrieve", "backup-123").Return(backupData, nil)
	
	// Check existing resources
	mockOVN.On("GetLogicalSwitch", ctx, "switch1").Return(nil, nil) // Not exists
	existingSwitch := &models.LogicalSwitch{UUID: "existing-sw2", Name: "switch2"}
	mockOVN.On("GetLogicalSwitch", ctx, "switch2").Return(existingSwitch, nil) // Exists
	
	// Dry run - no create/delete operations should be called
	
	options := &RestoreOptions{
		DryRun:         true,
		ConflictPolicy: ConflictPolicySkip,
	}
	
	result, err := service.RestoreBackup(ctx, "backup-123", options)
	
	// Assertions
	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.True(t, result.Success)
	assert.Equal(t, 1, result.RestoredCount) // Would restore switch1
	assert.Equal(t, 1, result.SkippedCount)  // Would skip switch2
	
	// Verify no actual restore operations were called
	mockOVN.AssertNotCalled(t, "CreateLogicalSwitch", mock.Anything, mock.Anything)
	mockOVN.AssertNotCalled(t, "DeleteLogicalSwitch", mock.Anything, mock.Anything)
	
	// Verify mocks
	mockOVN.AssertExpectations(t)
	mockStorage.AssertExpectations(t)
}