// Auth types
export type UserRole = 'admin' | 'operator' | 'viewer';

export interface User {
  id: string;
  email: string;
  name: string;
  picture?: string;
  provider: string;
  provider_id: string;
  role: UserRole;
  active: boolean;
  last_login_at?: string;
  created_at: string;
  updated_at: string;
}

export interface Session {
  access_token: string;
  refresh_token: string;
  expires_at: number;
  user: User;
}

// OVN types
export interface LogicalSwitch {
  uuid: string;
  name: string;
  ports?: string[];
  acls?: string[];
  load_balancers?: string[];
  other_config?: Record<string, string>;
  external_ids?: Record<string, string>;
  created_at?: string;
  updated_at?: string;
}

export interface LogicalRouter {
  uuid: string;
  name: string;
  ports?: string[];
  static_routes?: StaticRoute[];
  policies?: string[];
  nat?: string[];
  load_balancers?: string[];
  options?: Record<string, string>;
  external_ids?: Record<string, string>;
  created_at?: string;
  updated_at?: string;
}

export interface StaticRoute {
  ip_prefix: string;
  nexthop: string;
  output_port?: string;
  policy?: string;
}

export interface LogicalSwitchPort {
  uuid: string;
  name: string;
  type?: string;
  addresses?: string[];
  port_security?: string[];
  up?: boolean;
  enabled?: boolean;
  dhcpv4_options?: string;
  dhcpv6_options?: string;
  dynamic_addresses?: string;
  ha_chassis_group?: string;
  options?: Record<string, string>;
  external_ids?: Record<string, string>;
  created_at?: string;
  updated_at?: string;
}

export interface ACL {
  uuid: string;
  action: 'allow' | 'allow-related' | 'allow-stateless' | 'drop' | 'reject' | 'pass';
  direction: 'from-lport' | 'to-lport';
  match: string;
  priority: number;
  external_ids?: Record<string, string>;
  log?: boolean;
  meter?: string;
  name?: string;
  severity?: string;
  created_at?: string;
  updated_at?: string;
}

export interface LoadBalancer {
  uuid: string;
  name: string;
  vips: Record<string, string>;
  protocol?: 'tcp' | 'udp' | 'sctp';
  health_check?: HealthCheck[];
  ip_port_mappings?: Record<string, string>;
  selection_fields?: string[];
  external_ids?: Record<string, string>;
  created_at?: string;
  updated_at?: string;
}

export interface HealthCheck {
  vip: string;
  options: Record<string, string>;
}

// Transaction types
export interface TransactionOperation {
  id: string;
  type: 'create' | 'update' | 'delete';
  resource: 'logical-switch' | 'logical-router' | 'logical-switch-port' | 'acl' | 'load-balancer';
  resource_id?: string;
  switch_id?: string;
  data?: any;
}

export interface TransactionRequest {
  operations: TransactionOperation[];
}

export interface TransactionResult {
  success: boolean;
  results: Array<{
    operation_id: string;
    success: boolean;
    resource_id?: string;
    error?: string;
  }>;
}

// API Response types
export interface PaginatedResponse<T> {
  data: T[];
  total: number;
  limit: number;
  offset: number;
}

export interface ErrorResponse {
  error: string;
  details?: any;
}