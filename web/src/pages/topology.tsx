import { useState } from 'react';
import { useTopology } from '@/hooks/use-topology';
import NetworkGraph from '@/components/topology/network-graph';
import NodeDetails from '@/components/topology/node-details';
import { Card, CardContent } from '@/components/ui/card';
import { Loader2, AlertCircle } from 'lucide-react';
import { useToast } from '@/components/ui/use-toast';
import { api } from '@/lib/api';
import { useAuthStore } from '@/stores/auth';

export default function TopologyPage() {
  const { data, isLoading, error } = useTopology();
  const [selectedNode, setSelectedNode] = useState<any>(null);
  const { toast } = useToast();
  const user = useAuthStore((state) => state.user);
  const canEdit = user?.role === 'admin' || user?.role === 'operator';

  const handleNodeClick = (node: any) => {
    setSelectedNode(node);
  };

  const handleCloseDetails = () => {
    setSelectedNode(null);
  };

  const handleEdit = () => {
    if (!selectedNode) return;
    
    // Navigate to the appropriate edit page based on node type
    const { type, resource } = selectedNode;
    if (type === 'switch') {
      window.location.href = `/switches?edit=${resource.uuid}`;
    } else if (type === 'router') {
      window.location.href = `/routers?edit=${resource.uuid}`;
    } else if (type === 'port') {
      window.location.href = `/ports?edit=${resource.uuid}`;
    }
  };

  const handleDelete = async () => {
    if (!selectedNode || !canEdit) return;
    
    const { type, resource } = selectedNode;
    
    if (!confirm(`Are you sure you want to delete ${resource.name}?`)) {
      return;
    }

    try {
      if (type === 'switch') {
        await api.deleteLogicalSwitch(resource.uuid);
      } else if (type === 'router') {
        await api.deleteLogicalRouter(resource.uuid);
      } else if (type === 'port') {
        await api.deleteLogicalSwitchPort(resource.uuid);
      }
      
      toast({
        title: 'Success',
        description: `${resource.name} has been deleted.`,
      });
      
      setSelectedNode(null);
      // Refresh will happen automatically via React Query
    } catch (error) {
      toast({
        title: 'Error',
        description: error instanceof Error ? error.message : 'Failed to delete resource',
        variant: 'destructive',
      });
    }
  };

  if (isLoading) {
    return (
      <div className="h-[calc(100vh-10rem)] flex items-center justify-center">
        <Card>
          <CardContent className="flex items-center gap-2 p-6">
            <Loader2 className="h-4 w-4 animate-spin" />
            Loading topology data...
          </CardContent>
        </Card>
      </div>
    );
  }

  if (error) {
    return (
      <div className="h-[calc(100vh-10rem)] flex items-center justify-center">
        <Card>
          <CardContent className="flex items-center gap-2 p-6 text-destructive">
            <AlertCircle className="h-4 w-4" />
            Failed to load topology data
          </CardContent>
        </Card>
      </div>
    );
  }

  return (
    <div className="space-y-4">
      <div>
        <h1 className="text-3xl font-bold">Network Topology</h1>
        <p className="text-muted-foreground">
          Interactive visualization of your OVN network. Click on nodes to view details.
        </p>
      </div>
      
      <div className="relative h-[calc(100vh-12rem)] border rounded-lg overflow-hidden">
        {data && (
          <>
            <NetworkGraph
              data={data}
              onNodeClick={handleNodeClick}
            />
            <NodeDetails
              node={selectedNode}
              onClose={handleCloseDetails}
              onEdit={canEdit ? handleEdit : undefined}
              onDelete={canEdit ? handleDelete : undefined}
            />
          </>
        )}
      </div>
    </div>
  );
}