const API_BASE_URL = import.meta.env.PUBLIC_API_URL || 'http://localhost:8080';

export interface SystemStats {
  providers: {
    total: number;
    healthy: number;
    unhealthy: number;
  };
  requests: {
    total: number;
  };
  recent_api_calls: Array<{
    id: number;
    timestamp: string;
    path: string;
    model: string;
    provider_name: string;
    status_code: number;
    response_time_ms: number;
    prompt_tokens: number;
    completion_tokens: number;
    total_tokens: number;
    request_summary: string;
    error_message?: string;
  }>;
}

export interface Provider {
  id: number;
  name: string;
  base_url: string;
  api_key: string;
  test_model: string;
  enabled: boolean;
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

export interface ModelInfo {
  id: string;
  object: string;
}

export interface AvailableModelsResponse {
  provider_id: number;
  provider_name: string;
  models: ModelInfo[];
  total: number;
  fetched_at: string;
}

export interface ModelMapping {
  id: number;
  unified_model_id: number;
  provider_id: number;
  provider_name?: string;
  target_model: string;
  weight: number;
  priority: number;
  enabled: boolean;
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

export interface HealthCheckResult {
  healthy: boolean;
  response_time_ms: number;
  status_code?: number;
  error?: string;
  checked_at: string;
}

export interface ModelTestResult {
  provider_id: number;
  provider_name: string;
  model_name: string;
  success: boolean;
  response_time_ms: number;
  status_code?: number;
  error?: string;
  tested_at: string;
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

  async healthCheckProvider(id: number): Promise<HealthCheckResult> {
    const res = await fetch(`${API_BASE_URL}/api/providers/${id}/health-check`, {
      method: 'POST',
    });
    if (!res.ok) throw new Error('Failed to health check provider');
    return res.json();
  },

  async testProviderModel(providerId: number, modelName: string): Promise<ModelTestResult> {
    const res = await fetch(`${API_BASE_URL}/api/providers/${providerId}/test-model`, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ model_name: modelName }),
    });
    if (!res.ok) throw new Error('Failed to test provider model');
    return res.json();
  },

  // 模型
  async getModels(): Promise<UnifiedModel[]> {
    const res = await fetch(`${API_BASE_URL}/api/models`);
    if (!res.ok) throw new Error('Failed to fetch models');
    const data = await res.json();
    // 后端返回的是 { models: [...], pagination: {...} } 格式
    return data.models || [];
  },

  async createModel(data: Partial<UnifiedModel>): Promise<UnifiedModel> {
    const res = await fetch(`${API_BASE_URL}/api/models`, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify(data),
    });
    if (!res.ok) {
      const error = await res.json();
      throw new Error(error.error || 'Failed to create model');
    }
    return res.json();
  },

  async updateModel(id: number, data: Partial<UnifiedModel>): Promise<UnifiedModel> {
    const res = await fetch(`${API_BASE_URL}/api/models/${id}`, {
      method: 'PUT',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify(data),
    });
    if (!res.ok) {
      const error = await res.json();
      throw new Error(error.error || 'Failed to update model');
    }
    return res.json();
  },

  async deleteModel(id: number): Promise<void> {
    const res = await fetch(`${API_BASE_URL}/api/models/${id}`, {
      method: 'DELETE',
    });
    if (!res.ok) throw new Error('Failed to delete model');
  },

  // 供应商模型发现
  async getProviderModels(providerId: number): Promise<AvailableModelsResponse> {
    const res = await fetch(`${API_BASE_URL}/api/providers/${providerId}/models`);
    if (!res.ok) throw new Error('Failed to fetch provider models');
    return res.json();
  },

  // 模型映射
  async getModelMappings(modelId: number): Promise<ModelMapping[]> {
    const res = await fetch(`${API_BASE_URL}/api/models/${modelId}/mappings`);
    if (!res.ok) throw new Error('Failed to fetch mappings');
    const data = await res.json();
    // 后端返回的是 { mappings: [...], total: number } 格式
    return data.mappings || [];
  },

  async createMapping(modelId: number, data: Partial<ModelMapping>): Promise<ModelMapping> {
    const res = await fetch(`${API_BASE_URL}/api/models/${modelId}/mappings`, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify(data),
    });
    if (!res.ok) {
      const error = await res.json();
      throw new Error(error.error || 'Failed to create mapping');
    }
    return res.json();
  },

  async updateMapping(mappingId: number, data: Partial<ModelMapping>): Promise<ModelMapping> {
    const res = await fetch(`${API_BASE_URL}/api/mappings/${mappingId}`, {
      method: 'PUT',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify(data),
    });
    if (!res.ok) throw new Error('Failed to update mapping');
    return res.json();
  },

  async deleteMapping(mappingId: number): Promise<void> {
    const res = await fetch(`${API_BASE_URL}/api/mappings/${mappingId}`, {
      method: 'DELETE',
    });
    if (!res.ok) throw new Error('Failed to delete mapping');
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
