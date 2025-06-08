package middleware

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	
	"github.com/lspecian/ovncp/internal/auth"
	"github.com/lspecian/ovncp/internal/models"
)

const (
	AuthUserKey = "auth_user"
)

// AuthMiddleware validates the Bearer token and sets the user in context
func AuthMiddleware(authService auth.Service) gin.HandlerFunc {
	return func(c *gin.Context) {
		// Extract token from Authorization header
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Authorization header required"})
			c.Abort()
			return
		}
		
		// Check Bearer prefix
		parts := strings.SplitN(authHeader, " ", 2)
		if len(parts) != 2 || parts[0] != "Bearer" || parts[1] == "" {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid authorization header format"})
			c.Abort()
			return
		}
		
		token := parts[1]
		
		// Validate token
		user, err := authService.ValidateToken(c.Request.Context(), token)
		if err != nil {
			c.JSON(http.StatusUnauthorized, gin.H{"error": err.Error()})
			c.Abort()
			return
		}
		
		// Set user in context
		c.Set(AuthUserKey, user)
		c.Next()
	}
}

// RequireRole checks if the authenticated user has the required role
func RequireRole(roles ...models.UserRole) gin.HandlerFunc {
	return func(c *gin.Context) {
		userInterface, exists := c.Get(AuthUserKey)
		if !exists {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "User not authenticated"})
			c.Abort()
			return
		}
		
		user, ok := userInterface.(*models.User)
		if !ok {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Invalid user context"})
			c.Abort()
			return
		}
		
		// Check if user has any of the required roles
		hasRole := false
		for _, role := range roles {
			if user.Role == role {
				hasRole = true
				break
			}
		}
		
		// Admin always has access
		if user.Role == models.RoleAdmin {
			hasRole = true
		}
		
		if !hasRole {
			c.JSON(http.StatusForbidden, gin.H{"error": "Insufficient permissions"})
			c.Abort()
			return
		}
		
		c.Next()
	}
}

// OptionalAuth is like AuthMiddleware but doesn't require authentication
func OptionalAuth(authService auth.Service) gin.HandlerFunc {
	return func(c *gin.Context) {
		// Extract token from Authorization header
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			c.Next()
			return
		}
		
		// Check Bearer prefix
		parts := strings.SplitN(authHeader, " ", 2)
		if len(parts) != 2 || parts[0] != "Bearer" || parts[1] == "" {
			c.Next()
			return
		}
		
		token := parts[1]
		
		// Validate token
		user, err := authService.ValidateToken(c.Request.Context(), token)
		if err == nil && user != nil {
			// Set user in context if valid
			c.Set(AuthUserKey, user)
		}
		
		c.Next()
	}
}

// GetAuthUser retrieves the authenticated user from context
func GetAuthUser(c *gin.Context) (*models.User, bool) {
	userInterface, exists := c.Get(AuthUserKey)
	if !exists {
		return nil, false
	}
	
	user, ok := userInterface.(*models.User)
	return user, ok
}