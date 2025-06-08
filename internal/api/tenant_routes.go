package api

import (
	"github.com/gin-gonic/gin"
	"github.com/lspecian/ovncp/internal/api/handlers"
	"github.com/lspecian/ovncp/internal/db"
	"github.com/lspecian/ovncp/internal/middleware"
	"github.com/lspecian/ovncp/internal/services"
	"go.uber.org/zap"
)

// RegisterTenantRoutes registers tenant management routes
func RegisterTenantRoutes(v1 *gin.RouterGroup, db *db.DB, logger *zap.Logger) {
	// Create tenant service and handler
	tenantService := services.NewTenantService(db, logger)
	tenantHandler := handlers.NewTenantHandler(tenantService, logger)

	// Public tenant routes (no tenant context required)
	tenants := v1.Group("/tenants")
	{
		// List user's tenants
		tenants.GET("",
			middleware.RequirePermission("tenants:read"),
			tenantHandler.ListTenants)

		// Create new tenant
		tenants.POST("",
			middleware.RequirePermission("tenants:create"),
			tenantHandler.CreateTenant)

		// Get specific tenant
		tenants.GET("/:id",
			middleware.RequirePermission("tenants:read"),
			tenantHandler.GetTenant)

		// Update tenant (requires admin role in tenant)
		tenants.PUT("/:id",
			middleware.RequirePermission("tenants:write"),
			middleware.RequireTenantRole("admin"),
			tenantHandler.UpdateTenant)

		// Delete tenant (requires admin role)
		tenants.DELETE("/:id",
			middleware.RequirePermission("tenants:delete"),
			middleware.RequireTenantRole("admin"),
			tenantHandler.DeleteTenant)

		// Get resource usage
		tenants.GET("/:id/usage",
			middleware.RequirePermission("tenants:read"),
			tenantHandler.GetResourceUsage)

		// Member management
		members := tenants.Group("/:id/members")
		members.Use(middleware.RequireTenantRole("admin"))
		{
			members.GET("", tenantHandler.ListMembers)
			members.POST("", tenantHandler.AddMember)
			members.DELETE("/:user_id", tenantHandler.RemoveMember)
			members.PUT("/:user_id/role", tenantHandler.UpdateMemberRole)
		}

		// Invitation management
		invitations := tenants.Group("/:id/invitations")
		{
			invitations.POST("",
				middleware.RequireTenantRole("admin"),
				tenantHandler.CreateInvitation)
		}

		// API key management
		apiKeys := tenants.Group("/:id/api-keys")
		apiKeys.Use(middleware.RequireTenantRole("admin"))
		{
			apiKeys.GET("", tenantHandler.ListAPIKeys)
			apiKeys.POST("", tenantHandler.CreateAPIKey)
			apiKeys.DELETE("/:key_id", tenantHandler.DeleteAPIKey)
		}
	}

	// Accept invitation (no tenant context)
	v1.POST("/invitations/:token/accept",
		middleware.RequirePermission("tenants:join"),
		tenantHandler.AcceptInvitation)
}