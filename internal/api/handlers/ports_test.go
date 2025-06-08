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

func TestPortHandler_List(t *testing.T) {
	gin.SetMode(gin.TestMode)

	tests := []struct {
		name           string
		switchID       string
		mockReturn     []*models.LogicalSwitchPort
		mockError      error
		expectedStatus int
		expectedBody   map[string]interface{}
	}{
		{
			name:     "successful list",
			switchID: "switch-uuid",
			mockReturn: []*models.LogicalSwitchPort{
				{
					UUID:      "port1",
					Name:      "port1",
					Addresses: []string{"00:00:00:00:00:01"},
					CreatedAt: time.Now(),
					UpdatedAt: time.Now(),
				},
				{
					UUID:      "port2",
					Name:      "port2",
					Addresses: []string{"00:00:00:00:00:02"},
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
			switchID:       "switch-uuid",
			mockReturn:     []*models.LogicalSwitchPort{},
			mockError:      nil,
			expectedStatus: http.StatusOK,
			expectedBody: map[string]interface{}{
				"count": float64(0),
			},
		},
		{
			name:           "switch not found",
			switchID:       "nonexistent",
			mockReturn:     nil,
			mockError:      errors.New("switch not found"),
			expectedStatus: http.StatusNotFound,
			expectedBody: map[string]interface{}{
				"error": "switch not found",
			},
		},
		{
			name:           "empty switch ID",
			switchID:       "",
			mockReturn:     nil,
			mockError:      nil,
			expectedStatus: http.StatusBadRequest,
			expectedBody: map[string]interface{}{
				"error": "switch ID is required",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockService := new(MockOVNService)
			handler := NewPortHandler(mockService)

			if tt.switchID != "" {
				mockService.On("ListPorts", mock.Anything, tt.switchID).Return(tt.mockReturn, tt.mockError)
			}

			w := httptest.NewRecorder()
			c, _ := gin.CreateTestContext(w)
			c.Request = httptest.NewRequest("GET", "/api/v1/switches/"+tt.switchID+"/ports", nil)
			c.Params = gin.Params{{Key: "switchId", Value: tt.switchID}}

			handler.List(c)

			assert.Equal(t, tt.expectedStatus, w.Code)

			var response map[string]interface{}
			err := json.Unmarshal(w.Body.Bytes(), &response)
			assert.NoError(t, err)

			for key, value := range tt.expectedBody {
				assert.Equal(t, value, response[key])
			}

			if tt.switchID != "" {
				mockService.AssertExpectations(t)
			}
		})
	}
}

func TestPortHandler_Get(t *testing.T) {
	gin.SetMode(gin.TestMode)

	tests := []struct {
		name           string
		portID         string
		mockReturn     *models.LogicalSwitchPort
		mockError      error
		expectedStatus int
	}{
		{
			name:   "successful get",
			portID: "port-uuid",
			mockReturn: &models.LogicalSwitchPort{
				UUID:      "port-uuid",
				Name:      "test-port",
				Addresses: []string{"00:00:00:00:00:01"},
				CreatedAt: time.Now(),
				UpdatedAt: time.Now(),
			},
			mockError:      nil,
			expectedStatus: http.StatusOK,
		},
		{
			name:           "not found",
			portID:         "nonexistent",
			mockReturn:     nil,
			mockError:      errors.New("logical switch port nonexistent not found"),
			expectedStatus: http.StatusNotFound,
		},
		{
			name:           "empty id",
			portID:         "",
			mockReturn:     nil,
			mockError:      nil,
			expectedStatus: http.StatusBadRequest,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockService := new(MockOVNService)
			handler := NewPortHandler(mockService)

			if tt.portID != "" {
				mockService.On("GetPort", mock.Anything, tt.portID).Return(tt.mockReturn, tt.mockError)
			}

			w := httptest.NewRecorder()
			c, _ := gin.CreateTestContext(w)
			c.Request = httptest.NewRequest("GET", "/api/v1/ports/"+tt.portID, nil)
			c.Params = gin.Params{{Key: "id", Value: tt.portID}}

			handler.Get(c)

			assert.Equal(t, tt.expectedStatus, w.Code)

			if tt.portID != "" {
				mockService.AssertExpectations(t)
			}
		})
	}
}

func TestPortHandler_Create(t *testing.T) {
	gin.SetMode(gin.TestMode)

	tests := []struct {
		name           string
		switchID       string
		requestBody    interface{}
		mockReturn     *models.LogicalSwitchPort
		mockError      error
		expectedStatus int
	}{
		{
			name:     "successful create",
			switchID: "switch-uuid",
			requestBody: map[string]interface{}{
				"name":      "test-port",
				"addresses": []string{"00:00:00:00:00:01"},
			},
			mockReturn: &models.LogicalSwitchPort{
				UUID:      "new-uuid",
				Name:      "test-port",
				Addresses: []string{"00:00:00:00:00:01"},
				CreatedAt: time.Now(),
				UpdatedAt: time.Now(),
			},
			mockError:      nil,
			expectedStatus: http.StatusCreated,
		},
		{
			name:     "missing name",
			switchID: "switch-uuid",
			requestBody: map[string]interface{}{
				"addresses": []string{"00:00:00:00:00:01"},
			},
			mockReturn:     nil,
			mockError:      nil,
			expectedStatus: http.StatusBadRequest,
		},
		{
			name:     "invalid name",
			switchID: "switch-uuid",
			requestBody: map[string]interface{}{
				"name":      "test port", // Contains space
				"addresses": []string{"00:00:00:00:00:01"},
			},
			mockReturn:     nil,
			mockError:      nil,
			expectedStatus: http.StatusBadRequest,
		},
		{
			name:     "missing addresses",
			switchID: "switch-uuid",
			requestBody: map[string]interface{}{
				"name": "test-port",
			},
			mockReturn:     nil,
			mockError:      nil,
			expectedStatus: http.StatusBadRequest,
		},
		{
			name:     "invalid address format",
			switchID: "switch-uuid",
			requestBody: map[string]interface{}{
				"name":      "test-port",
				"addresses": []string{"invalid-mac"},
			},
			mockReturn:     nil,
			mockError:      nil,
			expectedStatus: http.StatusBadRequest,
		},
		{
			name:     "invalid port type",
			switchID: "switch-uuid",
			requestBody: map[string]interface{}{
				"name":      "test-port",
				"addresses": []string{"00:00:00:00:00:01"},
				"type":      "invalid-type",
			},
			mockReturn:     nil,
			mockError:      nil,
			expectedStatus: http.StatusBadRequest,
		},
		{
			name:     "port already exists",
			switchID: "switch-uuid",
			requestBody: map[string]interface{}{
				"name":      "existing-port",
				"addresses": []string{"00:00:00:00:00:01"},
			},
			mockReturn:     nil,
			mockError:      errors.New("port existing-port already exists"),
			expectedStatus: http.StatusConflict,
		},
		{
			name:     "switch not found",
			switchID: "nonexistent",
			requestBody: map[string]interface{}{
				"name":      "test-port",
				"addresses": []string{"00:00:00:00:00:01"},
			},
			mockReturn:     nil,
			mockError:      errors.New("switch not found"),
			expectedStatus: http.StatusNotFound,
		},
		{
			name:     "empty switch ID",
			switchID: "",
			requestBody: map[string]interface{}{
				"name":      "test-port",
				"addresses": []string{"00:00:00:00:00:01"},
			},
			mockReturn:     nil,
			mockError:      nil,
			expectedStatus: http.StatusBadRequest,
		},
		{
			name:           "invalid json",
			switchID:       "switch-uuid",
			requestBody:    "invalid json",
			mockReturn:     nil,
			mockError:      nil,
			expectedStatus: http.StatusBadRequest,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockService := new(MockOVNService)
			handler := NewPortHandler(mockService)

			body, _ := json.Marshal(tt.requestBody)
			
			// Only set up mock if we expect the service to be called
			if tt.expectedStatus == http.StatusCreated || tt.expectedStatus == http.StatusConflict || 
			   (tt.expectedStatus == http.StatusNotFound && tt.switchID != "") {
				mockService.On("CreatePort", mock.Anything, tt.switchID, mock.Anything).Return(tt.mockReturn, tt.mockError)
			}

			w := httptest.NewRecorder()
			c, _ := gin.CreateTestContext(w)
			c.Request = httptest.NewRequest("POST", "/api/v1/switches/"+tt.switchID+"/ports", bytes.NewReader(body))
			c.Request.Header.Set("Content-Type", "application/json")
			c.Params = gin.Params{{Key: "switchId", Value: tt.switchID}}

			handler.Create(c)

			assert.Equal(t, tt.expectedStatus, w.Code)

			if tt.expectedStatus == http.StatusCreated || tt.expectedStatus == http.StatusConflict || 
			   (tt.expectedStatus == http.StatusNotFound && tt.switchID != "") {
				mockService.AssertExpectations(t)
			}
		})
	}
}

func TestPortHandler_Update(t *testing.T) {
	gin.SetMode(gin.TestMode)

	tests := []struct {
		name           string
		portID         string
		requestBody    interface{}
		mockReturn     *models.LogicalSwitchPort
		mockError      error
		expectedStatus int
	}{
		{
			name:   "successful update",
			portID: "port-uuid",
			requestBody: map[string]interface{}{
				"name":      "updated-port",
				"addresses": []string{"00:00:00:00:00:02"},
			},
			mockReturn: &models.LogicalSwitchPort{
				UUID:      "port-uuid",
				Name:      "updated-port",
				Addresses: []string{"00:00:00:00:00:02"},
				CreatedAt: time.Now(),
				UpdatedAt: time.Now(),
			},
			mockError:      nil,
			expectedStatus: http.StatusOK,
		},
		{
			name:   "not found",
			portID: "nonexistent",
			requestBody: map[string]interface{}{
				"name": "updated-port",
			},
			mockReturn:     nil,
			mockError:      errors.New("logical switch port nonexistent not found"),
			expectedStatus: http.StatusNotFound,
		},
		{
			name:   "empty id",
			portID: "",
			requestBody: map[string]interface{}{
				"name": "updated-port",
			},
			mockReturn:     nil,
			mockError:      nil,
			expectedStatus: http.StatusBadRequest,
		},
		{
			name:   "invalid name",
			portID: "port-uuid",
			requestBody: map[string]interface{}{
				"name": "invalid name", // Contains space
			},
			mockReturn:     nil,
			mockError:      nil,
			expectedStatus: http.StatusBadRequest,
		},
		{
			name:   "invalid address",
			portID: "port-uuid",
			requestBody: map[string]interface{}{
				"addresses": []string{"invalid-mac"},
			},
			mockReturn:     nil,
			mockError:      nil,
			expectedStatus: http.StatusBadRequest,
		},
		{
			name:   "invalid port type",
			portID: "port-uuid",
			requestBody: map[string]interface{}{
				"type": "invalid-type",
			},
			mockReturn:     nil,
			mockError:      nil,
			expectedStatus: http.StatusBadRequest,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockService := new(MockOVNService)
			handler := NewPortHandler(mockService)

			body, _ := json.Marshal(tt.requestBody)

			// Only set up mock if we expect the service to be called
			if tt.expectedStatus == http.StatusOK || tt.expectedStatus == http.StatusNotFound {
				mockService.On("UpdatePort", mock.Anything, tt.portID, mock.Anything).Return(tt.mockReturn, tt.mockError)
			}

			w := httptest.NewRecorder()
			c, _ := gin.CreateTestContext(w)
			c.Request = httptest.NewRequest("PUT", "/api/v1/ports/"+tt.portID, bytes.NewReader(body))
			c.Request.Header.Set("Content-Type", "application/json")
			c.Params = gin.Params{{Key: "id", Value: tt.portID}}

			handler.Update(c)

			assert.Equal(t, tt.expectedStatus, w.Code)

			if tt.expectedStatus == http.StatusOK || tt.expectedStatus == http.StatusNotFound {
				mockService.AssertExpectations(t)
			}
		})
	}
}

func TestPortHandler_Delete(t *testing.T) {
	gin.SetMode(gin.TestMode)

	tests := []struct {
		name           string
		portID         string
		mockError      error
		expectedStatus int
	}{
		{
			name:           "successful delete",
			portID:         "port-uuid",
			mockError:      nil,
			expectedStatus: http.StatusNoContent,
		},
		{
			name:           "not found",
			portID:         "nonexistent",
			mockError:      errors.New("logical switch port nonexistent not found"),
			expectedStatus: http.StatusNotFound,
		},
		{
			name:           "empty id",
			portID:         "",
			mockError:      nil,
			expectedStatus: http.StatusBadRequest,
		},
		{
			name:           "service error",
			portID:         "port-uuid",
			mockError:      errors.New("internal error"),
			expectedStatus: http.StatusInternalServerError,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockService := new(MockOVNService)
			handler := NewPortHandler(mockService)

			if tt.portID != "" {
				mockService.On("DeletePort", mock.Anything, tt.portID).Return(tt.mockError)
			}

			w := httptest.NewRecorder()
			c, _ := gin.CreateTestContext(w)
			c.Request = httptest.NewRequest("DELETE", "/api/v1/ports/"+tt.portID, nil)
			c.Params = gin.Params{{Key: "id", Value: tt.portID}}

			handler.Delete(c)

			assert.Equal(t, tt.expectedStatus, w.Code)

			if tt.portID != "" {
				mockService.AssertExpectations(t)
			}
		})
	}
}