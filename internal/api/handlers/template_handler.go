package handlers

import (
	"encoding/json"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/lspecian/ovncp/internal/services"
	"go.uber.org/zap"
)

type TemplateHandler struct {
	templateService *services.TemplateService
	logger          *zap.Logger
}

func NewTemplateHandler(templateService *services.TemplateService, logger *zap.Logger) *TemplateHandler {
	return &TemplateHandler{
		templateService: templateService,
		logger:          logger,
	}
}

// ListTemplates returns all available policy templates
func (h *TemplateHandler) ListTemplates(c *gin.Context) {
	category := c.Query("category")
	tags := c.QueryArray("tag")

	var templates interface{}

	if category != "" {
		templates = h.templateService.ListTemplatesByCategory(category)
	} else if len(tags) > 0 {
		templates = h.templateService.SearchTemplates(tags)
	} else {
		templates = h.templateService.ListTemplates()
	}

	c.JSON(http.StatusOK, gin.H{
		"templates": templates,
	})
}

// GetTemplate returns a specific template by ID
func (h *TemplateHandler) GetTemplate(c *gin.Context) {
	templateID := c.Param("id")

	template, err := h.templateService.GetTemplate(templateID)
	if err != nil {
		h.logger.Error("Failed to get template", zap.Error(err))
		c.JSON(http.StatusNotFound, gin.H{
			"error": "Template not found",
		})
		return
	}

	c.JSON(http.StatusOK, template)
}

// ValidateTemplateRequest represents a template validation request
type ValidateTemplateRequest struct {
	TemplateID string                 `json:"template_id" binding:"required"`
	Variables  map[string]interface{} `json:"variables" binding:"required"`
}

// ValidateTemplate validates template variables
func (h *TemplateHandler) ValidateTemplate(c *gin.Context) {
	var req ValidateTemplateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Invalid request body",
		})
		return
	}

	result, err := h.templateService.ValidateTemplate(req.TemplateID, req.Variables)
	if err != nil {
		h.logger.Error("Failed to validate template", zap.Error(err))
		c.JSON(http.StatusBadRequest, gin.H{
			"error": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, result)
}

// InstantiateTemplateRequest represents a template instantiation request
type InstantiateTemplateRequest struct {
	TemplateID   string                 `json:"template_id" binding:"required"`
	Variables    map[string]interface{} `json:"variables" binding:"required"`
	TargetSwitch string                 `json:"target_switch"`
	DryRun       bool                   `json:"dry_run"`
}

// InstantiateTemplate creates ACL rules from a template
func (h *TemplateHandler) InstantiateTemplate(c *gin.Context) {
	var req InstantiateTemplateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Invalid request body",
		})
		return
	}

	// If dry run, just validate and return preview
	if req.DryRun {
		result, err := h.templateService.ValidateTemplate(req.TemplateID, req.Variables)
		if err != nil {
			h.logger.Error("Failed to validate template", zap.Error(err))
			c.JSON(http.StatusBadRequest, gin.H{
				"error": err.Error(),
			})
			return
		}

		if !result.Valid {
			c.JSON(http.StatusBadRequest, gin.H{
				"error":      "Template validation failed",
				"validation": result,
			})
			return
		}

		c.JSON(http.StatusOK, gin.H{
			"dry_run": true,
			"preview": result.Preview,
		})
		return
	}

	// Instantiate template
	instance, err := h.templateService.InstantiateTemplate(c.Request.Context(), req.TemplateID, req.Variables, req.TargetSwitch)
	if err != nil {
		h.logger.Error("Failed to instantiate template", zap.Error(err))
		c.JSON(http.StatusBadRequest, gin.H{
			"error": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"instance": instance,
		"message":  "Template instantiated successfully",
	})
}

// ImportTemplate imports a custom policy template
func (h *TemplateHandler) ImportTemplate(c *gin.Context) {
	var data json.RawMessage
	if err := c.ShouldBindJSON(&data); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Invalid JSON data",
		})
		return
	}

	template, err := h.templateService.ImportTemplate(data)
	if err != nil {
		h.logger.Error("Failed to import template", zap.Error(err))
		c.JSON(http.StatusBadRequest, gin.H{
			"error": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"template": template,
		"message":  "Template imported successfully",
	})
}

// ExportTemplate exports a template as JSON
func (h *TemplateHandler) ExportTemplate(c *gin.Context) {
	templateID := c.Param("id")

	data, err := h.templateService.ExportTemplate(templateID)
	if err != nil {
		h.logger.Error("Failed to export template", zap.Error(err))
		c.JSON(http.StatusNotFound, gin.H{
			"error": "Template not found",
		})
		return
	}

	c.Header("Content-Type", "application/json")
	c.Header("Content-Disposition", "attachment; filename="+templateID+".json")
	c.Data(http.StatusOK, "application/json", data)
}