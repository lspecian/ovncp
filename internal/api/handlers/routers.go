package handlers

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/lspecian/ovncp/internal/models"
	"github.com/lspecian/ovncp/internal/services"
)

type RouterHandler struct {
	ovnService services.OVNServiceInterface
}

func NewRouterHandler(ovnService services.OVNServiceInterface) *RouterHandler {
	return &RouterHandler{
		ovnService: ovnService,
	}
}

func (h *RouterHandler) List(c *gin.Context) {
	routers, err := h.ovnService.ListLogicalRouters(c.Request.Context())
	if err != nil {
		h.handleError(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"routers": routers,
		"count":   len(routers),
	})
}

func (h *RouterHandler) Get(c *gin.Context) {
	id := c.Param("id")
	if id == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "router ID is required"})
		return
	}
	
	router, err := h.ovnService.GetLogicalRouter(c.Request.Context(), id)
	if err != nil {
		if strings.Contains(err.Error(), "not found") {
			c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
			return
		}
		h.handleError(c, err)
		return
	}

	c.JSON(http.StatusOK, router)
}

func (h *RouterHandler) Create(c *gin.Context) {
	var router models.LogicalRouter
	if err := c.ShouldBindJSON(&router); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "invalid request body",
			"details": err.Error(),
		})
		return
	}

	// Validate required fields
	if router.Name == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "validation failed",
			"details": "name is required",
		})
		return
	}

	// Validate name format (alphanumeric, dash, underscore)
	if !isValidName(router.Name) {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "validation failed",
			"details": "name must contain only alphanumeric characters, dashes, and underscores",
		})
		return
	}

	// Validate static routes if provided
	for _, route := range router.StaticRoutes {
		if route.IPPrefix == "" || route.Nexthop == "" {
			c.JSON(http.StatusBadRequest, gin.H{
				"error": "validation failed",
				"details": "static routes must have ip_prefix and nexthop",
			})
			return
		}
		
		// Validate policy if provided
		if route.Policy != nil && *route.Policy != "" {
			validPolicies := []string{"dst-ip", "src-ip"}
			isValid := false
			for _, valid := range validPolicies {
				if *route.Policy == valid {
					isValid = true
					break
				}
			}
			if !isValid {
				c.JSON(http.StatusBadRequest, gin.H{
					"error": "validation failed",
					"details": "static route policy must be 'dst-ip' or 'src-ip'",
				})
				return
			}
		}
	}

	created, err := h.ovnService.CreateLogicalRouter(c.Request.Context(), &router)
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

func (h *RouterHandler) Update(c *gin.Context) {
	id := c.Param("id")
	if id == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "router ID is required"})
		return
	}
	
	var router models.LogicalRouter
	if err := c.ShouldBindJSON(&router); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "invalid request body",
			"details": err.Error(),
		})
		return
	}

	// Validate name if provided
	if router.Name != "" && !isValidName(router.Name) {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "validation failed",
			"details": "name must contain only alphanumeric characters, dashes, and underscores",
		})
		return
	}

	updated, err := h.ovnService.UpdateLogicalRouter(c.Request.Context(), id, &router)
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

func (h *RouterHandler) Delete(c *gin.Context) {
	id := c.Param("id")
	if id == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "router ID is required"})
		return
	}
	
	err := h.ovnService.DeleteLogicalRouter(c.Request.Context(), id)
	if err != nil {
		if strings.Contains(err.Error(), "not found") {
			c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
			return
		}
		if strings.Contains(err.Error(), "has") && strings.Contains(err.Error(), "ports") {
			c.JSON(http.StatusConflict, gin.H{
				"error": "cannot delete router",
				"details": err.Error(),
			})
			return
		}
		h.handleError(c, err)
		return
	}

	c.JSON(http.StatusNoContent, nil)
}

// handleError handles generic errors
func (h *RouterHandler) handleError(c *gin.Context, err error) {
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

