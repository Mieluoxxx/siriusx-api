import { useState, useEffect } from 'react';
import { api, type UnifiedModel, type ModelMapping, type Provider, type AvailableModelsResponse } from '../lib/api';
import Toast from './Toast';

interface ToastState {
  show: boolean;
  message: string;
  type: 'success' | 'error' | 'info';
}

export default function ModelManagement() {
  const [models, setModels] = useState<UnifiedModel[]>([]);
  const [providers, setProviders] = useState<Provider[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [toast, setToast] = useState<ToastState>({ show: false, message: '', type: 'info' });

  // 模态框状态
  const [showCreateModal, setShowCreateModal] = useState(false);
  const [editingModel, setEditingModel] = useState<UnifiedModel | null>(null);
  const [showAddMappingModal, setShowAddMappingModal] = useState(false);
  const [selectedModel, setSelectedModel] = useState<UnifiedModel | null>(null);

  // 映射管理状态
  const [expandedModelId, setExpandedModelId] = useState<number | null>(null);
  const [modelMappings, setModelMappings] = useState<Record<number, ModelMapping[]>>({});

  // AddMapping 相关状态
  const [selectedProviderId, setSelectedProviderId] = useState<number | null>(null);
  const [availableModels, setAvailableModels] = useState<AvailableModelsResponse | null>(null);
  const [loadingModels, setLoadingModels] = useState(false);
  const [mappingForm, setMappingForm] = useState({
    target_model: '',
    weight: 50,
    priority: 1,
  });

  const showToast = (message: string, type: 'success' | 'error' | 'info' = 'info') => {
    setToast({ show: true, message, type });
  };

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

  const fetchProviders = async () => {
    try {
      const response = await api.getProviders();
      setProviders(response.data);
    } catch (err) {
      console.error('Failed to fetch providers:', err);
    }
  };

  const fetchModelMappings = async (modelId: number) => {
    try {
      const mappings = await api.getModelMappings(modelId);
      setModelMappings(prev => ({ ...prev, [modelId]: mappings }));
    } catch (err) {
      showToast('获取映射失败: ' + (err instanceof Error ? err.message : '未知错误'), 'error');
    }
  };

  useEffect(() => {
    fetchModels();
    fetchProviders();
  }, []);

  const handleToggleExpand = (modelId: number) => {
    if (expandedModelId === modelId) {
      setExpandedModelId(null);
    } else {
      setExpandedModelId(modelId);
      if (!modelMappings[modelId]) {
        fetchModelMappings(modelId);
      }
    }
  };

  const handleCreateModel = () => {
    setEditingModel(null);
    setShowCreateModal(true);
  };

  const handleEditModel = (model: UnifiedModel) => {
    setEditingModel(model);
    setShowCreateModal(true);
  };

  const handleDeleteModel = async (id: number, name: string) => {
    if (!confirm(`确定要删除统一模型 "${name}" 吗？这将同时删除所有相关映射。`)) return;

    try {
      await api.deleteModel(id);
      await fetchModels();
      showToast('删除成功', 'success');
    } catch (err) {
      showToast('删除失败: ' + (err instanceof Error ? err.message : '未知错误'), 'error');
    }
  };

  const handleAddMapping = (model: UnifiedModel) => {
    setSelectedModel(model);
    setSelectedProviderId(null);
    setAvailableModels(null);
    setMappingForm({ target_model: '', weight: 50, priority: 1 });
    setShowAddMappingModal(true);
  };

  const handleSelectProvider = async (providerId: number) => {
    setSelectedProviderId(providerId);
    setLoadingModels(true);
    try {
      const models = await api.getProviderModels(providerId);
      setAvailableModels(models);
    } catch (err) {
      showToast('获取供应商模型失败: ' + (err instanceof Error ? err.message : '未知错误'), 'error');
    } finally {
      setLoadingModels(false);
    }
  };

  const handleSubmitMapping = async (e: React.FormEvent) => {
    e.preventDefault();
    if (!selectedModel || !selectedProviderId || !mappingForm.target_model) return;

    try {
      await api.createMapping(selectedModel.id, {
        provider_id: selectedProviderId,
        target_model: mappingForm.target_model,
        weight: mappingForm.weight,
        priority: mappingForm.priority,
        enabled: true,
      });
      showToast('映射添加成功', 'success');
      setShowAddMappingModal(false);
      await fetchModelMappings(selectedModel.id);
    } catch (err) {
      showToast('添加映射失败: ' + (err instanceof Error ? err.message : '未知错误'), 'error');
    }
  };

  const handleDeleteMapping = async (mappingId: number, modelId: number) => {
    if (!confirm('确定要删除此映射吗？')) return;

    try {
      await api.deleteMapping(mappingId);
      showToast('映射删除成功', 'success');
      await fetchModelMappings(modelId);
    } catch (err) {
      showToast('删除映射失败: ' + (err instanceof Error ? err.message : '未知错误'), 'error');
    }
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
      {toast.show && (
        <Toast
          message={toast.message}
          type={toast.type}
          onClose={() => setToast({ ...toast, show: false })}
        />
      )}

      <div className="max-w-7xl mx-auto">
        {/* 头部 */}
        <div className="flex items-center justify-between mb-8">
          <div>
            <h1 className="text-3xl font-bold text-gray-900">模型管理</h1>
            <p className="mt-2 text-sm text-gray-600">
              管理统一模型别名和供应商映射关系
            </p>
          </div>
          <div className="flex gap-3">
            <button
              onClick={handleCreateModel}
              className="px-4 py-2 text-sm font-medium text-white bg-blue-600 border border-transparent rounded-md hover:bg-blue-700"
            >
              创建统一模型
            </button>
            <a
              href="/"
              className="px-4 py-2 text-sm font-medium text-gray-700 bg-white border border-gray-300 rounded-md hover:bg-gray-50"
            >
              返回首页
            </a>
          </div>
        </div>

        {/* 模型列表 */}
        <div className="grid grid-cols-1 gap-6">
          {models.length > 0 ? (
            models.map((model) => (
              <div
                key={model.id}
                className="bg-white rounded-lg shadow hover:shadow-lg transition-shadow"
              >
                {/* 模型卡片头部 */}
                <div className="p-6">
                  <div className="flex items-start justify-between mb-4">
                    <div className="flex-1">
                      <div className="flex items-center gap-3 mb-2">
                        <h3 className="text-xl font-semibold text-gray-900">
                          {model.display_name}
                        </h3>
                        <span className="inline-flex items-center px-2.5 py-0.5 rounded-full text-xs font-medium bg-blue-100 text-blue-800">
                          {model.name}
                        </span>
                        {modelMappings[model.id] && (
                          <span className="inline-flex items-center px-2.5 py-0.5 rounded-full text-xs font-medium bg-green-100 text-green-800">
                            {modelMappings[model.id].length} 个映射
                          </span>
                        )}
                      </div>
                      {model.description && (
                        <p className="text-sm text-gray-600">{model.description}</p>
                      )}
                    </div>
                    <div className="flex gap-2">
                      <button
                        onClick={() => handleAddMapping(model)}
                        className="px-3 py-1.5 text-sm font-medium text-blue-600 bg-blue-50 rounded-md hover:bg-blue-100"
                      >
                        添加映射
                      </button>
                      <button
                        onClick={() => handleEditModel(model)}
                        className="px-3 py-1.5 text-sm font-medium text-gray-700 bg-gray-100 rounded-md hover:bg-gray-200"
                      >
                        编辑
                      </button>
                      <button
                        onClick={() => handleDeleteModel(model.id, model.name)}
                        className="px-3 py-1.5 text-sm font-medium text-red-600 bg-red-50 rounded-md hover:bg-red-100"
                      >
                        删除
                      </button>
                      <button
                        onClick={() => handleToggleExpand(model.id)}
                        className="px-3 py-1.5 text-sm font-medium text-gray-700 bg-gray-100 rounded-md hover:bg-gray-200"
                      >
                        {expandedModelId === model.id ? '收起' : '展开映射'}
                      </button>
                    </div>
                  </div>

                  <div className="flex items-center gap-4 text-xs text-gray-500">
                    <span>ID: {model.id}</span>
                    <span>创建于 {new Date(model.created_at).toLocaleString('zh-CN')}</span>
                  </div>
                </div>

                {/* 映射列表（展开式） */}
                {expandedModelId === model.id && (
                  <div className="border-t border-gray-200 bg-gray-50 p-6">
                    <h4 className="text-sm font-medium text-gray-900 mb-4">供应商映射</h4>
                    {modelMappings[model.id] && modelMappings[model.id].length > 0 ? (
                      <div className="space-y-3">
                        {modelMappings[model.id].map((mapping) => (
                          <div
                            key={mapping.id}
                            className="bg-white p-4 rounded-lg border border-gray-200 flex items-center justify-between"
                          >
                            <div className="flex-1">
                              <div className="flex items-center gap-3 mb-2">
                                <span className="text-sm font-mono text-gray-900">
                                  {providers.find(p => p.id === mapping.provider_id)?.name || `Provider #${mapping.provider_id}`}/{mapping.target_model}
                                </span>
                                {mapping.enabled ? (
                                  <span className="inline-flex items-center px-2 py-0.5 rounded text-xs font-medium bg-green-100 text-green-800">
                                    启用
                                  </span>
                                ) : (
                                  <span className="inline-flex items-center px-2 py-0.5 rounded text-xs font-medium bg-gray-100 text-gray-800">
                                    禁用
                                  </span>
                                )}
                              </div>
                              <div className="flex gap-4 text-xs text-gray-500">
                                <span>权重: {mapping.weight}</span>
                                <span>优先级: {mapping.priority}</span>
                              </div>
                            </div>
                            <div className="flex gap-2">
                              <button
                                onClick={() => handleDeleteMapping(mapping.id, model.id)}
                                className="px-3 py-1.5 text-sm font-medium text-red-600 bg-red-50 rounded-md hover:bg-red-100"
                              >
                                删除
                              </button>
                            </div>
                          </div>
                        ))}
                      </div>
                    ) : (
                      <div className="text-center py-8 text-gray-500">
                        暂无映射，点击上方"添加映射"按钮添加
                      </div>
                    )}
                  </div>
                )}
              </div>
            ))
          ) : (
            <div className="bg-white rounded-lg shadow p-12 text-center">
              <p className="text-gray-500 mb-4">暂无统一模型</p>
              <button
                onClick={handleCreateModel}
                className="px-4 py-2 text-sm font-medium text-white bg-blue-600 border border-transparent rounded-md hover:bg-blue-700"
              >
                创建第一个模型
              </button>
            </div>
          )}
        </div>

        {/* 创建/编辑模型模态框 */}
        {showCreateModal && (
          <CreateModelModal
            model={editingModel}
            onClose={() => setShowCreateModal(false)}
            onSuccess={() => {
              setShowCreateModal(false);
              fetchModels();
              showToast(editingModel ? '模型更新成功' : '模型创建成功', 'success');
            }}
            onError={(msg) => showToast(msg, 'error')}
          />
        )}

        {/* 添加映射模态框 */}
        {showAddMappingModal && selectedModel && (
          <AddMappingModal
            model={selectedModel}
            providers={providers}
            selectedProviderId={selectedProviderId}
            availableModels={availableModels}
            loadingModels={loadingModels}
            mappingForm={mappingForm}
            onSelectProvider={handleSelectProvider}
            onChangeMappingForm={setMappingForm}
            onSubmit={handleSubmitMapping}
            onClose={() => setShowAddMappingModal(false)}
          />
        )}
      </div>
    </div>
  );
}

// 创建/编辑模型模态框组件
function CreateModelModal({
  model,
  onClose,
  onSuccess,
  onError,
}: {
  model: UnifiedModel | null;
  onClose: () => void;
  onSuccess: () => void;
  onError: (msg: string) => void;
}) {
  const [formData, setFormData] = useState({
    name: model?.name || '',
    display_name: model?.display_name || '',
    description: model?.description || '',
  });

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    try {
      if (model) {
        await api.updateModel(model.id, formData);
      } else {
        await api.createModel(formData);
      }
      onSuccess();
    } catch (err) {
      onError(err instanceof Error ? err.message : '操作失败');
    }
  };

  return (
    <div className="fixed inset-0 bg-black bg-opacity-50 flex items-center justify-center z-50">
      <div className="bg-white rounded-lg p-6 max-w-md w-full mx-4">
        <h2 className="text-xl font-bold text-gray-900 mb-4">
          {model ? '编辑统一模型' : '创建统一模型'}
        </h2>
        <form onSubmit={handleSubmit} className="space-y-4">
          <div>
            <label className="block text-sm font-medium text-gray-700 mb-1">
              模型名称 (name) *
            </label>
            <input
              type="text"
              required
              value={formData.name}
              onChange={(e) => setFormData({ ...formData, name: e.target.value })}
              className="w-full px-3 py-2 border border-gray-300 rounded-md focus:outline-none focus:ring-2 focus:ring-blue-500"
              placeholder="gpt-4"
            />
            <p className="mt-1 text-xs text-gray-500">
              API 调用时使用的标识符
            </p>
          </div>
          <div>
            <label className="block text-sm font-medium text-gray-700 mb-1">
              显示名称 *
            </label>
            <input
              type="text"
              required
              value={formData.display_name}
              onChange={(e) => setFormData({ ...formData, display_name: e.target.value })}
              className="w-full px-3 py-2 border border-gray-300 rounded-md focus:outline-none focus:ring-2 focus:ring-blue-500"
              placeholder="GPT-4 通用模型"
            />
          </div>
          <div>
            <label className="block text-sm font-medium text-gray-700 mb-1">
              描述
            </label>
            <textarea
              value={formData.description}
              onChange={(e) => setFormData({ ...formData, description: e.target.value })}
              className="w-full px-3 py-2 border border-gray-300 rounded-md focus:outline-none focus:ring-2 focus:ring-blue-500"
              rows={3}
              placeholder="模型用途和特点说明"
            />
          </div>
          <div className="flex justify-end gap-3 mt-6">
            <button
              type="button"
              onClick={onClose}
              className="px-4 py-2 text-sm font-medium text-gray-700 bg-white border border-gray-300 rounded-md hover:bg-gray-50"
            >
              取消
            </button>
            <button
              type="submit"
              className="px-4 py-2 text-sm font-medium text-white bg-blue-600 border border-transparent rounded-md hover:bg-blue-700"
            >
              {model ? '更新' : '创建'}
            </button>
          </div>
        </form>
      </div>
    </div>
  );
}

// 添加映射模态框组件
function AddMappingModal({
  model,
  providers,
  selectedProviderId,
  availableModels,
  loadingModels,
  mappingForm,
  onSelectProvider,
  onChangeMappingForm,
  onSubmit,
  onClose,
}: {
  model: UnifiedModel;
  providers: Provider[];
  selectedProviderId: number | null;
  availableModels: AvailableModelsResponse | null;
  loadingModels: boolean;
  mappingForm: { target_model: string; weight: number; priority: number };
  onSelectProvider: (id: number) => void;
  onChangeMappingForm: (form: { target_model: string; weight: number; priority: number }) => void;
  onSubmit: (e: React.FormEvent) => void;
  onClose: () => void;
}) {
  return (
    <div className="fixed inset-0 bg-black bg-opacity-50 flex items-center justify-center z-50">
      <div className="bg-white rounded-lg p-6 max-w-2xl w-full mx-4 max-h-[90vh] overflow-y-auto">
        <h2 className="text-xl font-bold text-gray-900 mb-4">
          为 "{model.display_name}" 添加映射
        </h2>

        <form onSubmit={onSubmit} className="space-y-6">
          {/* Step 1: 选择供应商 */}
          <div>
            <label className="block text-sm font-medium text-gray-700 mb-2">
              1. 选择供应商 *
            </label>
            <div className="grid grid-cols-2 gap-3">
              {providers.map((provider) => (
                <button
                  key={provider.id}
                  type="button"
                  onClick={() => onSelectProvider(provider.id)}
                  className={`p-3 border-2 rounded-lg text-left transition-colors ${
                    selectedProviderId === provider.id
                      ? 'border-blue-500 bg-blue-50'
                      : 'border-gray-200 hover:border-gray-300'
                  }`}
                >
                  <div className="font-medium text-gray-900">{provider.name}</div>
                  <div className="text-xs text-gray-500 mt-1">{provider.base_url}</div>
                </button>
              ))}
            </div>
          </div>

          {/* Step 2: 选择模型 */}
          {selectedProviderId && (
            <div>
              <label className="block text-sm font-medium text-gray-700 mb-2">
                2. 选择目标模型 *
              </label>
              {loadingModels ? (
                <div className="text-center py-8 text-gray-500">加载模型列表中...</div>
              ) : availableModels ? (
                <div className="grid grid-cols-1 gap-2 max-h-64 overflow-y-auto border border-gray-200 rounded-lg p-3">
                  {availableModels.models.map((m) => (
                    <button
                      key={m.id}
                      type="button"
                      onClick={() => onChangeMappingForm({ ...mappingForm, target_model: m.id })}
                      className={`p-3 border rounded-lg text-left transition-colors ${
                        mappingForm.target_model === m.id
                          ? 'border-blue-500 bg-blue-50'
                          : 'border-gray-200 hover:border-gray-300'
                      }`}
                    >
                      <div className="font-mono text-sm text-gray-900">
                        {availableModels.provider_name}/{m.id}
                      </div>
                    </button>
                  ))}
                </div>
              ) : null}
            </div>
          )}

          {/* Step 3: 配置参数 */}
          {mappingForm.target_model && (
            <div className="space-y-4">
              <h3 className="text-sm font-medium text-gray-700">3. 配置映射参数</h3>
              <div>
                <label className="block text-sm font-medium text-gray-700 mb-1">
                  权重 (1-100)
                </label>
                <input
                  type="number"
                  min="1"
                  max="100"
                  value={mappingForm.weight}
                  onChange={(e) => onChangeMappingForm({ ...mappingForm, weight: parseInt(e.target.value) })}
                  className="w-full px-3 py-2 border border-gray-300 rounded-md focus:outline-none focus:ring-2 focus:ring-blue-500"
                />
                <p className="mt-1 text-xs text-gray-500">负载均衡时的权重值</p>
              </div>
              <div>
                <label className="block text-sm font-medium text-gray-700 mb-1">
                  优先级 (≥1)
                </label>
                <input
                  type="number"
                  min="1"
                  value={mappingForm.priority}
                  onChange={(e) => onChangeMappingForm({ ...mappingForm, priority: parseInt(e.target.value) })}
                  className="w-full px-3 py-2 border border-gray-300 rounded-md focus:outline-none focus:ring-2 focus:ring-blue-500"
                />
                <p className="mt-1 text-xs text-gray-500">数字越小优先级越高</p>
              </div>
            </div>
          )}

          <div className="flex justify-end gap-3 pt-4 border-t">
            <button
              type="button"
              onClick={onClose}
              className="px-4 py-2 text-sm font-medium text-gray-700 bg-white border border-gray-300 rounded-md hover:bg-gray-50"
            >
              取消
            </button>
            <button
              type="submit"
              disabled={!mappingForm.target_model}
              className="px-4 py-2 text-sm font-medium text-white bg-blue-600 border border-transparent rounded-md hover:bg-blue-700 disabled:opacity-50 disabled:cursor-not-allowed"
            >
              添加映射
            </button>
          </div>
        </form>
      </div>
    </div>
  );
}
