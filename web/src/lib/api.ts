import { Session, User, UserRole, LogicalSwitch, LogicalRouter, LogicalSwitchPort, ACL, LoadBalancer, PaginatedResponse } from '@/types';

const API_BASE_URL = '/api/v1';

class ApiClient {
  private token: string | null = null;

  constructor() {
    // Load token from localStorage on init
    const stored = localStorage.getItem('ovncp_session');
    if (stored) {
      try {
        const session = JSON.parse(stored);
        this.token = session.access_token;
      } catch (e) {
        // Invalid stored session
        localStorage.removeItem('ovncp_session');
      }
    }
  }

  setToken(token: string | null) {
    this.token = token;
  }

  private async request<T>(
    path: string,
    options: RequestInit = {}
  ): Promise<T> {
    const headers: Record<string, string> = {
      'Content-Type': 'application/json',
    };

    if (this.token) {
      headers['Authorization'] = `Bearer ${this.token}`;
    }

    // Merge with any additional headers from options
    if (options.headers) {
      Object.assign(headers, options.headers);
    }

    const response = await fetch(`${API_BASE_URL}${path}`, {
      ...options,
      headers,
    });

    if (!response.ok) {
      const error = await response.json().catch(() => ({ error: 'Unknown error' }));
      throw new Error(error.error || `HTTP ${response.status}`);
    }

    return response.json();
  }

  // Auth endpoints
  async login(provider: string): Promise<{ auth_url: string }> {
    return this.request('/auth/login', {
      method: 'POST',
      body: JSON.stringify({ provider }),
    });
  }

  async localLogin(username: string, password: string): Promise<Session> {
    const session = await this.request<Session>('/auth/login/local', {
      method: 'POST',
      body: JSON.stringify({ username, password }),
    });
    
    // Store session
    localStorage.setItem('ovncp_session', JSON.stringify(session));
    this.setToken(session.access_token);
    
    return session;
  }

  async handleCallback(provider: string, code: string, state: string): Promise<Session> {
    const session = await this.request<Session>(`/auth/callback/${provider}?code=${encodeURIComponent(code)}&state=${encodeURIComponent(state)}`);
    
    // Store session
    localStorage.setItem('ovncp_session', JSON.stringify(session));
    this.setToken(session.access_token);
    
    return session;
  }

  async logout(): Promise<void> {
    try {
      await this.request('/auth/logout', { method: 'POST' });
    } finally {
      localStorage.removeItem('ovncp_session');
      this.setToken(null);
    }
  }

  async getProfile(): Promise<User> {
    return this.request('/auth/profile');
  }

  async refreshToken(refreshToken: string): Promise<Session> {
    const session = await this.request<Session>('/auth/refresh', {
      method: 'POST',
      body: JSON.stringify({ refresh_token: refreshToken }),
    });
    
    // Update stored session
    localStorage.setItem('ovncp_session', JSON.stringify(session));
    this.setToken(session.access_token);
    
    return session;
  }

  // User management endpoints (admin only)
  async listUsers(limit = 10, offset = 0): Promise<{ users: User[]; total: number }> {
    return this.request(`/auth/users?limit=${limit}&offset=${offset}`);
  }

  async getUser(userId: string): Promise<User> {
    return this.request(`/auth/users/${userId}`);
  }

  async updateUserRole(userId: string, role: UserRole): Promise<void> {
    await this.request(`/auth/users/${userId}/role`, {
      method: 'PUT',
      body: JSON.stringify({ role }),
    });
  }

  async deactivateUser(userId: string): Promise<void> {
    await this.request(`/auth/users/${userId}`, {
      method: 'DELETE',
    });
  }

  // Logical Switches
  async listLogicalSwitches(limit = 20, offset = 0): Promise<PaginatedResponse<LogicalSwitch>> {
    return this.request(`/logical-switches?limit=${limit}&offset=${offset}`);
  }

  async getLogicalSwitch(id: string): Promise<LogicalSwitch> {
    return this.request(`/logical-switches/${id}`);
  }

  async createLogicalSwitch(data: Partial<LogicalSwitch>): Promise<LogicalSwitch> {
    return this.request('/logical-switches', {
      method: 'POST',
      body: JSON.stringify(data),
    });
  }

  async updateLogicalSwitch(id: string, data: Partial<LogicalSwitch>): Promise<LogicalSwitch> {
    return this.request(`/logical-switches/${id}`, {
      method: 'PUT',
      body: JSON.stringify(data),
    });
  }

  async deleteLogicalSwitch(id: string): Promise<void> {
    return this.request(`/logical-switches/${id}`, {
      method: 'DELETE',
    });
  }

  // Logical Routers
  async listLogicalRouters(limit = 20, offset = 0): Promise<PaginatedResponse<LogicalRouter>> {
    return this.request(`/logical-routers?limit=${limit}&offset=${offset}`);
  }

  async getLogicalRouter(id: string): Promise<LogicalRouter> {
    return this.request(`/logical-routers/${id}`);
  }

  async createLogicalRouter(data: Partial<LogicalRouter>): Promise<LogicalRouter> {
    return this.request('/logical-routers', {
      method: 'POST',
      body: JSON.stringify(data),
    });
  }

  async updateLogicalRouter(id: string, data: Partial<LogicalRouter>): Promise<LogicalRouter> {
    return this.request(`/logical-routers/${id}`, {
      method: 'PUT',
      body: JSON.stringify(data),
    });
  }

  async deleteLogicalRouter(id: string): Promise<void> {
    return this.request(`/logical-routers/${id}`, {
      method: 'DELETE',
    });
  }

  // Logical Switch Ports
  async listLogicalSwitchPorts(switchId?: string, limit = 20, offset = 0): Promise<PaginatedResponse<LogicalSwitchPort>> {
    const query = new URLSearchParams({
      limit: limit.toString(),
      offset: offset.toString(),
    });
    if (switchId) {
      query.append('switch_id', switchId);
    }
    return this.request(`/logical-switch-ports?${query}`);
  }

  async getLogicalSwitchPort(id: string): Promise<LogicalSwitchPort> {
    return this.request(`/logical-switch-ports/${id}`);
  }

  async createLogicalSwitchPort(data: Partial<LogicalSwitchPort>): Promise<LogicalSwitchPort> {
    return this.request('/logical-switch-ports', {
      method: 'POST',
      body: JSON.stringify(data),
    });
  }

  async updateLogicalSwitchPort(id: string, data: Partial<LogicalSwitchPort>): Promise<LogicalSwitchPort> {
    return this.request(`/logical-switch-ports/${id}`, {
      method: 'PUT',
      body: JSON.stringify(data),
    });
  }

  async deleteLogicalSwitchPort(id: string): Promise<void> {
    return this.request(`/logical-switch-ports/${id}`, {
      method: 'DELETE',
    });
  }

  // ACLs
  async listACLs(switchId?: string, limit = 20, offset = 0): Promise<PaginatedResponse<ACL>> {
    const query = new URLSearchParams({
      limit: limit.toString(),
      offset: offset.toString(),
    });
    if (switchId) {
      query.append('switch_id', switchId);
    }
    return this.request(`/acls?${query}`);
  }

  async getACL(id: string): Promise<ACL> {
    return this.request(`/acls/${id}`);
  }

  async createACL(data: any): Promise<ACL> {
    return this.request('/acls', {
      method: 'POST',
      body: JSON.stringify(data),
    });
  }

  async updateACL(id: string, data: any): Promise<ACL> {
    return this.request(`/acls/${id}`, {
      method: 'PUT',
      body: JSON.stringify(data),
    });
  }

  async deleteACL(id: string): Promise<void> {
    return this.request(`/acls/${id}`, {
      method: 'DELETE',
    });
  }

  // Load Balancers
  async listLoadBalancers(limit = 20, offset = 0): Promise<PaginatedResponse<LoadBalancer>> {
    return this.request(`/load-balancers?limit=${limit}&offset=${offset}`);
  }

  async getLoadBalancer(id: string): Promise<LoadBalancer> {
    return this.request(`/load-balancers/${id}`);
  }

  async createLoadBalancer(data: any): Promise<LoadBalancer> {
    return this.request('/load-balancers', {
      method: 'POST',
      body: JSON.stringify(data),
    });
  }

  async updateLoadBalancer(id: string, data: any): Promise<LoadBalancer> {
    return this.request(`/load-balancers/${id}`, {
      method: 'PUT',
      body: JSON.stringify(data),
    });
  }

  async deleteLoadBalancer(id: string): Promise<void> {
    return this.request(`/load-balancers/${id}`, {
      method: 'DELETE',
    });
  }

  // Transactions
  async executeTransaction(operations: any[]) {
    return this.request('/transactions', {
      method: 'POST',
      body: JSON.stringify({ operations }),
    });
  }
}

export const api = new ApiClient();