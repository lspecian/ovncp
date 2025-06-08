package handlers

import (
	"bytes"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"

	"github.com/lspecian/ovncp/internal/models"
)

func TestTransactionHandler_Execute(t *testing.T) {
	gin.SetMode(gin.TestMode)

	tests := []struct {
		name           string
		requestBody    interface{}
		setupMocks     func(*MockOVNService)
		expectedStatus int
		validateResponse func(*testing.T, map[string]interface{})
	}{
		{
			name: "successful transaction - create switch and router",
			requestBody: map[string]interface{}{
				"operations": []map[string]interface{}{
					{
						"id":       "op1",
						"type":     "create",
						"resource": "switch",
						"data": map[string]interface{}{
							"name": "test-switch",
						},
					},
					{
						"id":       "op2",
						"type":     "create",
						"resource": "router",
						"data": map[string]interface{}{
							"name": "test-router",
						},
					},
				},
			},
			setupMocks: func(m *MockOVNService) {
				m.On("CreateLogicalSwitch", mock.Anything, mock.Anything).Return(&models.LogicalSwitch{
					UUID:      "switch-uuid",
					Name:      "test-switch",
					CreatedAt: time.Now(),
					UpdatedAt: time.Now(),
				}, nil)
				m.On("CreateLogicalRouter", mock.Anything, mock.Anything).Return(&models.LogicalRouter{
					UUID:      "router-uuid",
					Name:      "test-router",
					CreatedAt: time.Now(),
					UpdatedAt: time.Now(),
				}, nil)
			},
			expectedStatus: http.StatusOK,
			validateResponse: func(t *testing.T, resp map[string]interface{}) {
				assert.True(t, resp["success"].(bool))
				results := resp["results"].([]interface{})
				assert.Len(t, results, 2)
				
				// Check first operation
				op1 := results[0].(map[string]interface{})
				assert.Equal(t, "op1", op1["id"])
				assert.True(t, op1["success"].(bool))
				assert.Equal(t, "switch-uuid", op1["resource_id"])
				
				// Check second operation
				op2 := results[1].(map[string]interface{})
				assert.Equal(t, "op2", op2["id"])
				assert.True(t, op2["success"].(bool))
				assert.Equal(t, "router-uuid", op2["resource_id"])
			},
		},
		{
			name: "successful transaction - create port and ACL",
			requestBody: map[string]interface{}{
				"operations": []map[string]interface{}{
					{
						"id":        "op1",
						"type":      "create",
						"resource":  "port",
						"switch_id": "switch-uuid",
						"data": map[string]interface{}{
							"name":      "test-port",
							"addresses": []string{"00:00:00:00:00:01"},
						},
					},
					{
						"id":        "op2",
						"type":      "create",
						"resource":  "acl",
						"switch_id": "switch-uuid",
						"data": map[string]interface{}{
							"priority":  100,
							"direction": "from-lport",
							"match":     "ip4.src == 10.0.0.1",
							"action":    "allow",
						},
					},
				},
			},
			setupMocks: func(m *MockOVNService) {
				m.On("CreatePort", mock.Anything, "switch-uuid", mock.Anything).Return(&models.LogicalSwitchPort{
					UUID:      "port-uuid",
					Name:      "test-port",
					Addresses: []string{"00:00:00:00:00:01"},
					CreatedAt: time.Now(),
					UpdatedAt: time.Now(),
				}, nil)
				m.On("CreateACL", mock.Anything, "switch-uuid", mock.Anything).Return(&models.ACL{
					UUID:      "acl-uuid",
					Priority:  100,
					Direction: "from-lport",
					Match:     "ip4.src == 10.0.0.1",
					Action:    "allow",
					CreatedAt: time.Now(),
					UpdatedAt: time.Now(),
				}, nil)
			},
			expectedStatus: http.StatusOK,
			validateResponse: func(t *testing.T, resp map[string]interface{}) {
				assert.True(t, resp["success"].(bool))
				results := resp["results"].([]interface{})
				assert.Len(t, results, 2)
			},
		},
		{
			name: "failed transaction with rollback",
			requestBody: map[string]interface{}{
				"operations": []map[string]interface{}{
					{
						"id":       "op1",
						"type":     "create",
						"resource": "switch",
						"data": map[string]interface{}{
							"name": "test-switch",
						},
					},
					{
						"id":       "op2",
						"type":     "create",
						"resource": "router",
						"data": map[string]interface{}{
							"name": "test-router",
						},
					},
				},
			},
			setupMocks: func(m *MockOVNService) {
				// First operation succeeds
				m.On("CreateLogicalSwitch", mock.Anything, mock.Anything).Return(&models.LogicalSwitch{
					UUID:      "switch-uuid",
					Name:      "test-switch",
					CreatedAt: time.Now(),
					UpdatedAt: time.Now(),
				}, nil)
				// Second operation fails
				m.On("CreateLogicalRouter", mock.Anything, mock.Anything).Return(nil, errors.New("router creation failed"))
				// Rollback
				m.On("DeleteLogicalSwitch", mock.Anything, "switch-uuid").Return(nil)
			},
			expectedStatus: http.StatusBadRequest,
			validateResponse: func(t *testing.T, resp map[string]interface{}) {
				assert.False(t, resp["success"].(bool))
				assert.Contains(t, resp["error"].(string), "router creation failed")
				results := resp["results"].([]interface{})
				assert.Len(t, results, 2)
				
				// First operation succeeded
				op1 := results[0].(map[string]interface{})
				assert.True(t, op1["success"].(bool))
				
				// Second operation failed
				op2 := results[1].(map[string]interface{})
				assert.False(t, op2["success"].(bool))
			},
		},
		{
			name: "update and delete operations",
			requestBody: map[string]interface{}{
				"operations": []map[string]interface{}{
					{
						"id":          "op1",
						"type":        "update",
						"resource":    "switch",
						"resource_id": "switch-uuid",
						"data": map[string]interface{}{
							"name": "updated-switch",
						},
					},
					{
						"id":          "op2",
						"type":        "delete",
						"resource":    "router",
						"resource_id": "router-uuid",
					},
				},
			},
			setupMocks: func(m *MockOVNService) {
				m.On("UpdateLogicalSwitch", mock.Anything, "switch-uuid", mock.Anything).Return(&models.LogicalSwitch{
					UUID:      "switch-uuid",
					Name:      "updated-switch",
					CreatedAt: time.Now(),
					UpdatedAt: time.Now(),
				}, nil)
				m.On("DeleteLogicalRouter", mock.Anything, "router-uuid").Return(nil)
			},
			expectedStatus: http.StatusOK,
			validateResponse: func(t *testing.T, resp map[string]interface{}) {
				assert.True(t, resp["success"].(bool))
				results := resp["results"].([]interface{})
				assert.Len(t, results, 2)
				
				// Both operations should succeed
				for _, result := range results {
					op := result.(map[string]interface{})
					assert.True(t, op["success"].(bool))
				}
			},
		},
		{
			name: "dry run validation",
			requestBody: map[string]interface{}{
				"dry_run": true,
				"operations": []map[string]interface{}{
					{
						"id":       "op1",
						"type":     "create",
						"resource": "switch",
						"data": map[string]interface{}{
							"name": "test-switch",
						},
					},
				},
			},
			setupMocks: func(m *MockOVNService) {
				// No mocks needed for dry run
			},
			expectedStatus: http.StatusOK,
			validateResponse: func(t *testing.T, resp map[string]interface{}) {
				assert.Equal(t, "validation successful", resp["message"])
				assert.Equal(t, float64(1), resp["operations"])
			},
		},
		{
			name: "validation error - empty operations",
			requestBody: map[string]interface{}{
				"operations": []map[string]interface{}{},
			},
			setupMocks:     func(m *MockOVNService) {},
			expectedStatus: http.StatusBadRequest,
			validateResponse: func(t *testing.T, resp map[string]interface{}) {
				assert.Equal(t, "at least one operation is required", resp["error"])
			},
		},
		{
			name: "validation error - too many operations",
			requestBody: map[string]interface{}{
				"operations": make([]map[string]interface{}, 101),
			},
			setupMocks:     func(m *MockOVNService) {},
			expectedStatus: http.StatusBadRequest,
			validateResponse: func(t *testing.T, resp map[string]interface{}) {
				assert.Equal(t, "maximum 100 operations per transaction", resp["error"])
			},
		},
		{
			name: "validation error - missing operation id",
			requestBody: map[string]interface{}{
				"operations": []map[string]interface{}{
					{
						"type":     "create",
						"resource": "switch",
						"data": map[string]interface{}{
							"name": "test-switch",
						},
					},
				},
			},
			setupMocks:     func(m *MockOVNService) {},
			expectedStatus: http.StatusBadRequest,
			validateResponse: func(t *testing.T, resp map[string]interface{}) {
				assert.Equal(t, "validation failed", resp["error"])
				assert.Contains(t, resp["details"], "id is required")
			},
		},
		{
			name: "validation error - duplicate operation id",
			requestBody: map[string]interface{}{
				"operations": []map[string]interface{}{
					{
						"id":       "op1",
						"type":     "create",
						"resource": "switch",
						"data": map[string]interface{}{
							"name": "test-switch1",
						},
					},
					{
						"id":       "op1", // Duplicate ID
						"type":     "create",
						"resource": "switch",
						"data": map[string]interface{}{
							"name": "test-switch2",
						},
					},
				},
			},
			setupMocks:     func(m *MockOVNService) {},
			expectedStatus: http.StatusBadRequest,
			validateResponse: func(t *testing.T, resp map[string]interface{}) {
				assert.Equal(t, "validation failed", resp["error"])
				assert.Contains(t, resp["details"], "duplicate operation id")
			},
		},
		{
			name: "validation error - invalid operation type",
			requestBody: map[string]interface{}{
				"operations": []map[string]interface{}{
					{
						"id":       "op1",
						"type":     "invalid",
						"resource": "switch",
					},
				},
			},
			setupMocks:     func(m *MockOVNService) {},
			expectedStatus: http.StatusBadRequest,
			validateResponse: func(t *testing.T, resp map[string]interface{}) {
				assert.Equal(t, "validation failed", resp["error"])
				assert.Contains(t, resp["details"], "invalid type")
			},
		},
		{
			name: "validation error - invalid resource type",
			requestBody: map[string]interface{}{
				"operations": []map[string]interface{}{
					{
						"id":       "op1",
						"type":     "create",
						"resource": "invalid",
					},
				},
			},
			setupMocks:     func(m *MockOVNService) {},
			expectedStatus: http.StatusBadRequest,
			validateResponse: func(t *testing.T, resp map[string]interface{}) {
				assert.Equal(t, "validation failed", resp["error"])
				assert.Contains(t, resp["details"], "invalid resource")
			},
		},
		{
			name: "validation error - create without data",
			requestBody: map[string]interface{}{
				"operations": []map[string]interface{}{
					{
						"id":       "op1",
						"type":     "create",
						"resource": "switch",
					},
				},
			},
			setupMocks:     func(m *MockOVNService) {},
			expectedStatus: http.StatusBadRequest,
			validateResponse: func(t *testing.T, resp map[string]interface{}) {
				assert.Equal(t, "validation failed", resp["error"])
				assert.Contains(t, resp["details"], "data is required for create operation")
			},
		},
		{
			name: "validation error - update without resource_id",
			requestBody: map[string]interface{}{
				"operations": []map[string]interface{}{
					{
						"id":       "op1",
						"type":     "update",
						"resource": "switch",
						"data": map[string]interface{}{
							"name": "updated-switch",
						},
					},
				},
			},
			setupMocks:     func(m *MockOVNService) {},
			expectedStatus: http.StatusBadRequest,
			validateResponse: func(t *testing.T, resp map[string]interface{}) {
				assert.Equal(t, "validation failed", resp["error"])
				assert.Contains(t, resp["details"], "resource_id is required for update operation")
			},
		},
		{
			name: "validation error - delete with data",
			requestBody: map[string]interface{}{
				"operations": []map[string]interface{}{
					{
						"id":          "op1",
						"type":        "delete",
						"resource":    "switch",
						"resource_id": "switch-uuid",
						"data": map[string]interface{}{
							"name": "should-not-be-here",
						},
					},
				},
			},
			setupMocks:     func(m *MockOVNService) {},
			expectedStatus: http.StatusBadRequest,
			validateResponse: func(t *testing.T, resp map[string]interface{}) {
				assert.Equal(t, "validation failed", resp["error"])
				assert.Contains(t, resp["details"], "data should not be provided for delete operation")
			},
		},
		{
			name: "validation error - create port without switch_id",
			requestBody: map[string]interface{}{
				"operations": []map[string]interface{}{
					{
						"id":       "op1",
						"type":     "create",
						"resource": "port",
						"data": map[string]interface{}{
							"name": "test-port",
						},
					},
				},
			},
			setupMocks:     func(m *MockOVNService) {},
			expectedStatus: http.StatusBadRequest,
			validateResponse: func(t *testing.T, resp map[string]interface{}) {
				assert.Equal(t, "validation failed", resp["error"])
				assert.Contains(t, resp["details"], "switch_id is required for port creation")
			},
		},
		{
			name:           "invalid json",
			requestBody:    "invalid json",
			setupMocks:     func(m *MockOVNService) {},
			expectedStatus: http.StatusBadRequest,
			validateResponse: func(t *testing.T, resp map[string]interface{}) {
				assert.Equal(t, "invalid request body", resp["error"])
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockService := new(MockOVNService)
			handler := NewTransactionHandler(mockService)

			tt.setupMocks(mockService)

			body, _ := json.Marshal(tt.requestBody)

			w := httptest.NewRecorder()
			c, _ := gin.CreateTestContext(w)
			c.Request = httptest.NewRequest("POST", "/api/v1/transactions", bytes.NewReader(body))
			c.Request.Header.Set("Content-Type", "application/json")

			handler.Execute(c)

			assert.Equal(t, tt.expectedStatus, w.Code)

			var response map[string]interface{}
			err := json.Unmarshal(w.Body.Bytes(), &response)
			assert.NoError(t, err)

			tt.validateResponse(t, response)

			mockService.AssertExpectations(t)
		})
	}
}