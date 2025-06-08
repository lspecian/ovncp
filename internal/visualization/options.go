package visualization

// DetailLevel represents the level of detail in visualization
type DetailLevel int

const (
	DetailLevelMinimal DetailLevel = iota
	DetailLevelMedium
	DetailLevelFull
)

// VisualizationOptions configures topology visualization
type VisualizationOptions struct {
	// Layout algorithm: hierarchical, force, circular, grid, none
	Layout string `json:"layout"`
	
	// Detail level
	DetailLevel DetailLevel `json:"detailLevel"`
	
	// Component filters
	IncludeSwitches      bool `json:"includeSwitches"`
	IncludeRouters       bool `json:"includeRouters"`
	IncludePorts         bool `json:"includePorts"`
	IncludeLoadBalancers bool `json:"includeLoadBalancers"`
	IncludeACLs          bool `json:"includeACLs"`
	IncludeNAT           bool `json:"includeNAT"`
	
	// Visual options
	ShowLabels       bool `json:"showLabels"`
	ShowIcons        bool `json:"showIcons"`
	AnimateTraffic   bool `json:"animateTraffic"`
	GroupNodes       bool `json:"groupNodes"`
	
	// Filtering
	FilterByName     string   `json:"filterByName,omitempty"`
	FilterByType     []string `json:"filterByType,omitempty"`
	FilterByProperty map[string]string `json:"filterByProperty,omitempty"`
	
	// Performance
	MaxNodes         int  `json:"maxNodes"`
	SimplifyPorts    bool `json:"simplifyPorts"`
	AggregateACLs    bool `json:"aggregateACLs"`
	
	// Export options
	Format           string `json:"format"` // json, dot, cytoscape, d3
}

// DefaultVisualizationOptions returns default visualization options
func DefaultVisualizationOptions() *VisualizationOptions {
	return &VisualizationOptions{
		Layout:               "hierarchical",
		DetailLevel:          DetailLevelMedium,
		IncludeSwitches:      true,
		IncludeRouters:       true,
		IncludePorts:         true,
		IncludeLoadBalancers: true,
		IncludeACLs:          false,
		IncludeNAT:           false,
		ShowLabels:           true,
		ShowIcons:            true,
		AnimateTraffic:       false,
		GroupNodes:           true,
		MaxNodes:             1000,
		SimplifyPorts:        true,
		AggregateACLs:        true,
		Format:               "json",
	}
}

// MinimalOptions returns options for minimal visualization
func MinimalOptions() *VisualizationOptions {
	return &VisualizationOptions{
		Layout:               "hierarchical",
		DetailLevel:          DetailLevelMinimal,
		IncludeSwitches:      true,
		IncludeRouters:       true,
		IncludePorts:         false,
		IncludeLoadBalancers: false,
		IncludeACLs:          false,
		IncludeNAT:           false,
		ShowLabels:           true,
		ShowIcons:            false,
		AnimateTraffic:       false,
		GroupNodes:           false,
		MaxNodes:             100,
		SimplifyPorts:        true,
		AggregateACLs:        true,
		Format:               "json",
	}
}

// FullOptions returns options for full visualization
func FullOptions() *VisualizationOptions {
	return &VisualizationOptions{
		Layout:               "force",
		DetailLevel:          DetailLevelFull,
		IncludeSwitches:      true,
		IncludeRouters:       true,
		IncludePorts:         true,
		IncludeLoadBalancers: true,
		IncludeACLs:          true,
		IncludeNAT:           true,
		ShowLabels:           true,
		ShowIcons:            true,
		AnimateTraffic:       true,
		GroupNodes:           true,
		MaxNodes:             5000,
		SimplifyPorts:        false,
		AggregateACLs:        false,
		Format:               "json",
	}
}