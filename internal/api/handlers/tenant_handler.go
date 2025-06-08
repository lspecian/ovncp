package handlers

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/lspecian/ovncp/internal/models"
	"github.com/lspecian/ovncp/internal/services"
	"go.uber.org/zap"
)

type TenantHandler struct {
	tenantService *services.TenantService
	logger        *zap.Logger
}

func NewTenantHandler(tenantService *services.TenantService, logger *zap.Logger) *TenantHandler {
	return &TenantHandler{
		tenantService: tenantService,
		logger:        logger,
	}
}

// CreateTenantRequest represents a tenant creation request
type CreateTenantRequest struct {
	Name        string                 `json:"name" binding:"required"`
	DisplayName string                 `json:"display_name"`
	Description string                 `json:"description"`
	Type        models.TenantType      `json:"type"`
	Parent      string                 `json:"parent,omitempty"`
	Settings    models.TenantSettings  `json:"settings"`
	Quotas      models.TenantQuotas    `json:"quotas"`
	Metadata    map[string]string      `json:"metadata"`
}

// CreateTenant creates a new tenant
func (h *TenantHandler) CreateTenant(c *gin.Context) {
	var req CreateTenantRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Invalid request body",
		})
		return
	}

	userID, _ := c.Get("user_id")

	tenant := &models.Tenant{
		Name:        req.Name,
		DisplayName: req.DisplayName,
		Description: req.Description,
		Type:        req.Type,
		Settings:    req.Settings,
		Quotas:      req.Quotas,
		Metadata:    req.Metadata,
	}

	if req.Parent != "" {
		tenant.Parent = &req.Parent
	}

	created, err := h.tenantService.CreateTenant(c.Request.Context(), tenant, userID.(string))
	if err != nil {
		h.logger.Error("Failed to create tenant", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": err.Error(),
		})
		return
	}

	c.JSON(http.StatusCreated, created)
}

// ListTenants lists tenants accessible to the user
func (h *TenantHandler) ListTenants(c *gin.Context) {
	userID, _ := c.Get("user_id")
	
	filter := &models.TenantFilter{
		UserID: userID.(string),
	}

	// Apply query filters
	if tenantType := c.Query("type"); tenantType != "" {
		filter.Type = models.TenantType(tenantType)
	}
	if status := c.Query("status"); status != "" {
		filter.Status = models.TenantStatus(status)
	}
	if parent := c.Query("parent"); parent != "" {
		filter.Parent = parent
	}

	tenants, err := h.tenantService.ListTenants(c.Request.Context(), filter)
	if err != nil {
		h.logger.Error("Failed to list tenants", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to list tenants",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"tenants": tenants,
		"total":   len(tenants),
	})
}

// GetTenant retrieves a specific tenant
func (h *TenantHandler) GetTenant(c *gin.Context) {
	tenantID := c.Param("id")

	tenant, err := h.tenantService.GetTenant(c.Request.Context(), tenantID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{
			"error": "Tenant not found",
		})
		return
	}

	// Check access
	userID, _ := c.Get("user_id")
	membership, err := h.tenantService.GetMembership(c.Request.Context(), tenantID, userID.(string))
	if err != nil || membership == nil {
		c.JSON(http.StatusForbidden, gin.H{
			"error": "Access denied",
		})
		return
	}

	c.JSON(http.StatusOK, tenant)
}

// UpdateTenantRequest represents a tenant update request
type UpdateTenantRequest struct {
	DisplayName string                 `json:"display_name,omitempty"`
	Description string                 `json:"description,omitempty"`
	Status      models.TenantStatus    `json:"status,omitempty"`
	Settings    models.TenantSettings  `json:"settings,omitempty"`
	Quotas      models.TenantQuotas    `json:"quotas,omitempty"`
	Metadata    map[string]string      `json:"metadata,omitempty"`
}

// UpdateTenant updates a tenant
func (h *TenantHandler) UpdateTenant(c *gin.Context) {
	tenantID := c.Param("id")

	var req UpdateTenantRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Invalid request body",
		})
		return
	}

	updates := &models.Tenant{
		DisplayName: req.DisplayName,
		Description: req.Description,
		Status:      req.Status,
		Settings:    req.Settings,
		Quotas:      req.Quotas,
		Metadata:    req.Metadata,
	}

	updated, err := h.tenantService.UpdateTenant(c.Request.Context(), tenantID, updates)
	if err != nil {
		h.logger.Error("Failed to update tenant", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, updated)
}

// DeleteTenant deletes a tenant
func (h *TenantHandler) DeleteTenant(c *gin.Context) {
	tenantID := c.Param("id")

	if err := h.tenantService.DeleteTenant(c.Request.Context(), tenantID); err != nil {
		h.logger.Error("Failed to delete tenant", zap.Error(err))
		c.JSON(http.StatusBadRequest, gin.H{
			"error": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "Tenant marked for deletion",
	})
}

// GetResourceUsage returns resource usage for a tenant
func (h *TenantHandler) GetResourceUsage(c *gin.Context) {
	tenantID := c.Param("id")

	usage, err := h.tenantService.GetResourceUsage(c.Request.Context(), tenantID)
	if err != nil {
		h.logger.Error("Failed to get resource usage", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to get resource usage",
		})
		return
	}

	// Get tenant for quotas
	tenant, err := h.tenantService.GetTenant(c.Request.Context(), tenantID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{
			"error": "Tenant not found",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"usage":  usage,
		"quotas": tenant.Quotas,
	})
}

// Member management

// AddMemberRequest represents a request to add a member
type AddMemberRequest struct {
	UserID string `json:"user_id" binding:"required"`
	Role   string `json:"role" binding:"required"`
}

// AddMember adds a member to a tenant
func (h *TenantHandler) AddMember(c *gin.Context) {
	tenantID := c.Param("id")
	
	var req AddMemberRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Invalid request body",
		})
		return
	}

	addedBy, _ := c.Get("user_id")

	if err := h.tenantService.AddMember(c.Request.Context(), tenantID, req.UserID, req.Role, addedBy.(string)); err != nil {
		h.logger.Error("Failed to add member", zap.Error(err))
		c.JSON(http.StatusBadRequest, gin.H{
			"error": err.Error(),
		})
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"message": "Member added successfully",
	})
}

// RemoveMember removes a member from a tenant
func (h *TenantHandler) RemoveMember(c *gin.Context) {
	tenantID := c.Param("id")
	userID := c.Param("user_id")

	if err := h.tenantService.RemoveMember(c.Request.Context(), tenantID, userID); err != nil {
		h.logger.Error("Failed to remove member", zap.Error(err))
		c.JSON(http.StatusBadRequest, gin.H{
			"error": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "Member removed successfully",
	})
}

// UpdateMemberRole updates a member's role
func (h *TenantHandler) UpdateMemberRole(c *gin.Context) {
	tenantID := c.Param("id")
	userID := c.Param("user_id")

	var req struct {
		Role string `json:"role" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Invalid request body",
		})
		return
	}

	if err := h.tenantService.UpdateMemberRole(c.Request.Context(), tenantID, userID, req.Role); err != nil {
		h.logger.Error("Failed to update member role", zap.Error(err))
		c.JSON(http.StatusBadRequest, gin.H{
			"error": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "Role updated successfully",
	})
}

// ListMembers lists tenant members
func (h *TenantHandler) ListMembers(c *gin.Context) {
	tenantID := c.Param("id")

	members, err := h.tenantService.ListMembers(c.Request.Context(), tenantID)
	if err != nil {
		h.logger.Error("Failed to list members", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to list members",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"members": members,
		"total":   len(members),
	})
}

// Invitation management

// CreateInvitationRequest represents an invitation request
type CreateInvitationRequest struct {
	Email string `json:"email" binding:"required,email"`
	Role  string `json:"role" binding:"required"`
}

// CreateInvitation creates a tenant invitation
func (h *TenantHandler) CreateInvitation(c *gin.Context) {
	tenantID := c.Param("id")
	
	var req CreateInvitationRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Invalid request body",
		})
		return
	}

	createdBy, _ := c.Get("user_id")

	invitation, err := h.tenantService.CreateInvitation(c.Request.Context(), tenantID, req.Email, req.Role, createdBy.(string))
	if err != nil {
		h.logger.Error("Failed to create invitation", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": err.Error(),
		})
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"invitation": invitation,
		"message":    "Invitation created successfully",
	})
}

// AcceptInvitation accepts a tenant invitation
func (h *TenantHandler) AcceptInvitation(c *gin.Context) {
	token := c.Param("token")
	userID, _ := c.Get("user_id")

	if err := h.tenantService.AcceptInvitation(c.Request.Context(), token, userID.(string)); err != nil {
		h.logger.Error("Failed to accept invitation", zap.Error(err))
		c.JSON(http.StatusBadRequest, gin.H{
			"error": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "Invitation accepted successfully",
	})
}

// API Key management

// CreateAPIKeyRequest represents an API key creation request
type CreateAPIKeyRequest struct {
	Name        string   `json:"name" binding:"required"`
	Description string   `json:"description"`
	Scopes      []string `json:"scopes"`
	ExpiresIn   int      `json:"expires_in"` // Days
}

// CreateAPIKey creates a new API key for a tenant
func (h *TenantHandler) CreateAPIKey(c *gin.Context) {
	tenantID := c.Param("id")
	
	var req CreateAPIKeyRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Invalid request body",
		})
		return
	}

	createdBy, _ := c.Get("user_id")

	key := &models.TenantAPIKey{
		Name:        req.Name,
		Description: req.Description,
		Scopes:      req.Scopes,
		CreatedBy:   createdBy.(string),
	}

	// Set expiration if specified
	if req.ExpiresIn > 0 {
		expiresAt := time.Now().Add(time.Duration(req.ExpiresIn) * 24 * time.Hour)
		key.ExpiresAt = &expiresAt
	}

	apiKey, err := h.tenantService.CreateAPIKey(c.Request.Context(), tenantID, key)
	if err != nil {
		h.logger.Error("Failed to create API key", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": err.Error(),
		})
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"key":     apiKey,
		"message": "API key created successfully. Please save the key, it won't be shown again.",
	})
}

// ListAPIKeys lists API keys for a tenant
func (h *TenantHandler) ListAPIKeys(c *gin.Context) {
	tenantID := c.Param("id")

	keys, err := h.tenantService.ListAPIKeys(c.Request.Context(), tenantID)
	if err != nil {
		h.logger.Error("Failed to list API keys", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to list API keys",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"keys":  keys,
		"total": len(keys),
	})
}

// DeleteAPIKey deletes an API key
func (h *TenantHandler) DeleteAPIKey(c *gin.Context) {
	tenantID := c.Param("id")
	keyID := c.Param("key_id")

	if err := h.tenantService.DeleteAPIKey(c.Request.Context(), tenantID, keyID); err != nil {
		h.logger.Error("Failed to delete API key", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "API key deleted successfully",
	})
}