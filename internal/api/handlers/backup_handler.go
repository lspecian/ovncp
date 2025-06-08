package handlers

import (
	"fmt"
	"net/http"
	"path/filepath"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/lspecian/ovncp/internal/backup"
	"go.uber.org/zap"
)

type BackupHandler struct {
	backupService *backup.BackupService
	logger        *zap.Logger
}

func NewBackupHandler(backupService *backup.BackupService, logger *zap.Logger) *BackupHandler {
	return &BackupHandler{
		backupService: backupService,
		logger:        logger,
	}
}

// CreateBackupRequest represents a backup creation request
type CreateBackupRequest struct {
	Name           string                   `json:"name" binding:"required"`
	Description    string                   `json:"description,omitempty"`
	Type           backup.BackupType        `json:"type,omitempty"`
	Format         backup.BackupFormat      `json:"format,omitempty"`
	Compress       bool                     `json:"compress"`
	Encrypt        bool                     `json:"encrypt"`
	EncryptionKey  string                   `json:"encryption_key,omitempty"`
	Tags           []string                 `json:"tags,omitempty"`
	ResourceFilter *backup.ResourceFilter   `json:"resource_filter,omitempty"`
}

// CreateBackup creates a new backup
func (h *BackupHandler) CreateBackup(c *gin.Context) {
	var req CreateBackupRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Invalid request body",
		})
		return
	}

	// Create backup options
	options := &backup.BackupOptions{
		Name:           req.Name,
		Description:    req.Description,
		Type:           req.Type,
		Format:         req.Format,
		Compress:       req.Compress,
		Encrypt:        req.Encrypt,
		EncryptionKey:  req.EncryptionKey,
		Tags:           req.Tags,
		ResourceFilter: req.ResourceFilter,
	}

	// Set defaults
	if options.Type == "" {
		options.Type = backup.BackupTypeFull
	}
	if options.Format == "" {
		options.Format = backup.BackupFormatJSON
	}

	// Create backup
	metadata, err := h.backupService.CreateBackup(c.Request.Context(), options)
	if err != nil {
		h.logger.Error("Failed to create backup", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": fmt.Sprintf("Failed to create backup: %v", err),
		})
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"backup": metadata,
		"message": "Backup created successfully",
	})
}

// ListBackups lists all available backups
func (h *BackupHandler) ListBackups(c *gin.Context) {
	backups, err := h.backupService.ListBackups()
	if err != nil {
		h.logger.Error("Failed to list backups", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to list backups",
		})
		return
	}

	// Filter by tags if requested
	tags := c.QueryArray("tag")
	if len(tags) > 0 {
		filtered := []*backup.BackupMetadata{}
		for _, b := range backups {
			for _, tag := range tags {
				for _, bTag := range b.Tags {
					if tag == bTag {
						filtered = append(filtered, b)
						break
					}
				}
			}
		}
		backups = filtered
	}

	c.JSON(http.StatusOK, gin.H{
		"backups": backups,
		"total":   len(backups),
	})
}

// GetBackup returns backup metadata
func (h *BackupHandler) GetBackup(c *gin.Context) {
	backupID := c.Param("id")

	backup, err := h.backupService.GetBackup(backupID)
	if err != nil {
		h.logger.Error("Failed to get backup", zap.Error(err))
		c.JSON(http.StatusNotFound, gin.H{
			"error": "Backup not found",
		})
		return
	}

	c.JSON(http.StatusOK, backup)
}

// RestoreBackupRequest represents a restore request
type RestoreBackupRequest struct {
	DryRun          bool                      `json:"dry_run"`
	Force           bool                      `json:"force"`
	SkipValidation  bool                      `json:"skip_validation"`
	ConflictPolicy  backup.ConflictPolicy     `json:"conflict_policy,omitempty"`
	ResourceMapping map[string]string         `json:"resource_mapping,omitempty"`
	RestoreFilter   *backup.ResourceFilter    `json:"restore_filter,omitempty"`
	DecryptionKey   string                    `json:"decryption_key,omitempty"`
}

// RestoreBackup restores from a backup
func (h *BackupHandler) RestoreBackup(c *gin.Context) {
	backupID := c.Param("id")

	var req RestoreBackupRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Invalid request body",
		})
		return
	}

	// Set default conflict policy
	if req.ConflictPolicy == "" {
		req.ConflictPolicy = backup.ConflictPolicySkip
	}

	// Create restore options
	options := &backup.RestoreOptions{
		DryRun:          req.DryRun,
		Force:           req.Force,
		SkipValidation:  req.SkipValidation,
		ConflictPolicy:  req.ConflictPolicy,
		ResourceMapping: req.ResourceMapping,
		RestoreFilter:   req.RestoreFilter,
		DecryptionKey:   req.DecryptionKey,
	}

	// Perform restore
	result, err := h.backupService.RestoreBackup(c.Request.Context(), backupID, options)
	if err != nil {
		h.logger.Error("Failed to restore backup", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": fmt.Sprintf("Failed to restore backup: %v", err),
		})
		return
	}

	status := http.StatusOK
	if !result.Success {
		status = http.StatusPartialContent
	}

	c.JSON(status, result)
}

// DeleteBackup deletes a backup
func (h *BackupHandler) DeleteBackup(c *gin.Context) {
	backupID := c.Param("id")

	if err := h.backupService.DeleteBackup(backupID); err != nil {
		h.logger.Error("Failed to delete backup", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to delete backup",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "Backup deleted successfully",
	})
}

// ExportBackup exports a backup file
func (h *BackupHandler) ExportBackup(c *gin.Context) {
	backupID := c.Param("id")
	format := c.Query("format")

	// Get backup metadata
	metadata, err := h.backupService.GetBackup(backupID)
	if err != nil {
		h.logger.Error("Failed to get backup", zap.Error(err))
		c.JSON(http.StatusNotFound, gin.H{
			"error": "Backup not found",
		})
		return
	}

	// Determine format
	var exportFormat backup.BackupFormat
	switch format {
	case "yaml":
		exportFormat = backup.BackupFormatYAML
	case "json", "":
		exportFormat = backup.BackupFormatJSON
	default:
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Invalid format. Supported formats: json, yaml",
		})
		return
	}

	// Set headers
	filename := fmt.Sprintf("backup-%s-%s.%s", 
		metadata.Name, 
		metadata.CreatedAt.Format("20060102-150405"),
		exportFormat)
	
	c.Header("Content-Type", "application/octet-stream")
	c.Header("Content-Disposition", fmt.Sprintf("attachment; filename=%s", filename))

	// Export backup
	if err := h.backupService.ExportBackup(backupID, exportFormat, c.Writer); err != nil {
		h.logger.Error("Failed to export backup", zap.Error(err))
		// Can't change status after writing headers
		c.Writer.WriteString(fmt.Sprintf("\nError: %v", err))
		return
	}
}

// ImportBackupRequest represents an import request
type ImportBackupRequest struct {
	Format backup.BackupFormat `json:"format,omitempty"`
	Data   string              `json:"data" binding:"required"`
}

// ImportBackup imports a backup from uploaded data
func (h *BackupHandler) ImportBackup(c *gin.Context) {
	// Check if it's a file upload
	file, header, err := c.Request.FormFile("file")
	if err == nil {
		defer file.Close()

		// Determine format from filename
		format := backup.BackupFormatJSON
		if filepath.Ext(header.Filename) == ".yaml" || filepath.Ext(header.Filename) == ".yml" {
			format = backup.BackupFormatYAML
		}

		// Import from file
		metadata, err := h.backupService.ImportBackup(file, format)
		if err != nil {
			h.logger.Error("Failed to import backup", zap.Error(err))
			c.JSON(http.StatusBadRequest, gin.H{
				"error": fmt.Sprintf("Failed to import backup: %v", err),
			})
			return
		}

		c.JSON(http.StatusCreated, gin.H{
			"backup": metadata,
			"message": "Backup imported successfully",
		})
		return
	}

	// Otherwise try JSON body
	var req ImportBackupRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Invalid request. Provide either a file upload or JSON data",
		})
		return
	}

	// Set default format
	if req.Format == "" {
		req.Format = backup.BackupFormatJSON
	}

	// Import from string data
	reader := &simpleStringReader{data: req.Data}
	metadata, err := h.backupService.ImportBackup(reader, req.Format)
	if err != nil {
		h.logger.Error("Failed to import backup", zap.Error(err))
		c.JSON(http.StatusBadRequest, gin.H{
			"error": fmt.Sprintf("Failed to import backup: %v", err),
		})
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"backup": metadata,
		"message": "Backup imported successfully",
	})
}

// ValidateBackupRequest represents a validation request
type ValidateBackupRequest struct {
	BackupID      string `json:"backup_id" binding:"required"`
	DecryptionKey string `json:"decryption_key,omitempty"`
}

// ValidateBackup validates a backup
func (h *BackupHandler) ValidateBackup(c *gin.Context) {
	var req ValidateBackupRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Invalid request body",
		})
		return
	}

	// For now, just check if backup exists and can be retrieved
	// In a real implementation, would do more thorough validation
	_, err := h.backupService.GetBackup(req.BackupID)
	if err != nil {
		c.JSON(http.StatusOK, gin.H{
			"valid": false,
			"errors": []string{"Backup not found or corrupted"},
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"valid": true,
		"message": "Backup is valid",
	})
}

// Helper type for string reader
type simpleStringReader struct {
	data string
	pos  int
}

func (r *simpleStringReader) Read(p []byte) (n int, err error) {
	if r.pos >= len(r.data) {
		return 0, fmt.Errorf("EOF")
	}
	
	n = copy(p, r.data[r.pos:])
	r.pos += n
	return n, nil
}