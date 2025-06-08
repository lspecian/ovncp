package routes

import (
	"github.com/gin-gonic/gin"
	
	"github.com/lspecian/ovncp/internal/api/handlers"
	"github.com/lspecian/ovncp/internal/api/middleware"
	"github.com/lspecian/ovncp/internal/auth"
	"github.com/lspecian/ovncp/internal/config"
	"github.com/lspecian/ovncp/internal/models"
	"github.com/lspecian/ovncp/pkg/ovn"
)

func SetupRoutes(router *gin.Engine, cfg *config.Config, ovnService ovn.Service, authService auth.Service) {
	// Health check
	router.GET("/health", func(c *gin.Context) {
		c.JSON(200, gin.H{"status": "ok"})
	})
	
	// API v1 group
	v1 := router.Group("/api/v1")
	
	// Auth routes (public)
	authHandler := handlers.NewAuthHandler(authService)
	authGroup := v1.Group("/auth")
	{
		authGroup.POST("/login", authHandler.Login)
		authGroup.GET("/callback/:provider", authHandler.Callback)
		authGroup.POST("/refresh", authHandler.Refresh)
	}
	
	// Apply auth middleware to all routes below
	if cfg.Auth.Enabled {
		v1.Use(middleware.AuthMiddleware(authService))
	}
	
	// Auth routes (authenticated)
	if cfg.Auth.Enabled {
		authGroup.POST("/logout", authHandler.Logout)
		authGroup.GET("/profile", authHandler.GetProfile)
		
		// Admin only routes
		authGroup.GET("/users", middleware.RequireRole(models.RoleAdmin), authHandler.ListUsers)
		authGroup.GET("/users/:id", middleware.RequireRole(models.RoleAdmin), authHandler.GetUser)
		authGroup.PUT("/users/:id/role", middleware.RequireRole(models.RoleAdmin), authHandler.UpdateUserRole)
		authGroup.DELETE("/users/:id", middleware.RequireRole(models.RoleAdmin), authHandler.DeactivateUser)
	}
	
	// Logical Switches
	switchHandler := handlers.NewLogicalSwitchHandler(ovnService)
	switchGroup := v1.Group("/logical-switches")
	if cfg.Auth.Enabled {
		switchGroup.Use(middleware.RequireRole(models.RoleViewer))
	}
	{
		switchGroup.GET("", switchHandler.List)
		switchGroup.GET("/:id", switchHandler.Get)
		if cfg.Auth.Enabled {
			switchGroup.POST("", middleware.RequireRole(models.RoleOperator), switchHandler.Create)
			switchGroup.PUT("/:id", middleware.RequireRole(models.RoleOperator), switchHandler.Update)
			switchGroup.DELETE("/:id", middleware.RequireRole(models.RoleOperator), switchHandler.Delete)
		} else {
			switchGroup.POST("", switchHandler.Create)
			switchGroup.PUT("/:id", switchHandler.Update)
			switchGroup.DELETE("/:id", switchHandler.Delete)
		}
	}
	
	// Load Balancers
	lbHandler := handlers.NewLoadBalancerHandler(ovnService)
	lbGroup := v1.Group("/load-balancers")
	if cfg.Auth.Enabled {
		lbGroup.Use(middleware.RequireRole(models.RoleViewer))
	}
	{
		lbGroup.GET("", lbHandler.List)
		lbGroup.GET("/:id", lbHandler.Get)
		if cfg.Auth.Enabled {
			lbGroup.POST("", middleware.RequireRole(models.RoleOperator), lbHandler.Create)
			lbGroup.PUT("/:id", middleware.RequireRole(models.RoleOperator), lbHandler.Update)
			lbGroup.DELETE("/:id", middleware.RequireRole(models.RoleOperator), lbHandler.Delete)
		} else {
			lbGroup.POST("", lbHandler.Create)
			lbGroup.PUT("/:id", lbHandler.Update)
			lbGroup.DELETE("/:id", lbHandler.Delete)
		}
	}
	
	// Logical Routers
	routerHandler := handlers.NewLogicalRouterHandler(ovnService)
	routerGroup := v1.Group("/logical-routers")
	if cfg.Auth.Enabled {
		routerGroup.Use(middleware.RequireRole(models.RoleViewer))
	}
	{
		routerGroup.GET("", routerHandler.List)
		routerGroup.GET("/:id", routerHandler.Get)
		if cfg.Auth.Enabled {
			routerGroup.POST("", middleware.RequireRole(models.RoleOperator), routerHandler.Create)
			routerGroup.PUT("/:id", middleware.RequireRole(models.RoleOperator), routerHandler.Update)
			routerGroup.DELETE("/:id", middleware.RequireRole(models.RoleOperator), routerHandler.Delete)
		} else {
			routerGroup.POST("", routerHandler.Create)
			routerGroup.PUT("/:id", routerHandler.Update)
			routerGroup.DELETE("/:id", routerHandler.Delete)
		}
	}
	
	// Logical Switch Ports
	portHandler := handlers.NewLogicalSwitchPortHandler(ovnService)
	portGroup := v1.Group("/logical-switch-ports")
	if cfg.Auth.Enabled {
		portGroup.Use(middleware.RequireRole(models.RoleViewer))
	}
	{
		portGroup.GET("", portHandler.List)
		portGroup.GET("/:id", portHandler.Get)
		if cfg.Auth.Enabled {
			portGroup.POST("", middleware.RequireRole(models.RoleOperator), portHandler.Create)
			portGroup.PUT("/:id", middleware.RequireRole(models.RoleOperator), portHandler.Update)
			portGroup.DELETE("/:id", middleware.RequireRole(models.RoleOperator), portHandler.Delete)
		} else {
			portGroup.POST("", portHandler.Create)
			portGroup.PUT("/:id", portHandler.Update)
			portGroup.DELETE("/:id", portHandler.Delete)
		}
	}
	
	// ACLs
	aclHandler := handlers.NewACLHandler(ovnService)
	aclGroup := v1.Group("/acls")
	if cfg.Auth.Enabled {
		aclGroup.Use(middleware.RequireRole(models.RoleViewer))
	}
	{
		aclGroup.GET("", aclHandler.List)
		aclGroup.GET("/:id", aclHandler.Get)
		if cfg.Auth.Enabled {
			aclGroup.POST("", middleware.RequireRole(models.RoleOperator), aclHandler.Create)
			aclGroup.PUT("/:id", middleware.RequireRole(models.RoleOperator), aclHandler.Update)
			aclGroup.DELETE("/:id", middleware.RequireRole(models.RoleOperator), aclHandler.Delete)
		} else {
			aclGroup.POST("", aclHandler.Create)
			aclGroup.PUT("/:id", aclHandler.Update)
			aclGroup.DELETE("/:id", aclHandler.Delete)
		}
	}
	
	// Transactions
	txHandler := handlers.NewTransactionHandler(ovnService)
	if cfg.Auth.Enabled {
		v1.POST("/transactions", middleware.RequireRole(models.RoleOperator), txHandler.Execute)
	} else {
		v1.POST("/transactions", txHandler.Execute)
	}
}