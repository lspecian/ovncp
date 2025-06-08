import { useQuery } from '@tanstack/react-query';
import { api } from '@/lib/api';
import { LogicalSwitch, LogicalRouter, LogicalSwitchPort } from '@/types';

export interface TopologyData {
  switches: LogicalSwitch[];
  routers: LogicalRouter[];
  ports: LogicalSwitchPort[];
}

export function useTopology() {
  return useQuery({
    queryKey: ['topology'],
    queryFn: async (): Promise<TopologyData> => {
      const [switches, routers, ports] = await Promise.all([
        api.listLogicalSwitches(100, 0),
        api.listLogicalRouters(100, 0),
        api.listLogicalSwitchPorts(undefined, 100, 0),
      ]);
      
      return {
        switches: switches.data || [],
        routers: routers.data || [],
        ports: ports.data || [],
      };
    },
    refetchInterval: 30000, // Refresh every 30 seconds
  });
}