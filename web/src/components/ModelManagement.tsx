import { useState, useEffect } from 'react';
import { api, type UnifiedModel } from '../lib/api';

export default function ModelManagement() {
  const [models, setModels] = useState<UnifiedModel[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);

  const fetchModels = async () => {
    try {
      const data = await api.getModels();
      setModels(data);
      setError(null);
    } catch (err) {
      setError(err instanceof Error ? err.message : '获取模型列表失败');
    } finally {
      setLoading(false);
    }
  };

  useEffect(() => {
    fetchModels();
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
    <div className="min-h-screen bg-gray-50 py-8 px-4 sm:px-6 lg:px-8">
      <div className="max-w-7xl mx-auto">
        {/* 头部 */}
        <div className="flex items-center justify-between mb-8">
          <div>
            <h1 className="text-3xl font-bold text-gray-900">模型管理</h1>
            <p className="mt-2 text-sm text-gray-600">
              查看统一模型配置和映射关系
            </p>
          </div>
          <a
            href="/"
            className="px-4 py-2 text-sm font-medium text-gray-700 bg-white border border-gray-300 rounded-md hover:bg-gray-50"
          >
            返回首页
          </a>
        </div>

        {/* 模型列表 */}
        <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-6">
          {models.length > 0 ? (
            models.map((model) => (
              <div
                key={model.id}
                className="bg-white rounded-lg shadow hover:shadow-lg transition-shadow p-6"
              >
                <div className="flex items-start justify-between mb-4">
                  <div className="flex-1">
                    <h3 className="text-lg font-semibold text-gray-900 mb-1">
                      {model.display_name}
                    </h3>
                    <p className="text-sm text-gray-500 font-mono">
                      {model.name}
                    </p>
                  </div>
                  <span className="inline-flex items-center px-2.5 py-0.5 rounded-full text-xs font-medium bg-blue-100 text-blue-800">
                    ID: {model.id}
                  </span>
                </div>

                {model.description && (
                  <p className="text-sm text-gray-600 mb-4">
                    {model.description}
                  </p>
                )}

                <div className="pt-4 border-t border-gray-200">
                  <div className="flex items-center justify-between text-xs text-gray-500">
                    <span>创建时间</span>
                    <span>{new Date(model.created_at).toLocaleDateString('zh-CN')}</span>
                  </div>
                  {model.updated_at !== model.created_at && (
                    <div className="flex items-center justify-between text-xs text-gray-500 mt-1">
                      <span>更新时间</span>
                      <span>{new Date(model.updated_at).toLocaleDateString('zh-CN')}</span>
                    </div>
                  )}
                </div>
              </div>
            ))
          ) : (
            <div className="col-span-full bg-white rounded-lg shadow p-12 text-center">
              <p className="text-gray-500">暂无模型配置</p>
            </div>
          )}
        </div>

        {/* 提示信息 */}
        <div className="mt-8 bg-blue-50 border border-blue-200 rounded-lg p-4">
          <div className="flex">
            <div className="flex-shrink-0">
              <svg className="h-5 w-5 text-blue-400" fill="currentColor" viewBox="0 0 20 20">
                <path fillRule="evenodd" d="M18 10a8 8 0 11-16 0 8 8 0 0116 0zm-7-4a1 1 0 11-2 0 1 1 0 012 0zM9 9a1 1 0 000 2v3a1 1 0 001 1h1a1 1 0 100-2v-3a1 1 0 00-1-1H9z" clipRule="evenodd" />
              </svg>
            </div>
            <div className="ml-3">
              <h3 className="text-sm font-medium text-blue-800">
                关于模型管理
              </h3>
              <div className="mt-2 text-sm text-blue-700">
                <p>
                  统一模型用于抽象不同供应商的模型接口。模型映射关系在数据库中配置，
                  当前页面仅展示已配置的统一模型。如需添加或修改模型映射，
                  请使用后端 API 或数据库工具进行操作。
                </p>
              </div>
            </div>
          </div>
        </div>
      </div>
    </div>
  );
}
