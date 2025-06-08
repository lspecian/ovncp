package ovn

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/ovn-org/libovsdb/client"
	"github.com/ovn-org/libovsdb/model"
	"github.com/ovn-org/libovsdb/ovsdb"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

// Mock OVSDB client
type mockOVSDBClient struct {
	mock.Mock
}

func (m *mockOVSDBClient) Connect(ctx context.Context) error {
	args := m.Called(ctx)
	return args.Error(0)
}

func (m *mockOVSDBClient) Disconnect() {
	m.Called()
}

func (m *mockOVSDBClient) List(ctx context.Context, result interface{}) error {
	args := m.Called(ctx, result)
	return args.Error(0)
}

func (m *mockOVSDBClient) Get(ctx context.Context, model model.Model) error {
	args := m.Called(ctx, model)
	return args.Error(0)
}

func (m *mockOVSDBClient) Create(models ...model.Model) ([]ovsdb.Operation, error) {
	args := m.Called(models)
	return args.Get(0).([]ovsdb.Operation), args.Error(1)
}

func (m *mockOVSDBClient) Update(model model.Model, fields ...interface{}) ([]ovsdb.Operation, error) {
	args := m.Called(model, fields)
	return args.Get(0).([]ovsdb.Operation), args.Error(1)
}

func (m *mockOVSDBClient) Delete(models ...model.Model) ([]ovsdb.Operation, error) {
	args := m.Called(models)
	return args.Get(0).([]ovsdb.Operation), args.Error(1)
}

func (m *mockOVSDBClient) Transact(ctx context.Context, operations ...ovsdb.Operation) ([]ovsdb.OperationResult, error) {
	args := m.Called(ctx, operations)
	return args.Get(0).([]ovsdb.OperationResult), args.Error(1)
}

func (m *mockOVSDBClient) WhereCache(predicate interface{}) client.ConditionalAPI {
	args := m.Called(predicate)
	return args.Get(0).(client.ConditionalAPI)
}

// Test Client
func TestClient_Connect(t *testing.T) {
	mockDB := new(mockOVSDBClient)
	client := &Client{
		northbound: mockDB,
		southbound: mockDB,
	}
	
	ctx := context.Background()
	mockDB.On("Connect", ctx).Return(nil)
	
	err := client.Connect(ctx)
	assert.NoError(t, err)
	
	mockDB.AssertExpectations(t)
}

func TestClient_ListLogicalSwitches(t *testing.T) {
	mockDB := new(mockOVSDBClient)
	client := &Client{
		northbound: mockDB,
		connected:  true,
	}
	
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
	
	ctx := context.Background()
	mockDB.On("List", ctx, mock.AnythingOfType("*[]ovn.LogicalSwitch")).
		Run(func(args mock.Arguments) {
			result := args.Get(1).(*[]LogicalSwitch)
			*result = switches
		}).Return(nil)
	
	result, err := client.ListLogicalSwitches()
	assert.NoError(t, err)
	assert.Equal(t, len(switches), len(result))
	assert.Equal(t, switches[0].Name, result[0].Name)
	
	mockDB.AssertExpectations(t)
}

func TestClient_GetLogicalSwitch(t *testing.T) {
	mockDB := new(mockOVSDBClient)
	client := &Client{
		northbound: mockDB,
		connected:  true,
	}
	
	switchID := uuid.New().String()
	ls := &LogicalSwitch{
		UUID:        switchID,
		Name:        "switch1",
		Description: "Test switch",
	}
	
	ctx := context.Background()
	mockDB.On("Get", ctx, mock.AnythingOfType("*ovn.LogicalSwitch")).
		Run(func(args mock.Arguments) {
			result := args.Get(1).(*LogicalSwitch)
			*result = *ls
		}).Return(nil)
	
	result, err := client.GetLogicalSwitch(switchID)
	assert.NoError(t, err)
	assert.Equal(t, ls.UUID, result.UUID)
	assert.Equal(t, ls.Name, result.Name)
	
	mockDB.AssertExpectations(t)
}

func TestClient_CreateLogicalSwitch(t *testing.T) {
	mockDB := new(mockOVSDBClient)
	client := &Client{
		northbound: mockDB,
		connected:  true,
	}
	
	name := "new-switch"
	description := "New test switch"
	newUUID := uuid.New().String()
	
	ls := &LogicalSwitch{
		UUID:        newUUID,
		Name:        name,
		Description: description,
	}
	
	ops := []ovsdb.Operation{
		{
			Op:    "insert",
			Table: "Logical_Switch",
			Row: ovsdb.Row{
				"name":        name,
				"description": description,
			},
			UUIDName: &newUUID,
		},
	}
	
	results := []ovsdb.OperationResult{
		{
			UUID: &ovsdb.UUID{GoUUID: newUUID},
		},
	}
	
	ctx := context.Background()
	mockDB.On("Create", mock.AnythingOfType("[]model.Model")).Return(ops, nil)
	mockDB.On("Transact", ctx, ops).Return(results, nil)
	mockDB.On("Get", ctx, mock.AnythingOfType("*ovn.LogicalSwitch")).
		Run(func(args mock.Arguments) {
			result := args.Get(1).(*LogicalSwitch)
			*result = *ls
		}).Return(nil)
	
	result, err := client.CreateLogicalSwitch(name, description)
	assert.NoError(t, err)
	assert.Equal(t, name, result.Name)
	assert.Equal(t, description, result.Description)
	
	mockDB.AssertExpectations(t)
}

func TestClient_UpdateLogicalSwitch(t *testing.T) {
	mockDB := new(mockOVSDBClient)
	client := &Client{
		northbound: mockDB,
		connected:  true,
	}
	
	switchID := uuid.New().String()
	updates := map[string]interface{}{
		"name":        "updated-switch",
		"description": "Updated description",
	}
	
	existing := &LogicalSwitch{
		UUID:        switchID,
		Name:        "old-switch",
		Description: "Old description",
	}
	
	updated := &LogicalSwitch{
		UUID:        switchID,
		Name:        updates["name"].(string),
		Description: updates["description"].(string),
	}
	
	ops := []ovsdb.Operation{
		{
			Op:    "update",
			Table: "Logical_Switch",
			Where: []ovsdb.Condition{
				{
					Column:   "_uuid",
					Function: "==",
					Value:    ovsdb.UUID{GoUUID: switchID},
				},
			},
			Row: updates,
		},
	}
	
	results := []ovsdb.OperationResult{
		{Count: 1},
	}
	
	ctx := context.Background()
	
	// First Get to find existing
	mockDB.On("Get", ctx, mock.AnythingOfType("*ovn.LogicalSwitch")).
		Run(func(args mock.Arguments) {
			result := args.Get(1).(*LogicalSwitch)
			*result = *existing
		}).Return(nil).Once()
	
	mockDB.On("Update", mock.Anything, mock.Anything).Return(ops, nil)
	mockDB.On("Transact", ctx, ops).Return(results, nil)
	
	// Second Get to return updated
	mockDB.On("Get", ctx, mock.AnythingOfType("*ovn.LogicalSwitch")).
		Run(func(args mock.Arguments) {
			result := args.Get(1).(*LogicalSwitch)
			*result = *updated
		}).Return(nil).Once()
	
	result, err := client.UpdateLogicalSwitch(switchID, updates)
	assert.NoError(t, err)
	assert.Equal(t, updated.Name, result.Name)
	assert.Equal(t, updated.Description, result.Description)
	
	mockDB.AssertExpectations(t)
}

func TestClient_DeleteLogicalSwitch(t *testing.T) {
	mockDB := new(mockOVSDBClient)
	client := &Client{
		northbound: mockDB,
		connected:  true,
	}
	
	switchID := uuid.New().String()
	ls := &LogicalSwitch{
		UUID: switchID,
		Name: "switch-to-delete",
	}
	
	ops := []ovsdb.Operation{
		{
			Op:    "delete",
			Table: "Logical_Switch",
			Where: []ovsdb.Condition{
				{
					Column:   "_uuid",
					Function: "==",
					Value:    ovsdb.UUID{GoUUID: switchID},
				},
			},
		},
	}
	
	results := []ovsdb.OperationResult{
		{Count: 1},
	}
	
	ctx := context.Background()
	mockDB.On("Get", ctx, mock.AnythingOfType("*ovn.LogicalSwitch")).
		Run(func(args mock.Arguments) {
			result := args.Get(1).(*LogicalSwitch)
			*result = *ls
		}).Return(nil)
	mockDB.On("Delete", mock.AnythingOfType("[]model.Model")).Return(ops, nil)
	mockDB.On("Transact", ctx, ops).Return(results, nil)
	
	err := client.DeleteLogicalSwitch(switchID)
	assert.NoError(t, err)
	
	mockDB.AssertExpectations(t)
}

// Transaction Tests
func TestClient_CreateTransaction(t *testing.T) {
	mockDB := new(mockOVSDBClient)
	client := &Client{
		northbound: mockDB,
		connected:  true,
	}
	
	tx := client.CreateTransaction()
	assert.NotNil(t, tx)
	assert.IsType(t, &Transaction{}, tx)
}

func TestTransaction_AddCreateSwitch(t *testing.T) {
	tx := &Transaction{
		operations: []ovsdb.Operation{},
	}
	
	name := "tx-switch"
	description := "Transaction test switch"
	
	tx.AddCreateSwitch(name, description)
	
	assert.Len(t, tx.operations, 1)
	assert.Equal(t, "insert", tx.operations[0].Op)
	assert.Equal(t, "Logical_Switch", tx.operations[0].Table)
	assert.Equal(t, name, tx.operations[0].Row["name"])
}

func TestTransaction_AddDeleteSwitch(t *testing.T) {
	tx := &Transaction{
		operations: []ovsdb.Operation{},
	}
	
	switchID := uuid.New().String()
	tx.AddDeleteSwitch(switchID)
	
	assert.Len(t, tx.operations, 1)
	assert.Equal(t, "delete", tx.operations[0].Op)
	assert.Equal(t, "Logical_Switch", tx.operations[0].Table)
}

func TestTransaction_Commit(t *testing.T) {
	mockDB := new(mockOVSDBClient)
	client := &Client{
		northbound: mockDB,
		connected:  true,
	}
	
	tx := &Transaction{
		client:     client,
		operations: []ovsdb.Operation{},
	}
	
	// Add some operations
	tx.AddCreateSwitch("switch1", "desc1")
	tx.AddCreateSwitch("switch2", "desc2")
	
	results := []ovsdb.OperationResult{
		{UUID: &ovsdb.UUID{GoUUID: uuid.New().String()}},
		{UUID: &ovsdb.UUID{GoUUID: uuid.New().String()}},
	}
	
	ctx := context.Background()
	mockDB.On("Transact", ctx, tx.operations).Return(results, nil)
	
	err := tx.Commit()
	assert.NoError(t, err)
	
	mockDB.AssertExpectations(t)
}

func TestTransaction_Commit_WithError(t *testing.T) {
	mockDB := new(mockOVSDBClient)
	client := &Client{
		northbound: mockDB,
		connected:  true,
	}
	
	tx := &Transaction{
		client:     client,
		operations: []ovsdb.Operation{},
	}
	
	tx.AddCreateSwitch("switch1", "desc1")
	
	results := []ovsdb.OperationResult{
		{
			Error: &ovsdb.OperationError{
				Error:   "constraint violation",
				Details: "duplicate name",
			},
		},
	}
	
	ctx := context.Background()
	mockDB.On("Transact", ctx, tx.operations).Return(results, nil)
	
	err := tx.Commit()
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "constraint violation")
	
	mockDB.AssertExpectations(t)
}

// Connection Tests
func TestClient_ConnectionRetry(t *testing.T) {
	mockDB := new(mockOVSDBClient)
	client := &Client{
		northbound: mockDB,
		southbound: mockDB,
		config: Config{
			MaxRetries:    3,
			RetryInterval: 10 * time.Millisecond,
		},
	}
	
	ctx := context.Background()
	
	// Fail twice, then succeed
	mockDB.On("Connect", ctx).Return(assert.AnError).Twice()
	mockDB.On("Connect", ctx).Return(nil).Once()
	
	err := client.Connect(ctx)
	assert.NoError(t, err)
	
	mockDB.AssertExpectations(t)
}

func TestClient_ConnectionMaxRetries(t *testing.T) {
	mockDB := new(mockOVSDBClient)
	client := &Client{
		northbound: mockDB,
		southbound: mockDB,
		config: Config{
			MaxRetries:    2,
			RetryInterval: 10 * time.Millisecond,
		},
	}
	
	ctx := context.Background()
	
	// Always fail
	mockDB.On("Connect", ctx).Return(assert.AnError)
	
	err := client.Connect(ctx)
	assert.Error(t, err)
	
	// Should have tried MaxRetries + 1 times
	mockDB.AssertNumberOfCalls(t, "Connect", 3)
}