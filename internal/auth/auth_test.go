package auth

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/coreos/go-oidc/v3/oidc"
	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"golang.org/x/oauth2"
)

// Mock OIDC verifier
type mockVerifier struct {
	mock.Mock
}

func (m *mockVerifier) Verify(ctx context.Context, rawIDToken string) (*oidc.IDToken, error) {
	args := m.Called(ctx, rawIDToken)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*oidc.IDToken), args.Error(1)
}

// Test Auth Service
func TestNewAuthService(t *testing.T) {
	cfg := Config{
		Enabled:      true,
		Provider:     "test-provider",
		ClientID:     "test-client",
		ClientSecret: "test-secret",
		IssuerURL:    "https://test.issuer.com",
		RedirectURL:  "https://app.example.com/callback",
	}
	
	svc := NewAuthService(cfg)
	assert.NotNil(t, svc)
	assert.Equal(t, cfg.ClientID, svc.config.ClientID)
}

func TestAuthService_ValidateAPIKey(t *testing.T) {
	svc := &Service{
		apiKeys: map[string]*User{
			"valid-key": {
				ID:    "user1",
				Email: "user1@example.com",
				Roles: []string{"admin"},
			},
		},
	}
	
	// Valid key
	user, err := svc.ValidateAPIKey("valid-key")
	assert.NoError(t, err)
	assert.Equal(t, "user1", user.ID)
	
	// Invalid key
	user, err = svc.ValidateAPIKey("invalid-key")
	assert.Error(t, err)
	assert.Nil(t, user)
}

func TestAuthService_ValidateJWT(t *testing.T) {
	// Create a test JWT
	claims := jwt.MapClaims{
		"sub":   "user123",
		"email": "user@example.com",
		"roles": []string{"user", "admin"},
		"exp":   time.Now().Add(time.Hour).Unix(),
	}
	
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	tokenString, err := token.SignedString([]byte("test-secret"))
	assert.NoError(t, err)
	
	svc := &Service{
		config: Config{
			JWTSecret: "test-secret",
		},
	}
	
	// Valid token
	user, err := svc.ValidateJWT(tokenString)
	assert.NoError(t, err)
	assert.Equal(t, "user123", user.ID)
	assert.Equal(t, "user@example.com", user.Email)
	assert.Contains(t, user.Roles, "admin")
	
	// Invalid token
	user, err = svc.ValidateJWT("invalid.token.here")
	assert.Error(t, err)
	assert.Nil(t, user)
}

func TestAuthService_ValidateJWT_Expired(t *testing.T) {
	// Create an expired JWT
	claims := jwt.MapClaims{
		"sub":   "user123",
		"email": "user@example.com",
		"exp":   time.Now().Add(-time.Hour).Unix(), // Expired
	}
	
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	tokenString, err := token.SignedString([]byte("test-secret"))
	assert.NoError(t, err)
	
	svc := &Service{
		config: Config{
			JWTSecret: "test-secret",
		},
	}
	
	user, err := svc.ValidateJWT(tokenString)
	assert.Error(t, err)
	assert.Nil(t, user)
}

// Test Middleware
func TestAuthMiddleware_Disabled(t *testing.T) {
	svc := &Service{
		config: Config{
			Enabled: false,
		},
	}
	
	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.Use(svc.Middleware())
	
	router.GET("/test", func(c *gin.Context) {
		user, exists := c.Get("user")
		assert.True(t, exists)
		assert.NotNil(t, user)
		c.JSON(200, gin.H{"status": "ok"})
	})
	
	req, _ := http.NewRequest("GET", "/test", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)
	
	assert.Equal(t, http.StatusOK, w.Code)
}

func TestAuthMiddleware_WithBearer(t *testing.T) {
	// Create a valid JWT
	claims := jwt.MapClaims{
		"sub":   "user123",
		"email": "user@example.com",
		"roles": []string{"admin"},
		"exp":   time.Now().Add(time.Hour).Unix(),
	}
	
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	tokenString, err := token.SignedString([]byte("test-secret"))
	assert.NoError(t, err)
	
	svc := &Service{
		config: Config{
			Enabled:   true,
			JWTSecret: "test-secret",
		},
	}
	
	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.Use(svc.Middleware())
	
	router.GET("/test", func(c *gin.Context) {
		user, exists := c.Get("user")
		assert.True(t, exists)
		assert.NotNil(t, user)
		
		u := user.(*User)
		assert.Equal(t, "user123", u.ID)
		c.JSON(200, gin.H{"status": "ok"})
	})
	
	req, _ := http.NewRequest("GET", "/test", nil)
	req.Header.Set("Authorization", "Bearer "+tokenString)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)
	
	assert.Equal(t, http.StatusOK, w.Code)
}

func TestAuthMiddleware_WithAPIKey(t *testing.T) {
	svc := &Service{
		config: Config{
			Enabled: true,
		},
		apiKeys: map[string]*User{
			"test-api-key": {
				ID:    "api-user",
				Email: "api@example.com",
				Roles: []string{"api"},
			},
		},
	}
	
	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.Use(svc.Middleware())
	
	router.GET("/test", func(c *gin.Context) {
		user, exists := c.Get("user")
		assert.True(t, exists)
		assert.NotNil(t, user)
		
		u := user.(*User)
		assert.Equal(t, "api-user", u.ID)
		c.JSON(200, gin.H{"status": "ok"})
	})
	
	req, _ := http.NewRequest("GET", "/test", nil)
	req.Header.Set("X-API-Key", "test-api-key")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)
	
	assert.Equal(t, http.StatusOK, w.Code)
}

func TestAuthMiddleware_Unauthorized(t *testing.T) {
	svc := &Service{
		config: Config{
			Enabled: true,
		},
	}
	
	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.Use(svc.Middleware())
	
	router.GET("/test", func(c *gin.Context) {
		c.JSON(200, gin.H{"status": "ok"})
	})
	
	// No auth header
	req, _ := http.NewRequest("GET", "/test", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)
	
	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

// Test RBAC Middleware
func TestRequireRole(t *testing.T) {
	gin.SetMode(gin.TestMode)
	
	tests := []struct {
		name       string
		userRoles  []string
		required   string
		shouldPass bool
	}{
		{
			name:       "has required role",
			userRoles:  []string{"admin", "user"},
			required:   "admin",
			shouldPass: true,
		},
		{
			name:       "missing required role",
			userRoles:  []string{"user"},
			required:   "admin",
			shouldPass: false,
		},
		{
			name:       "no user",
			userRoles:  nil,
			required:   "admin",
			shouldPass: false,
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			router := gin.New()
			
			// Add user to context if roles provided
			if tt.userRoles != nil {
				router.Use(func(c *gin.Context) {
					c.Set("user", &User{
						ID:    "test-user",
						Roles: tt.userRoles,
					})
					c.Next()
				})
			}
			
			router.GET("/test", RequireRole(tt.required), func(c *gin.Context) {
				c.JSON(200, gin.H{"status": "ok"})
			})
			
			req, _ := http.NewRequest("GET", "/test", nil)
			w := httptest.NewRecorder()
			router.ServeHTTP(w, req)
			
			if tt.shouldPass {
				assert.Equal(t, http.StatusOK, w.Code)
			} else {
				assert.Equal(t, http.StatusForbidden, w.Code)
			}
		})
	}
}

func TestRequireAnyRole(t *testing.T) {
	gin.SetMode(gin.TestMode)
	
	tests := []struct {
		name       string
		userRoles  []string
		required   []string
		shouldPass bool
	}{
		{
			name:       "has one required role",
			userRoles:  []string{"user"},
			required:   []string{"admin", "user"},
			shouldPass: true,
		},
		{
			name:       "has multiple required roles",
			userRoles:  []string{"admin", "user"},
			required:   []string{"admin", "user"},
			shouldPass: true,
		},
		{
			name:       "missing all required roles",
			userRoles:  []string{"guest"},
			required:   []string{"admin", "user"},
			shouldPass: false,
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			router := gin.New()
			
			router.Use(func(c *gin.Context) {
				c.Set("user", &User{
					ID:    "test-user",
					Roles: tt.userRoles,
				})
				c.Next()
			})
			
			router.GET("/test", RequireAnyRole(tt.required...), func(c *gin.Context) {
				c.JSON(200, gin.H{"status": "ok"})
			})
			
			req, _ := http.NewRequest("GET", "/test", nil)
			w := httptest.NewRecorder()
			router.ServeHTTP(w, req)
			
			if tt.shouldPass {
				assert.Equal(t, http.StatusOK, w.Code)
			} else {
				assert.Equal(t, http.StatusForbidden, w.Code)
			}
		})
	}
}

// Test OAuth2 Flow
func TestAuthService_GetAuthURL(t *testing.T) {
	svc := &Service{
		config: Config{
			ClientID:    "test-client",
			RedirectURL: "https://app.example.com/callback",
		},
		oauth2Config: &oauth2.Config{
			ClientID:     "test-client",
			ClientSecret: "test-secret",
			RedirectURL:  "https://app.example.com/callback",
			Endpoint: oauth2.Endpoint{
				AuthURL:  "https://auth.example.com/auth",
				TokenURL: "https://auth.example.com/token",
			},
			Scopes: []string{"openid", "email", "profile"},
		},
	}
	
	state := "test-state"
	url := svc.GetAuthURL(state)
	
	assert.Contains(t, url, "https://auth.example.com/auth")
	assert.Contains(t, url, "client_id=test-client")
	assert.Contains(t, url, "state=test-state")
	assert.Contains(t, url, "redirect_uri=")
}

// Test User Methods
func TestUser_HasRole(t *testing.T) {
	user := &User{
		ID:    "user1",
		Roles: []string{"admin", "user"},
	}
	
	assert.True(t, user.HasRole("admin"))
	assert.True(t, user.HasRole("user"))
	assert.False(t, user.HasRole("guest"))
}

func TestUser_HasAnyRole(t *testing.T) {
	user := &User{
		ID:    "user1",
		Roles: []string{"admin", "user"},
	}
	
	assert.True(t, user.HasAnyRole("admin", "guest"))
	assert.True(t, user.HasAnyRole("user", "guest"))
	assert.False(t, user.HasAnyRole("guest", "visitor"))
}