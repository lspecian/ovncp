package visualization

import (
	"fmt"
	"math"
	"strings"

	"github.com/lspecian/ovncp/internal/models"
	"github.com/lspecian/ovncp/internal/services"
)

// NodeType represents the type of a graph node
type NodeType string

const (
	NodeTypeSwitch       NodeType = "switch"
	NodeTypeRouter       NodeType = "router"
	NodeTypePort         NodeType = "port"
	NodeTypeLoadBalancer NodeType = "loadbalancer"
	NodeTypeNAT          NodeType = "nat"
	NodeTypeACL          NodeType = "acl"
)

// GraphNode represents a node in the topology graph
type GraphNode struct {
	ID         string                 `json:"id"`
	Label      string                 `json:"label"`
	Type       NodeType               `json:"type"`
	Group      string                 `json:"group,omitempty"`
	Properties map[string]interface{} `json:"properties"`
	Position   *Position              `json:"position,omitempty"`
	Style      *NodeStyle             `json:"style,omitempty"`
}

// GraphEdge represents an edge in the topology graph
type GraphEdge struct {
	ID         string                 `json:"id"`
	Source     string                 `json:"source"`
	Target     string                 `json:"target"`
	Label      string                 `json:"label,omitempty"`
	Type       string                 `json:"type"`
	Properties map[string]interface{} `json:"properties"`
	Style      *EdgeStyle             `json:"style,omitempty"`
}

// Position represents node position in 2D space
type Position struct {
	X float64 `json:"x"`
	Y float64 `json:"y"`
}

// NodeStyle represents visual styling for nodes
type NodeStyle struct {
	Shape       string `json:"shape,omitempty"`
	Color       string `json:"color,omitempty"`
	BorderColor string `json:"borderColor,omitempty"`
	Size        int    `json:"size,omitempty"`
	Icon        string `json:"icon,omitempty"`
}

// EdgeStyle represents visual styling for edges
type EdgeStyle struct {
	Color     string  `json:"color,omitempty"`
	Width     int     `json:"width,omitempty"`
	Style     string  `json:"style,omitempty"` // solid, dashed, dotted
	Animated  bool    `json:"animated,omitempty"`
	Curvature float64 `json:"curvature,omitempty"`
}

// TopologyGraph represents the complete network topology graph
type TopologyGraph struct {
	Nodes      []GraphNode            `json:"nodes"`
	Edges      []GraphEdge            `json:"edges"`
	Groups     []Group                `json:"groups"`
	Layout     string                 `json:"layout"`
	Properties map[string]interface{} `json:"properties"`
}

// Group represents a group of nodes
type Group struct {
	ID    string     `json:"id"`
	Label string     `json:"label"`
	Nodes []string   `json:"nodes"`
	Style GroupStyle `json:"style"`
}

// GroupStyle represents visual styling for groups
type GroupStyle struct {
	BackgroundColor string `json:"backgroundColor,omitempty"`
	BorderColor     string `json:"borderColor,omitempty"`
	BorderStyle     string `json:"borderStyle,omitempty"`
}

// TopologyVisualizer generates visual representations of network topology
type TopologyVisualizer struct {
	topology *services.Topology
}

// NewTopologyVisualizer creates a new topology visualizer
func NewTopologyVisualizer(topology *services.Topology) *TopologyVisualizer {
	return &TopologyVisualizer{
		topology: topology,
	}
}

// GenerateGraph generates a graph representation of the topology
func (v *TopologyVisualizer) GenerateGraph(options *VisualizationOptions) (*TopologyGraph, error) {
	if options == nil {
		options = DefaultVisualizationOptions()
	}

	graph := &TopologyGraph{
		Nodes:      []GraphNode{},
		Edges:      []GraphEdge{},
		Groups:     []Group{},
		Layout:     options.Layout,
		Properties: make(map[string]interface{}),
	}

	// Add switches
	if options.IncludeSwitches {
		v.addSwitches(graph, options)
	}

	// Add routers
	if options.IncludeRouters {
		v.addRouters(graph, options)
	}

	// Add ports and connections
	if options.IncludePorts {
		v.addPorts(graph, options)
	}

	// Add load balancers
	// NOTE: LoadBalancers field needs to be added to Topology struct if needed
	// if options.IncludeLoadBalancers {
	// 	v.addLoadBalancers(graph, options)
	// }

	// Add ACLs
	if options.IncludeACLs && options.DetailLevel == DetailLevelFull {
		v.addACLs(graph, options)
	}

	// Apply layout
	v.applyLayout(graph, options)

	// Add metadata
	graph.Properties["nodeCount"] = len(graph.Nodes)
	graph.Properties["edgeCount"] = len(graph.Edges)
	graph.Properties["timestamp"] = v.topology.Timestamp

	return graph, nil
}

// addSwitches adds switch nodes to the graph
func (v *TopologyVisualizer) addSwitches(graph *TopologyGraph, options *VisualizationOptions) {
	for _, sw := range v.topology.Switches {
		node := GraphNode{
			ID:    "switch:" + sw.UUID,
			Label: sw.Name,
			Type:  NodeTypeSwitch,
			Group: "switches",
			Properties: map[string]interface{}{
				"uuid":        sw.UUID,
				"description": sw.Description,
				"vlan":        sw.VLAN,
				"portCount":   len(sw.Ports),
			},
			Style: &NodeStyle{
				Shape:       "rectangle",
				Color:       "#4FC3F7",
				BorderColor: "#29B6F6",
				Size:        60,
				Icon:        "switch",
			},
		}

		// Add metadata based on detail level
		if options.DetailLevel >= DetailLevelMedium {
			node.Properties["metadata"] = sw.ExternalIDs
		}

		graph.Nodes = append(graph.Nodes, node)
	}
}

// addRouters adds router nodes to the graph
func (v *TopologyVisualizer) addRouters(graph *TopologyGraph, options *VisualizationOptions) {
	for _, router := range v.topology.Routers {
		node := GraphNode{
			ID:    "router:" + router.UUID,
			Label: router.Name,
			Type:  NodeTypeRouter,
			Group: "routers",
			Properties: map[string]interface{}{
				"uuid":        router.UUID,
				"description": router.Description,
				"portCount":   len(router.Ports),
				"natRules":    len(router.NAT),
			},
			Style: &NodeStyle{
				Shape:       "circle",
				Color:       "#66BB6A",
				BorderColor: "#4CAF50",
				Size:        70,
				Icon:        "router",
			},
		}

		// Add policies if detailed view
		if options.DetailLevel >= DetailLevelMedium {
			node.Properties["policies"] = router.Policies
			node.Properties["staticRoutes"] = router.StaticRoutes
		}

		graph.Nodes = append(graph.Nodes, node)
	}
}

// addPorts adds port nodes and connections to the graph
func (v *TopologyVisualizer) addPorts(graph *TopologyGraph, options *VisualizationOptions) {
	// Process all ports from topology
	for _, port := range v.topology.Ports {
		if options.DetailLevel < DetailLevelFull && !v.isSignificantPort(port) {
			continue
		}

		portNode := GraphNode{
			ID:    "port:" + port.UUID,
			Label: port.Name,
			Type:  NodeTypePort,
			Group: "ports",
			Properties: map[string]interface{}{
				"uuid":       port.UUID,
				"type":       port.Type,
				"macAddress": port.MAC,
				"ipAddress":  strings.Join(port.Addresses, ", "),
			},
			Style: &NodeStyle{
				Shape:       "dot",
				Color:       "#FFB74D",
				BorderColor: "#FFA726",
				Size:        20,
			},
		}

		graph.Nodes = append(graph.Nodes, portNode)

		// Add edge from switch to port if we have a switch ID
		if port.SwitchID != "" {
			edge := GraphEdge{
				ID:     fmt.Sprintf("edge:%s-%s", port.SwitchID, port.UUID),
				Source: "switch:" + port.SwitchID,
				Target: "port:" + port.UUID,
				Type:   "contains",
				Style: &EdgeStyle{
					Color: "#757575",
					Width: 2,
					Style: "solid",
				},
			}
			graph.Edges = append(graph.Edges, edge)
		}

		// Add router connections
		if port.Type == "router" && port.Options["router-port"] != "" {
			routerPortID := port.Options["router-port"]
			routerEdge := GraphEdge{
				ID:     fmt.Sprintf("edge:router-%s-%s", port.UUID, routerPortID),
				Source: "port:" + port.UUID,
				Target: "port:" + routerPortID,
				Type:   "connected",
				Label:  "L3",
				Style: &EdgeStyle{
					Color:    "#4CAF50",
					Width:    3,
					Style:    "solid",
					Animated: true,
				},
			}
			graph.Edges = append(graph.Edges, routerEdge)
		}
	}

	// Process router ports
	for _, rp := range v.topology.RouterPorts {
		if options.DetailLevel < DetailLevelFull {
			continue
		}

		portNode := GraphNode{
			ID:    "port:" + rp.UUID,
			Label: rp.Name,
			Type:  NodeTypePort,
			Group: "ports",
			Properties: map[string]interface{}{
				"uuid":       rp.UUID,
				"type":       "router",
				"macAddress": rp.MAC,
				"networks":   rp.Networks,
			},
			Style: &NodeStyle{
				Shape:       "dot",
				Color:       "#81C784",
				BorderColor: "#66BB6A",
				Size:        20,
			},
		}

		graph.Nodes = append(graph.Nodes, portNode)

		// Note: We'd need to determine which router owns this port
		// This is a simplified version - in real implementation you'd need proper mapping
	}
}

// addLoadBalancers adds load balancer nodes to the graph
// NOTE: This function is commented out until LoadBalancers field is added to Topology struct
/*
func (v *TopologyVisualizer) addLoadBalancers(graph *TopologyGraph, options *VisualizationOptions) {
	for _, lb := range v.topology.LoadBalancers {
		node := GraphNode{
			ID:    "lb:" + lb.UUID,
			Label: lb.Name,
			Type:  NodeTypeLoadBalancer,
			Group: "loadbalancers",
			Properties: map[string]interface{}{
				"uuid":     lb.UUID,
				"protocol": lb.Protocol,
				"vips":     lb.VIPs,
			},
			Style: &NodeStyle{
				Shape:       "hexagon",
				Color:       "#BA68C8",
				BorderColor: "#AB47BC",
				Size:        50,
				Icon:        "loadbalancer",
			},
		}

		graph.Nodes = append(graph.Nodes, node)

		// Connect to switches
		for _, swID := range lb.Switches {
			edge := GraphEdge{
				ID:     fmt.Sprintf("edge:lb-%s-%s", lb.UUID, swID),
				Source: "lb:" + lb.UUID,
				Target: "switch:" + swID,
				Type:   "serves",
				Label:  "LB",
				Style: &EdgeStyle{
					Color:    "#BA68C8",
					Width:    2,
					Style:    "dashed",
					Animated: options.AnimateTraffic,
				},
			}
			graph.Edges = append(graph.Edges, edge)
		}
	}
}
*/

// addACLs adds ACL representations to the graph
func (v *TopologyVisualizer) addACLs(graph *TopologyGraph, options *VisualizationOptions) {
	// Create a single ACL summary node for all ACLs
	if len(v.topology.ACLs) == 0 {
		return
	}

	// Create a summary node for all ACLs
	node := GraphNode{
		ID:    "acl-group:all",
		Label: fmt.Sprintf("ACLs (%d rules)", len(v.topology.ACLs)),
		Type:  NodeTypeACL,
		Group: "acls",
		Properties: map[string]interface{}{
			"ruleCount": len(v.topology.ACLs),
			"rules":     v.summarizeACLs(v.topology.ACLs),
		},
		Style: &NodeStyle{
			Shape:       "shield",
			Color:       "#FF7043",
			BorderColor: "#FF5722",
			Size:        30,
			Icon:        "security",
		},
	}

	graph.Nodes = append(graph.Nodes, node)
}

// summarizeACLs creates a summary of ACL rules
func (v *TopologyVisualizer) summarizeACLs(acls []*models.ACL) []map[string]interface{} {
	summary := []map[string]interface{}{}
	
	for _, acl := range acls {
		rule := map[string]interface{}{
			"priority":  acl.Priority,
			"direction": acl.Direction,
			"action":    acl.Action,
			"match":     acl.Match,
		}
		summary = append(summary, rule)
	}
	
	return summary
}

// isSignificantPort determines if a port should be shown based on detail level
func (v *TopologyVisualizer) isSignificantPort(port *models.LogicalSwitchPort) bool {
	// Always show router ports
	if port.Type == "router" {
		return true
	}
	
	// Show ports with IP addresses
	if len(port.Addresses) > 0 && port.Addresses[0] != "unknown" {
		return true
	}
	
	// Show ports with special types
	specialTypes := []string{"localnet", "localport", "l2gateway", "vtep"}
	for _, t := range specialTypes {
		if port.Type == t {
			return true
		}
	}
	
	return false
}

// applyLayout applies the specified layout algorithm to the graph
func (v *TopologyVisualizer) applyLayout(graph *TopologyGraph, options *VisualizationOptions) {
	switch options.Layout {
	case "hierarchical":
		v.applyHierarchicalLayout(graph)
	case "force":
		v.applyForceLayout(graph)
	case "circular":
		v.applyCircularLayout(graph)
	case "grid":
		v.applyGridLayout(graph)
	default:
		// No layout, let the client handle it
	}
}

// applyHierarchicalLayout arranges nodes in a hierarchical structure
func (v *TopologyVisualizer) applyHierarchicalLayout(graph *TopologyGraph) {
	// Layer 1: Routers
	routerCount := 0
	for i, node := range graph.Nodes {
		if node.Type == NodeTypeRouter {
			graph.Nodes[i].Position = &Position{
				X: float64(routerCount * 200),
				Y: 0,
			}
			routerCount++
		}
	}

	// Layer 2: Switches
	switchCount := 0
	for i, node := range graph.Nodes {
		if node.Type == NodeTypeSwitch {
			graph.Nodes[i].Position = &Position{
				X: float64(switchCount * 150),
				Y: 200,
			}
			switchCount++
		}
	}

	// Layer 3: Ports
	portCount := 0
	for i, node := range graph.Nodes {
		if node.Type == NodeTypePort {
			graph.Nodes[i].Position = &Position{
				X: float64(portCount * 100),
				Y: 400,
			}
			portCount++
		}
	}

	// Layer 4: Services (LB, ACL)
	serviceCount := 0
	for i, node := range graph.Nodes {
		if node.Type == NodeTypeLoadBalancer || node.Type == NodeTypeACL {
			graph.Nodes[i].Position = &Position{
				X: float64(serviceCount * 150),
				Y: 600,
			}
			serviceCount++
		}
	}
}

// applyForceLayout would implement force-directed layout
func (v *TopologyVisualizer) applyForceLayout(graph *TopologyGraph) {
	// This would implement a force-directed algorithm
	// For now, we'll leave positioning to the client
}

// applyCircularLayout arranges nodes in a circle
func (v *TopologyVisualizer) applyCircularLayout(graph *TopologyGraph) {
	nodeCount := len(graph.Nodes)
	if nodeCount == 0 {
		return
	}

	radius := float64(nodeCount * 30)
	angleStep := 2 * 3.14159 / float64(nodeCount)

	for i := range graph.Nodes {
		angle := float64(i) * angleStep
		graph.Nodes[i].Position = &Position{
			X: radius * cos(angle),
			Y: radius * sin(angle),
		}
	}
}

// applyGridLayout arranges nodes in a grid
func (v *TopologyVisualizer) applyGridLayout(graph *TopologyGraph) {
	nodeCount := len(graph.Nodes)
	if nodeCount == 0 {
		return
	}

	// Calculate grid dimensions
	cols := int(sqrt(float64(nodeCount))) + 1
	spacing := 150.0

	for i := range graph.Nodes {
		row := i / cols
		col := i % cols
		graph.Nodes[i].Position = &Position{
			X: float64(col) * spacing,
			Y: float64(row) * spacing,
		}
	}
}

// Math helper functions
func cos(angle float64) float64 {
	return math.Cos(angle)
}

func sin(angle float64) float64 {
	return math.Sin(angle)
}

func sqrt(x float64) float64 {
	return math.Sqrt(x)
}