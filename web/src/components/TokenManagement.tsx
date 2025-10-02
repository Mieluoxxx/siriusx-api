import { useState, useEffect } from 'react';
import { api, type Token } from '../lib/api';

export default function TokenManagement() {
  const [tokens, setTokens] = useState<Token[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [showCreateModal, setShowCreateModal] = useState(false);
  const [createdToken, setCreatedToken] = useState<string | null>(null);
  const [visibleTokens, setVisibleTokens] = useState<Set<number>>(new Set());
  const [fullTokens, setFullTokens] = useState<Map<number, string>>(new Map());

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

  const handleCopyToken = async (tokenId: number) => {
    // 总是复制完整的 Token
    let fullToken = fullTokens.get(tokenId);

    // 如果还没有获取过完整 Token，先获取
    if (!fullToken) {
      try {
        const result = await api.getToken(tokenId);
        fullToken = result.token;
        const newFullTokens = new Map(fullTokens);
        newFullTokens.set(tokenId, fullToken);
        setFullTokens(newFullTokens);
      } catch (err) {
        alert('获取 Token 失败: ' + (err instanceof Error ? err.message : '未知错误'));
        return;
      }
    }

    navigator.clipboard.writeText(fullToken);
    alert('Token 已复制到剪贴板！');
  };

  const toggleTokenVisibility = async (id: number) => {
    const newVisibleTokens = new Set(visibleTokens);

    if (visibleTokens.has(id)) {
      // 隐藏
      newVisibleTokens.delete(id);
      setVisibleTokens(newVisibleTokens);
    } else {
      // 显示 - 需要先获取完整 Token
      if (!fullTokens.has(id)) {
        try {
          const result = await api.getToken(id);
          const newFullTokens = new Map(fullTokens);
          newFullTokens.set(id, result.token);
          setFullTokens(newFullTokens);
        } catch (err) {
          alert('获取 Token 失败: ' + (err instanceof Error ? err.message : '未知错误'));
          return;
        }
      }
      newVisibleTokens.add(id);
      setVisibleTokens(newVisibleTokens);
    }
  };

  const getDisplayToken = (token: Token) => {
    if (visibleTokens.has(token.id) && fullTokens.has(token.id)) {
      return fullTokens.get(token.id)!;
    }
    return token.token_display;
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
                        <div className="flex items-center space-x-2">
                          <span className="text-sm font-mono text-gray-900">
                            {getDisplayToken(token)}
                          </span>
                          <button
                            onClick={() => toggleTokenVisibility(token.id)}
                            className="text-gray-500 hover:text-gray-700 focus:outline-none"
                            title={visibleTokens.has(token.id) ? '隐藏 Token' : '显示 Token'}
                          >
                            {visibleTokens.has(token.id) ? (
                              <svg className="w-5 h-5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                                <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M13.875 18.825A10.05 10.05 0 0112 19c-4.478 0-8.268-2.943-9.543-7a9.97 9.97 0 011.563-3.029m5.858.908a3 3 0 114.243 4.243M9.878 9.878l4.242 4.242M9.88 9.88l-3.29-3.29m7.532 7.532l3.29 3.29M3 3l3.59 3.59m0 0A9.953 9.953 0 0112 5c4.478 0 8.268 2.943 9.543 7a10.025 10.025 0 01-4.132 5.411m0 0L21 21" />
                              </svg>
                            ) : (
                              <svg className="w-5 h-5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                                <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M15 12a3 3 0 11-6 0 3 3 0 016 0z" />
                                <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M2.458 12C3.732 7.943 7.523 5 12 5c4.478 0 8.268 2.943 9.542 7-1.274 4.057-5.064 7-9.542 7-4.477 0-8.268-2.943-9.542-7z" />
                              </svg>
                            )}
                          </button>
                          <button
                            onClick={() => handleCopyToken(token.id)}
                            className="text-gray-500 hover:text-gray-700 focus:outline-none"
                            title="复制完整 Token"
                          >
                            <svg className="w-5 h-5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                              <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M8 16H6a2 2 0 01-2-2V6a2 2 0 012-2h8a2 2 0 012 2v2m-6 12h8a2 2 0 002-2v-8a2 2 0 00-2-2h-8a2 2 0 00-2 2v8a2 2 0 002 2z" />
                            </svg>
                          </button>
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
                  <li>不要在公共场合、聊天工具或代码仓库中暴露 Token</li>
                  <li>建议定期轮换 Token 以提高安全性</li>
                  <li>删除的 Token 无法恢复，使用该 Token 的请求将立即失败</li>
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
    custom_token: '',
  });
  const [submitting, setSubmitting] = useState(false);
  const [showAdvanced, setShowAdvanced] = useState(false);

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    setSubmitting(true);

    try {
      const result = await api.createToken({
        name: formData.name,
        expires_at: formData.expires_at || undefined,
        custom_token: formData.custom_token || undefined,
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
              Token 详情
            </h3>
          </div>

          <div className="px-6 py-4">
            <div className="bg-blue-50 border border-blue-200 rounded-lg p-4 mb-4">
              <p className="text-sm text-blue-800 mb-2">
                ✅ Token 创建成功！请立即复制保存到安全的地方。
              </p>
              <p className="text-sm text-blue-700 mt-1">
                💡 提示：您可以随时在 Token 列表中点击"眼睛"图标查看完整 Token。
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

          {/* 高级模式切换 */}
          <div className="pt-2 border-t border-gray-200">
            <button
              type="button"
              onClick={() => setShowAdvanced(!showAdvanced)}
              className="text-sm text-blue-600 hover:text-blue-800 flex items-center"
            >
              {showAdvanced ? '▼' : '▶'} 高级选项
            </button>
          </div>

          {/* 自定义 Token 字段（高级模式） */}
          {showAdvanced && (
            <div className="bg-yellow-50 border border-yellow-200 rounded-lg p-4">
              <label className="block text-sm font-medium text-gray-700 mb-1">
                自定义 Token 值 (可选)
              </label>
              <input
                type="text"
                value={formData.custom_token}
                onChange={(e) => setFormData({ ...formData, custom_token: e.target.value })}
                className="w-full px-3 py-2 border border-gray-300 rounded-md focus:outline-none focus:ring-2 focus:ring-blue-500 font-mono text-sm"
                placeholder="sk-your-custom-token-here"
              />
              <p className="mt-2 text-xs text-yellow-800">
                ⚠️ <strong>高级功能：</strong>自定义 Token 必须以 "sk-" 开头，长度至少 8 个字符。留空则自动生成随机 Token。
              </p>
              <p className="mt-1 text-xs text-yellow-800">
                示例: sk-123456、sk-my-custom-key-2024
              </p>
            </div>
          )}

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
