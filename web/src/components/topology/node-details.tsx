import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/card';
import { Badge } from '@/components/ui/badge';
import { Button } from '@/components/ui/button';
import { Tabs, TabsContent, TabsList, TabsTrigger } from '@/components/ui/tabs';
import { LogicalSwitch, LogicalRouter, LogicalSwitchPort } from '@/types';
import { X, Edit, Trash2 } from 'lucide-react';

interface NodeDetailsProps {
  node: {
    id: string;
    type: 'switch' | 'router' | 'port';
    resource: LogicalSwitch | LogicalRouter | LogicalSwitchPort;
  } | null;
  onClose: () => void;
  onEdit?: () => void;
  onDelete?: () => void;
}

export default function NodeDetails({ node, onClose, onEdit, onDelete }: NodeDetailsProps) {
  if (!node) return null;

  const { type, resource } = node;

  const renderSwitchDetails = (sw: LogicalSwitch) => (
    <>
      <div className="space-y-2">
        <div>
          <span className="text-sm font-medium">UUID:</span>
          <p className="text-sm text-muted-foreground font-mono">{sw.uuid}</p>
        </div>
        <div>
          <span className="text-sm font-medium">Ports:</span>
          <p className="text-sm text-muted-foreground">{sw.ports?.length || 0} ports</p>
        </div>
        <div>
          <span className="text-sm font-medium">ACLs:</span>
          <p className="text-sm text-muted-foreground">{sw.acls?.length || 0} rules</p>
        </div>
        <div>
          <span className="text-sm font-medium">Load Balancers:</span>
          <p className="text-sm text-muted-foreground">{sw.load_balancers?.length || 0} configured</p>
        </div>
      </div>
    </>
  );

  const renderRouterDetails = (router: LogicalRouter) => (
    <>
      <div className="space-y-2">
        <div>
          <span className="text-sm font-medium">UUID:</span>
          <p className="text-sm text-muted-foreground font-mono">{router.uuid}</p>
        </div>
        <div>
          <span className="text-sm font-medium">Ports:</span>
          <p className="text-sm text-muted-foreground">{router.ports?.length || 0} ports</p>
        </div>
        <div>
          <span className="text-sm font-medium">Static Routes:</span>
          <p className="text-sm text-muted-foreground">{router.static_routes?.length || 0} routes</p>
        </div>
        <div>
          <span className="text-sm font-medium">NAT Rules:</span>
          <p className="text-sm text-muted-foreground">{router.nat?.length || 0} rules</p>
        </div>
      </div>
    </>
  );

  const renderPortDetails = (port: LogicalSwitchPort) => (
    <>
      <div className="space-y-2">
        <div>
          <span className="text-sm font-medium">UUID:</span>
          <p className="text-sm text-muted-foreground font-mono">{port.uuid}</p>
        </div>
        <div>
          <span className="text-sm font-medium">Type:</span>
          <p className="text-sm text-muted-foreground">{port.type || 'Regular'}</p>
        </div>
        <div>
          <span className="text-sm font-medium">Status:</span>
          <Badge variant={port.up ? 'default' : 'secondary'}>
            {port.up ? 'Up' : 'Down'}
          </Badge>
        </div>
        <div>
          <span className="text-sm font-medium">Enabled:</span>
          <Badge variant={port.enabled !== false ? 'default' : 'secondary'}>
            {port.enabled !== false ? 'Enabled' : 'Disabled'}
          </Badge>
        </div>
        {port.addresses && port.addresses.length > 0 && (
          <div>
            <span className="text-sm font-medium">Addresses:</span>
            <div className="mt-1 space-y-1">
              {port.addresses.map((addr, i) => (
                <p key={i} className="text-sm text-muted-foreground font-mono">{addr}</p>
              ))}
            </div>
          </div>
        )}
      </div>
    </>
  );

  const renderMetadata = (metadata: Record<string, string> | undefined, title: string) => {
    if (!metadata || Object.keys(metadata).length === 0) return null;
    
    return (
      <div className="space-y-2">
        <h4 className="text-sm font-medium">{title}</h4>
        <div className="space-y-1">
          {Object.entries(metadata).map(([key, value]) => (
            <div key={key} className="text-sm">
              <span className="font-mono text-muted-foreground">{key}:</span>{' '}
              <span className="text-muted-foreground">{value}</span>
            </div>
          ))}
        </div>
      </div>
    );
  };

  return (
    <Card className="absolute top-4 left-4 w-96 max-h-[calc(100vh-8rem)] overflow-auto z-10">
      <CardHeader className="pb-4">
        <div className="flex items-start justify-between">
          <div>
            <CardTitle className="text-lg flex items-center gap-2">
              {resource.name}
              <Badge variant="outline" className="text-xs">
                {type}
              </Badge>
            </CardTitle>
            <CardDescription>
              {type === 'switch' && 'Logical Switch'}
              {type === 'router' && 'Logical Router'}
              {type === 'port' && 'Logical Switch Port'}
            </CardDescription>
          </div>
          <Button
            variant="ghost"
            size="icon"
            className="h-8 w-8"
            onClick={onClose}
          >
            <X className="h-4 w-4" />
          </Button>
        </div>
      </CardHeader>
      <CardContent className="space-y-4">
        <Tabs defaultValue="details" className="w-full">
          <TabsList className="grid w-full grid-cols-2">
            <TabsTrigger value="details">Details</TabsTrigger>
            <TabsTrigger value="metadata">Metadata</TabsTrigger>
          </TabsList>
          <TabsContent value="details" className="mt-4 space-y-4">
            {type === 'switch' && renderSwitchDetails(resource as LogicalSwitch)}
            {type === 'router' && renderRouterDetails(resource as LogicalRouter)}
            {type === 'port' && renderPortDetails(resource as LogicalSwitchPort)}
            
            {resource.created_at && (
              <div>
                <span className="text-sm font-medium">Created:</span>
                <p className="text-sm text-muted-foreground">
                  {new Date(resource.created_at).toLocaleString()}
                </p>
              </div>
            )}
            
            {resource.updated_at && (
              <div>
                <span className="text-sm font-medium">Updated:</span>
                <p className="text-sm text-muted-foreground">
                  {new Date(resource.updated_at).toLocaleString()}
                </p>
              </div>
            )}
          </TabsContent>
          <TabsContent value="metadata" className="mt-4 space-y-4">
            {type === 'switch' && (
              <>
                {renderMetadata((resource as LogicalSwitch).other_config, 'Other Config')}
                {renderMetadata((resource as LogicalSwitch).external_ids, 'External IDs')}
              </>
            )}
            {type === 'router' && (
              <>
                {renderMetadata((resource as LogicalRouter).options, 'Options')}
                {renderMetadata((resource as LogicalRouter).external_ids, 'External IDs')}
              </>
            )}
            {type === 'port' && (
              <>
                {renderMetadata((resource as LogicalSwitchPort).options, 'Options')}
                {renderMetadata((resource as LogicalSwitchPort).external_ids, 'External IDs')}
              </>
            )}
          </TabsContent>
        </Tabs>
        
        <div className="flex gap-2 pt-4 border-t">
          {onEdit && (
            <Button variant="outline" size="sm" onClick={onEdit}>
              <Edit className="h-4 w-4 mr-2" />
              Edit
            </Button>
          )}
          {onDelete && (
            <Button variant="outline" size="sm" onClick={onDelete}>
              <Trash2 className="h-4 w-4 mr-2" />
              Delete
            </Button>
          )}
        </div>
      </CardContent>
    </Card>
  );
}