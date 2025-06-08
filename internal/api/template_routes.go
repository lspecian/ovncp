package api

import (
	"github.com/gin-gonic/gin"
	"github.com/lspecian/ovncp/internal/api/handlers"
	"github.com/lspecian/ovncp/internal/middleware"
	"github.com/lspecian/ovncp/internal/services"
	"go.uber.org/zap"
)

// RegisterTemplateRoutes registers policy template routes
func RegisterTemplateRoutes(v1 *gin.RouterGroup, ovnService services.OVNServiceInterface, logger *zap.Logger) {
	// Create template service and handler
	templateService := services.NewTemplateService(ovnService, logger)
	templateHandler := handlers.NewTemplateHandler(templateService, logger)

	// Template routes
	templates := v1.Group("/templates")
	templates.Use(middleware.RequirePermission("templates:read"))
	{
		// List all templates
		templates.GET("", templateHandler.ListTemplates)
		
		// Get specific template
		templates.GET("/:id", templateHandler.GetTemplate)
		
		// Validate template with variables
		templates.POST("/validate", 
			middleware.RequirePermission("templates:validate"),
			templateHandler.ValidateTemplate)
		
		// Instantiate template (create ACLs)
		templates.POST("/instantiate",
			middleware.RequirePermission("templates:write"),
			middleware.RequirePermission("acls:write"),
			templateHandler.InstantiateTemplate)
		
		// Import custom template
		templates.POST("/import",
			middleware.RequirePermission("templates:admin"),
			templateHandler.ImportTemplate)
		
		// Export template
		templates.GET("/:id/export",
			middleware.RequirePermission("templates:read"),
			templateHandler.ExportTemplate)
	}
}