package handlers

import (
	"net/http"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/lspecian/ovncp/internal/models"
	"github.com/lspecian/ovncp/internal/services"
)

type ACLHandler struct {
	ovnService services.OVNServiceInterface
}

func NewACLHandler(ovnService services.OVNServiceInterface) *ACLHandler {
	return &ACLHandler{
		ovnService: ovnService,
	}
}

func (h *ACLHandler) List(c *gin.Context) {
	switchID := c.Query("switch_id")
	if switchID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "switch_id query parameter is required"})
		return
	}

	// Pagination parameters
	page := 1
	limit := 20 // Default limit
	
	if pageStr := c.Query("page"); pageStr != "" {
		if p, err := strconv.Atoi(pageStr); err == nil && p > 0 {
			page = p
		}
	}
	
	if limitStr := c.Query("limit"); limitStr != "" {
		if l, err := strconv.Atoi(limitStr); err == nil && l > 0 && l <= 100 {
			limit = l
		}
	}

	acls, err := h.ovnService.ListACLs(c.Request.Context(), switchID)
	if err != nil {
		if strings.Contains(err.Error(), "not found") {
			c.JSON(http.StatusNotFound, gin.H{"error": "switch not found"})
			return
		}
		h.handleError(c, err)
		return
	}

	// Apply pagination
	totalCount := len(acls)
	start := (page - 1) * limit
	end := start + limit
	
	if start >= totalCount {
		c.JSON(http.StatusOK, gin.H{
			"acls":       []*models.ACL{},
			"pagination": gin.H{
				"page":        page,
				"limit":       limit,
				"total_count": totalCount,
				"total_pages": (totalCount + limit - 1) / limit,
			},
		})
		return
	}
	
	if end > totalCount {
		end = totalCount
	}
	
	paginatedACLs := acls[start:end]

	c.JSON(http.StatusOK, gin.H{
		"acls": paginatedACLs,
		"pagination": gin.H{
			"page":        page,
			"limit":       limit,
			"total_count": totalCount,
			"total_pages": (totalCount + limit - 1) / limit,
		},
	})
}

func (h *ACLHandler) Get(c *gin.Context) {
	id := c.Param("id")
	if id == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "ACL ID is required"})
		return
	}
	
	acl, err := h.ovnService.GetACL(c.Request.Context(), id)
	if err != nil {
		if strings.Contains(err.Error(), "not found") {
			c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
			return
		}
		h.handleError(c, err)
		return
	}

	c.JSON(http.StatusOK, acl)
}

func (h *ACLHandler) Create(c *gin.Context) {
	switchID := c.Query("switch_id")
	if switchID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "switch_id query parameter is required"})
		return
	}

	var acl models.ACL
	if err := c.ShouldBindJSON(&acl); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "invalid request body",
			"details": err.Error(),
		})
		return
	}

	// Validate required fields
	if acl.Match == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "validation failed",
			"details": "match expression is required",
		})
		return
	}

	if acl.Action == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "validation failed",
			"details": "action is required",
		})
		return
	}

	if acl.Direction == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "validation failed",
			"details": "direction is required",
		})
		return
	}

	// Validate action
	validActions := []string{"allow", "allow-related", "allow-stateless", "drop", "reject", "pass"}
	isValidAction := false
	for _, valid := range validActions {
		if acl.Action == valid {
			isValidAction = true
			break
		}
	}
	if !isValidAction {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "validation failed",
			"details": "action must be one of: allow, allow-related, allow-stateless, drop, reject, pass",
		})
		return
	}

	// Validate direction
	if acl.Direction != "from-lport" && acl.Direction != "to-lport" {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "validation failed",
			"details": "direction must be 'from-lport' or 'to-lport'",
		})
		return
	}

	// Validate priority
	if acl.Priority < 0 || acl.Priority > 65535 {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "validation failed",
			"details": "priority must be between 0 and 65535",
		})
		return
	}

	// Validate severity if provided
	if acl.Severity != "" {
		validSeverities := []string{"alert", "warning", "notice", "info", "debug"}
		isValidSeverity := false
		for _, valid := range validSeverities {
			if acl.Severity == valid {
				isValidSeverity = true
				break
			}
		}
		if !isValidSeverity {
			c.JSON(http.StatusBadRequest, gin.H{
				"error": "validation failed",
				"details": "severity must be one of: alert, warning, notice, info, debug",
			})
			return
		}
	}

	// TODO: Add match expression syntax validation

	created, err := h.ovnService.CreateACL(c.Request.Context(), switchID, &acl)
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

func (h *ACLHandler) Update(c *gin.Context) {
	id := c.Param("id")
	if id == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "ACL ID is required"})
		return
	}
	
	var acl models.ACL
	if err := c.ShouldBindJSON(&acl); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "invalid request body",
			"details": err.Error(),
		})
		return
	}

	// Validate action if provided
	if acl.Action != "" {
		validActions := []string{"allow", "allow-related", "allow-stateless", "drop", "reject", "pass"}
		isValidAction := false
		for _, valid := range validActions {
			if acl.Action == valid {
				isValidAction = true
				break
			}
		}
		if !isValidAction {
			c.JSON(http.StatusBadRequest, gin.H{
				"error": "validation failed",
				"details": "action must be one of: allow, allow-related, allow-stateless, drop, reject, pass",
			})
			return
		}
	}

	// Validate direction if provided
	if acl.Direction != "" && acl.Direction != "from-lport" && acl.Direction != "to-lport" {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "validation failed",
			"details": "direction must be 'from-lport' or 'to-lport'",
		})
		return
	}

	// Validate priority if provided
	if acl.Priority != 0 && (acl.Priority < 0 || acl.Priority > 65535) {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "validation failed",
			"details": "priority must be between 0 and 65535",
		})
		return
	}

	// Validate severity if provided
	if acl.Severity != "" {
		validSeverities := []string{"alert", "warning", "notice", "info", "debug"}
		isValidSeverity := false
		for _, valid := range validSeverities {
			if acl.Severity == valid {
				isValidSeverity = true
				break
			}
		}
		if !isValidSeverity {
			c.JSON(http.StatusBadRequest, gin.H{
				"error": "validation failed",
				"details": "severity must be one of: alert, warning, notice, info, debug",
			})
			return
		}
	}

	updated, err := h.ovnService.UpdateACL(c.Request.Context(), id, &acl)
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

func (h *ACLHandler) Delete(c *gin.Context) {
	id := c.Param("id")
	if id == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "ACL ID is required"})
		return
	}
	
	err := h.ovnService.DeleteACL(c.Request.Context(), id)
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
func (h *ACLHandler) handleError(c *gin.Context, err error) {
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