package handlers

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/lspecian/ovncp/internal/services"
)

// TopologyHandler handles topology-related requests
type TopologyHandler struct {
	service services.OVNServiceInterface
}

// NewTopologyHandler creates a new topology handler
func NewTopologyHandler(service services.OVNServiceInterface) *TopologyHandler {
	return &TopologyHandler{
		service: service,
	}
}

// GetTopology handles GET /api/v1/topology
func (h *TopologyHandler) GetTopology(c *gin.Context) {
	ctx := c.Request.Context()

	topology, err := h.service.GetTopology(ctx)
	if err != nil {
		h.handleError(c, err)
		return
	}

	c.JSON(http.StatusOK, topology)
}

// handleError handles generic errors
func (h *TopologyHandler) handleError(c *gin.Context, err error) {
	c.JSON(http.StatusInternalServerError, gin.H{
		"error": "internal server error",
		"details": err.Error(),
	})
}