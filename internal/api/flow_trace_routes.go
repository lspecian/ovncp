package api

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/lspecian/ovncp/internal/services"
	"github.com/lspecian/ovncp/pkg/ovn"
	"go.uber.org/zap"
)

// FlowTraceHandler handles flow tracing endpoints
type FlowTraceHandler struct {
	traceService *services.FlowTraceService
	ovnService   *services.OVNService
	logger       *zap.Logger
}

// NewFlowTraceHandler creates a new flow trace handler
func NewFlowTraceHandler(ovnClient *ovn.Client, ovnService *services.OVNService, logger *zap.Logger) *FlowTraceHandler {
	traceService := services.NewFlowTraceService(ovnClient, logger)
	return &FlowTraceHandler{
		traceService: traceService,
		ovnService:   ovnService,
		logger:       logger,
	}
}

// RegisterFlowTraceRoutes registers flow trace routes
func (h *FlowTraceHandler) RegisterFlowTraceRoutes(router *gin.RouterGroup) {
	trace := router.Group("/trace")
	{
		trace.POST("/flow", h.traceFlow)
		trace.POST("/multi-path", h.traceMultiplePaths)
		trace.POST("/connectivity", h.analyzeConnectivity)
		trace.GET("/ports/:port/addresses", h.getPortAddresses)
		trace.POST("/simulate", h.simulateFlow)
	}
}

// traceFlow handles flow trace requests
func (h *FlowTraceHandler) traceFlow(c *gin.Context) {
	var req ovn.FlowTraceRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Invalid request: " + err.Error(),
		})
		return
	}

	ctx := c.Request.Context()

	// Validate source port exists
	if req.SourcePort != "" {
		port, err := h.ovnService.GetPort(ctx, req.SourcePort)
		if err != nil || port == nil {
			c.JSON(http.StatusBadRequest, gin.H{
				"error": "Source port not found",
			})
			return
		}
		
		// Auto-fill MAC if not provided
		if req.SourceMAC == "" && port.MAC != "" {
			req.SourceMAC = port.MAC
		}
	}

	// Perform trace
	result, err := h.traceService.TraceFlow(ctx, &req)
	if err != nil {
		h.logger.Error("Flow trace failed", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Flow trace failed: " + err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, result)
}

// traceMultiplePaths handles multi-path trace requests
func (h *FlowTraceHandler) traceMultiplePaths(c *gin.Context) {
	var req services.MultiPathTraceRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Invalid request: " + err.Error(),
		})
		return
	}

	ctx := c.Request.Context()

	// Perform multi-path trace
	result, err := h.traceService.TraceMultiplePaths(ctx, &req)
	if err != nil {
		h.logger.Error("Multi-path trace failed", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Multi-path trace failed: " + err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, result)
}

// analyzeConnectivity handles connectivity analysis requests
func (h *FlowTraceHandler) analyzeConnectivity(c *gin.Context) {
	var req services.ConnectivityAnalysisRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Invalid request: " + err.Error(),
		})
		return
	}

	ctx := c.Request.Context()

	// Perform connectivity analysis
	result, err := h.traceService.AnalyzeConnectivity(ctx, &req)
	if err != nil {
		h.logger.Error("Connectivity analysis failed", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Connectivity analysis failed: " + err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, result)
}

// getPortAddresses returns the addresses for a port
func (h *FlowTraceHandler) getPortAddresses(c *gin.Context) {
	portID := c.Param("port")
	ctx := c.Request.Context()

	// Get port details
	port, err := h.ovnService.GetPort(ctx, portID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{
			"error": "Port not found",
		})
		return
	}

	// Extract addresses
	addresses := make([]map[string]string, 0)
	for _, addr := range port.Addresses {
		// Parse "MAC IP" format
		parts := strings.Fields(addr)
		if len(parts) >= 2 {
			addresses = append(addresses, map[string]string{
				"mac": parts[0],
				"ip":  parts[1],
			})
		} else if len(parts) == 1 && strings.Contains(parts[0], ":") {
			// Just MAC
			addresses = append(addresses, map[string]string{
				"mac": parts[0],
			})
		}
	}

	c.JSON(http.StatusOK, gin.H{
		"port_id":   port.UUID,
		"port_name": port.Name,
		"addresses": addresses,
		"type":      port.Type,
	})
}

// simulateFlow handles flow simulation requests (for testing)
func (h *FlowTraceHandler) simulateFlow(c *gin.Context) {
	var req ovn.FlowTraceRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Invalid request: " + err.Error(),
		})
		return
	}

	ctx := c.Request.Context()

	// Use simulation method
	result, err := h.ovnService.GetOVNClient().SimulateFlowTrace(ctx, &req)
	if err != nil {
		h.logger.Error("Flow simulation failed", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Flow simulation failed: " + err.Error(),
		})
		return
	}

	// Mark as simulation
	result.Summary = "[SIMULATION] " + result.Summary

	c.JSON(http.StatusOK, result)
}