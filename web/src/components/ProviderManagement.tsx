import { useState, useEffect } from 'react';
import { api, type Provider } from '../lib/api';
import Toast from './Toast';

interface ToastState {
  show: boolean;
  message: string;
  type: 'success' | 'error' | 'info';
}

export default function ProviderManagement() {
  const [providers, setProviders] = useState<Provider[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [showCreateModal, setShowCreateModal] = useState(false);
  const [editingProvider, setEditingProvider] = useState<Provider | null>(null);
  const [toast, setToast] = useState<ToastState>({ show: false, message: '', type: 'info' });

  const showToast = (message: string, type: 'success' | 'error' | 'info' = 'info') => {
    setToast({ show: true, message, type });
  };

  const fetchProviders = async () => {
    try {
      const response = await api.getProviders();
      setProviders(response.data);
      setError(null);
    } catch (err) {
      setError(err instanceof Error ? err.message : '获取供应商列表失败');
    } finally {
      setLoading(false);
    }
  };

  useEffect(() => {
    fetchProviders();
  }, []);

  const handleToggleEnabled = async (id: number, enabled: boolean) => {
    try {
      await api.toggleProviderEnabled(id, enabled);
      await fetchProviders();
      showToast('状态切换成功', 'success');
    } catch (err) {
      showToast('切换状态失败: ' + (err instanceof Error ? err.message : '未知错误'), 'error');
    }
  };

  const handleHealthCheck = async (id: number) => {
    try {
      const result = await api.healthCheckProvider(id);
      const statusText = result.healthy ? '健康 ✓' : '异常 ✗';

      // 构建详细的提示信息
      let message = `健康检查完成 - 状态: ${statusText}, 响应时间: ${result.response_time_ms}ms`;

      // 如果有错误信息或状态码，添加到提示中
      if (!result.healthy) {
        if (result.status_code) {
          message += `, HTTP ${result.status_code}`;
        }
        if (result.error) {
          message += ` (${result.error})`;
        }
      }

      showToast(message, result.healthy ? 'success' : 'error');
      await fetchProviders();
    } catch (err) {
      showToast('健康检查失败: ' + (err instanceof Error ? err.message : '未知错误'), 'error');
    }
  };

  const handleDelete = async (id: number, name: string) => {
    if (!confirm(`确定要删除供应商 "${name}" 吗？`)) return;

    try {
      await api.deleteProvider(id);
      await fetchProviders();
      showToast('删除成功', 'success');
    } catch (err) {
      showToast('删除失败: ' + (err instanceof Error ? err.message : '未知错误'), 'error');
    }
  };

  const handleCreate = () => {
    setEditingProvider(null);
    setShowCreateModal(true);
  };

  const handleEdit = (provider: Provider) => {
    setEditingProvider(provider);
    setShowCreateModal(true);
  };

  if (loading) {
    return (
      <div className="min-h-screen flex items-center justify-center">
        <div className="text-gray-600 text-lg">加载中...</div>
      </div>
    );
  }

  if (error) {
    return (
      <div className="min-h-screen flex items-center justify-center">
        <div className="bg-red-50 text-red-600 px-6 py-4 rounded-lg">
          错误: {error}
        </div>
      </div>
    );
  }

  return (
    <div className="min-h-screen bg-gray-50 py-8 px-4 sm:px-6 lg:px-8">
      <div className="max-w-7xl mx-auto">
        {/* 头部 */}
        <div className="flex items-center justify-between mb-8">
          <div>
            <h1 className="text-3xl font-bold text-gray-900">供应商管理</h1>
            <p className="mt-2 text-sm text-gray-600">
              管理 API 供应商配置和健康状态
            </p>
          </div>
          <div className="flex space-x-4">
            <a
              href="/"
              className="px-4 py-2 text-sm font-medium text-gray-700 bg-white border border-gray-300 rounded-md hover:bg-gray-50"
            >
              返回首页
            </a>
            <button
              onClick={handleCreate}
              className="px-4 py-2 text-sm font-medium text-white bg-blue-600 rounded-md hover:bg-blue-700"
            >
              + 添加供应商
            </button>
          </div>
        </div>

        {/* 供应商列表 */}
        <div className="bg-white shadow rounded-lg overflow-hidden">
          <table className="min-w-full divide-y divide-gray-200">
            <thead className="bg-gray-50">
              <tr>
                <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">
                  名称
                </th>
                <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">
                  Base URL
                </th>
                <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">
                  健康状态
                </th>
                <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">
                  启用状态
                </th>
                <th className="px-6 py-3 text-right text-xs font-medium text-gray-500 uppercase tracking-wider">
                  操作
                </th>
              </tr>
            </thead>
            <tbody className="bg-white divide-y divide-gray-200">
              {providers.length > 0 ? (
                providers.map((provider) => (
                  <tr key={provider.id} className="hover:bg-gray-50">
                    <td className="px-6 py-4 whitespace-nowrap">
                      <div className="text-sm font-medium text-gray-900">
                        {provider.name}
                      </div>
                    </td>
                    <td className="px-6 py-4">
                      <div className="text-sm text-gray-900">
                        {provider.base_url}
                      </div>
                    </td>
                    <td className="px-6 py-4 whitespace-nowrap">
                      <span className={`inline-flex px-2 py-1 text-xs font-semibold rounded-full ${
                        provider.health_status === 'healthy'
                          ? 'bg-green-100 text-green-800'
                          : provider.health_status === 'unhealthy'
                          ? 'bg-red-100 text-red-800'
                          : 'bg-gray-100 text-gray-800'
                      }`}>
                        {provider.health_status}
                      </span>
                    </td>
                    <td className="px-6 py-4 whitespace-nowrap">
                      <button
                        onClick={() => handleToggleEnabled(provider.id, !provider.enabled)}
                        className={`relative inline-flex h-6 w-11 items-center rounded-full transition-colors ${
                          provider.enabled ? 'bg-blue-600' : 'bg-gray-200'
                        }`}
                      >
                        <span
                          className={`inline-block h-4 w-4 transform rounded-full bg-white transition-transform ${
                            provider.enabled ? 'translate-x-6' : 'translate-x-1'
                          }`}
                        />
                      </button>
                    </td>
                    <td className="px-6 py-4 whitespace-nowrap text-right text-sm font-medium">
                      <button
                        onClick={() => handleHealthCheck(provider.id)}
                        className="text-blue-600 hover:text-blue-900 mr-4"
                      >
                        检查
                      </button>
                      <button
                        onClick={() => handleEdit(provider)}
                        className="text-indigo-600 hover:text-indigo-900 mr-4"
                      >
                        编辑
                      </button>
                      <button
                        onClick={() => handleDelete(provider.id, provider.name)}
                        className="text-red-600 hover:text-red-900"
                      >
                        删除
                      </button>
                    </td>
                  </tr>
                ))
              ) : (
                <tr>
                  <td colSpan={6} className="px-6 py-8 text-center text-gray-500">
                    暂无供应商，点击右上角添加
                  </td>
                </tr>
              )}
            </tbody>
          </table>
        </div>
      </div>

      {/* 创建/编辑模态框 */}
      {showCreateModal && (
        <ProviderModal
          provider={editingProvider}
          onClose={() => setShowCreateModal(false)}
          onSuccess={() => {
            setShowCreateModal(false);
            fetchProviders();
            showToast(editingProvider ? '供应商更新成功' : '供应商创建成功', 'success');
          }}
          onError={(message) => showToast(message, 'error')}
        />
      )}

      {/* Toast 通知 */}
      {toast.show && (
        <Toast
          message={toast.message}
          type={toast.type}
          onClose={() => setToast({ ...toast, show: false })}
        />
      )}
    </div>
  );
}

// 供应商创建/编辑模态框组件
function ProviderModal({
  provider,
  onClose,
  onSuccess,
  onError,
}: {
  provider: Provider | null;
  onClose: () => void;
  onSuccess: () => void;
  onError: (message: string) => void;
}) {
  const [formData, setFormData] = useState({
    name: provider?.name || '',
    base_url: provider?.base_url || '',
    api_key: provider?.api_key || '',
    test_model: provider?.test_model || 'gpt-3.5-turbo',
    enabled: provider?.enabled ?? true,
  });
  const [submitting, setSubmitting] = useState(false);

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    setSubmitting(true);

    try {
      if (provider) {
        await api.updateProvider(provider.id, formData);
      } else {
        await api.createProvider(formData);
      }
      onSuccess();
    } catch (err) {
      onError('保存失败: ' + (err instanceof Error ? err.message : '未知错误'));
    } finally {
      setSubmitting(false);
    }
  };

  return (
    <div className="fixed inset-0 bg-gray-600 bg-opacity-50 flex items-center justify-center z-50">
      <div className="bg-white rounded-lg shadow-xl max-w-md w-full mx-4">
        <div className="px-6 py-4 border-b border-gray-200">
          <h3 className="text-lg font-medium text-gray-900">
            {provider ? '编辑供应商' : '添加供应商'}
          </h3>
        </div>

        <form onSubmit={handleSubmit} className="px-6 py-4 space-y-4">
          <div>
            <label className="block text-sm font-medium text-gray-700 mb-1">
              名称 *
            </label>
            <input
              type="text"
              required
              value={formData.name}
              onChange={(e) => setFormData({ ...formData, name: e.target.value })}
              className="w-full px-3 py-2 border border-gray-300 rounded-md focus:outline-none focus:ring-2 focus:ring-blue-500"
            />
          </div>

          <div>
            <label className="block text-sm font-medium text-gray-700 mb-1">
              Base URL *
            </label>
            <input
              type="url"
              required
              value={formData.base_url}
              onChange={(e) => setFormData({ ...formData, base_url: e.target.value })}
              className="w-full px-3 py-2 border border-gray-300 rounded-md focus:outline-none focus:ring-2 focus:ring-blue-500"
              placeholder="https://api.example.com"
            />
            <p className="mt-1 text-xs text-gray-500">
              ⚠️ 请勿在 URL 末尾添加 / 或 /v1 等路径
            </p>
          </div>

          <div>
            <label className="block text-sm font-medium text-gray-700 mb-1">
              API Key *
            </label>
            <input
              type="password"
              required
              value={formData.api_key}
              onChange={(e) => setFormData({ ...formData, api_key: e.target.value })}
              className="w-full px-3 py-2 border border-gray-300 rounded-md focus:outline-none focus:ring-2 focus:ring-blue-500"
            />
          </div>

          <div>
            <label className="block text-sm font-medium text-gray-700 mb-1">
              测试模型 *
            </label>
            <input
              type="text"
              required
              value={formData.test_model}
              onChange={(e) => setFormData({ ...formData, test_model: e.target.value })}
              className="w-full px-3 py-2 border border-gray-300 rounded-md focus:outline-none focus:ring-2 focus:ring-blue-500"
              placeholder="gpt-3.5-turbo"
            />
            <p className="mt-1 text-xs text-gray-500">
              健康检查将使用此模型发送测试请求
            </p>
          </div>

          <div className="flex items-center">
            <input
              type="checkbox"
              id="enabled"
              checked={formData.enabled}
              onChange={(e) => setFormData({ ...formData, enabled: e.target.checked })}
              className="h-4 w-4 text-blue-600 focus:ring-blue-500 border-gray-300 rounded"
            />
            <label htmlFor="enabled" className="ml-2 block text-sm text-gray-900">
              启用该供应商
            </label>
          </div>

          <div className="flex justify-end space-x-3 pt-4">
            <button
              type="button"
              onClick={onClose}
              disabled={submitting}
              className="px-4 py-2 text-sm font-medium text-gray-700 bg-white border border-gray-300 rounded-md hover:bg-gray-50 disabled:opacity-50"
            >
              取消
            </button>
            <button
              type="submit"
              disabled={submitting}
              className="px-4 py-2 text-sm font-medium text-white bg-blue-600 rounded-md hover:bg-blue-700 disabled:opacity-50"
            >
              {submitting ? '保存中...' : '保存'}
            </button>
          </div>
        </form>
      </div>
    </div>
  );
}
