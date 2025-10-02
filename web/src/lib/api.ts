const API_BASE_URL = import.meta.env.PUBLIC_API_URL || 'http://localhost:8080';

export interface SystemStats {
  providers: {
    total: number;
    healthy: number;
    unhealthy: number;
  };
  requests: {
    total: number;
    current_qps: number;
  };
  recent_events: Array<{
    timestamp: string;
    type: string;
    message: string;
  }>;
}

export interface Provider {
  id: number;
  name: string;
  base_url: string;
  api_key: string;
  enabled: boolean;
  priority: number;
  health_status: string;
  created_at: string;
  updated_at: string;
}

export interface UnifiedModel {
  id: number;
  name: string;
  display_name: string;
  description: string;
  created_at: string;
  updated_at: string;
}

export interface Token {
  id: number;
  name: string;
  token_display: string;
  enabled: boolean;
  expires_at: string | null;
  created_at: string;
  updated_at: string;
}

// API 客户端
export const api = {
  // 统计
  async getStats(): Promise<SystemStats> {
    const res = await fetch(`${API_BASE_URL}/api/stats`);
    if (!res.ok) throw new Error('Failed to fetch stats');
    return res.json();
  },

  // 供应商
  async getProviders(): Promise<{ data: Provider[] }> {
    const res = await fetch(`${API_BASE_URL}/api/providers`);
    if (!res.ok) throw new Error('Failed to fetch providers');
    return res.json();
  },

  async createProvider(data: Partial<Provider>): Promise<Provider> {
    const res = await fetch(`${API_BASE_URL}/api/providers`, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify(data),
    });
    if (!res.ok) throw new Error('Failed to create provider');
    return res.json();
  },

  async updateProvider(id: number, data: Partial<Provider>): Promise<Provider> {
    const res = await fetch(`${API_BASE_URL}/api/providers/${id}`, {
      method: 'PUT',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify(data),
    });
    if (!res.ok) throw new Error('Failed to update provider');
    return res.json();
  },

  async deleteProvider(id: number): Promise<void> {
    const res = await fetch(`${API_BASE_URL}/api/providers/${id}`, {
      method: 'DELETE',
    });
    if (!res.ok) throw new Error('Failed to delete provider');
  },

  async toggleProviderEnabled(id: number, enabled: boolean): Promise<Provider> {
    const res = await fetch(`${API_BASE_URL}/api/providers/${id}/enabled`, {
      method: 'PATCH',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ enabled }),
    });
    if (!res.ok) throw new Error('Failed to toggle provider');
    return res.json();
  },

  async healthCheckProvider(id: number) {
    const res = await fetch(`${API_BASE_URL}/api/providers/${id}/health-check`, {
      method: 'POST',
    });
    if (!res.ok) throw new Error('Failed to health check provider');
    return res.json();
  },

  // 模型
  async getModels(): Promise<UnifiedModel[]> {
    const res = await fetch(`${API_BASE_URL}/api/models`);
    if (!res.ok) throw new Error('Failed to fetch models');
    return res.json();
  },

  // Token
  async getTokens(): Promise<Token[]> {
    const res = await fetch(`${API_BASE_URL}/api/tokens`);
    if (!res.ok) throw new Error('Failed to fetch tokens');
    return res.json();
  },

  async getToken(id: number): Promise<Token & { token: string }> {
    const res = await fetch(`${API_BASE_URL}/api/tokens/${id}`);
    if (!res.ok) throw new Error('Failed to fetch token');
    return res.json();
  },

  async createToken(data: { name: string; expires_at?: string; custom_token?: string }): Promise<Token & { token: string }> {
    const res = await fetch(`${API_BASE_URL}/api/tokens`, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify(data),
    });
    if (!res.ok) {
      const error = await res.json();
      throw new Error(error.error?.message || 'Failed to create token');
    }
    return res.json();
  },

  async deleteToken(id: number): Promise<void> {
    const res = await fetch(`${API_BASE_URL}/api/tokens/${id}`, {
      method: 'DELETE',
    });
    if (!res.ok) throw new Error('Failed to delete token');
  },
};
