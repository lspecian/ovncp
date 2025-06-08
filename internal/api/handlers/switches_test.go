package handlers

import (
	"bytes"
	"context"
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

// MockOVNService is a mock implementation of the OVN service interface
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

// Implement remaining interface methods with default behavior
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

func TestSwitchHandler_List(t *testing.T) {
	gin.SetMode(gin.TestMode)

	tests := []struct {
		name           string
		mockReturn     []*models.LogicalSwitch
		mockError      error
		expectedStatus int
		expectedBody   map[string]interface{}
	}{
		{
			name: "successful list",
			mockReturn: []*models.LogicalSwitch{
				{
					UUID:      "uuid1",
					Name:      "switch1",
					CreatedAt: time.Now(),
					UpdatedAt: time.Now(),
				},
				{
					UUID:      "uuid2",
					Name:      "switch2",
					CreatedAt: time.Now(),
					UpdatedAt: time.Now(),
				},
			},
			mockError:      nil,
			expectedStatus: http.StatusOK,
			expectedBody: map[string]interface{}{
				"count": float64(2),
			},
		},
		{
			name:           "empty list",
			mockReturn:     []*models.LogicalSwitch{},
			mockError:      nil,
			expectedStatus: http.StatusOK,
			expectedBody: map[string]interface{}{
				"count": float64(0),
			},
		},
		{
			name:           "service error",
			mockReturn:     nil,
			mockError:      errors.New("service error"),
			expectedStatus: http.StatusInternalServerError,
			expectedBody: map[string]interface{}{
				"error": "internal server error",
			},
		},
		{
			name:           "not connected error",
			mockReturn:     nil,
			mockError:      errors.New("client not connected"),
			expectedStatus: http.StatusServiceUnavailable,
			expectedBody: map[string]interface{}{
				"error": "OVN service unavailable",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockService := new(MockOVNService)
			handler := NewSwitchHandler(mockService)

			mockService.On("ListLogicalSwitches", mock.Anything).Return(tt.mockReturn, tt.mockError)

			w := httptest.NewRecorder()
			c, _ := gin.CreateTestContext(w)
			c.Request = httptest.NewRequest("GET", "/api/v1/switches", nil)

			handler.List(c)

			assert.Equal(t, tt.expectedStatus, w.Code)

			var response map[string]interface{}
			err := json.Unmarshal(w.Body.Bytes(), &response)
			assert.NoError(t, err)

			for key, value := range tt.expectedBody {
				assert.Equal(t, value, response[key])
			}

			mockService.AssertExpectations(t)
		})
	}
}

func TestSwitchHandler_Get(t *testing.T) {
	gin.SetMode(gin.TestMode)

	tests := []struct {
		name           string
		switchID       string
		mockReturn     *models.LogicalSwitch
		mockError      error
		expectedStatus int
	}{
		{
			name:     "successful get",
			switchID: "uuid1",
			mockReturn: &models.LogicalSwitch{
				UUID:      "uuid1",
				Name:      "switch1",
				CreatedAt: time.Now(),
				UpdatedAt: time.Now(),
			},
			mockError:      nil,
			expectedStatus: http.StatusOK,
		},
		{
			name:           "not found",
			switchID:       "nonexistent",
			mockReturn:     nil,
			mockError:      errors.New("logical switch nonexistent not found"),
			expectedStatus: http.StatusNotFound,
		},
		{
			name:           "empty id",
			switchID:       "",
			mockReturn:     nil,
			mockError:      nil,
			expectedStatus: http.StatusBadRequest,
		},
		{
			name:           "service error",
			switchID:       "uuid1",
			mockReturn:     nil,
			mockError:      errors.New("service error"),
			expectedStatus: http.StatusInternalServerError,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockService := new(MockOVNService)
			handler := NewSwitchHandler(mockService)

			if tt.switchID != "" {
				mockService.On("GetLogicalSwitch", mock.Anything, tt.switchID).Return(tt.mockReturn, tt.mockError)
			}

			w := httptest.NewRecorder()
			c, _ := gin.CreateTestContext(w)
			c.Request = httptest.NewRequest("GET", "/api/v1/switches/"+tt.switchID, nil)
			c.Params = gin.Params{{Key: "id", Value: tt.switchID}}

			handler.Get(c)

			assert.Equal(t, tt.expectedStatus, w.Code)

			if tt.switchID != "" {
				mockService.AssertExpectations(t)
			}
		})
	}
}

func TestSwitchHandler_Create(t *testing.T) {
	gin.SetMode(gin.TestMode)

	tests := []struct {
		name           string
		requestBody    interface{}
		mockReturn     *models.LogicalSwitch
		mockError      error
		expectedStatus int
	}{
		{
			name: "successful create",
			requestBody: map[string]interface{}{
				"name": "test-switch",
				"other_config": map[string]string{
					"subnet": "192.168.1.0/24",
				},
			},
			mockReturn: &models.LogicalSwitch{
				UUID:      "new-uuid",
				Name:      "test-switch",
				CreatedAt: time.Now(),
				UpdatedAt: time.Now(),
			},
			mockError:      nil,
			expectedStatus: http.StatusCreated,
		},
		{
			name: "missing name",
			requestBody: map[string]interface{}{
				"other_config": map[string]string{
					"subnet": "192.168.1.0/24",
				},
			},
			mockReturn:     nil,
			mockError:      nil,
			expectedStatus: http.StatusBadRequest,
		},
		{
			name: "invalid name",
			requestBody: map[string]interface{}{
				"name": "test switch", // Contains space
			},
			mockReturn:     nil,
			mockError:      nil,
			expectedStatus: http.StatusBadRequest,
		},
		{
			name: "already exists",
			requestBody: map[string]interface{}{
				"name": "existing-switch",
			},
			mockReturn:     nil,
			mockError:      errors.New("switch already exists"),
			expectedStatus: http.StatusConflict,
		},
		{
			name:           "invalid json",
			requestBody:    "invalid json",
			mockReturn:     nil,
			mockError:      nil,
			expectedStatus: http.StatusBadRequest,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockService := new(MockOVNService)
			handler := NewSwitchHandler(mockService)

			body, _ := json.Marshal(tt.requestBody)
			
			// Only set up mock if we expect the service to be called
			if tt.expectedStatus == http.StatusCreated || tt.expectedStatus == http.StatusConflict {
				mockService.On("CreateLogicalSwitch", mock.Anything, mock.Anything).Return(tt.mockReturn, tt.mockError)
			}

			w := httptest.NewRecorder()
			c, _ := gin.CreateTestContext(w)
			c.Request = httptest.NewRequest("POST", "/api/v1/switches", bytes.NewReader(body))
			c.Request.Header.Set("Content-Type", "application/json")

			handler.Create(c)

			assert.Equal(t, tt.expectedStatus, w.Code)

			if tt.expectedStatus == http.StatusCreated || tt.expectedStatus == http.StatusConflict {
				mockService.AssertExpectations(t)
			}
		})
	}
}

func TestSwitchHandler_Update(t *testing.T) {
	gin.SetMode(gin.TestMode)

	tests := []struct {
		name           string
		switchID       string
		requestBody    interface{}
		mockReturn     *models.LogicalSwitch
		mockError      error
		expectedStatus int
	}{
		{
			name:     "successful update",
			switchID: "uuid1",
			requestBody: map[string]interface{}{
				"name": "updated-switch",
			},
			mockReturn: &models.LogicalSwitch{
				UUID:      "uuid1",
				Name:      "updated-switch",
				CreatedAt: time.Now(),
				UpdatedAt: time.Now(),
			},
			mockError:      nil,
			expectedStatus: http.StatusOK,
		},
		{
			name:     "not found",
			switchID: "nonexistent",
			requestBody: map[string]interface{}{
				"name": "updated-switch",
			},
			mockReturn:     nil,
			mockError:      errors.New("logical switch nonexistent not found"),
			expectedStatus: http.StatusNotFound,
		},
		{
			name:     "empty id",
			switchID: "",
			requestBody: map[string]interface{}{
				"name": "updated-switch",
			},
			mockReturn:     nil,
			mockError:      nil,
			expectedStatus: http.StatusBadRequest,
		},
		{
			name:     "invalid name",
			switchID: "uuid1",
			requestBody: map[string]interface{}{
				"name": "invalid name", // Contains space
			},
			mockReturn:     nil,
			mockError:      nil,
			expectedStatus: http.StatusBadRequest,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockService := new(MockOVNService)
			handler := NewSwitchHandler(mockService)

			body, _ := json.Marshal(tt.requestBody)

			// Only set up mock if we expect the service to be called
			if tt.expectedStatus == http.StatusOK || tt.expectedStatus == http.StatusNotFound {
				mockService.On("UpdateLogicalSwitch", mock.Anything, tt.switchID, mock.Anything).Return(tt.mockReturn, tt.mockError)
			}

			w := httptest.NewRecorder()
			c, _ := gin.CreateTestContext(w)
			c.Request = httptest.NewRequest("PUT", "/api/v1/switches/"+tt.switchID, bytes.NewReader(body))
			c.Request.Header.Set("Content-Type", "application/json")
			c.Params = gin.Params{{Key: "id", Value: tt.switchID}}

			handler.Update(c)

			assert.Equal(t, tt.expectedStatus, w.Code)

			if tt.expectedStatus == http.StatusOK || tt.expectedStatus == http.StatusNotFound {
				mockService.AssertExpectations(t)
			}
		})
	}
}

func TestSwitchHandler_Delete(t *testing.T) {
	gin.SetMode(gin.TestMode)

	tests := []struct {
		name           string
		switchID       string
		mockError      error
		expectedStatus int
	}{
		{
			name:           "successful delete",
			switchID:       "uuid1",
			mockError:      nil,
			expectedStatus: http.StatusNoContent,
		},
		{
			name:           "not found",
			switchID:       "nonexistent",
			mockError:      errors.New("logical switch nonexistent not found"),
			expectedStatus: http.StatusNotFound,
		},
		{
			name:           "in use",
			switchID:       "uuid1",
			mockError:      errors.New("switch in use"),
			expectedStatus: http.StatusConflict,
		},
		{
			name:           "empty id",
			switchID:       "",
			mockError:      nil,
			expectedStatus: http.StatusBadRequest,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockService := new(MockOVNService)
			handler := NewSwitchHandler(mockService)

			if tt.switchID != "" {
				mockService.On("DeleteLogicalSwitch", mock.Anything, tt.switchID).Return(tt.mockError)
			}

			w := httptest.NewRecorder()
			c, _ := gin.CreateTestContext(w)
			c.Request = httptest.NewRequest("DELETE", "/api/v1/switches/"+tt.switchID, nil)
			c.Params = gin.Params{{Key: "id", Value: tt.switchID}}

			handler.Delete(c)

			assert.Equal(t, tt.expectedStatus, w.Code)

			if tt.switchID != "" {
				mockService.AssertExpectations(t)
			}
		})
	}
}