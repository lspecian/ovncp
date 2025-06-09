package middleware

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
)

// RequireAuth middleware checks for valid JWT token
func RequireAuth() gin.HandlerFunc {
	return func(c *gin.Context) {
		// Skip auth in development if AUTH_ENABLED is false
		if c.GetString("AUTH_ENABLED") == "false" {
			c.Next()
			return
		}

		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Authorization header required"})
			c.Abort()
			return
		}

		// Extract token from Bearer scheme
		parts := strings.Split(authHeader, " ")
		if len(parts) != 2 || parts[0] != "Bearer" {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid authorization header format"})
			c.Abort()
			return
		}

		tokenString := parts[1]
		
		// Parse and validate token
		token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
			// Validate signing method
			if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
				return nil, jwt.ErrSignatureInvalid
			}
			// Return the secret key (should be from config)
			return []byte("your-secret-key-here-min-32-chars"), nil
		})

		if err != nil || !token.Valid {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid token"})
			c.Abort()
			return
		}

		// Extract claims
		if claims, ok := token.Claims.(jwt.MapClaims); ok {
			c.Set("user_id", claims["sub"])
			c.Set("user_email", claims["email"])
			c.Set("user_roles", claims["roles"])
		}

		c.Next()
	}
}

// RequirePermission middleware checks if user has required permission
func RequirePermission(permission string) gin.HandlerFunc {
	return func(c *gin.Context) {
		// Skip permission check in development if AUTH_ENABLED is false
		if c.GetString("AUTH_ENABLED") == "false" {
			c.Next()
			return
		}

		// Get user roles from context
		rolesInterface, exists := c.Get("user_roles")
		if !exists {
			c.JSON(http.StatusForbidden, gin.H{"error": "No roles found"})
			c.Abort()
			return
		}

		// Convert roles to string slice
		var userRoles []string
		switch v := rolesInterface.(type) {
		case []string:
			userRoles = v
		case []interface{}:
			for _, role := range v {
				if roleStr, ok := role.(string); ok {
					userRoles = append(userRoles, roleStr)
				}
			}
		}

		// Check if user has admin role (admin has all permissions)
		for _, role := range userRoles {
			if role == "admin" {
				c.Next()
				return
			}
		}

		// Check specific permission
		hasPermission := false
		for _, role := range userRoles {
			if checkRolePermission(role, permission) {
				hasPermission = true
				break
			}
		}

		if !hasPermission {
			c.JSON(http.StatusForbidden, gin.H{"error": "Insufficient permissions"})
			c.Abort()
			return
		}

		c.Next()
	}
}

// checkRolePermission checks if a role has a specific permission
func checkRolePermission(role, permission string) bool {
	// Define role-permission mappings
	rolePermissions := map[string][]string{
		"operator": {
			"switches:read", "switches:write",
			"routers:read", "routers:write",
			"ports:read", "ports:write",
			"acls:read", "acls:write",
			"backups:read", "backups:write",
			"topology:read",
		},
		"viewer": {
			"switches:read",
			"routers:read",
			"ports:read",
			"acls:read",
			"backups:read",
			"topology:read",
		},
	}

	if permissions, ok := rolePermissions[role]; ok {
		for _, p := range permissions {
			if p == permission {
				return true
			}
		}
	}

	return false
}

// AuthConfig holds configuration for authentication middleware
type AuthConfig struct {
	Enabled      bool
	JWTSecret    string
	SkipPaths    []string
	PublicPaths  []string
}

// Auth creates an authentication middleware with the given config
func Auth(cfg AuthConfig) gin.HandlerFunc {
	return func(c *gin.Context) {
		// Skip auth if disabled
		if !cfg.Enabled {
			c.Next()
			return
		}

		// Skip auth for public paths
		path := c.Request.URL.Path
		for _, publicPath := range cfg.PublicPaths {
			if path == publicPath || strings.HasPrefix(path, publicPath) {
				c.Next()
				return
			}
		}

		// Skip auth for configured paths
		for _, skipPath := range cfg.SkipPaths {
			if path == skipPath {
				c.Next()
				return
			}
		}

		// Use RequireAuth for other paths
		RequireAuth()(c)
	}
}

// TenantContext middleware adds tenant context to requests
func TenantContext() gin.HandlerFunc {
	return func(c *gin.Context) {
		// Extract tenant from header or user context
		tenantID := c.GetHeader("X-Tenant-ID")
		
		// If not in header, try to get from user's default tenant
		if tenantID == "" {
			if userID, exists := c.Get("user_id"); exists {
				// TODO: Look up user's default tenant
				_ = userID
			}
		}

		// Set tenant in context if found
		if tenantID != "" {
			c.Set("tenant_id", tenantID)
		}

		c.Next()
	}
}
