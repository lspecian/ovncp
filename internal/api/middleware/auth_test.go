package middleware

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	
	"github.com/lspecian/ovncp/internal/models"
)

// Mock Auth Service
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

func TestAuthMiddleware(t *testing.T) {
	gin.SetMode(gin.TestMode)
	
	tests := []struct {
		name           string
		authHeader     string
		setupMock      func(*mockAuthService)
		expectedStatus int
		expectedBody   string
		expectUserSet  bool
	}{
		{
			name:           "No authorization header",
			authHeader:     "",
			expectedStatus: http.StatusUnauthorized,
			expectedBody:   `{"error":"Authorization header required"}`,
		},
		{
			name:           "Invalid authorization format",
			authHeader:     "InvalidFormat token",
			expectedStatus: http.StatusUnauthorized,
			expectedBody:   `{"error":"Invalid authorization header format"}`,
		},
		{
			name:           "Missing token",
			authHeader:     "Bearer ",
			expectedStatus: http.StatusUnauthorized,
			expectedBody:   `{"error":"Invalid authorization header format"}`,
		},
		{
			name:       "Invalid token",
			authHeader: "Bearer invalid-token",
			setupMock: func(m *mockAuthService) {
				m.On("ValidateToken", mock.Anything, "invalid-token").
					Return(nil, errors.New("invalid or expired token"))
			},
			expectedStatus: http.StatusUnauthorized,
			expectedBody:   `{"error":"invalid or expired token"}`,
		},
		{
			name:       "Valid token",
			authHeader: "Bearer valid-token",
			setupMock: func(m *mockAuthService) {
				m.On("ValidateToken", mock.Anything, "valid-token").
					Return(&models.User{
						ID:    "user-123",
						Email: "test@example.com",
						Role:  models.RoleAdmin,
					}, nil)
			},
			expectedStatus: http.StatusOK,
			expectedBody:   `{"message":"success"}`,
			expectUserSet:  true,
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockAuth := new(mockAuthService)
			if tt.setupMock != nil {
				tt.setupMock(mockAuth)
			}
			
			router := gin.New()
			router.Use(AuthMiddleware(mockAuth))
			router.GET("/test", func(c *gin.Context) {
				if tt.expectUserSet {
					user, exists := GetAuthUser(c)
					assert.True(t, exists)
					assert.Equal(t, "user-123", user.ID)
				}
				c.JSON(http.StatusOK, gin.H{"message": "success"})
			})
			
			w := httptest.NewRecorder()
			req, _ := http.NewRequest("GET", "/test", nil)
			if tt.authHeader != "" {
				req.Header.Set("Authorization", tt.authHeader)
			}
			
			router.ServeHTTP(w, req)
			
			assert.Equal(t, tt.expectedStatus, w.Code)
			assert.JSONEq(t, tt.expectedBody, w.Body.String())
			
			mockAuth.AssertExpectations(t)
		})
	}
}

func TestRequireRole(t *testing.T) {
	gin.SetMode(gin.TestMode)
	
	tests := []struct {
		name           string
		user           *models.User
		requiredRoles  []models.UserRole
		expectedStatus int
		expectedBody   string
	}{
		{
			name:           "No user in context",
			user:           nil,
			requiredRoles:  []models.UserRole{models.RoleAdmin},
			expectedStatus: http.StatusUnauthorized,
			expectedBody:   `{"error":"User not authenticated"}`,
		},
		{
			name: "User lacks required role",
			user: &models.User{
				ID:   "user-123",
				Role: models.RoleViewer,
			},
			requiredRoles:  []models.UserRole{models.RoleOperator},
			expectedStatus: http.StatusForbidden,
			expectedBody:   `{"error":"Insufficient permissions"}`,
		},
		{
			name: "User has required role",
			user: &models.User{
				ID:   "user-123",
				Role: models.RoleOperator,
			},
			requiredRoles:  []models.UserRole{models.RoleOperator, models.RoleAdmin},
			expectedStatus: http.StatusOK,
			expectedBody:   `{"message":"success"}`,
		},
		{
			name: "Admin bypasses role check",
			user: &models.User{
				ID:   "admin-123",
				Role: models.RoleAdmin,
			},
			requiredRoles:  []models.UserRole{models.RoleOperator},
			expectedStatus: http.StatusOK,
			expectedBody:   `{"message":"success"}`,
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			router := gin.New()
			
			// Middleware to set user if provided
			router.Use(func(c *gin.Context) {
				if tt.user != nil {
					c.Set(AuthUserKey, tt.user)
				}
				c.Next()
			})
			
			router.Use(RequireRole(tt.requiredRoles...))
			router.GET("/test", func(c *gin.Context) {
				c.JSON(http.StatusOK, gin.H{"message": "success"})
			})
			
			w := httptest.NewRecorder()
			req, _ := http.NewRequest("GET", "/test", nil)
			
			router.ServeHTTP(w, req)
			
			assert.Equal(t, tt.expectedStatus, w.Code)
			assert.JSONEq(t, tt.expectedBody, w.Body.String())
		})
	}
}

func TestOptionalAuth(t *testing.T) {
	gin.SetMode(gin.TestMode)
	
	tests := []struct {
		name          string
		authHeader    string
		setupMock     func(*mockAuthService)
		expectUserSet bool
	}{
		{
			name:          "No authorization header",
			authHeader:    "",
			expectUserSet: false,
		},
		{
			name:          "Invalid authorization format",
			authHeader:    "InvalidFormat token",
			expectUserSet: false,
		},
		{
			name:       "Invalid token - continues without user",
			authHeader: "Bearer invalid-token",
			setupMock: func(m *mockAuthService) {
				m.On("ValidateToken", mock.Anything, "invalid-token").
					Return(nil, errors.New("invalid token"))
			},
			expectUserSet: false,
		},
		{
			name:       "Valid token - sets user",
			authHeader: "Bearer valid-token",
			setupMock: func(m *mockAuthService) {
				m.On("ValidateToken", mock.Anything, "valid-token").
					Return(&models.User{
						ID:    "user-123",
						Email: "test@example.com",
						Role:  models.RoleViewer,
					}, nil)
			},
			expectUserSet: true,
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockAuth := new(mockAuthService)
			if tt.setupMock != nil {
				tt.setupMock(mockAuth)
			}
			
			router := gin.New()
			router.Use(OptionalAuth(mockAuth))
			router.GET("/test", func(c *gin.Context) {
				user, exists := GetAuthUser(c)
				assert.Equal(t, tt.expectUserSet, exists)
				if tt.expectUserSet {
					assert.NotNil(t, user)
					assert.Equal(t, "user-123", user.ID)
				}
				c.JSON(http.StatusOK, gin.H{"message": "success"})
			})
			
			w := httptest.NewRecorder()
			req, _ := http.NewRequest("GET", "/test", nil)
			if tt.authHeader != "" {
				req.Header.Set("Authorization", tt.authHeader)
			}
			
			router.ServeHTTP(w, req)
			
			assert.Equal(t, http.StatusOK, w.Code)
			assert.JSONEq(t, `{"message":"success"}`, w.Body.String())
			
			mockAuth.AssertExpectations(t)
		})
	}
}