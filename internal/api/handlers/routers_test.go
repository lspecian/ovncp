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

func TestRouterHandler_List(t *testing.T) {
	gin.SetMode(gin.TestMode)

	tests := []struct {
		name           string
		mockReturn     []*models.LogicalRouter
		mockError      error
		expectedStatus int
		expectedBody   map[string]interface{}
	}{
		{
			name: "successful list",
			mockReturn: []*models.LogicalRouter{
				{
					UUID:      "uuid1",
					Name:      "router1",
					CreatedAt: time.Now(),
					UpdatedAt: time.Now(),
				},
				{
					UUID:      "uuid2",
					Name:      "router2",
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
			mockReturn:     []*models.LogicalRouter{},
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
			handler := NewRouterHandler(mockService)

			mockService.On("ListLogicalRouters", mock.Anything).Return(tt.mockReturn, tt.mockError)

			w := httptest.NewRecorder()
			c, _ := gin.CreateTestContext(w)
			c.Request = httptest.NewRequest("GET", "/api/v1/routers", nil)

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

func TestRouterHandler_Get(t *testing.T) {
	gin.SetMode(gin.TestMode)

	tests := []struct {
		name           string
		routerID       string
		mockReturn     *models.LogicalRouter
		mockError      error
		expectedStatus int
	}{
		{
			name:     "successful get",
			routerID: "uuid1",
			mockReturn: &models.LogicalRouter{
				UUID:      "uuid1",
				Name:      "router1",
				CreatedAt: time.Now(),
				UpdatedAt: time.Now(),
			},
			mockError:      nil,
			expectedStatus: http.StatusOK,
		},
		{
			name:           "not found",
			routerID:       "nonexistent",
			mockReturn:     nil,
			mockError:      errors.New("logical router nonexistent not found"),
			expectedStatus: http.StatusNotFound,
		},
		{
			name:           "empty id",
			routerID:       "",
			mockReturn:     nil,
			mockError:      nil,
			expectedStatus: http.StatusBadRequest,
		},
		{
			name:           "service error",
			routerID:       "uuid1",
			mockReturn:     nil,
			mockError:      errors.New("service error"),
			expectedStatus: http.StatusInternalServerError,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockService := new(MockOVNService)
			handler := NewRouterHandler(mockService)

			if tt.routerID != "" {
				mockService.On("GetLogicalRouter", mock.Anything, tt.routerID).Return(tt.mockReturn, tt.mockError)
			}

			w := httptest.NewRecorder()
			c, _ := gin.CreateTestContext(w)
			c.Request = httptest.NewRequest("GET", "/api/v1/routers/"+tt.routerID, nil)
			c.Params = gin.Params{{Key: "id", Value: tt.routerID}}

			handler.Get(c)

			assert.Equal(t, tt.expectedStatus, w.Code)

			if tt.routerID != "" {
				mockService.AssertExpectations(t)
			}
		})
	}
}

func TestRouterHandler_Create(t *testing.T) {
	gin.SetMode(gin.TestMode)

	tests := []struct {
		name           string
		requestBody    interface{}
		mockReturn     *models.LogicalRouter
		mockError      error
		expectedStatus int
	}{
		{
			name: "successful create",
			requestBody: map[string]interface{}{
				"name": "test-router",
				"options": map[string]string{
					"router_preference": "high",
				},
			},
			mockReturn: &models.LogicalRouter{
				UUID:      "new-uuid",
				Name:      "test-router",
				CreatedAt: time.Now(),
				UpdatedAt: time.Now(),
			},
			mockError:      nil,
			expectedStatus: http.StatusCreated,
		},
		{
			name: "create with static routes",
			requestBody: map[string]interface{}{
				"name": "test-router",
				"static_routes": []map[string]interface{}{
					{
						"ip_prefix": "192.168.1.0/24",
						"nexthop":   "10.0.0.1",
					},
				},
			},
			mockReturn: &models.LogicalRouter{
				UUID:      "new-uuid",
				Name:      "test-router",
				CreatedAt: time.Now(),
				UpdatedAt: time.Now(),
			},
			mockError:      nil,
			expectedStatus: http.StatusCreated,
		},
		{
			name: "missing name",
			requestBody: map[string]interface{}{
				"options": map[string]string{
					"router_preference": "high",
				},
			},
			mockReturn:     nil,
			mockError:      nil,
			expectedStatus: http.StatusBadRequest,
		},
		{
			name: "invalid name",
			requestBody: map[string]interface{}{
				"name": "test router", // Contains space
			},
			mockReturn:     nil,
			mockError:      nil,
			expectedStatus: http.StatusBadRequest,
		},
		{
			name: "invalid static route - missing fields",
			requestBody: map[string]interface{}{
				"name": "test-router",
				"static_routes": []map[string]interface{}{
					{
						"ip_prefix": "192.168.1.0/24",
						// Missing nexthop
					},
				},
			},
			mockReturn:     nil,
			mockError:      nil,
			expectedStatus: http.StatusBadRequest,
		},
		{
			name: "invalid static route policy",
			requestBody: map[string]interface{}{
				"name": "test-router",
				"static_routes": []map[string]interface{}{
					{
						"ip_prefix": "192.168.1.0/24",
						"nexthop":   "10.0.0.1",
						"policy":    "invalid-policy",
					},
				},
			},
			mockReturn:     nil,
			mockError:      nil,
			expectedStatus: http.StatusBadRequest,
		},
		{
			name: "already exists",
			requestBody: map[string]interface{}{
				"name": "existing-router",
			},
			mockReturn:     nil,
			mockError:      errors.New("router already exists"),
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
			handler := NewRouterHandler(mockService)

			body, _ := json.Marshal(tt.requestBody)
			
			// Only set up mock if we expect the service to be called
			if tt.expectedStatus == http.StatusCreated || tt.expectedStatus == http.StatusConflict {
				mockService.On("CreateLogicalRouter", mock.Anything, mock.Anything).Return(tt.mockReturn, tt.mockError)
			}

			w := httptest.NewRecorder()
			c, _ := gin.CreateTestContext(w)
			c.Request = httptest.NewRequest("POST", "/api/v1/routers", bytes.NewReader(body))
			c.Request.Header.Set("Content-Type", "application/json")

			handler.Create(c)

			assert.Equal(t, tt.expectedStatus, w.Code)

			if tt.expectedStatus == http.StatusCreated || tt.expectedStatus == http.StatusConflict {
				mockService.AssertExpectations(t)
			}
		})
	}
}

func TestRouterHandler_Update(t *testing.T) {
	gin.SetMode(gin.TestMode)

	tests := []struct {
		name           string
		routerID       string
		requestBody    interface{}
		mockReturn     *models.LogicalRouter
		mockError      error
		expectedStatus int
	}{
		{
			name:     "successful update",
			routerID: "uuid1",
			requestBody: map[string]interface{}{
				"name": "updated-router",
			},
			mockReturn: &models.LogicalRouter{
				UUID:      "uuid1",
				Name:      "updated-router",
				CreatedAt: time.Now(),
				UpdatedAt: time.Now(),
			},
			mockError:      nil,
			expectedStatus: http.StatusOK,
		},
		{
			name:     "not found",
			routerID: "nonexistent",
			requestBody: map[string]interface{}{
				"name": "updated-router",
			},
			mockReturn:     nil,
			mockError:      errors.New("logical router nonexistent not found"),
			expectedStatus: http.StatusNotFound,
		},
		{
			name:     "empty id",
			routerID: "",
			requestBody: map[string]interface{}{
				"name": "updated-router",
			},
			mockReturn:     nil,
			mockError:      nil,
			expectedStatus: http.StatusBadRequest,
		},
		{
			name:     "invalid name",
			routerID: "uuid1",
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
			handler := NewRouterHandler(mockService)

			body, _ := json.Marshal(tt.requestBody)

			// Only set up mock if we expect the service to be called
			if tt.expectedStatus == http.StatusOK || tt.expectedStatus == http.StatusNotFound {
				mockService.On("UpdateLogicalRouter", mock.Anything, tt.routerID, mock.Anything).Return(tt.mockReturn, tt.mockError)
			}

			w := httptest.NewRecorder()
			c, _ := gin.CreateTestContext(w)
			c.Request = httptest.NewRequest("PUT", "/api/v1/routers/"+tt.routerID, bytes.NewReader(body))
			c.Request.Header.Set("Content-Type", "application/json")
			c.Params = gin.Params{{Key: "id", Value: tt.routerID}}

			handler.Update(c)

			assert.Equal(t, tt.expectedStatus, w.Code)

			if tt.expectedStatus == http.StatusOK || tt.expectedStatus == http.StatusNotFound {
				mockService.AssertExpectations(t)
			}
		})
	}
}

func TestRouterHandler_Delete(t *testing.T) {
	gin.SetMode(gin.TestMode)

	tests := []struct {
		name           string
		routerID       string
		mockError      error
		expectedStatus int
	}{
		{
			name:           "successful delete",
			routerID:       "uuid1",
			mockError:      nil,
			expectedStatus: http.StatusNoContent,
		},
		{
			name:           "not found",
			routerID:       "nonexistent",
			mockError:      errors.New("logical router nonexistent not found"),
			expectedStatus: http.StatusNotFound,
		},
		{
			name:           "has ports",
			routerID:       "uuid1",
			mockError:      errors.New("cannot delete router: router has 2 ports attached"),
			expectedStatus: http.StatusConflict,
		},
		{
			name:           "empty id",
			routerID:       "",
			mockError:      nil,
			expectedStatus: http.StatusBadRequest,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockService := new(MockOVNService)
			handler := NewRouterHandler(mockService)

			if tt.routerID != "" {
				mockService.On("DeleteLogicalRouter", mock.Anything, tt.routerID).Return(tt.mockError)
			}

			w := httptest.NewRecorder()
			c, _ := gin.CreateTestContext(w)
			c.Request = httptest.NewRequest("DELETE", "/api/v1/routers/"+tt.routerID, nil)
			c.Params = gin.Params{{Key: "id", Value: tt.routerID}}

			handler.Delete(c)

			assert.Equal(t, tt.expectedStatus, w.Code)

			if tt.routerID != "" {
				mockService.AssertExpectations(t)
			}
		})
	}
}