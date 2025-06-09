import { useEffect, useRef, useState } from 'react';
import cytoscape, { Core, StylesheetStyle } from 'cytoscape';
import cola from 'cytoscape-cola';
import { useTheme } from '@/components/theme-provider';
import { TopologyData } from '@/hooks/use-topology';
import { Button } from '@/components/ui/button';
import { ZoomIn, ZoomOut, Maximize2, Download } from 'lucide-react';

// Register the cola layout
cytoscape.use(cola);

interface NetworkGraphProps {
  data: TopologyData;
  onNodeClick?: (node: cytoscape.NodeSingular) => void;
}

export default function NetworkGraph({ data, onNodeClick }: NetworkGraphProps) {
  const containerRef = useRef<HTMLDivElement>(null);
  const cyRef = useRef<Core | null>(null);
  const { theme } = useTheme();
  const [, setSelectedNode] = useState<string | null>(null);

  useEffect(() => {
    if (!containerRef.current) return;

    // Create nodes and edges from topology data
    const elements: cytoscape.ElementDefinition[] = [];

    // Add switches as nodes
    data.switches.forEach((sw) => {
      elements.push({
        data: {
          id: `switch-${sw.uuid}`,
          label: sw.name,
          type: 'switch',
          resource: sw,
        },
        classes: 'switch',
      });
    });

    // Add routers as nodes
    data.routers.forEach((router) => {
      elements.push({
        data: {
          id: `router-${router.uuid}`,
          label: router.name,
          type: 'router',
          resource: router,
        },
        classes: 'router',
      });
    });

    // Add ports and create edges
    data.ports.forEach((port) => {
      // Find the switch this port belongs to
      const parentSwitch = data.switches.find(sw => sw.ports?.includes(port.uuid));
      if (!parentSwitch) return;

      elements.push({
        data: {
          id: `port-${port.uuid}`,
          label: port.name,
          type: 'port',
          parent: `switch-${parentSwitch.uuid}`,
          resource: port,
        },
        classes: 'port',
      });

      // If port is connected to a router, create an edge
      if (port.type === 'router' && port.options?.['router-port']) {
        const routerPortName = port.options['router-port'];
        // Find the router that has this port
        const connectedRouter = data.routers.find(r => 
          r.ports?.some(p => p === routerPortName)
        );
        
        if (connectedRouter) {
          elements.push({
            data: {
              id: `edge-${port.uuid}`,
              source: `switch-${parentSwitch.uuid}`,
              target: `router-${connectedRouter.uuid}`,
              label: port.name,
            },
            classes: 'router-connection',
          });
        }
      }
    });

    // Define styles based on theme
    const isDark = theme === 'dark';
    const style: StylesheetStyle[] = [
      {
        selector: 'node',
        style: {
          'background-color': isDark ? '#1e293b' : '#e2e8f0',
          'border-color': isDark ? '#475569' : '#94a3b8',
          'border-width': 2,
          'label': 'data(label)',
          'text-valign': 'center',
          'text-halign': 'center',
          'color': isDark ? '#f1f5f9' : '#1e293b',
          'font-size': '12px',
          'text-outline-color': isDark ? '#1e293b' : '#ffffff',
          'text-outline-width': 2,
          'width': 60,
          'height': 60,
        },
      },
      {
        selector: 'node.switch',
        style: {
          'background-color': isDark ? '#1e40af' : '#3b82f6',
          'shape': 'round-rectangle',
        },
      },
      {
        selector: 'node.router',
        style: {
          'background-color': isDark ? '#059669' : '#10b981',
          'shape': 'diamond',
          'width': 70,
          'height': 70,
        },
      },
      {
        selector: 'node.port',
        style: {
          'background-color': isDark ? '#7c3aed' : '#8b5cf6',
          'shape': 'ellipse',
          'width': 40,
          'height': 40,
          'font-size': '10px',
        },
      },
      {
        selector: 'edge',
        style: {
          'width': 2,
          'line-color': isDark ? '#64748b' : '#94a3b8',
          'target-arrow-color': isDark ? '#64748b' : '#94a3b8',
          'target-arrow-shape': 'triangle',
          'curve-style': 'bezier',
          'label': 'data(label)',
          'font-size': '10px',
          'text-rotation': 'autorotate',
          'text-background-color': isDark ? '#0f172a' : '#ffffff',
          'text-background-opacity': 0.8,
          'text-background-padding': '2px',
          'color': isDark ? '#cbd5e1' : '#475569',
        },
      },
      {
        selector: 'node:selected',
        style: {
          'border-color': '#f59e0b',
          'border-width': 4,
          'background-color': isDark ? '#f59e0b' : '#fbbf24',
        },
      },
      {
        selector: 'edge:selected',
        style: {
          'line-color': '#f59e0b',
          'target-arrow-color': '#f59e0b',
          'width': 3,
        },
      },
    ];

    // Initialize Cytoscape
    const cy = cytoscape({
      container: containerRef.current,
      elements,
      style,
      layout: {
        name: 'cola',
        animate: true,
        randomize: false,
        convergenceThreshold: 0.0001,
        edgeLength: 150,
        nodeSpacing: 50,
        maxSimulationTime: 2000,
      } as cytoscape.LayoutOptions,
      minZoom: 0.1,
      maxZoom: 4,
      wheelSensitivity: 0.1,
    });

    // Event handlers
    cy.on('tap', 'node', (event) => {
      const node = event.target;
      const nodeData = node.data();
      setSelectedNode(nodeData.id);
      
      if (onNodeClick) {
        onNodeClick(node);
      }
    });

    cy.on('tap', (event) => {
      if (event.target === cy) {
        setSelectedNode(null);
      }
    });

    cyRef.current = cy;

    return () => {
      cy.destroy();
    };
  }, [data, theme, onNodeClick]);

  const handleZoomIn = () => {
    if (cyRef.current) {
      cyRef.current.zoom(cyRef.current.zoom() * 1.2);
    }
  };

  const handleZoomOut = () => {
    if (cyRef.current) {
      cyRef.current.zoom(cyRef.current.zoom() * 0.8);
    }
  };

  const handleFit = () => {
    if (cyRef.current) {
      cyRef.current.fit(undefined, 50);
    }
  };

  const handleExport = () => {
    if (cyRef.current) {
      const png = cyRef.current.png({ scale: 2, bg: theme === 'dark' ? '#0f172a' : '#ffffff' });
      const link = document.createElement('a');
      link.download = 'network-topology.png';
      link.href = png;
      link.click();
    }
  };

  return (
    <div className="relative h-full">
      <div ref={containerRef} className="w-full h-full bg-background" />
      
      {/* Controls */}
      <div className="absolute top-4 right-4 flex flex-col gap-2">
        <Button
          variant="outline"
          size="icon"
          onClick={handleZoomIn}
          title="Zoom In"
        >
          <ZoomIn className="h-4 w-4" />
        </Button>
        <Button
          variant="outline"
          size="icon"
          onClick={handleZoomOut}
          title="Zoom Out"
        >
          <ZoomOut className="h-4 w-4" />
        </Button>
        <Button
          variant="outline"
          size="icon"
          onClick={handleFit}
          title="Fit to Screen"
        >
          <Maximize2 className="h-4 w-4" />
        </Button>
        <Button
          variant="outline"
          size="icon"
          onClick={handleExport}
          title="Export as PNG"
        >
          <Download className="h-4 w-4" />
        </Button>
      </div>
      
      {/* Legend */}
      <div className="absolute bottom-4 left-4 bg-card p-4 rounded-lg border shadow-sm">
        <h4 className="text-sm font-medium mb-2">Legend</h4>
        <div className="space-y-2 text-xs">
          <div className="flex items-center gap-2">
            <div className="w-4 h-4 bg-blue-500 rounded" />
            <span>Logical Switch</span>
          </div>
          <div className="flex items-center gap-2">
            <div className="w-4 h-4 bg-green-500 transform rotate-45" />
            <span>Logical Router</span>
          </div>
          <div className="flex items-center gap-2">
            <div className="w-4 h-4 bg-purple-500 rounded-full" />
            <span>Switch Port</span>
          </div>
        </div>
      </div>
    </div>
  );
}