# Network Topology Visualization

The OVN Control Platform provides powerful visualization capabilities to help you understand and manage your network topology.

## Features

- **Interactive Graph Visualization**: View your entire OVN network as an interactive graph
- **Multiple Layout Algorithms**: Choose from hierarchical, force-directed, circular, or grid layouts
- **Component Filtering**: Show/hide switches, routers, ports, load balancers, ACLs, and NAT rules
- **Export Formats**: Export topology in various formats (JSON, DOT/Graphviz, Cytoscape, D3, Mermaid, HTML)
- **Path Finding**: Find the shortest path between any two nodes
- **Detail Levels**: Choose from minimal, medium, or full detail views

## API Endpoints

### Get Topology Visualization

```http
GET /api/v1/visualization/topology
```

Query Parameters:
- `layout` - Layout algorithm: `hierarchical`, `force`, `circular`, `grid` (default: `hierarchical`)
- `detail` - Detail level: `minimal`, `medium`, `full` (default: `medium`)
- `switches` - Include switches: `true`, `false` (default: `true`)
- `routers` - Include routers: `true`, `false` (default: `true`)
- `ports` - Include ports: `true`, `false` (default: `true`)
- `loadbalancers` - Include load balancers: `true`, `false` (default: `true`)
- `acls` - Include ACLs: `true`, `false` (default: `false`)
- `nat` - Include NAT rules: `true`, `false` (default: `false`)
- `labels` - Show labels: `true`, `false` (default: `true`)
- `icons` - Show icons: `true`, `false` (default: `true`)
- `animate` - Animate traffic: `true`, `false` (default: `false`)
- `maxNodes` - Maximum number of nodes to display (default: `1000`)

Example:
```bash
curl -H "Authorization: Bearer $TOKEN" \
  "https://ovncp.example.com/api/v1/visualization/topology?layout=force&detail=medium"
```

Response:
```json
{
  "nodes": [
    {
      "id": "switch:uuid-1234",
      "label": "web-switch",
      "type": "switch",
      "group": "switches",
      "properties": {
        "uuid": "uuid-1234",
        "description": "Web tier switch",
        "portCount": 5
      },
      "style": {
        "shape": "rectangle",
        "color": "#4FC3F7",
        "size": 60
      }
    }
  ],
  "edges": [
    {
      "id": "edge:uuid-1234-uuid-5678",
      "source": "switch:uuid-1234",
      "target": "port:uuid-5678",
      "type": "contains",
      "style": {
        "color": "#757575",
        "width": 2
      }
    }
  ],
  "layout": "force",
  "properties": {
    "nodeCount": 25,
    "edgeCount": 30,
    "timestamp": "2024-01-20T10:30:00Z"
  }
}
```

### Export Topology

```http
GET /api/v1/visualization/topology/export
```

Query Parameters:
- `format` - Export format: `json`, `dot`, `cytoscape`, `d3`, `mermaid`, `html` (default: `json`)
- All visualization parameters from above

Example:
```bash
# Export as Graphviz DOT
curl -H "Authorization: Bearer $TOKEN" \
  "https://ovncp.example.com/api/v1/visualization/topology/export?format=dot" \
  -o topology.dot

# Generate image
dot -Tpng topology.dot -o topology.png
```

### Custom Topology Visualization

```http
POST /api/v1/visualization/topology/custom
```

Request Body:
```json
{
  "layout": "hierarchical",
  "detailLevel": 2,
  "includeSwitches": true,
  "includeRouters": true,
  "includePorts": false,
  "includeLoadBalancers": true,
  "includeACLs": false,
  "showLabels": true,
  "animateTraffic": true,
  "filterByName": "web-*",
  "maxNodes": 500
}
```

### Get Node Details

```http
GET /api/v1/visualization/topology/node/:id
```

Example:
```bash
curl -H "Authorization: Bearer $TOKEN" \
  "https://ovncp.example.com/api/v1/visualization/topology/node/switch:uuid-1234"
```

Response:
```json
{
  "id": "uuid-1234",
  "type": "switch",
  "name": "web-switch",
  "description": "Web tier switch",
  "ports": [
    {
      "uuid": "uuid-5678",
      "name": "web-port-1",
      "addresses": ["192.168.1.10"]
    }
  ],
  "metadata": {
    "created": "2024-01-15T08:00:00Z",
    "tier": "web"
  }
}
```

### Find Path Between Nodes

```http
GET /api/v1/visualization/topology/path/:source/:target
```

Example:
```bash
curl -H "Authorization: Bearer $TOKEN" \
  "https://ovncp.example.com/api/v1/visualization/topology/path/switch:uuid-1234/switch:uuid-9012"
```

Response:
```json
{
  "source": "switch:uuid-1234",
  "target": "switch:uuid-9012",
  "path": [
    "switch:uuid-1234",
    "port:uuid-5678",
    "port:uuid-3456",
    "router:uuid-7890",
    "port:uuid-2345",
    "port:uuid-6789",
    "switch:uuid-9012"
  ],
  "hops": 6
}
```

## Export Formats

### Graphviz DOT

The DOT format can be used with Graphviz tools to generate high-quality network diagrams:

```bash
# Export topology as DOT
curl -H "Authorization: Bearer $TOKEN" \
  "https://ovncp.example.com/api/v1/visualization/topology/export?format=dot" \
  -o topology.dot

# Generate various image formats
dot -Tpng topology.dot -o topology.png
dot -Tsvg topology.dot -o topology.svg
dot -Tpdf topology.dot -o topology.pdf

# Use different layout engines
neato -Tpng topology.dot -o topology-neato.png
fdp -Tpng topology.dot -o topology-fdp.png
circo -Tpng topology.dot -o topology-circular.png
```

### Cytoscape.js

Export for use with Cytoscape.js visualization library:

```javascript
// Fetch topology in Cytoscape format
const response = await fetch('/api/v1/visualization/topology/export?format=cytoscape', {
  headers: { 'Authorization': `Bearer ${token}` }
});
const cytoscapeData = await response.json();

// Initialize Cytoscape
const cy = cytoscape({
  container: document.getElementById('cy'),
  elements: cytoscapeData.elements,
  style: cytoscapeData.style,
  layout: cytoscapeData.layout
});
```

### D3.js

Export for use with D3.js visualization library:

```javascript
// Fetch topology in D3 format
const response = await fetch('/api/v1/visualization/topology/export?format=d3', {
  headers: { 'Authorization': `Bearer ${token}` }
});
const d3Data = await response.json();

// Use with D3 force simulation
const simulation = d3.forceSimulation(d3Data.nodes)
  .force("link", d3.forceLink(d3Data.links).id(d => d.id))
  .force("charge", d3.forceManyBody())
  .force("center", d3.forceCenter(width / 2, height / 2));
```

### Mermaid

Export as Mermaid diagram for documentation:

```bash
# Export as Mermaid
curl -H "Authorization: Bearer $TOKEN" \
  "https://ovncp.example.com/api/v1/visualization/topology/export?format=mermaid" \
  -o topology.mmd

# Include in Markdown documentation
cat topology.mmd >> README.md
```

### Interactive HTML

Export as standalone HTML file with interactive visualization:

```bash
# Export as HTML
curl -H "Authorization: Bearer $TOKEN" \
  "https://ovncp.example.com/api/v1/visualization/topology/export?format=html" \
  -o topology.html

# Open in browser
open topology.html
```

## Visualization Options

### Detail Levels

1. **Minimal** (`detail=minimal`)
   - Shows only switches and routers
   - No ports or services
   - Ideal for high-level overview

2. **Medium** (`detail=medium`)
   - Shows switches, routers, and significant ports
   - Includes load balancers
   - Good balance of detail and clarity

3. **Full** (`detail=full`)
   - Shows all components including all ports
   - Includes ACLs and NAT rules
   - Maximum detail for troubleshooting

### Layout Algorithms

1. **Hierarchical** (`layout=hierarchical`)
   - Arranges nodes in layers
   - Routers at top, switches in middle, ports at bottom
   - Best for understanding network tiers

2. **Force-Directed** (`layout=force`)
   - Physics-based layout
   - Nodes repel, edges attract
   - Good for discovering clusters

3. **Circular** (`layout=circular`)
   - Nodes arranged in a circle
   - Edges cross the center
   - Useful for dense networks

4. **Grid** (`layout=grid`)
   - Nodes in a regular grid
   - Simple and predictable
   - Good for documentation

## Use Cases

### Network Documentation

Generate network diagrams for documentation:

```bash
# Export detailed topology as SVG
curl -H "Authorization: Bearer $TOKEN" \
  "https://ovncp.example.com/api/v1/visualization/topology/export?format=dot&detail=full" | \
  dot -Tsvg -o network-topology.svg
```

### Troubleshooting

Find paths and dependencies:

```bash
# Find path between two switches
curl -H "Authorization: Bearer $TOKEN" \
  "https://ovncp.example.com/api/v1/visualization/topology/path/switch:web-switch/switch:db-switch"

# Get detailed view of a specific component
curl -H "Authorization: Bearer $TOKEN" \
  "https://ovncp.example.com/api/v1/visualization/topology/node/router:edge-router"
```

### Monitoring Integration

Use the visualization API in monitoring dashboards:

```javascript
// Periodic topology refresh
setInterval(async () => {
  const topology = await fetchTopology();
  updateVisualization(topology);
  checkForChanges(topology);
}, 30000); // Every 30 seconds
```

### Change Detection

Compare topology over time:

```bash
# Save current topology
curl -H "Authorization: Bearer $TOKEN" \
  "https://ovncp.example.com/api/v1/visualization/topology" \
  -o topology-$(date +%Y%m%d).json

# Compare with previous
diff topology-20240119.json topology-20240120.json
```

## Performance Considerations

### Large Networks

For networks with thousands of components:

1. Use filtering to reduce the dataset:
   ```bash
   curl -H "Authorization: Bearer $TOKEN" \
     "https://ovncp.example.com/api/v1/visualization/topology?name=prod-*&maxNodes=500"
   ```

2. Use minimal detail level:
   ```bash
   curl -H "Authorization: Bearer $TOKEN" \
     "https://ovncp.example.com/api/v1/visualization/topology?detail=minimal"
   ```

3. Disable unnecessary components:
   ```bash
   curl -H "Authorization: Bearer $TOKEN" \
     "https://ovncp.example.com/api/v1/visualization/topology?ports=false&acls=false"
   ```

### Caching

The topology endpoint is cached for performance. Cache is invalidated when:
- Network components are created/updated/deleted
- Manual cache clear is triggered
- Cache TTL expires (default: 30 seconds)

## Client Libraries

### JavaScript/TypeScript

```typescript
import { OVNCPClient } from '@ovncp/client';

const client = new OVNCPClient({
  baseURL: 'https://ovncp.example.com',
  token: 'your-api-token'
});

// Get topology
const topology = await client.visualization.getTopology({
  layout: 'force',
  detail: 'medium'
});

// Export as DOT
const dot = await client.visualization.export('dot');

// Find path
const path = await client.visualization.findPath(
  'switch:uuid-1234',
  'switch:uuid-5678'
);
```

### Python

```python
from ovncp import Client

client = Client(
    base_url='https://ovncp.example.com',
    token='your-api-token'
)

# Get topology
topology = client.visualization.get_topology(
    layout='hierarchical',
    detail='full'
)

# Export as Mermaid
mermaid = client.visualization.export(format='mermaid')

# Save as HTML
html = client.visualization.export(format='html')
with open('topology.html', 'w') as f:
    f.write(html)
```

### Go

```go
client := ovncp.NewClient("https://ovncp.example.com", "your-api-token")

// Get topology
topology, err := client.Visualization.GetTopology(&ovncp.TopologyOptions{
    Layout: "force",
    Detail: "medium",
})

// Export as DOT
dot, err := client.Visualization.Export("dot", nil)

// Find path
path, err := client.Visualization.FindPath("switch:uuid-1234", "switch:uuid-5678")
```