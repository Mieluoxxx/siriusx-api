import { useState, useEffect } from 'react';
import { api, type Token } from '../lib/api';

export default function TokenManagement() {
  const [tokens, setTokens] = useState<Token[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [showCreateModal, setShowCreateModal] = useState(false);
  const [createdToken, setCreatedToken] = useState<string | null>(null);

  const fetchTokens = async () => {
    try {
      const data = await api.getTokens();
      setTokens(data);
      setError(null);
    } catch (err) {
      setError(err instanceof Error ? err.message : '获取 Token 列表失败');
    } finally {
      setLoading(false);
    }
  };

  useEffect(() => {
    fetchTokens();
  }, []);

  const handleDelete = async (id: number, name: string) => {
    if (!confirm(`确定要删除 Token "${name}" 吗？删除后无法恢复！`)) return;

    try {
      await api.deleteToken(id);
      await fetchTokens();
    } catch (err) {
      alert('删除失败: ' + (err instanceof Error ? err.message : '未知错误'));
    }
  };

  const handleCreate = () => {
    setCreatedToken(null);
    setShowCreateModal(true);
  };

  const handleCopyToken = (token: string) => {
    navigator.clipboard.writeText(token);
    alert('Token 已复制到剪贴板！');
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
            <h1 className="text-3xl font-bold text-gray-900">Token 管理</h1>
            <p className="mt-2 text-sm text-gray-600">
              创建和管理 API 访问令牌
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
              + 创建 Token
            </button>
          </div>
        </div>

        {/* Token 列表 */}
        <div className="bg-white shadow rounded-lg overflow-hidden">
          <table className="min-w-full divide-y divide-gray-200">
            <thead className="bg-gray-50">
              <tr>
                <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">
                  名称
                </th>
                <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">
                  Token (部分)
                </th>
                <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">
                  状态
                </th>
                <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">
                  过期时间
                </th>
                <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">
                  创建时间
                </th>
                <th className="px-6 py-3 text-right text-xs font-medium text-gray-500 uppercase tracking-wider">
                  操作
                </th>
              </tr>
            </thead>
            <tbody className="bg-white divide-y divide-gray-200">
              {tokens.length > 0 ? (
                tokens.map((token) => {
                  const isExpired = token.expires_at && new Date(token.expires_at) < new Date();
                  const isDisabled = !token.enabled;

                  return (
                    <tr key={token.id} className="hover:bg-gray-50">
                      <td className="px-6 py-4 whitespace-nowrap">
                        <div className="text-sm font-medium text-gray-900">
                          {token.name}
                        </div>
                      </td>
                      <td className="px-6 py-4">
                        <div className="text-sm font-mono text-gray-900">
                          {token.token_display}
                        </div>
                      </td>
                      <td className="px-6 py-4 whitespace-nowrap">
                        <span className={`inline-flex px-2 py-1 text-xs font-semibold rounded-full ${
                          isDisabled
                            ? 'bg-gray-100 text-gray-800'
                            : isExpired
                            ? 'bg-red-100 text-red-800'
                            : 'bg-green-100 text-green-800'
                        }`}>
                          {isDisabled ? '已禁用' : isExpired ? '已过期' : '有效'}
                        </span>
                      </td>
                      <td className="px-6 py-4 whitespace-nowrap text-sm text-gray-900">
                        {token.expires_at ? (
                          <span className={isExpired ? 'text-red-600' : ''}>
                            {new Date(token.expires_at).toLocaleString('zh-CN')}
                          </span>
                        ) : (
                          <span className="text-gray-500">永不过期</span>
                        )}
                      </td>
                      <td className="px-6 py-4 whitespace-nowrap text-sm text-gray-500">
                        {new Date(token.created_at).toLocaleString('zh-CN')}
                      </td>
                      <td className="px-6 py-4 whitespace-nowrap text-right text-sm font-medium">
                        <button
                          onClick={() => handleDelete(token.id, token.name)}
                          className="text-red-600 hover:text-red-900"
                        >
                          删除
                        </button>
                      </td>
                    </tr>
                  );
                })
              ) : (
                <tr>
                  <td colSpan={6} className="px-6 py-8 text-center text-gray-500">
                    暂无 Token，点击右上角创建
                  </td>
                </tr>
              )}
            </tbody>
          </table>
        </div>

        {/* 安全提示 */}
        <div className="mt-8 bg-yellow-50 border border-yellow-200 rounded-lg p-4">
          <div className="flex">
            <div className="flex-shrink-0">
              <svg className="h-5 w-5 text-yellow-400" fill="currentColor" viewBox="0 0 20 20">
                <path fillRule="evenodd" d="M8.257 3.099c.765-1.36 2.722-1.36 3.486 0l5.58 9.92c.75 1.334-.213 2.98-1.742 2.98H4.42c-1.53 0-2.493-1.646-1.743-2.98l5.58-9.92zM11 13a1 1 0 11-2 0 1 1 0 012 0zm-1-8a1 1 0 00-1 1v3a1 1 0 002 0V6a1 1 0 00-1-1z" clipRule="evenodd" />
              </svg>
            </div>
            <div className="ml-3">
              <h3 className="text-sm font-medium text-yellow-800">
                安全提示
              </h3>
              <div className="mt-2 text-sm text-yellow-700">
                <ul className="list-disc list-inside space-y-1">
                  <li>Token 创建后只会显示一次，请妥善保存</li>
                  <li>不要在公共场合或代码仓库中暴露 Token</li>
                  <li>建议定期轮换 Token 以提高安全性</li>
                  <li>删除的 Token 无法恢复，使用该 Token 的请求将失败</li>
                </ul>
              </div>
            </div>
          </div>
        </div>
      </div>

      {/* 创建模态框 */}
      {showCreateModal && (
        <CreateTokenModal
          onClose={() => {
            setShowCreateModal(false);
            setCreatedToken(null);
          }}
          onSuccess={(token) => {
            setCreatedToken(token);
            fetchTokens();
          }}
          createdToken={createdToken}
          onCopyToken={handleCopyToken}
        />
      )}
    </div>
  );
}

// Token 创建模态框组件
function CreateTokenModal({
  onClose,
  onSuccess,
  createdToken,
  onCopyToken,
}: {
  onClose: () => void;
  onSuccess: (token: string) => void;
  createdToken: string | null;
  onCopyToken: (token: string) => void;
}) {
  const [formData, setFormData] = useState({
    name: '',
    expires_at: '',
  });
  const [submitting, setSubmitting] = useState(false);

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    setSubmitting(true);

    try {
      const result = await api.createToken({
        name: formData.name,
        expires_at: formData.expires_at || undefined,
      });
      onSuccess(result.token);
    } catch (err) {
      alert('创建失败: ' + (err instanceof Error ? err.message : '未知错误'));
    } finally {
      setSubmitting(false);
    }
  };

  // 如果已创建 Token，显示成功界面
  if (createdToken) {
    return (
      <div className="fixed inset-0 bg-gray-600 bg-opacity-50 flex items-center justify-center z-50">
        <div className="bg-white rounded-lg shadow-xl max-w-md w-full mx-4">
          <div className="px-6 py-4 border-b border-gray-200">
            <h3 className="text-lg font-medium text-gray-900">
              Token 创建成功
            </h3>
          </div>

          <div className="px-6 py-4">
            <div className="bg-green-50 border border-green-200 rounded-lg p-4 mb-4">
              <p className="text-sm text-green-800 mb-2">
                ✓ Token 已创建！请复制保存，关闭后将无法再次查看完整 Token。
              </p>
            </div>

            <div className="bg-gray-50 rounded-lg p-4">
              <label className="block text-sm font-medium text-gray-700 mb-2">
                完整 Token
              </label>
              <div className="flex items-center space-x-2">
                <input
                  type="text"
                  readOnly
                  value={createdToken}
                  className="flex-1 px-3 py-2 border border-gray-300 rounded-md bg-white font-mono text-sm"
                />
                <button
                  onClick={() => onCopyToken(createdToken)}
                  className="px-4 py-2 text-sm font-medium text-white bg-blue-600 rounded-md hover:bg-blue-700"
                >
                  复制
                </button>
              </div>
            </div>
          </div>

          <div className="px-6 py-4 border-t border-gray-200 flex justify-end">
            <button
              onClick={onClose}
              className="px-4 py-2 text-sm font-medium text-white bg-gray-600 rounded-md hover:bg-gray-700"
            >
              关闭
            </button>
          </div>
        </div>
      </div>
    );
  }

  // 创建表单
  return (
    <div className="fixed inset-0 bg-gray-600 bg-opacity-50 flex items-center justify-center z-50">
      <div className="bg-white rounded-lg shadow-xl max-w-md w-full mx-4">
        <div className="px-6 py-4 border-b border-gray-200">
          <h3 className="text-lg font-medium text-gray-900">
            创建新 Token
          </h3>
        </div>

        <form onSubmit={handleSubmit} className="px-6 py-4 space-y-4">
          <div>
            <label className="block text-sm font-medium text-gray-700 mb-1">
              Token 名称 *
            </label>
            <input
              type="text"
              required
              value={formData.name}
              onChange={(e) => setFormData({ ...formData, name: e.target.value })}
              className="w-full px-3 py-2 border border-gray-300 rounded-md focus:outline-none focus:ring-2 focus:ring-blue-500"
              placeholder="例如: 生产环境 API Key"
            />
          </div>

          <div>
            <label className="block text-sm font-medium text-gray-700 mb-1">
              过期时间 (可选)
            </label>
            <input
              type="datetime-local"
              value={formData.expires_at}
              onChange={(e) => setFormData({ ...formData, expires_at: e.target.value })}
              className="w-full px-3 py-2 border border-gray-300 rounded-md focus:outline-none focus:ring-2 focus:ring-blue-500"
            />
            <p className="mt-1 text-xs text-gray-500">
              留空则永不过期
            </p>
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
              {submitting ? '创建中...' : '创建'}
            </button>
          </div>
        </form>
      </div>
    </div>
  );
}
