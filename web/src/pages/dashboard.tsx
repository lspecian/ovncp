import { useQuery } from '@tanstack/react-query';
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card';
import { api } from '@/lib/api';
import { cn } from '@/lib/utils';
import { Network, Router, Server, Shield, LoaderIcon } from 'lucide-react';

export default function DashboardPage() {
  // Fetch counts for each resource type
  const { data: switches } = useQuery({
    queryKey: ['switches-count'],
    queryFn: () => api.listLogicalSwitches(1, 0),
  });
  
  const { data: routers } = useQuery({
    queryKey: ['routers-count'],
    queryFn: () => api.listLogicalRouters(1, 0),
  });
  
  const { data: ports } = useQuery({
    queryKey: ['ports-count'],
    queryFn: () => api.listLogicalSwitchPorts(undefined, 1, 0),
  });
  
  const { data: acls } = useQuery({
    queryKey: ['acls-count'],
    queryFn: () => api.listACLs(undefined, 1, 0),
  });
  
  const { data: loadBalancers } = useQuery({
    queryKey: ['load-balancers-count'],
    queryFn: () => api.listLoadBalancers(1, 0),
  });
  
  const stats = [
    {
      name: 'Logical Switches',
      value: switches?.total || 0,
      icon: Network,
      color: 'text-blue-600',
      bgColor: 'bg-blue-100',
    },
    {
      name: 'Logical Routers',
      value: routers?.total || 0,
      icon: Router,
      color: 'text-green-600',
      bgColor: 'bg-green-100',
    },
    {
      name: 'Switch Ports',
      value: ports?.total || 0,
      icon: Server,
      color: 'text-purple-600',
      bgColor: 'bg-purple-100',
    },
    {
      name: 'ACL Rules',
      value: acls?.total || 0,
      icon: Shield,
      color: 'text-orange-600',
      bgColor: 'bg-orange-100',
    },
    {
      name: 'Load Balancers',
      value: loadBalancers?.total || 0,
      icon: LoaderIcon,
      color: 'text-pink-600',
      bgColor: 'bg-pink-100',
    },
  ];
  
  return (
    <div className="space-y-6">
      <div>
        <h1 className="text-3xl font-bold">Dashboard</h1>
        <p className="text-muted-foreground">
          Overview of your OVN infrastructure
        </p>
      </div>
      
      <div className="grid gap-4 md:grid-cols-2 lg:grid-cols-3 xl:grid-cols-5">
        {stats.map((stat) => (
          <Card key={stat.name}>
            <CardHeader className="flex flex-row items-center justify-between space-y-0 pb-2">
              <CardTitle className="text-sm font-medium">
                {stat.name}
              </CardTitle>
              <div className={cn('rounded-full p-2', stat.bgColor)}>
                <stat.icon className={cn('h-4 w-4', stat.color)} />
              </div>
            </CardHeader>
            <CardContent>
              <div className="text-2xl font-bold">{stat.value}</div>
            </CardContent>
          </Card>
        ))}
      </div>
      
      <div className="grid gap-4 md:grid-cols-2">
        <Card>
          <CardHeader>
            <CardTitle>Recent Activity</CardTitle>
          </CardHeader>
          <CardContent>
            <p className="text-sm text-muted-foreground">
              Activity tracking will be implemented in a future update.
            </p>
          </CardContent>
        </Card>
        
        <Card>
          <CardHeader>
            <CardTitle>System Health</CardTitle>
          </CardHeader>
          <CardContent>
            <p className="text-sm text-muted-foreground">
              Health monitoring will be implemented in a future update.
            </p>
          </CardContent>
        </Card>
      </div>
    </div>
  );
}