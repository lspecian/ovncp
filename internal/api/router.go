package api

import (
	"time"

	"github.com/gin-gonic/gin"
	"github.com/lspecian/ovncp/internal/api/handlers"
	"github.com/lspecian/ovncp/internal/config"
	"github.com/lspecian/ovncp/internal/db"
	"github.com/lspecian/ovncp/internal/middleware"
	"github.com/lspecian/ovncp/internal/services"
	"go.uber.org/zap"
)

type Router struct {
	engine              *gin.Engine
	ovnService          services.OVNServiceInterface
	tenantService       *services.TenantService
	switchHandler       *handlers.SwitchHandler
	routerHandler       *handlers.RouterHandler
	portHandler         *handlers.PortHandler
	aclHandler          *handlers.ACLHandler
	transactionHandler  *handlers.TransactionHandler
	topologyHandler     *handlers.TopologyHandler
	config              *config.Config
	db                  *db.DB
	logger              *zap.Logger
}

func NewRouter(ovnService services.OVNServiceInterface, cfg *config.Config, database *db.DB, logger *zap.Logger) *Router {
	// Set Gin mode based on environment
	if cfg.Environment == "production" {
		gin.SetMode(gin.ReleaseMode)
	}

	// Create tenant service
	tenantService := services.NewTenantService(database, logger)

	// Create tenant-aware OVN service wrapper
	tenantAwareOVN := services.NewTenantOVNService(ovnService, tenantService, logger)

	r := &Router{
		engine:             gin.New(),
		ovnService:         tenantAwareOVN,
		tenantService:      tenantService,
		switchHandler:      handlers.NewSwitchHandler(tenantAwareOVN),
		routerHandler:      handlers.NewRouterHandler(tenantAwareOVN),
		portHandler:        handlers.NewPortHandler(tenantAwareOVN),
		aclHandler:         handlers.NewACLHandler(tenantAwareOVN),
		transactionHandler: handlers.NewTransactionHandler(tenantAwareOVN),
		topologyHandler:    handlers.NewTopologyHandler(tenantAwareOVN),
		config:             cfg,
		db:                 database,
		logger:             logger,
	}

	r.setupMiddleware()
	r.setupRoutes()
	r.SetupSwaggerRoutes()
	r.SetupReDocRoutes()

	return r
}

func (r *Router) setupMiddleware() {
	// Basic middleware
	r.engine.Use(middleware.Recovery())
	r.engine.Use(middleware.RequestID())
	
	// Security headers - should be first
	r.engine.Use(middleware.SecurityHeaders(middleware.DefaultSecurityConfig()))
	
	// HTTPS redirect in production
	if r.config.Environment == "production" {
		r.engine.Use(middleware.SecureRedirect())
	}
	
	// CORS configuration
	corsConfig := middleware.SecurityConfig{
		CORSEnabled:      true,
		CORSAllowOrigins: r.config.Security.CORSAllowOrigins,
		CORSAllowMethods: []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		CORSAllowHeaders: []string{"Authorization", "Content-Type", "X-Request-ID"},
		CORSExposeHeaders: []string{"X-Request-ID", "X-RateLimit-Limit", "X-RateLimit-Remaining"},
		CORSAllowCredentials: true,
		CORSMaxAge: 86400,
	}
	r.engine.Use(middleware.CORS(corsConfig))
	
	// Logging with context
	r.engine.Use(middleware.LoggerWithConfig(middleware.LoggerConfig{
		Logger: r.logger,
		SkipPaths: []string{"/health", "/metrics"},
	}))
	
	// Rate limiting
	if r.config.Security.RateLimitEnabled {
		rateLimitConfig := middleware.RateLimitConfig{
			Enabled:           true,
			RequestsPerSecond: float64(r.config.Security.RateLimitRPS),
			Burst:            r.config.Security.RateLimitBurst,
			TTL:              5 * time.Minute,
			ByIP:             true,
			ByUser:           true,
		}
		r.engine.Use(middleware.RateLimit(rateLimitConfig))
	}
	
	// Audit logging
	if r.config.Security.AuditEnabled {
		auditLogger := middleware.NewDatabaseAuditLogger(r.db, r.logger)
		auditConfig := middleware.AuditConfig{
			Enabled:          true,
			Logger:           auditLogger,
			LogRequestBody:   true,
			LogResponseBody:  false, // Don't log response bodies by default
			MaxBodySize:      1024 * 1024, // 1MB
			ExcludePaths:     []string{"/health", "/metrics"},
			SensitiveFields:  []string{"password", "token", "secret", "key"},
		}
		r.engine.Use(middleware.Audit(auditConfig))
	}
	
	// Error handler should be last
	r.engine.Use(middleware.ErrorHandler())
}

func (r *Router) setupRoutes() {
	// Health check (no auth required)
	r.engine.GET("/health", r.healthCheck)
	
	// Metrics endpoint (no auth required)
	r.engine.GET("/metrics", middleware.PrometheusHandler())
	
	// CSP violation reports (no auth required)
	r.engine.POST("/api/csp-report", middleware.CSPReportHandler())

	// API v1 - all routes require authentication
	v1 := r.engine.Group("/api/v1")
	
	// Apply authentication middleware to all v1 routes
	authMiddleware := middleware.Auth(middleware.AuthConfig{
		JWTSecret:        r.config.Auth.JWTSecret,
		TokenExpiration:  r.config.Auth.TokenExpiration,
		RefreshExpiration: r.config.Auth.RefreshExpiration,
	})
	v1.Use(authMiddleware)
	
	// Apply tenant context middleware
	tenantMiddleware := middleware.TenantContext(r.tenantService, r.logger)
	v1.Use(tenantMiddleware)
	
	// Register tenant management routes (no tenant context required)
	RegisterTenantRoutes(v1, r.db, r.logger)
	
	{
		// Logical Switches
		switches := v1.Group("/switches")
		switches.Use(middleware.RequirePermission("switches:read"))
		{
			switches.GET("", r.switchHandler.List)
			switches.GET("/:id", r.switchHandler.Get)
			
			// Write operations require additional permission
			switches.POST("", 
				middleware.RequirePermission("switches:write"),
				middleware.EndpointRateLimit(10, 100), // 10 req/s, burst 100
				r.switchHandler.Create)
			switches.PUT("/:id", 
				middleware.RequirePermission("switches:write"),
				r.switchHandler.Update)
			switches.DELETE("/:id", 
				middleware.RequirePermission("switches:delete"),
				middleware.EndpointRateLimit(5, 10), // 5 req/s, burst 10
				r.switchHandler.Delete)
		}

		// Logical Routers
		routers := v1.Group("/routers")
		routers.Use(middleware.RequirePermission("routers:read"))
		{
			routers.GET("", r.routerHandler.List)
			routers.GET("/:id", r.routerHandler.Get)
			
			routers.POST("", 
				middleware.RequirePermission("routers:write"),
				middleware.EndpointRateLimit(10, 100),
				r.routerHandler.Create)
			routers.PUT("/:id", 
				middleware.RequirePermission("routers:write"),
				r.routerHandler.Update)
			routers.DELETE("/:id", 
				middleware.RequirePermission("routers:delete"),
				middleware.EndpointRateLimit(5, 10),
				r.routerHandler.Delete)
		}

		// Ports (under switches)
		switches.GET("/:switchId/ports", 
			middleware.RequirePermission("ports:read"),
			r.portHandler.List)
		switches.POST("/:switchId/ports", 
			middleware.RequirePermission("ports:write"),
			middleware.EndpointRateLimit(20, 200),
			r.portHandler.Create)
		
		// Ports (standalone)
		ports := v1.Group("/ports")
		ports.Use(middleware.RequirePermission("ports:read"))
		{
			ports.GET("/:id", r.portHandler.Get)
			ports.PUT("/:id", 
				middleware.RequirePermission("ports:write"),
				r.portHandler.Update)
			ports.DELETE("/:id", 
				middleware.RequirePermission("ports:delete"),
				middleware.EndpointRateLimit(10, 50),
				r.portHandler.Delete)
		}

		// ACLs
		acls := v1.Group("/acls")
		acls.Use(middleware.RequirePermission("acls:read"))
		{
			acls.GET("", r.aclHandler.List)
			acls.GET("/:id", r.aclHandler.Get)
			
			acls.POST("", 
				middleware.RequirePermission("acls:write"),
				middleware.EndpointRateLimit(10, 100),
				r.aclHandler.Create)
			acls.PUT("/:id", 
				middleware.RequirePermission("acls:write"),
				r.aclHandler.Update)
			acls.DELETE("/:id", 
				middleware.RequirePermission("acls:delete"),
				middleware.EndpointRateLimit(5, 20),
				r.aclHandler.Delete)
		}

		// Transactions - requires admin permission
		v1.POST("/transactions", 
			middleware.RequirePermission("admin"),
			middleware.EndpointRateLimit(5, 10),
			r.transactionHandler.Execute)

		// Topology
		v1.GET("/topology",
			middleware.RequirePermission("topology:read"),
			r.topologyHandler.GetTopology)

		// Visualization routes
		vizHandler := NewVisualizationHandler(r.ovnService, r.logger)
		vizHandler.RegisterVisualizationRoutes(v1)

		// Flow trace routes
		// Note: We need the OVN client directly for flow tracing
		// This would require updating the Router struct to include ovnClient
		// For now, we'll skip this integration

		// Template routes
		RegisterTemplateRoutes(v1, r.ovnService, r.logger)

		// Backup routes
		if err := RegisterBackupRoutes(v1, r.ovnService, r.config, r.logger); err != nil {
			r.logger.Error("Failed to register backup routes", zap.Error(err))
		}
	}
}

func (r *Router) Engine() *gin.Engine {
	return r.engine
}

func (r *Router) healthCheck(c *gin.Context) {
	c.JSON(200, gin.H{
		"status":  "healthy",
		"service": "ovncp-api",
		"version": "0.2.0",
	})
}

