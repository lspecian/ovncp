package handlers

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/lspecian/ovncp/internal/models"
	"github.com/lspecian/ovncp/internal/services"
)

type PortHandler struct {
	ovnService services.OVNServiceInterface
}

func NewPortHandler(ovnService services.OVNServiceInterface) *PortHandler {
	return &PortHandler{
		ovnService: ovnService,
	}
}

func (h *PortHandler) List(c *gin.Context) {
	switchID := c.Param("switchId")
	if switchID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "switch ID is required"})
		return
	}

	ports, err := h.ovnService.ListPorts(c.Request.Context(), switchID)
	if err != nil {
		if strings.Contains(err.Error(), "not found") {
			c.JSON(http.StatusNotFound, gin.H{"error": "switch not found"})
			return
		}
		h.handleError(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"ports": ports,
		"count": len(ports),
	})
}

func (h *PortHandler) Get(c *gin.Context) {
	id := c.Param("id")
	if id == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "port ID is required"})
		return
	}
	
	port, err := h.ovnService.GetPort(c.Request.Context(), id)
	if err != nil {
		if strings.Contains(err.Error(), "not found") {
			c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
			return
		}
		h.handleError(c, err)
		return
	}

	c.JSON(http.StatusOK, port)
}

func (h *PortHandler) Create(c *gin.Context) {
	switchID := c.Param("switchId")
	if switchID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "switch ID is required"})
		return
	}

	var port models.LogicalSwitchPort
	if err := c.ShouldBindJSON(&port); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "invalid request body",
			"details": err.Error(),
		})
		return
	}

	// Validate required fields
	if port.Name == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "validation failed",
			"details": "name is required",
		})
		return
	}

	// Validate name format
	if !isValidName(port.Name) {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "validation failed",
			"details": "name must contain only alphanumeric characters, dashes, and underscores",
		})
		return
	}

	// Validate addresses if provided
	if len(port.Addresses) == 0 {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "validation failed",
			"details": "at least one address is required",
		})
		return
	}

	// Validate addresses format
	for _, addr := range port.Addresses {
		if !isValidAddress(addr) {
			c.JSON(http.StatusBadRequest, gin.H{
				"error": "validation failed",
				"details": "invalid address format: " + addr,
			})
			return
		}
	}

	// Validate port type if provided
	if port.Type != "" && !isValidPortType(port.Type) {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "validation failed",
			"details": "invalid port type: " + port.Type,
		})
		return
	}

	created, err := h.ovnService.CreatePort(c.Request.Context(), switchID, &port)
	if err != nil {
		if strings.Contains(err.Error(), "already exists") {
			c.JSON(http.StatusConflict, gin.H{"error": err.Error()})
			return
		}
		if strings.Contains(err.Error(), "not found") {
			c.JSON(http.StatusNotFound, gin.H{"error": "switch not found"})
			return
		}
		h.handleError(c, err)
		return
	}

	c.JSON(http.StatusCreated, created)
}

func (h *PortHandler) Update(c *gin.Context) {
	id := c.Param("id")
	if id == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "port ID is required"})
		return
	}
	
	var port models.LogicalSwitchPort
	if err := c.ShouldBindJSON(&port); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "invalid request body",
			"details": err.Error(),
		})
		return
	}

	// Validate name if provided
	if port.Name != "" && !isValidName(port.Name) {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "validation failed",
			"details": "name must contain only alphanumeric characters, dashes, and underscores",
		})
		return
	}

	// Validate addresses if provided
	for _, addr := range port.Addresses {
		if !isValidAddress(addr) {
			c.JSON(http.StatusBadRequest, gin.H{
				"error": "validation failed",
				"details": "invalid address format: " + addr,
			})
			return
		}
	}

	// Validate port type if provided
	if port.Type != "" && !isValidPortType(port.Type) {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "validation failed",
			"details": "invalid port type: " + port.Type,
		})
		return
	}

	updated, err := h.ovnService.UpdatePort(c.Request.Context(), id, &port)
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

func (h *PortHandler) Delete(c *gin.Context) {
	id := c.Param("id")
	if id == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "port ID is required"})
		return
	}
	
	err := h.ovnService.DeletePort(c.Request.Context(), id)
	if err != nil {
		if strings.Contains(err.Error(), "not found") {
			c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
			return
		}
		h.handleError(c, err)
		return
	}

	c.JSON(http.StatusNoContent, nil)
}

// handleError handles generic errors
func (h *PortHandler) handleError(c *gin.Context, err error) {
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

// isValidAddress validates address formats (MAC or "dynamic")
func isValidAddress(addr string) bool {
	// Allow "dynamic" as a special address
	if addr == "dynamic" {
		return true
	}

	// Allow "unknown" as a special address
	if addr == "unknown" {
		return true
	}

	// Check if it's a MAC address (simplified validation)
	// Format: XX:XX:XX:XX:XX:XX
	if len(addr) == 17 {
		for i, ch := range addr {
			if i%3 == 2 {
				if ch != ':' {
					return false
				}
			} else {
				if !((ch >= '0' && ch <= '9') || (ch >= 'a' && ch <= 'f') || (ch >= 'A' && ch <= 'F')) {
					return false
				}
			}
		}
		return true
	}

	// Could be an IP address or CIDR
	// Simple validation - just check for basic format
	return strings.Contains(addr, ".") || strings.Contains(addr, ":")
}

// isValidPortType validates OVN port types
func isValidPortType(portType string) bool {
	validTypes := []string{
		"",            // normal port
		"localnet",    // physical network connection
		"localport",   // local port
		"l2gateway",   // L2 gateway port
		"router",      // router attachment
		"vtep",        // VTEP gateway
		"virtual",     // virtual port
	}

	for _, valid := range validTypes {
		if portType == valid {
			return true
		}
	}
	return false
}