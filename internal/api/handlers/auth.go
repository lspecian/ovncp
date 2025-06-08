package handlers

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	
	"github.com/lspecian/ovncp/internal/api/middleware"
	"github.com/lspecian/ovncp/internal/auth"
	"github.com/lspecian/ovncp/internal/models"
)

type AuthHandler struct {
	authService auth.Service
}

func NewAuthHandler(authService auth.Service) *AuthHandler {
	return &AuthHandler{
		authService: authService,
	}
}

// LoginRequest represents the OAuth login request
type LoginRequest struct {
	Provider string `json:"provider" binding:"required"`
}

// LoginResponse contains the OAuth authorization URL
type LoginResponse struct {
	AuthURL string `json:"auth_url"`
}

// CallbackRequest represents the OAuth callback parameters
type CallbackRequest struct {
	Code  string `form:"code" binding:"required"`
	State string `form:"state" binding:"required"`
}

// TokenResponse contains the authentication tokens
type TokenResponse struct {
	AccessToken  string       `json:"access_token"`
	RefreshToken string       `json:"refresh_token"`
	ExpiresAt    int64        `json:"expires_at"`
	User         *models.User `json:"user"`
}

// RefreshRequest contains the refresh token
type RefreshRequest struct {
	RefreshToken string `json:"refresh_token" binding:"required"`
}

// UpdateRoleRequest contains the new role for a user
type UpdateRoleRequest struct {
	Role string `json:"role" binding:"required,oneof=admin operator viewer"`
}

// Login initiates the OAuth flow
func (h *AuthHandler) Login(c *gin.Context) {
	var req LoginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	
	// Generate state for CSRF protection
	state := uuid.New().String()
	
	// Store state in session/cache (in production, use proper session storage)
	// For now, we'll pass it through the OAuth flow
	
	authURL, err := h.authService.GetAuthURL(req.Provider, state)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	
	c.JSON(http.StatusOK, LoginResponse{
		AuthURL: authURL,
	})
}

// Callback handles the OAuth callback
func (h *AuthHandler) Callback(c *gin.Context) {
	provider := c.Param("provider")
	
	var req CallbackRequest
	if err := c.ShouldBindQuery(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	
	// In production, verify state against stored value for CSRF protection
	
	// Exchange code for token
	session, err := h.authService.ExchangeCode(c.Request.Context(), provider, req.Code)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": err.Error()})
		return
	}
	
	c.JSON(http.StatusOK, TokenResponse{
		AccessToken:  session.AccessToken,
		RefreshToken: session.RefreshToken,
		ExpiresAt:    session.ExpiresAt.Unix(),
		User:         session.User,
	})
}

// Refresh exchanges a refresh token for new tokens
func (h *AuthHandler) Refresh(c *gin.Context) {
	var req RefreshRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	
	session, err := h.authService.RefreshToken(c.Request.Context(), req.RefreshToken)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": err.Error()})
		return
	}
	
	c.JSON(http.StatusOK, TokenResponse{
		AccessToken:  session.AccessToken,
		RefreshToken: session.RefreshToken,
		ExpiresAt:    session.ExpiresAt.Unix(),
		User:         session.User,
	})
}

// Logout invalidates the current session
func (h *AuthHandler) Logout(c *gin.Context) {
	user, ok := middleware.GetAuthUser(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Not authenticated"})
		return
	}
	
	// Get token from header
	authHeader := c.GetHeader("Authorization")
	if len(authHeader) > 7 {
		token := authHeader[7:] // Remove "Bearer " prefix
		if err := h.authService.Logout(c.Request.Context(), token); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
	}
	
	c.JSON(http.StatusOK, gin.H{"message": "Logged out successfully", "user_id": user.ID})
}

// GetProfile returns the current user's profile
func (h *AuthHandler) GetProfile(c *gin.Context) {
	user, ok := middleware.GetAuthUser(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Not authenticated"})
		return
	}
	
	c.JSON(http.StatusOK, user)
}

// ListUsers returns a paginated list of users (admin only)
func (h *AuthHandler) ListUsers(c *gin.Context) {
	// Parse pagination
	limit, offset := parsePagination(c)
	
	users, total, err := h.authService.ListUsers(c.Request.Context(), limit, offset)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	
	c.JSON(http.StatusOK, gin.H{
		"users": users,
		"total": total,
		"limit": limit,
		"offset": offset,
	})
}

// GetUser returns a specific user (admin only)
func (h *AuthHandler) GetUser(c *gin.Context) {
	userID := c.Param("id")
	
	user, err := h.authService.GetUser(c.Request.Context(), userID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		return
	}
	
	c.JSON(http.StatusOK, user)
}

// UpdateUserRole updates a user's role (admin only)
func (h *AuthHandler) UpdateUserRole(c *gin.Context) {
	userID := c.Param("id")
	
	var req UpdateRoleRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	
	// Prevent self role change
	authUser, _ := middleware.GetAuthUser(c)
	if authUser.ID == userID {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Cannot change your own role"})
		return
	}
	
	role := models.UserRole(req.Role)
	if err := h.authService.UpdateUserRole(c.Request.Context(), userID, role); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	
	c.JSON(http.StatusOK, gin.H{"message": "Role updated successfully"})
}

// DeactivateUser deactivates a user account (admin only)
func (h *AuthHandler) DeactivateUser(c *gin.Context) {
	userID := c.Param("id")
	
	// Prevent self deactivation
	authUser, _ := middleware.GetAuthUser(c)
	if authUser.ID == userID {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Cannot deactivate your own account"})
		return
	}
	
	if err := h.authService.DeactivateUser(c.Request.Context(), userID); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	
	c.JSON(http.StatusOK, gin.H{"message": "User deactivated successfully"})
}