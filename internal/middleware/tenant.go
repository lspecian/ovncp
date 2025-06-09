package middleware

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/lspecian/ovncp/internal/models"
	"github.com/lspecian/ovncp/internal/services"
	"go.uber.org/zap"
)

const (
	// TenantContextKey is the key for tenant ID in context
	TenantContextKey = "tenant_id"
	// TenantHeaderKey is the HTTP header for tenant ID
	TenantHeaderKey = "X-Tenant-ID"
)

// TenantMiddleware extracts and validates tenant context
func TenantMiddleware(tenantService *services.TenantService, logger *zap.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		// Get tenant ID from header
		tenantID := c.GetHeader(TenantHeaderKey)
		
		// If not in header, check query parameter
		if tenantID == "" {
			tenantID = c.Query("tenant_id")
		}
		
		// If not in query, check if it's in the URL path
		if tenantID == "" {
			tenantID = c.Param("tenant_id")
		}
		
		// For API key authentication, get tenant from API key
		if tenantID == "" {
			if apiKey := extractAPIKey(c); apiKey != "" {
				key, err := tenantService.ValidateAPIKey(c.Request.Context(), apiKey)
				if err == nil {
					tenantID = key.TenantID
				}
			}
		}
		
		// If still no tenant ID, check user's default tenant
		if tenantID == "" {
			if userID, exists := c.Get("user_id"); exists {
				// Get user's tenants
				filter := &models.TenantFilter{
					UserID: userID.(string),
				}
				tenants, err := tenantService.ListTenants(c.Request.Context(), filter)
				if err == nil && len(tenants) == 1 {
					// If user has only one tenant, use it as default
					tenantID = tenants[0].ID
				}
			}
		}
		
		// Validate tenant exists and user has access
		if tenantID != "" {
			tenant, err := tenantService.GetTenant(c.Request.Context(), tenantID)
			if err != nil {
				logger.Warn("Invalid tenant ID",
					zap.String("tenant_id", tenantID),
					zap.Error(err))
				c.JSON(http.StatusBadRequest, gin.H{
					"error": "Invalid tenant ID",
				})
				c.Abort()
				return
			}
			
			// Check if user has access to this tenant
			if userID, exists := c.Get("user_id"); exists {
				membership, err := tenantService.GetMembership(c.Request.Context(), tenantID, userID.(string))
				if err != nil || membership == nil {
					c.JSON(http.StatusForbidden, gin.H{
						"error": "Access denied to tenant",
					})
					c.Abort()
					return
				}
				
				// Set tenant role in context
				c.Set("tenant_role", membership.Role)
			}
			
			// Set tenant in context
			c.Set(TenantContextKey, tenantID)
			c.Set("tenant", tenant)
		}
		
		c.Next()
	}
}

// RequireTenant ensures a tenant context is present
func RequireTenant() gin.HandlerFunc {
	return func(c *gin.Context) {
		if _, exists := c.Get(TenantContextKey); !exists {
			c.JSON(http.StatusBadRequest, gin.H{
				"error": "Tenant context required",
			})
			c.Abort()
			return
		}
		c.Next()
	}
}

// RequireTenantRole checks if user has specific role in tenant
func RequireTenantRole(roles ...string) gin.HandlerFunc {
	return func(c *gin.Context) {
		tenantRole, exists := c.Get("tenant_role")
		if !exists {
			c.JSON(http.StatusForbidden, gin.H{
				"error": "No tenant role found",
			})
			c.Abort()
			return
		}
		
		userRole := tenantRole.(string)
		for _, role := range roles {
			if userRole == role {
				c.Next()
				return
			}
		}
		
		c.JSON(http.StatusForbidden, gin.H{
			"error": "Insufficient tenant permissions",
		})
		c.Abort()
	}
}

// TenantResourceFilter adds tenant filtering to OVN queries
func TenantResourceFilter() gin.HandlerFunc {
	return func(c *gin.Context) {
		if tenantID, exists := c.Get(TenantContextKey); exists {
			// Add tenant filter to query context
			c.Set("resource_filter", map[string]interface{}{
				"tenant_id": tenantID,
			})
		}
		c.Next()
	}
}

// extractAPIKey extracts API key from request
func extractAPIKey(c *gin.Context) string {
	// Check Authorization header
	auth := c.GetHeader("Authorization")
	if strings.HasPrefix(auth, "Bearer ovncp_") {
		return strings.TrimPrefix(auth, "Bearer ")
	}
	
	// Check X-API-Key header
	if key := c.GetHeader("X-API-Key"); key != "" {
		return key
	}
	
	// Check query parameter
	if key := c.Query("api_key"); key != "" {
		return key
	}
	
	return ""
}

// GetTenantID returns the current tenant ID from context
func GetTenantID(c *gin.Context) string {
	if tenantID, exists := c.Get(TenantContextKey); exists {
		return tenantID.(string)
	}
	return ""
}

// GetTenantRole returns the current user's role in the tenant
func GetTenantRole(c *gin.Context) string {
	if role, exists := c.Get("tenant_role"); exists {
		return role.(string)
	}
	return ""
}