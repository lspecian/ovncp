package visualization

import (
	"bytes"
	"encoding/json"
	"fmt"
	"strings"
	"text/template"
)

// Exporter handles exporting topology graphs to various formats
type Exporter struct {
	graph *TopologyGraph
}

// NewExporter creates a new exporter
func NewExporter(graph *TopologyGraph) *Exporter {
	return &Exporter{graph: graph}
}

// Export exports the graph to the specified format
func (e *Exporter) Export(format string) ([]byte, error) {
	switch strings.ToLower(format) {
	case "json":
		return e.exportJSON()
	case "dot", "graphviz":
		return e.exportDOT()
	case "cytoscape":
		return e.exportCytoscape()
	case "d3":
		return e.exportD3()
	case "mermaid":
		return e.exportMermaid()
	default:
		return nil, fmt.Errorf("unsupported export format: %s", format)
	}
}

// exportJSON exports to JSON format
func (e *Exporter) exportJSON() ([]byte, error) {
	return json.MarshalIndent(e.graph, "", "  ")
}

// exportDOT exports to Graphviz DOT format
func (e *Exporter) exportDOT() ([]byte, error) {
	var buf bytes.Buffer
	
	buf.WriteString("digraph OVNTopology {\n")
	buf.WriteString("  rankdir=TB;\n")
	buf.WriteString("  node [shape=box, style=rounded];\n")
	buf.WriteString("  edge [color=gray];\n\n")
	
	// Define node styles by type
	buf.WriteString("  /* Node Styles */\n")
	buf.WriteString("  node [shape=box, style=\"rounded,filled\"];\n")
	
	// Add nodes
	buf.WriteString("\n  /* Nodes */\n")
	for _, node := range e.graph.Nodes {
		nodeID := strings.ReplaceAll(node.ID, ":", "_")
		label := strings.ReplaceAll(node.Label, "\"", "\\\"")
		
		style := e.getDOTNodeStyle(node.Type)
		buf.WriteString(fmt.Sprintf("  %s [label=\"%s\"%s];\n", nodeID, label, style))
	}
	
	// Add edges
	buf.WriteString("\n  /* Edges */\n")
	for _, edge := range e.graph.Edges {
		sourceID := strings.ReplaceAll(edge.Source, ":", "_")
		targetID := strings.ReplaceAll(edge.Target, ":", "_")
		
		edgeStyle := e.getDOTEdgeStyle(edge.Type)
		if edge.Label != "" {
			edgeStyle += fmt.Sprintf(", label=\"%s\"", edge.Label)
		}
		
		buf.WriteString(fmt.Sprintf("  %s -> %s [%s];\n", sourceID, targetID, edgeStyle))
	}
	
	// Add subgraphs for groups
	if len(e.graph.Groups) > 0 {
		buf.WriteString("\n  /* Groups */\n")
		for i, group := range e.graph.Groups {
			buf.WriteString(fmt.Sprintf("  subgraph cluster_%d {\n", i))
			buf.WriteString(fmt.Sprintf("    label=\"%s\";\n", group.Label))
			buf.WriteString("    style=dotted;\n")
			
			for _, nodeID := range group.Nodes {
				nodeID = strings.ReplaceAll(nodeID, ":", "_")
				buf.WriteString(fmt.Sprintf("    %s;\n", nodeID))
			}
			
			buf.WriteString("  }\n")
		}
	}
	
	buf.WriteString("}\n")
	
	return buf.Bytes(), nil
}

// getDOTNodeStyle returns DOT style attributes for a node type
func (e *Exporter) getDOTNodeStyle(nodeType NodeType) string {
	styles := map[NodeType]string{
		NodeTypeRouter:       ", fillcolor=\"#66BB6A\", shape=circle",
		NodeTypeSwitch:       ", fillcolor=\"#4FC3F7\", shape=box",
		NodeTypePort:         ", fillcolor=\"#FFB74D\", shape=ellipse, width=0.5, height=0.5",
		NodeTypeLoadBalancer: ", fillcolor=\"#BA68C8\", shape=hexagon",
		NodeTypeACL:          ", fillcolor=\"#FF7043\", shape=diamond",
		NodeTypeNAT:          ", fillcolor=\"#9CCC65\", shape=trapezium",
	}
	
	if style, ok := styles[nodeType]; ok {
		return style
	}
	return ", fillcolor=\"#E0E0E0\""
}

// getDOTEdgeStyle returns DOT style attributes for an edge type
func (e *Exporter) getDOTEdgeStyle(edgeType string) string {
	styles := map[string]string{
		"contains":  "color=\"#757575\"",
		"connected": "color=\"#4CAF50\", penwidth=2",
		"serves":    "color=\"#BA68C8\", style=dashed",
		"protects":  "color=\"#FF7043\", style=dotted",
	}
	
	if style, ok := styles[edgeType]; ok {
		return style
	}
	return "color=\"#9E9E9E\""
}

// exportCytoscape exports to Cytoscape.js format
func (e *Exporter) exportCytoscape() ([]byte, error) {
	cyto := map[string]interface{}{
		"elements": map[string]interface{}{
			"nodes": []map[string]interface{}{},
			"edges": []map[string]interface{}{},
		},
		"style": e.getCytoscapeStyles(),
		"layout": map[string]interface{}{
			"name": e.graph.Layout,
		},
	}
	
	// Convert nodes
	nodes := []map[string]interface{}{}
	for _, node := range e.graph.Nodes {
		cyNode := map[string]interface{}{
			"data": map[string]interface{}{
				"id":         node.ID,
				"label":      node.Label,
				"type":       string(node.Type),
				"properties": node.Properties,
			},
		}
		
		if node.Position != nil {
			cyNode["position"] = map[string]float64{
				"x": node.Position.X,
				"y": node.Position.Y,
			}
		}
		
		nodes = append(nodes, cyNode)
	}
	
	// Convert edges
	edges := []map[string]interface{}{}
	for _, edge := range e.graph.Edges {
		cyEdge := map[string]interface{}{
			"data": map[string]interface{}{
				"id":         edge.ID,
				"source":     edge.Source,
				"target":     edge.Target,
				"label":      edge.Label,
				"type":       edge.Type,
				"properties": edge.Properties,
			},
		}
		
		edges = append(edges, cyEdge)
	}
	
	cyto["elements"].(map[string]interface{})["nodes"] = nodes
	cyto["elements"].(map[string]interface{})["edges"] = edges
	
	return json.MarshalIndent(cyto, "", "  ")
}

// getCytoscapeStyles returns Cytoscape.js style definitions
func (e *Exporter) getCytoscapeStyles() []map[string]interface{} {
	return []map[string]interface{}{
		{
			"selector": "node",
			"style": map[string]interface{}{
				"label":              "data(label)",
				"text-valign":        "center",
				"text-halign":        "center",
				"background-opacity": 0.8,
				"border-width":       2,
			},
		},
		{
			"selector": "node[type='router']",
			"style": map[string]interface{}{
				"shape":               "ellipse",
				"background-color":    "#66BB6A",
				"border-color":        "#4CAF50",
				"width":               60,
				"height":              60,
			},
		},
		{
			"selector": "node[type='switch']",
			"style": map[string]interface{}{
				"shape":               "roundrectangle",
				"background-color":    "#4FC3F7",
				"border-color":        "#29B6F6",
				"width":               80,
				"height":              40,
			},
		},
		{
			"selector": "node[type='port']",
			"style": map[string]interface{}{
				"shape":               "ellipse",
				"background-color":    "#FFB74D",
				"border-color":        "#FFA726",
				"width":               20,
				"height":              20,
				"font-size":           10,
			},
		},
		{
			"selector": "edge",
			"style": map[string]interface{}{
				"width":                3,
				"line-color":           "#9E9E9E",
				"target-arrow-color":   "#9E9E9E",
				"target-arrow-shape":   "triangle",
				"curve-style":          "bezier",
			},
		},
		{
			"selector": "edge[type='connected']",
			"style": map[string]interface{}{
				"line-color":         "#4CAF50",
				"target-arrow-color": "#4CAF50",
				"width":              4,
			},
		},
	}
}

// exportD3 exports to D3.js format
func (e *Exporter) exportD3() ([]byte, error) {
	d3Data := map[string]interface{}{
		"nodes": []map[string]interface{}{},
		"links": []map[string]interface{}{},
	}
	
	// Convert nodes
	nodeIndex := make(map[string]int)
	nodes := []map[string]interface{}{}
	
	for i, node := range e.graph.Nodes {
		nodeIndex[node.ID] = i
		
		d3Node := map[string]interface{}{
			"id":    node.ID,
			"label": node.Label,
			"type":  string(node.Type),
			"group": node.Group,
		}
		
		// Add visual properties
		if node.Style != nil {
			d3Node["color"] = node.Style.Color
			d3Node["size"] = node.Style.Size
		}
		
		// Add position if available
		if node.Position != nil {
			d3Node["x"] = node.Position.X
			d3Node["y"] = node.Position.Y
		}
		
		nodes = append(nodes, d3Node)
	}
	
	// Convert edges to links
	links := []map[string]interface{}{}
	for _, edge := range e.graph.Edges {
		sourceIdx, sourceOk := nodeIndex[edge.Source]
		targetIdx, targetOk := nodeIndex[edge.Target]
		
		if sourceOk && targetOk {
			link := map[string]interface{}{
				"source": sourceIdx,
				"target": targetIdx,
				"type":   edge.Type,
				"label":  edge.Label,
			}
			
			if edge.Style != nil {
				link["color"] = edge.Style.Color
				link["width"] = edge.Style.Width
			}
			
			links = append(links, link)
		}
	}
	
	d3Data["nodes"] = nodes
	d3Data["links"] = links
	
	return json.MarshalIndent(d3Data, "", "  ")
}

// exportMermaid exports to Mermaid diagram format
func (e *Exporter) exportMermaid() ([]byte, error) {
	var buf bytes.Buffer
	
	buf.WriteString("graph TB\n")
	
	// Add nodes
	for _, node := range e.graph.Nodes {
		nodeID := strings.ReplaceAll(node.ID, ":", "_")
		label := strings.ReplaceAll(node.Label, "\"", "'")
		
		shape := e.getMermaidShape(node.Type)
		buf.WriteString(fmt.Sprintf("    %s%s\n", nodeID, shape(label)))
	}
	
	buf.WriteString("\n")
	
	// Add edges
	for _, edge := range e.graph.Edges {
		sourceID := strings.ReplaceAll(edge.Source, ":", "_")
		targetID := strings.ReplaceAll(edge.Target, ":", "_")
		
		arrow := e.getMermaidArrow(edge.Type)
		if edge.Label != "" {
			arrow += fmt.Sprintf("|%s|", edge.Label)
		}
		
		buf.WriteString(fmt.Sprintf("    %s %s %s\n", sourceID, arrow, targetID))
	}
	
	// Add styling
	buf.WriteString("\n")
	buf.WriteString("    classDef router fill:#66BB6A,stroke:#4CAF50,stroke-width:2px;\n")
	buf.WriteString("    classDef switch fill:#4FC3F7,stroke:#29B6F6,stroke-width:2px;\n")
	buf.WriteString("    classDef port fill:#FFB74D,stroke:#FFA726,stroke-width:2px;\n")
	buf.WriteString("    classDef lb fill:#BA68C8,stroke:#AB47BC,stroke-width:2px;\n")
	buf.WriteString("    classDef acl fill:#FF7043,stroke:#FF5722,stroke-width:2px;\n")
	
	// Apply classes
	for _, node := range e.graph.Nodes {
		nodeID := strings.ReplaceAll(node.ID, ":", "_")
		class := strings.ToLower(string(node.Type))
		buf.WriteString(fmt.Sprintf("    class %s %s;\n", nodeID, class))
	}
	
	return buf.Bytes(), nil
}

// getMermaidShape returns the Mermaid shape function for a node type
func (e *Exporter) getMermaidShape(nodeType NodeType) func(string) string {
	switch nodeType {
	case NodeTypeRouter:
		return func(label string) string { return fmt.Sprintf("((%s))", label) }
	case NodeTypeSwitch:
		return func(label string) string { return fmt.Sprintf("[%s]", label) }
	case NodeTypePort:
		return func(label string) string { return fmt.Sprintf("((%s))", label) }
	case NodeTypeLoadBalancer:
		return func(label string) string { return fmt.Sprintf("{{%s}}", label) }
	case NodeTypeACL:
		return func(label string) string { return fmt.Sprintf("{%s}", label) }
	default:
		return func(label string) string { return fmt.Sprintf("[%s]", label) }
	}
}

// getMermaidArrow returns the Mermaid arrow style for an edge type
func (e *Exporter) getMermaidArrow(edgeType string) string {
	switch edgeType {
	case "connected":
		return "==>"
	case "contains":
		return "-->"
	case "serves":
		return "-.->"
	case "protects":
		return "-..->"
	default:
		return "-->"
	}
}

// ExportHTML exports an interactive HTML visualization
func (e *Exporter) ExportHTML(options *HTMLExportOptions) ([]byte, error) {
	if options == nil {
		options = DefaultHTMLExportOptions()
	}
	
	tmpl := template.Must(template.New("topology").Parse(htmlTemplate))
	
	// Prepare data
	jsonData, err := e.exportJSON()
	if err != nil {
		return nil, err
	}
	
	data := struct {
		Title       string
		GraphData   string
		IncludeD3   bool
		IncludeCyto bool
		Width       string
		Height      string
	}{
		Title:       options.Title,
		GraphData:   string(jsonData),
		IncludeD3:   options.Library == "d3",
		IncludeCyto: options.Library == "cytoscape",
		Width:       options.Width,
		Height:      options.Height,
	}
	
	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, data); err != nil {
		return nil, err
	}
	
	return buf.Bytes(), nil
}

// HTMLExportOptions configures HTML export
type HTMLExportOptions struct {
	Title      string
	Library    string // d3, cytoscape
	Width      string
	Height     string
	Responsive bool
}

// DefaultHTMLExportOptions returns default HTML export options
func DefaultHTMLExportOptions() *HTMLExportOptions {
	return &HTMLExportOptions{
		Title:      "OVN Network Topology",
		Library:    "cytoscape",
		Width:      "100%",
		Height:     "600px",
		Responsive: true,
	}
}

const htmlTemplate = `<!DOCTYPE html>
<html>
<head>
    <title>{{.Title}}</title>
    <style>
        body {
            font-family: Arial, sans-serif;
            margin: 0;
            padding: 20px;
            background-color: #f5f5f5;
        }
        #topology {
            width: {{.Width}};
            height: {{.Height}};
            background-color: white;
            border: 1px solid #ddd;
            border-radius: 4px;
            box-shadow: 0 2px 4px rgba(0,0,0,0.1);
        }
        .controls {
            margin-bottom: 20px;
            padding: 10px;
            background-color: white;
            border: 1px solid #ddd;
            border-radius: 4px;
        }
        button {
            margin-right: 10px;
            padding: 5px 15px;
            border: 1px solid #ddd;
            border-radius: 4px;
            background-color: #f8f8f8;
            cursor: pointer;
        }
        button:hover {
            background-color: #e8e8e8;
        }
    </style>
    {{if .IncludeCyto}}
    <script src="https://cdnjs.cloudflare.com/ajax/libs/cytoscape/3.21.1/cytoscape.min.js"></script>
    {{end}}
    {{if .IncludeD3}}
    <script src="https://d3js.org/d3.v7.min.js"></script>
    {{end}}
</head>
<body>
    <h1>{{.Title}}</h1>
    <div class="controls">
        <button onclick="resetView()">Reset View</button>
        <button onclick="fitToScreen()">Fit to Screen</button>
        <button onclick="exportImage()">Export Image</button>
    </div>
    <div id="topology"></div>
    
    <script>
        const graphData = {{.GraphData}};
        
        {{if .IncludeCyto}}
        // Cytoscape.js implementation
        const cy = cytoscape({
            container: document.getElementById('topology'),
            elements: convertToCytoscape(graphData),
            style: getCytoscapeStyles(),
            layout: {
                name: graphData.layout || 'cose',
                animate: false
            }
        });
        
        function resetView() {
            cy.reset();
        }
        
        function fitToScreen() {
            cy.fit();
        }
        
        function exportImage() {
            const png = cy.png();
            const link = document.createElement('a');
            link.href = png;
            link.download = 'topology.png';
            link.click();
        }
        
        function convertToCytoscape(data) {
            const elements = {
                nodes: data.nodes.map(node => ({
                    data: {
                        id: node.id,
                        label: node.label,
                        type: node.type
                    }
                })),
                edges: data.edges.map(edge => ({
                    data: {
                        id: edge.id,
                        source: edge.source,
                        target: edge.target,
                        label: edge.label,
                        type: edge.type
                    }
                }))
            };
            return elements;
        }
        
        function getCytoscapeStyles() {
            return [
                {
                    selector: 'node',
                    style: {
                        'label': 'data(label)',
                        'text-valign': 'center',
                        'text-halign': 'center'
                    }
                },
                {
                    selector: 'edge',
                    style: {
                        'curve-style': 'bezier',
                        'target-arrow-shape': 'triangle'
                    }
                }
            ];
        }
        {{end}}
        
        {{if .IncludeD3}}
        // D3.js implementation
        // ... D3 visualization code ...
        {{end}}
    </script>
</body>
</html>`