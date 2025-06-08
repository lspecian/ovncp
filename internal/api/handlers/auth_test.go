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
	
	"github.com/lspecian/ovncp/internal/api/middleware"
	"github.com/lspecian/ovncp/internal/models"
)

// Mock Auth Service (reusing from middleware tests)
type mockAuthService struct {
	mock.Mock
}

func (m *mockAuthService) GetAuthURL(provider string, state string) (string, error) {
	args := m.Called(provider, state)
	return args.String(0), args.Error(1)
}

func (m *mockAuthService) ExchangeCode(ctx context.Context, provider string, code string) (*models.Session, error) {
	args := m.Called(ctx, provider, code)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.Session), args.Error(1)
}

func (m *mockAuthService) ValidateToken(ctx context.Context, token string) (*models.User, error) {
	args := m.Called(ctx, token)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.User), args.Error(1)
}

func (m *mockAuthService) RefreshToken(ctx context.Context, refreshToken string) (*models.Session, error) {
	args := m.Called(ctx, refreshToken)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.Session), args.Error(1)
}

func (m *mockAuthService) Logout(ctx context.Context, token string) error {
	args := m.Called(ctx, token)
	return args.Error(0)
}

func (m *mockAuthService) GetUser(ctx context.Context, userID string) (*models.User, error) {
	args := m.Called(ctx, userID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.User), args.Error(1)
}

func (m *mockAuthService) UpdateUserRole(ctx context.Context, userID string, role models.UserRole) error {
	args := m.Called(ctx, userID, role)
	return args.Error(0)
}

func (m *mockAuthService) ListUsers(ctx context.Context, limit, offset int) ([]*models.User, int, error) {
	args := m.Called(ctx, limit, offset)
	if args.Get(0) == nil {
		return nil, args.Int(1), args.Error(2)
	}
	return args.Get(0).([]*models.User), args.Int(1), args.Error(2)
}

func (m *mockAuthService) DeactivateUser(ctx context.Context, userID string) error {
	args := m.Called(ctx, userID)
	return args.Error(0)
}

func TestLogin(t *testing.T) {
	gin.SetMode(gin.TestMode)
	
	tests := []struct {
		name           string
		requestBody    interface{}
		setupMock      func(*mockAuthService)
		expectedStatus int
		expectedBody   map[string]interface{}
	}{
		{
			name:           "Invalid request body",
			requestBody:    `{"invalid": "json"`,
			expectedStatus: http.StatusBadRequest,
			expectedBody: map[string]interface{}{
				"error": "unexpected EOF",
			},
		},
		{
			name:           "Missing provider",
			requestBody:    map[string]string{},
			expectedStatus: http.StatusBadRequest,
			expectedBody: map[string]interface{}{
				"error": "Key: 'LoginRequest.Provider' Error:Field validation for 'Provider' failed on the 'required' tag",
			},
		},
		{
			name: "Provider not found",
			requestBody: map[string]string{
				"provider": "nonexistent",
			},
			setupMock: func(m *mockAuthService) {
				m.On("GetAuthURL", "nonexistent", mock.AnythingOfType("string")).
					Return("", errors.New("provider nonexistent not found"))
			},
			expectedStatus: http.StatusBadRequest,
			expectedBody: map[string]interface{}{
				"error": "provider nonexistent not found",
			},
		},
		{
			name: "Success",
			requestBody: map[string]string{
				"provider": "github",
			},
			setupMock: func(m *mockAuthService) {
				m.On("GetAuthURL", "github", mock.AnythingOfType("string")).
					Return("https://github.com/login/oauth/authorize?client_id=123&state=abc", nil)
			},
			expectedStatus: http.StatusOK,
			expectedBody: map[string]interface{}{
				"auth_url": "https://github.com/login/oauth/authorize?client_id=123&state=abc",
			},
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockAuth := new(mockAuthService)
			if tt.setupMock != nil {
				tt.setupMock(mockAuth)
			}
			
			handler := NewAuthHandler(mockAuth)
			router := gin.New()
			router.POST("/auth/login", handler.Login)
			
			var body []byte
			if str, ok := tt.requestBody.(string); ok {
				body = []byte(str)
			} else {
				body, _ = json.Marshal(tt.requestBody)
			}
			
			w := httptest.NewRecorder()
			req, _ := http.NewRequest("POST", "/auth/login", bytes.NewBuffer(body))
			req.Header.Set("Content-Type", "application/json")
			
			router.ServeHTTP(w, req)
			
			assert.Equal(t, tt.expectedStatus, w.Code)
			
			var actualBody map[string]interface{}
			json.Unmarshal(w.Body.Bytes(), &actualBody)
			assert.Equal(t, tt.expectedBody, actualBody)
			
			mockAuth.AssertExpectations(t)
		})
	}
}

func TestCallback(t *testing.T) {
	gin.SetMode(gin.TestMode)
	
	now := time.Now()
	tests := []struct {
		name           string
		provider       string
		queryParams    string
		setupMock      func(*mockAuthService)
		expectedStatus int
		checkResponse  func(*testing.T, map[string]interface{})
	}{
		{
			name:           "Missing code",
			provider:       "github",
			queryParams:    "state=test-state",
			expectedStatus: http.StatusBadRequest,
			checkResponse: func(t *testing.T, resp map[string]interface{}) {
				assert.Contains(t, resp["error"], "required")
			},
		},
		{
			name:           "Missing state",
			provider:       "github",
			queryParams:    "code=test-code",
			expectedStatus: http.StatusBadRequest,
			checkResponse: func(t *testing.T, resp map[string]interface{}) {
				assert.Contains(t, resp["error"], "required")
			},
		},
		{
			name:        "Exchange code error",
			provider:    "github",
			queryParams: "code=invalid-code&state=test-state",
			setupMock: func(m *mockAuthService) {
				m.On("ExchangeCode", mock.Anything, "github", "invalid-code").
					Return(nil, errors.New("invalid authorization code"))
			},
			expectedStatus: http.StatusUnauthorized,
			checkResponse: func(t *testing.T, resp map[string]interface{}) {
				assert.Equal(t, "invalid authorization code", resp["error"])
			},
		},
		{
			name:        "Success",
			provider:    "github",
			queryParams: "code=valid-code&state=test-state",
			setupMock: func(m *mockAuthService) {
				m.On("ExchangeCode", mock.Anything, "github", "valid-code").
					Return(&models.Session{
						ID:           "session-123",
						UserID:       "user-123",
						AccessToken:  "access-token",
						RefreshToken: "refresh-token",
						ExpiresAt:    now.Add(24 * time.Hour),
						User: &models.User{
							ID:    "user-123",
							Email: "test@example.com",
							Role:  models.RoleViewer,
						},
					}, nil)
			},
			expectedStatus: http.StatusOK,
			checkResponse: func(t *testing.T, resp map[string]interface{}) {
				assert.Equal(t, "access-token", resp["access_token"])
				assert.Equal(t, "refresh-token", resp["refresh_token"])
				assert.NotNil(t, resp["expires_at"])
				assert.NotNil(t, resp["user"])
				user := resp["user"].(map[string]interface{})
				assert.Equal(t, "user-123", user["id"])
				assert.Equal(t, "test@example.com", user["email"])
			},
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockAuth := new(mockAuthService)
			if tt.setupMock != nil {
				tt.setupMock(mockAuth)
			}
			
			handler := NewAuthHandler(mockAuth)
			router := gin.New()
			router.GET("/auth/callback/:provider", handler.Callback)
			
			w := httptest.NewRecorder()
			req, _ := http.NewRequest("GET", "/auth/callback/"+tt.provider+"?"+tt.queryParams, nil)
			
			router.ServeHTTP(w, req)
			
			assert.Equal(t, tt.expectedStatus, w.Code)
			
			var resp map[string]interface{}
			json.Unmarshal(w.Body.Bytes(), &resp)
			tt.checkResponse(t, resp)
			
			mockAuth.AssertExpectations(t)
		})
	}
}

func TestUpdateUserRole(t *testing.T) {
	gin.SetMode(gin.TestMode)
	
	tests := []struct {
		name           string
		userID         string
		authUser       *models.User
		requestBody    interface{}
		setupMock      func(*mockAuthService)
		expectedStatus int
		expectedBody   map[string]interface{}
	}{
		{
			name:   "Invalid role",
			userID: "user-123",
			authUser: &models.User{
				ID:   "admin-123",
				Role: models.RoleAdmin,
			},
			requestBody: map[string]string{
				"role": "superadmin",
			},
			expectedStatus: http.StatusBadRequest,
			expectedBody: map[string]interface{}{
				"error": "Key: 'UpdateRoleRequest.Role' Error:Field validation for 'Role' failed on the 'oneof' tag",
			},
		},
		{
			name:   "Cannot change own role",
			userID: "admin-123",
			authUser: &models.User{
				ID:   "admin-123",
				Role: models.RoleAdmin,
			},
			requestBody: map[string]string{
				"role": "viewer",
			},
			expectedStatus: http.StatusBadRequest,
			expectedBody: map[string]interface{}{
				"error": "Cannot change your own role",
			},
		},
		{
			name:   "User not found",
			userID: "nonexistent",
			authUser: &models.User{
				ID:   "admin-123",
				Role: models.RoleAdmin,
			},
			requestBody: map[string]string{
				"role": "operator",
			},
			setupMock: func(m *mockAuthService) {
				m.On("UpdateUserRole", mock.Anything, "nonexistent", models.RoleOperator).
					Return(errors.New("user not found"))
			},
			expectedStatus: http.StatusInternalServerError,
			expectedBody: map[string]interface{}{
				"error": "user not found",
			},
		},
		{
			name:   "Success",
			userID: "user-123",
			authUser: &models.User{
				ID:   "admin-123",
				Role: models.RoleAdmin,
			},
			requestBody: map[string]string{
				"role": "operator",
			},
			setupMock: func(m *mockAuthService) {
				m.On("UpdateUserRole", mock.Anything, "user-123", models.RoleOperator).
					Return(nil)
			},
			expectedStatus: http.StatusOK,
			expectedBody: map[string]interface{}{
				"message": "Role updated successfully",
			},
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockAuth := new(mockAuthService)
			if tt.setupMock != nil {
				tt.setupMock(mockAuth)
			}
			
			handler := NewAuthHandler(mockAuth)
			router := gin.New()
			
			// Add middleware to set auth user
			router.Use(func(c *gin.Context) {
				c.Set(middleware.AuthUserKey, tt.authUser)
				c.Next()
			})
			
			router.PUT("/users/:id/role", handler.UpdateUserRole)
			
			body, _ := json.Marshal(tt.requestBody)
			w := httptest.NewRecorder()
			req, _ := http.NewRequest("PUT", "/users/"+tt.userID+"/role", bytes.NewBuffer(body))
			req.Header.Set("Content-Type", "application/json")
			
			router.ServeHTTP(w, req)
			
			assert.Equal(t, tt.expectedStatus, w.Code)
			
			var actualBody map[string]interface{}
			json.Unmarshal(w.Body.Bytes(), &actualBody)
			assert.Equal(t, tt.expectedBody, actualBody)
			
			mockAuth.AssertExpectations(t)
		})
	}
}

func TestListUsers(t *testing.T) {
	gin.SetMode(gin.TestMode)
	
	now := time.Now()
	users := []*models.User{
		{
			ID:        "user-1",
			Email:     "admin@example.com",
			Name:      "Admin User",
			Role:      models.RoleAdmin,
			Active:    true,
			CreatedAt: now,
		},
		{
			ID:        "user-2",
			Email:     "viewer@example.com",
			Name:      "Viewer User",
			Role:      models.RoleViewer,
			Active:    true,
			CreatedAt: now,
		},
	}
	
	tests := []struct {
		name           string
		queryParams    string
		setupMock      func(*mockAuthService)
		expectedStatus int
		checkResponse  func(*testing.T, map[string]interface{})
	}{
		{
			name:        "Default pagination",
			queryParams: "",
			setupMock: func(m *mockAuthService) {
				m.On("ListUsers", mock.Anything, 10, 0).
					Return(users, 2, nil)
			},
			expectedStatus: http.StatusOK,
			checkResponse: func(t *testing.T, resp map[string]interface{}) {
				assert.Equal(t, float64(2), resp["total"])
				assert.Equal(t, float64(10), resp["limit"])
				assert.Equal(t, float64(0), resp["offset"])
				usersList := resp["users"].([]interface{})
				assert.Len(t, usersList, 2)
			},
		},
		{
			name:        "Custom pagination",
			queryParams: "limit=5&offset=10",
			setupMock: func(m *mockAuthService) {
				m.On("ListUsers", mock.Anything, 5, 10).
					Return([]*models.User{}, 2, nil)
			},
			expectedStatus: http.StatusOK,
			checkResponse: func(t *testing.T, resp map[string]interface{}) {
				assert.Equal(t, float64(2), resp["total"])
				assert.Equal(t, float64(5), resp["limit"])
				assert.Equal(t, float64(10), resp["offset"])
				usersList := resp["users"].([]interface{})
				assert.Len(t, usersList, 0)
			},
		},
		{
			name:        "Service error",
			queryParams: "",
			setupMock: func(m *mockAuthService) {
				m.On("ListUsers", mock.Anything, 10, 0).
					Return(nil, 0, errors.New("database error"))
			},
			expectedStatus: http.StatusInternalServerError,
			checkResponse: func(t *testing.T, resp map[string]interface{}) {
				assert.Equal(t, "database error", resp["error"])
			},
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockAuth := new(mockAuthService)
			if tt.setupMock != nil {
				tt.setupMock(mockAuth)
			}
			
			handler := NewAuthHandler(mockAuth)
			router := gin.New()
			router.GET("/users", handler.ListUsers)
			
			w := httptest.NewRecorder()
			req, _ := http.NewRequest("GET", "/users?"+tt.queryParams, nil)
			
			router.ServeHTTP(w, req)
			
			assert.Equal(t, tt.expectedStatus, w.Code)
			
			var resp map[string]interface{}
			json.Unmarshal(w.Body.Bytes(), &resp)
			tt.checkResponse(t, resp)
			
			mockAuth.AssertExpectations(t)
		})
	}
}