package api

import (
	"github.com/gin-gonic/gin"
	"github.com/lspecian/ovncp/internal/api/handlers"
	"github.com/lspecian/ovncp/internal/backup"
	"github.com/lspecian/ovncp/internal/config"
	"github.com/lspecian/ovncp/internal/middleware"
	"github.com/lspecian/ovncp/internal/services"
	"go.uber.org/zap"
)

// RegisterBackupRoutes registers backup and restore routes
func RegisterBackupRoutes(v1 *gin.RouterGroup, ovnService services.OVNServiceInterface, cfg *config.Config, logger *zap.Logger) error {
	// Create backup storage
	storagePath := cfg.GetBackupPath()
	storage, err := backup.NewFileStorage(storagePath)
	if err != nil {
		return err
	}

	// Create backup service and handler
	backupService := backup.NewBackupService(ovnService, storage, logger)
	backupHandler := handlers.NewBackupHandler(backupService, logger)

	// Backup routes
	backups := v1.Group("/backups")
	{
		// List backups (read permission)
		backups.GET("",
			middleware.RequirePermission("backups:read"),
			backupHandler.ListBackups)

		// Get backup details (read permission)
		backups.GET("/:id",
			middleware.RequirePermission("backups:read"),
			backupHandler.GetBackup)

		// Create backup (write permission)
		backups.POST("",
			middleware.RequirePermission("backups:write"),
			middleware.EndpointRateLimit(5, 10), // 5 req/s, burst 10
			backupHandler.CreateBackup)

		// Restore from backup (admin permission due to potential impact)
		backups.POST("/:id/restore",
			middleware.RequirePermission("backups:restore"),
			middleware.RequirePermission("admin"),
			middleware.EndpointRateLimit(1, 5), // 1 req/s, burst 5
			backupHandler.RestoreBackup)

		// Delete backup (delete permission)
		backups.DELETE("/:id",
			middleware.RequirePermission("backups:delete"),
			middleware.EndpointRateLimit(5, 10),
			backupHandler.DeleteBackup)

		// Export backup (read permission)
		backups.GET("/:id/export",
			middleware.RequirePermission("backups:read"),
			backupHandler.ExportBackup)

		// Import backup (write permission)
		backups.POST("/import",
			middleware.RequirePermission("backups:write"),
			middleware.EndpointRateLimit(5, 10),
			backupHandler.ImportBackup)

		// Validate backup (read permission)
		backups.POST("/validate",
			middleware.RequirePermission("backups:read"),
			backupHandler.ValidateBackup)
	}

	return nil
}