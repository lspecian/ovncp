package handlers

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/lspecian/ovncp/internal/models"
	"github.com/lspecian/ovncp/internal/services"
)

type SwitchHandler struct {
	ovnService services.OVNServiceInterface
}

func NewSwitchHandler(ovnService services.OVNServiceInterface) *SwitchHandler {
	return &SwitchHandler{
		ovnService: ovnService,
	}
}

func (h *SwitchHandler) List(c *gin.Context) {
	switches, err := h.ovnService.ListLogicalSwitches(c.Request.Context())
	if err != nil {
		h.handleError(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"switches": switches,
		"count":    len(switches),
	})
}

func (h *SwitchHandler) Get(c *gin.Context) {
	id := c.Param("id")
	if id == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "switch ID is required"})
		return
	}
	
	sw, err := h.ovnService.GetLogicalSwitch(c.Request.Context(), id)
	if err != nil {
		if strings.Contains(err.Error(), "not found") {
			c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
			return
		}
		h.handleError(c, err)
		return
	}

	c.JSON(http.StatusOK, sw)
}

func (h *SwitchHandler) Create(c *gin.Context) {
	var sw models.LogicalSwitch
	if err := c.ShouldBindJSON(&sw); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "invalid request body",
			"details": err.Error(),
		})
		return
	}

	// Validate required fields
	if sw.Name == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "validation failed",
			"details": "name is required",
		})
		return
	}

	// Validate name format (alphanumeric, dash, underscore)
	if !isValidName(sw.Name) {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "validation failed",
			"details": "name must contain only alphanumeric characters, dashes, and underscores",
		})
		return
	}

	created, err := h.ovnService.CreateLogicalSwitch(c.Request.Context(), &sw)
	if err != nil {
		if strings.Contains(err.Error(), "already exists") {
			c.JSON(http.StatusConflict, gin.H{"error": err.Error()})
			return
		}
		h.handleError(c, err)
		return
	}

	c.JSON(http.StatusCreated, created)
}

func (h *SwitchHandler) Update(c *gin.Context) {
	id := c.Param("id")
	if id == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "switch ID is required"})
		return
	}
	
	var sw models.LogicalSwitch
	if err := c.ShouldBindJSON(&sw); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "invalid request body",
			"details": err.Error(),
		})
		return
	}

	// Validate name if provided
	if sw.Name != "" && !isValidName(sw.Name) {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "validation failed",
			"details": "name must contain only alphanumeric characters, dashes, and underscores",
		})
		return
	}

	updated, err := h.ovnService.UpdateLogicalSwitch(c.Request.Context(), id, &sw)
	if err != nil {
		if strings.Contains(err.Error(), "not found") {
			c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
			return
		}
		h.handleError(c, err)
		return
	}

	c.JSON(http.StatusOK, updated)
}

func (h *SwitchHandler) Delete(c *gin.Context) {
	id := c.Param("id")
	if id == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "switch ID is required"})
		return
	}
	
	err := h.ovnService.DeleteLogicalSwitch(c.Request.Context(), id)
	if err != nil {
		if strings.Contains(err.Error(), "not found") {
			c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
			return
		}
		if strings.Contains(err.Error(), "in use") {
			c.JSON(http.StatusConflict, gin.H{
				"error": "cannot delete switch",
				"details": "switch has associated ports or resources",
			})
			return
		}
		h.handleError(c, err)
		return
	}

	c.JSON(http.StatusNoContent, nil)
}

// handleError handles generic errors
func (h *SwitchHandler) handleError(c *gin.Context, err error) {
	// Check if client is not connected
	if strings.Contains(err.Error(), "not connected") {
		c.JSON(http.StatusServiceUnavailable, gin.H{
			"error": "OVN service unavailable",
			"details": "unable to connect to OVN northbound database",
		})
		return
	}

	// Default error response
	c.JSON(http.StatusInternalServerError, gin.H{
		"error": "internal server error",
		"details": err.Error(),
	})
}

