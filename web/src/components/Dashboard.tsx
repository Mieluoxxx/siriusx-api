import { useState, useEffect } from 'react';
import { api, type SystemStats } from '../lib/api';

export default function Dashboard() {
  const [stats, setStats] = useState<SystemStats | null>(null);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);

  const fetchStats = async () => {
    try {
      const data = await api.getStats();
      setStats(data);
      setError(null);
    } catch (err) {
      setError(err instanceof Error ? err.message : '获取数据失败');
    } finally {
      setLoading(false);
    }
  };

  useEffect(() => {
    fetchStats();
    // 每 5 秒自动刷新
    const interval = setInterval(fetchStats, 5000);
    return () => clearInterval(interval);
  }, []);

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
    <div className="min-h-screen bg-gradient-to-br from-gray-50 to-gray-100 py-8 px-4 sm:px-6 lg:px-8">
      <div className="max-w-7xl mx-auto">
        <div className="mb-8">
          <h1 className="text-3xl font-bold text-gray-900">
            Siriusx-API 管理界面
          </h1>
          <p className="mt-2 text-sm text-gray-600">
            实时监控系统状态 • 自动刷新间隔 5 秒
          </p>
        </div>

        {/* 供应商统计卡片 */}
        <div className="grid grid-cols-1 md:grid-cols-3 gap-6 mb-8">
          <div className="bg-white rounded-lg shadow hover:shadow-md transition-shadow p-6 border-l-4 border-gray-400">
            <div className="text-sm font-medium text-gray-500 mb-2">
              总供应商数
            </div>
            <div className="text-3xl font-bold text-gray-900">
              {stats?.providers.total || 0}
            </div>
          </div>

          <div className="bg-white rounded-lg shadow hover:shadow-md transition-shadow p-6 border-l-4 border-green-500">
            <div className="text-sm font-medium text-gray-500 mb-2">
              健康供应商
            </div>
            <div className="text-3xl font-bold text-green-600">
              {stats?.providers.healthy || 0}
            </div>
          </div>

          <div className="bg-white rounded-lg shadow hover:shadow-md transition-shadow p-6 border-l-4 border-red-500">
            <div className="text-sm font-medium text-gray-500 mb-2">
              异常供应商
            </div>
            <div className="text-3xl font-bold text-red-600">
              {stats?.providers.unhealthy || 0}
            </div>
          </div>
        </div>

        {/* 请求统计卡片 */}
        <div className="grid grid-cols-1 md:grid-cols-2 gap-6 mb-8">
          <div className="bg-white rounded-lg shadow hover:shadow-md transition-shadow p-6 border-l-4 border-indigo-500">
            <div className="text-sm font-medium text-gray-500 mb-2">
              总请求数
            </div>
            <div className="text-3xl font-bold text-gray-900">
              {stats?.requests.total.toLocaleString() || 0}
            </div>
          </div>

          <div className="bg-white rounded-lg shadow hover:shadow-md transition-shadow p-6 border-l-4 border-blue-500">
            <div className="text-sm font-medium text-gray-500 mb-2">
              当前 QPS
            </div>
            <div className="text-3xl font-bold text-blue-600">
              {stats?.requests.current_qps.toFixed(2) || '0.00'}
            </div>
          </div>
        </div>

        {/* 最近事件 */}
        <div className="bg-white rounded-lg shadow">
          <div className="px-6 py-4 border-b border-gray-200">
            <h2 className="text-lg font-semibold text-gray-900">最近事件</h2>
          </div>
          <div className="divide-y divide-gray-200">
            {stats?.recent_events && stats.recent_events.length > 0 ? (
              stats.recent_events.map((event, index) => (
                <div key={index} className="px-6 py-4 hover:bg-gray-50">
                  <div className="flex items-start justify-between">
                    <div className="flex-1">
                      <div className="flex items-center space-x-3">
                        <span className={`inline-flex items-center px-2.5 py-0.5 rounded-full text-xs font-medium ${
                          event.type === 'failover' ? 'bg-red-100 text-red-800' :
                          event.type === 'health_check' ? 'bg-green-100 text-green-800' :
                          'bg-blue-100 text-blue-800'
                        }`}>
                          {event.type}
                        </span>
                        <span className="text-sm text-gray-900">
                          {event.message}
                        </span>
                      </div>
                      <div className="mt-1 text-xs text-gray-500">
                        {new Date(event.timestamp).toLocaleString('zh-CN')}
                      </div>
                    </div>
                  </div>
                </div>
              ))
            ) : (
              <div className="px-6 py-8 text-center text-gray-500">
                暂无事件记录
              </div>
            )}
          </div>
        </div>

        {/* 快速操作链接 */}
        <div className="mt-8 grid grid-cols-1 md:grid-cols-3 gap-4">
          <a
            href="/providers"
            className="block bg-white rounded-lg shadow p-6 hover:shadow-xl hover:scale-105 transition-all duration-200 border-t-4 border-blue-500"
          >
            <h3 className="text-lg font-semibold text-gray-900 mb-2">
              📡 供应商管理
            </h3>
            <p className="text-sm text-gray-600">
              管理 API 供应商配置和健康状态
            </p>
          </a>

          <a
            href="/models"
            className="block bg-white rounded-lg shadow p-6 hover:shadow-xl hover:scale-105 transition-all duration-200 border-t-4 border-purple-500"
          >
            <h3 className="text-lg font-semibold text-gray-900 mb-2">
              🤖 模型管理
            </h3>
            <p className="text-sm text-gray-600">
              配置统一模型和映射关系
            </p>
          </a>

          <a
            href="/tokens"
            className="block bg-white rounded-lg shadow p-6 hover:shadow-xl hover:scale-105 transition-all duration-200 border-t-4 border-green-500"
          >
            <h3 className="text-lg font-semibold text-gray-900 mb-2">
              🔑 Token 管理
            </h3>
            <p className="text-sm text-gray-600">
              创建和管理 API 访问令牌
            </p>
          </a>
        </div>
      </div>
    </div>
  );
}
