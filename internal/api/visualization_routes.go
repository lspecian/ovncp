package api

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/lspecian/ovncp/internal/services"
	"github.com/lspecian/ovncp/internal/visualization"
	"go.uber.org/zap"
)

// VisualizationHandler handles topology visualization endpoints
type VisualizationHandler struct {
	service *services.OVNService
	logger  *zap.Logger
}

// NewVisualizationHandler creates a new visualization handler
func NewVisualizationHandler(service *services.OVNService, logger *zap.Logger) *VisualizationHandler {
	return &VisualizationHandler{
		service: service,
		logger:  logger,
	}
}

// RegisterVisualizationRoutes registers visualization routes
func (h *VisualizationHandler) RegisterVisualizationRoutes(router *gin.RouterGroup) {
	viz := router.Group("/visualization")
	{
		viz.GET("/topology", h.getTopologyVisualization)
		viz.GET("/topology/export", h.exportTopology)
		viz.POST("/topology/custom", h.getCustomTopology)
		viz.GET("/topology/node/:id", h.getNodeDetails)
		viz.GET("/topology/path/:source/:target", h.getPath)
	}
}

// getTopologyVisualization returns the network topology visualization
func (h *VisualizationHandler) getTopologyVisualization(c *gin.Context) {
	ctx := c.Request.Context()

	// Parse query parameters
	options := h.parseVisualizationOptions(c)

	// Get topology from service
	topology, err := h.service.GetTopology(ctx)
	if err != nil {
		h.logger.Error("Failed to get topology", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to retrieve topology",
		})
		return
	}

	// Create visualizer
	visualizer := visualization.NewTopologyVisualizer(topology)

	// Generate graph
	graph, err := visualizer.GenerateGraph(options)
	if err != nil {
		h.logger.Error("Failed to generate visualization", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to generate visualization",
		})
		return
	}

	c.JSON(http.StatusOK, graph)
}

// exportTopology exports the topology in various formats
func (h *VisualizationHandler) exportTopology(c *gin.Context) {
	ctx := c.Request.Context()

	// Get format parameter
	format := c.DefaultQuery("format", "json")

	// Parse visualization options
	options := h.parseVisualizationOptions(c)

	// Get topology
	topology, err := h.service.GetTopology(ctx)
	if err != nil {
		h.logger.Error("Failed to get topology", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to retrieve topology",
		})
		return
	}

	// Generate graph
	visualizer := visualization.NewTopologyVisualizer(topology)
	graph, err := visualizer.GenerateGraph(options)
	if err != nil {
		h.logger.Error("Failed to generate visualization", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to generate visualization",
		})
		return
	}

	// Export to requested format
	exporter := visualization.NewExporter(graph)
	data, err := exporter.Export(format)
	if err != nil {
		h.logger.Error("Failed to export topology", zap.Error(err))
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Unsupported export format: " + format,
		})
		return
	}

	// Set appropriate content type
	contentType := h.getContentType(format)
	filename := "topology." + h.getFileExtension(format)

	c.Header("Content-Disposition", "attachment; filename="+filename)
	c.Data(http.StatusOK, contentType, data)
}

// getCustomTopology generates topology with custom options
func (h *VisualizationHandler) getCustomTopology(c *gin.Context) {
	ctx := c.Request.Context()

	// Parse custom options from request body
	var options visualization.VisualizationOptions
	if err := c.ShouldBindJSON(&options); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Invalid visualization options",
		})
		return
	}

	// Get topology
	topology, err := h.service.GetTopology(ctx)
	if err != nil {
		h.logger.Error("Failed to get topology", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to retrieve topology",
		})
		return
	}

	// Generate graph with custom options
	visualizer := visualization.NewTopologyVisualizer(topology)
	graph, err := visualizer.GenerateGraph(&options)
	if err != nil {
		h.logger.Error("Failed to generate visualization", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to generate visualization",
		})
		return
	}

	c.JSON(http.StatusOK, graph)
}

// getNodeDetails returns detailed information about a specific node
func (h *VisualizationHandler) getNodeDetails(c *gin.Context) {
	ctx := c.Request.Context()
	nodeID := c.Param("id")

	// Get topology
	topology, err := h.service.GetTopology(ctx)
	if err != nil {
		h.logger.Error("Failed to get topology", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to retrieve topology",
		})
		return
	}

	// Find node details
	nodeDetails := h.findNodeDetails(topology, nodeID)
	if nodeDetails == nil {
		c.JSON(http.StatusNotFound, gin.H{
			"error": "Node not found",
		})
		return
	}

	c.JSON(http.StatusOK, nodeDetails)
}

// getPath finds the path between two nodes
func (h *VisualizationHandler) getPath(c *gin.Context) {
	ctx := c.Request.Context()
	source := c.Param("source")
	target := c.Param("target")

	// Get topology
	topology, err := h.service.GetTopology(ctx)
	if err != nil {
		h.logger.Error("Failed to get topology", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to retrieve topology",
		})
		return
	}

	// Find path between nodes
	path := h.findPath(topology, source, target)
	if len(path) == 0 {
		c.JSON(http.StatusNotFound, gin.H{
			"error": "No path found between nodes",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"source": source,
		"target": target,
		"path":   path,
		"hops":   len(path) - 1,
	})
}

// parseVisualizationOptions parses visualization options from query parameters
func (h *VisualizationHandler) parseVisualizationOptions(c *gin.Context) *visualization.VisualizationOptions {
	options := visualization.DefaultVisualizationOptions()

	// Layout
	if layout := c.Query("layout"); layout != "" {
		options.Layout = layout
	}

	// Detail level
	if detail := c.Query("detail"); detail != "" {
		switch detail {
		case "minimal":
			options.DetailLevel = visualization.DetailLevelMinimal
		case "medium":
			options.DetailLevel = visualization.DetailLevelMedium
		case "full":
			options.DetailLevel = visualization.DetailLevelFull
		}
	}

	// Component filters
	options.IncludeSwitches = h.parseBool(c.Query("switches"), true)
	options.IncludeRouters = h.parseBool(c.Query("routers"), true)
	options.IncludePorts = h.parseBool(c.Query("ports"), true)
	options.IncludeLoadBalancers = h.parseBool(c.Query("loadbalancers"), true)
	options.IncludeACLs = h.parseBool(c.Query("acls"), false)
	options.IncludeNAT = h.parseBool(c.Query("nat"), false)

	// Visual options
	options.ShowLabels = h.parseBool(c.Query("labels"), true)
	options.ShowIcons = h.parseBool(c.Query("icons"), true)
	options.AnimateTraffic = h.parseBool(c.Query("animate"), false)
	options.GroupNodes = h.parseBool(c.Query("group"), true)

	// Performance options
	if maxNodes := c.Query("maxNodes"); maxNodes != "" {
		if n, err := strconv.Atoi(maxNodes); err == nil {
			options.MaxNodes = n
		}
	}

	options.SimplifyPorts = h.parseBool(c.Query("simplifyPorts"), true)
	options.AggregateACLs = h.parseBool(c.Query("aggregateACLs"), true)

	// Filters
	options.FilterByName = c.Query("name")

	return options
}

// parseBool parses a boolean query parameter
func (h *VisualizationHandler) parseBool(value string, defaultValue bool) bool {
	if value == "" {
		return defaultValue
	}
	return value == "true" || value == "1" || value == "yes"
}

// getContentType returns the content type for an export format
func (h *VisualizationHandler) getContentType(format string) string {
	contentTypes := map[string]string{
		"json":      "application/json",
		"dot":       "text/vnd.graphviz",
		"graphviz":  "text/vnd.graphviz",
		"cytoscape": "application/json",
		"d3":        "application/json",
		"mermaid":   "text/plain",
		"html":      "text/html",
	}

	if ct, ok := contentTypes[format]; ok {
		return ct
	}
	return "application/octet-stream"
}

// getFileExtension returns the file extension for an export format
func (h *VisualizationHandler) getFileExtension(format string) string {
	extensions := map[string]string{
		"json":      "json",
		"dot":       "dot",
		"graphviz":  "dot",
		"cytoscape": "json",
		"d3":        "json",
		"mermaid":   "mmd",
		"html":      "html",
	}

	if ext, ok := extensions[format]; ok {
		return ext
	}
	return "txt"
}

// findNodeDetails finds detailed information about a node
func (h *VisualizationHandler) findNodeDetails(topology *services.Topology, nodeID string) map[string]interface{} {
	// Check switches
	for _, sw := range topology.Switches {
		if sw.UUID == nodeID || "switch:"+sw.UUID == nodeID {
			return map[string]interface{}{
				"id":          sw.UUID,
				"type":        "switch",
				"name":        sw.Name,
				"description": sw.Description,
				"ports":       sw.Ports,
				"metadata":    sw.ExternalIDs,
			}
		}
	}

	// Check routers
	for _, router := range topology.Routers {
		if router.UUID == nodeID || "router:"+router.UUID == nodeID {
			return map[string]interface{}{
				"id":           router.UUID,
				"type":         "router",
				"name":         router.Name,
				"description":  router.Description,
				"ports":        router.Ports,
				"nat":          router.NAT,
				"staticRoutes": router.StaticRoutes,
				"policies":     router.Policies,
			}
		}
	}

	// Check ports
	for _, sw := range topology.Switches {
		for _, port := range sw.Ports {
			if port.UUID == nodeID || "port:"+port.UUID == nodeID {
				return map[string]interface{}{
					"id":         port.UUID,
					"type":       "port",
					"name":       port.Name,
					"parentType": "switch",
					"parentID":   sw.UUID,
					"parentName": sw.Name,
					"macAddress": port.MAC,
					"addresses":  port.Addresses,
					"options":    port.Options,
				}
			}
		}
	}

	for _, router := range topology.Routers {
		for _, port := range router.Ports {
			if port.UUID == nodeID || "port:"+port.UUID == nodeID {
				return map[string]interface{}{
					"id":         port.UUID,
					"type":       "port",
					"name":       port.Name,
					"parentType": "router",
					"parentID":   router.UUID,
					"parentName": router.Name,
					"macAddress": port.MAC,
					"networks":   port.Networks,
				}
			}
		}
	}

	return nil
}

// findPath finds the path between two nodes using BFS
func (h *VisualizationHandler) findPath(topology *services.Topology, source, target string) []string {
	// Build adjacency map
	adj := make(map[string][]string)

	// Add switch-port connections
	for _, sw := range topology.Switches {
		swID := "switch:" + sw.UUID
		for _, port := range sw.Ports {
			portID := "port:" + port.UUID
			adj[swID] = append(adj[swID], portID)
			adj[portID] = append(adj[portID], swID)

			// Add router connections
			if port.Type == "router" && port.Options["router-port"] != "" {
				routerPortID := "port:" + port.Options["router-port"]
				adj[portID] = append(adj[portID], routerPortID)
				adj[routerPortID] = append(adj[routerPortID], portID)
			}
		}
	}

	// Add router-port connections
	for _, router := range topology.Routers {
		routerID := "router:" + router.UUID
		for _, port := range router.Ports {
			portID := "port:" + port.UUID
			adj[routerID] = append(adj[routerID], portID)
			adj[portID] = append(adj[portID], routerID)
		}
	}

	// BFS to find shortest path
	visited := make(map[string]bool)
	parent := make(map[string]string)
	queue := []string{source}
	visited[source] = true

	for len(queue) > 0 {
		current := queue[0]
		queue = queue[1:]

		if current == target {
			// Reconstruct path
			path := []string{}
			for node := target; node != ""; node = parent[node] {
				path = append([]string{node}, path...)
				if node == source {
					break
				}
			}
			return path
		}

		for _, neighbor := range adj[current] {
			if !visited[neighbor] {
				visited[neighbor] = true
				parent[neighbor] = current
				queue = append(queue, neighbor)
			}
		}
	}

	return []string{} // No path found
}