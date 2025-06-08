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

func TestACLHandler_List(t *testing.T) {
	gin.SetMode(gin.TestMode)

	tests := []struct {
		name           string
		switchID       string
		queryParams    map[string]string
		mockReturn     []*models.ACL
		mockError      error
		expectedStatus int
		expectedBody   map[string]interface{}
		checkPagination bool
	}{
		{
			name:     "successful list",
			switchID: "switch-uuid",
			queryParams: map[string]string{
				"switch_id": "switch-uuid",
			},
			mockReturn: []*models.ACL{
				{
					UUID:      "acl1",
					Priority:  100,
					Direction: "from-lport",
					Match:     "ip4.src == 10.0.0.1",
					Action:    "allow",
					CreatedAt: time.Now(),
					UpdatedAt: time.Now(),
				},
				{
					UUID:      "acl2",
					Priority:  200,
					Direction: "to-lport",
					Match:     "ip4.dst == 10.0.0.2",
					Action:    "drop",
					CreatedAt: time.Now(),
					UpdatedAt: time.Now(),
				},
			},
			mockError:      nil,
			expectedStatus: http.StatusOK,
			checkPagination: true,
		},
		{
			name:     "with pagination",
			switchID: "switch-uuid",
			queryParams: map[string]string{
				"switch_id": "switch-uuid",
				"page":      "2",
				"limit":     "1",
			},
			mockReturn: []*models.ACL{
				{
					UUID:      "acl1",
					Priority:  100,
					Direction: "from-lport",
					Match:     "ip4.src == 10.0.0.1",
					Action:    "allow",
				},
				{
					UUID:      "acl2",
					Priority:  200,
					Direction: "to-lport",
					Match:     "ip4.dst == 10.0.0.2",
					Action:    "drop",
				},
			},
			mockError:      nil,
			expectedStatus: http.StatusOK,
			checkPagination: true,
		},
		{
			name:     "empty list",
			switchID: "switch-uuid",
			queryParams: map[string]string{
				"switch_id": "switch-uuid",
			},
			mockReturn:     []*models.ACL{},
			mockError:      nil,
			expectedStatus: http.StatusOK,
			checkPagination: true,
		},
		{
			name:     "switch not found",
			switchID: "nonexistent",
			queryParams: map[string]string{
				"switch_id": "nonexistent",
			},
			mockReturn:     nil,
			mockError:      errors.New("switch not found"),
			expectedStatus: http.StatusNotFound,
			expectedBody: map[string]interface{}{
				"error": "switch not found",
			},
		},
		{
			name:           "missing switch_id",
			switchID:       "",
			queryParams:    map[string]string{},
			mockReturn:     nil,
			mockError:      nil,
			expectedStatus: http.StatusBadRequest,
			expectedBody: map[string]interface{}{
				"error": "switch_id query parameter is required",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockService := new(MockOVNService)
			handler := NewACLHandler(mockService)

			if tt.switchID != "" {
				mockService.On("ListACLs", mock.Anything, tt.switchID).Return(tt.mockReturn, tt.mockError)
			}

			w := httptest.NewRecorder()
			c, _ := gin.CreateTestContext(w)
			
			// Build URL with query params
			url := "/api/v1/acls"
			if len(tt.queryParams) > 0 {
				url += "?"
				first := true
				for k, v := range tt.queryParams {
					if !first {
						url += "&"
					}
					url += k + "=" + v
					first = false
				}
			}
			
			c.Request = httptest.NewRequest("GET", url, nil)

			handler.List(c)

			assert.Equal(t, tt.expectedStatus, w.Code)

			var response map[string]interface{}
			err := json.Unmarshal(w.Body.Bytes(), &response)
			assert.NoError(t, err)

			if tt.checkPagination {
				assert.Contains(t, response, "pagination")
			}

			for key, value := range tt.expectedBody {
				assert.Equal(t, value, response[key])
			}

			if tt.switchID != "" {
				mockService.AssertExpectations(t)
			}
		})
	}
}

func TestACLHandler_Get(t *testing.T) {
	gin.SetMode(gin.TestMode)

	tests := []struct {
		name           string
		aclID          string
		mockReturn     *models.ACL
		mockError      error
		expectedStatus int
	}{
		{
			name:  "successful get",
			aclID: "acl-uuid",
			mockReturn: &models.ACL{
				UUID:      "acl-uuid",
				Priority:  100,
				Direction: "from-lport",
				Match:     "ip4.src == 10.0.0.1",
				Action:    "allow",
				CreatedAt: time.Now(),
				UpdatedAt: time.Now(),
			},
			mockError:      nil,
			expectedStatus: http.StatusOK,
		},
		{
			name:           "not found",
			aclID:          "nonexistent",
			mockReturn:     nil,
			mockError:      errors.New("ACL nonexistent not found"),
			expectedStatus: http.StatusNotFound,
		},
		{
			name:           "empty id",
			aclID:          "",
			mockReturn:     nil,
			mockError:      nil,
			expectedStatus: http.StatusBadRequest,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockService := new(MockOVNService)
			handler := NewACLHandler(mockService)

			if tt.aclID != "" {
				mockService.On("GetACL", mock.Anything, tt.aclID).Return(tt.mockReturn, tt.mockError)
			}

			w := httptest.NewRecorder()
			c, _ := gin.CreateTestContext(w)
			c.Request = httptest.NewRequest("GET", "/api/v1/acls/"+tt.aclID, nil)
			c.Params = gin.Params{{Key: "id", Value: tt.aclID}}

			handler.Get(c)

			assert.Equal(t, tt.expectedStatus, w.Code)

			if tt.aclID != "" {
				mockService.AssertExpectations(t)
			}
		})
	}
}

func TestACLHandler_Create(t *testing.T) {
	gin.SetMode(gin.TestMode)

	tests := []struct {
		name           string
		switchID       string
		requestBody    interface{}
		mockReturn     *models.ACL
		mockError      error
		expectedStatus int
	}{
		{
			name:     "successful create",
			switchID: "switch-uuid",
			requestBody: map[string]interface{}{
				"priority":  100,
				"direction": "from-lport",
				"match":     "ip4.src == 10.0.0.1",
				"action":    "allow",
			},
			mockReturn: &models.ACL{
				UUID:      "new-uuid",
				Priority:  100,
				Direction: "from-lport",
				Match:     "ip4.src == 10.0.0.1",
				Action:    "allow",
				CreatedAt: time.Now(),
				UpdatedAt: time.Now(),
			},
			mockError:      nil,
			expectedStatus: http.StatusCreated,
		},
		{
			name:     "with optional fields",
			switchID: "switch-uuid",
			requestBody: map[string]interface{}{
				"name":      "test-acl",
				"priority":  100,
				"direction": "from-lport",
				"match":     "ip4.src == 10.0.0.1",
				"action":    "allow",
				"log":       true,
				"severity":  "info",
			},
			mockReturn: &models.ACL{
				UUID:      "new-uuid",
				Name:      "test-acl",
				Priority:  100,
				Direction: "from-lport",
				Match:     "ip4.src == 10.0.0.1",
				Action:    "allow",
				Log:       true,
				Severity:  "info",
				CreatedAt: time.Now(),
				UpdatedAt: time.Now(),
			},
			mockError:      nil,
			expectedStatus: http.StatusCreated,
		},
		{
			name:     "missing match",
			switchID: "switch-uuid",
			requestBody: map[string]interface{}{
				"priority":  100,
				"direction": "from-lport",
				"action":    "allow",
			},
			mockReturn:     nil,
			mockError:      nil,
			expectedStatus: http.StatusBadRequest,
		},
		{
			name:     "missing action",
			switchID: "switch-uuid",
			requestBody: map[string]interface{}{
				"priority":  100,
				"direction": "from-lport",
				"match":     "ip4.src == 10.0.0.1",
			},
			mockReturn:     nil,
			mockError:      nil,
			expectedStatus: http.StatusBadRequest,
		},
		{
			name:     "missing direction",
			switchID: "switch-uuid",
			requestBody: map[string]interface{}{
				"priority": 100,
				"match":    "ip4.src == 10.0.0.1",
				"action":   "allow",
			},
			mockReturn:     nil,
			mockError:      nil,
			expectedStatus: http.StatusBadRequest,
		},
		{
			name:     "invalid action",
			switchID: "switch-uuid",
			requestBody: map[string]interface{}{
				"priority":  100,
				"direction": "from-lport",
				"match":     "ip4.src == 10.0.0.1",
				"action":    "invalid-action",
			},
			mockReturn:     nil,
			mockError:      nil,
			expectedStatus: http.StatusBadRequest,
		},
		{
			name:     "invalid direction",
			switchID: "switch-uuid",
			requestBody: map[string]interface{}{
				"priority":  100,
				"direction": "invalid-direction",
				"match":     "ip4.src == 10.0.0.1",
				"action":    "allow",
			},
			mockReturn:     nil,
			mockError:      nil,
			expectedStatus: http.StatusBadRequest,
		},
		{
			name:     "invalid priority - negative",
			switchID: "switch-uuid",
			requestBody: map[string]interface{}{
				"priority":  -1,
				"direction": "from-lport",
				"match":     "ip4.src == 10.0.0.1",
				"action":    "allow",
			},
			mockReturn:     nil,
			mockError:      nil,
			expectedStatus: http.StatusBadRequest,
		},
		{
			name:     "invalid priority - too high",
			switchID: "switch-uuid",
			requestBody: map[string]interface{}{
				"priority":  65536,
				"direction": "from-lport",
				"match":     "ip4.src == 10.0.0.1",
				"action":    "allow",
			},
			mockReturn:     nil,
			mockError:      nil,
			expectedStatus: http.StatusBadRequest,
		},
		{
			name:     "invalid severity",
			switchID: "switch-uuid",
			requestBody: map[string]interface{}{
				"priority":  100,
				"direction": "from-lport",
				"match":     "ip4.src == 10.0.0.1",
				"action":    "allow",
				"severity":  "invalid-severity",
			},
			mockReturn:     nil,
			mockError:      nil,
			expectedStatus: http.StatusBadRequest,
		},
		{
			name:     "switch not found",
			switchID: "nonexistent",
			requestBody: map[string]interface{}{
				"priority":  100,
				"direction": "from-lport",
				"match":     "ip4.src == 10.0.0.1",
				"action":    "allow",
			},
			mockReturn:     nil,
			mockError:      errors.New("switch not found"),
			expectedStatus: http.StatusNotFound,
		},
		{
			name:     "missing switch_id",
			switchID: "",
			requestBody: map[string]interface{}{
				"priority":  100,
				"direction": "from-lport",
				"match":     "ip4.src == 10.0.0.1",
				"action":    "allow",
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
			handler := NewACLHandler(mockService)

			body, _ := json.Marshal(tt.requestBody)
			
			// Only set up mock if we expect the service to be called
			if tt.expectedStatus == http.StatusCreated || 
			   (tt.expectedStatus == http.StatusNotFound && tt.switchID != "") {
				mockService.On("CreateACL", mock.Anything, tt.switchID, mock.Anything).Return(tt.mockReturn, tt.mockError)
			}

			w := httptest.NewRecorder()
			c, _ := gin.CreateTestContext(w)
			
			url := "/api/v1/acls"
			if tt.switchID != "" {
				url += "?switch_id=" + tt.switchID
			}
			
			c.Request = httptest.NewRequest("POST", url, bytes.NewReader(body))
			c.Request.Header.Set("Content-Type", "application/json")

			handler.Create(c)

			assert.Equal(t, tt.expectedStatus, w.Code)

			if tt.expectedStatus == http.StatusCreated || 
			   (tt.expectedStatus == http.StatusNotFound && tt.switchID != "") {
				mockService.AssertExpectations(t)
			}
		})
	}
}

func TestACLHandler_Update(t *testing.T) {
	gin.SetMode(gin.TestMode)

	tests := []struct {
		name           string
		aclID          string
		requestBody    interface{}
		mockReturn     *models.ACL
		mockError      error
		expectedStatus int
	}{
		{
			name:  "successful update",
			aclID: "acl-uuid",
			requestBody: map[string]interface{}{
				"priority": 200,
				"action":   "drop",
			},
			mockReturn: &models.ACL{
				UUID:      "acl-uuid",
				Priority:  200,
				Direction: "from-lport",
				Match:     "ip4.src == 10.0.0.1",
				Action:    "drop",
				CreatedAt: time.Now(),
				UpdatedAt: time.Now(),
			},
			mockError:      nil,
			expectedStatus: http.StatusOK,
		},
		{
			name:  "not found",
			aclID: "nonexistent",
			requestBody: map[string]interface{}{
				"priority": 200,
			},
			mockReturn:     nil,
			mockError:      errors.New("ACL nonexistent not found"),
			expectedStatus: http.StatusNotFound,
		},
		{
			name:  "empty id",
			aclID: "",
			requestBody: map[string]interface{}{
				"priority": 200,
			},
			mockReturn:     nil,
			mockError:      nil,
			expectedStatus: http.StatusBadRequest,
		},
		{
			name:  "invalid action",
			aclID: "acl-uuid",
			requestBody: map[string]interface{}{
				"action": "invalid-action",
			},
			mockReturn:     nil,
			mockError:      nil,
			expectedStatus: http.StatusBadRequest,
		},
		{
			name:  "invalid direction",
			aclID: "acl-uuid",
			requestBody: map[string]interface{}{
				"direction": "invalid-direction",
			},
			mockReturn:     nil,
			mockError:      nil,
			expectedStatus: http.StatusBadRequest,
		},
		{
			name:  "invalid priority",
			aclID: "acl-uuid",
			requestBody: map[string]interface{}{
				"priority": 65536,
			},
			mockReturn:     nil,
			mockError:      nil,
			expectedStatus: http.StatusBadRequest,
		},
		{
			name:  "invalid severity",
			aclID: "acl-uuid",
			requestBody: map[string]interface{}{
				"severity": "invalid-severity",
			},
			mockReturn:     nil,
			mockError:      nil,
			expectedStatus: http.StatusBadRequest,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockService := new(MockOVNService)
			handler := NewACLHandler(mockService)

			body, _ := json.Marshal(tt.requestBody)

			// Only set up mock if we expect the service to be called
			if tt.expectedStatus == http.StatusOK || tt.expectedStatus == http.StatusNotFound {
				mockService.On("UpdateACL", mock.Anything, tt.aclID, mock.Anything).Return(tt.mockReturn, tt.mockError)
			}

			w := httptest.NewRecorder()
			c, _ := gin.CreateTestContext(w)
			c.Request = httptest.NewRequest("PUT", "/api/v1/acls/"+tt.aclID, bytes.NewReader(body))
			c.Request.Header.Set("Content-Type", "application/json")
			c.Params = gin.Params{{Key: "id", Value: tt.aclID}}

			handler.Update(c)

			assert.Equal(t, tt.expectedStatus, w.Code)

			if tt.expectedStatus == http.StatusOK || tt.expectedStatus == http.StatusNotFound {
				mockService.AssertExpectations(t)
			}
		})
	}
}

func TestACLHandler_Delete(t *testing.T) {
	gin.SetMode(gin.TestMode)

	tests := []struct {
		name           string
		aclID          string
		mockError      error
		expectedStatus int
	}{
		{
			name:           "successful delete",
			aclID:          "acl-uuid",
			mockError:      nil,
			expectedStatus: http.StatusNoContent,
		},
		{
			name:           "not found",
			aclID:          "nonexistent",
			mockError:      errors.New("ACL nonexistent not found"),
			expectedStatus: http.StatusNotFound,
		},
		{
			name:           "empty id",
			aclID:          "",
			mockError:      nil,
			expectedStatus: http.StatusBadRequest,
		},
		{
			name:           "service error",
			aclID:          "acl-uuid",
			mockError:      errors.New("internal error"),
			expectedStatus: http.StatusInternalServerError,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockService := new(MockOVNService)
			handler := NewACLHandler(mockService)

			if tt.aclID != "" {
				mockService.On("DeleteACL", mock.Anything, tt.aclID).Return(tt.mockError)
			}

			w := httptest.NewRecorder()
			c, _ := gin.CreateTestContext(w)
			c.Request = httptest.NewRequest("DELETE", "/api/v1/acls/"+tt.aclID, nil)
			c.Params = gin.Params{{Key: "id", Value: tt.aclID}}

			handler.Delete(c)

			assert.Equal(t, tt.expectedStatus, w.Code)

			if tt.aclID != "" {
				mockService.AssertExpectations(t)
			}
		})
	}
}