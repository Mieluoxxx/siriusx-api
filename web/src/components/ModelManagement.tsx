import { useState, useEffect, useMemo, Fragment } from 'react';
import { api, type UnifiedModel, type ModelMapping, type Provider, type AvailableModelsResponse } from '../lib/api';
import Toast from './Toast';

type ViewMode = 'card' | 'list' | 'compact';

interface ToastState {
  show: boolean;
  message: string;
  type: 'success' | 'error' | 'info';
}

interface ConfirmDialogState {
  show: boolean;
  title: string;
  message: string;
  onConfirm: () => void;
}

const CLAUDE_CODE_MODEL_NAMES = [
  'claude-3-5-haiku-20241022',
  'claude-sonnet-4-5-20250929',
  'claude-opus-4-1-20250805',
];

export default function ModelManagement() {
  const [models, setModels] = useState<UnifiedModel[]>([]);
  const [providers, setProviders] = useState<Provider[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [toast, setToast] = useState<ToastState>({ show: false, message: '', type: 'info' });
  const [deletingId, setDeletingId] = useState<number | null>(null);
  const [confirmDialog, setConfirmDialog] = useState<ConfirmDialogState>({
    show: false,
    title: '',
    message: '',
    onConfirm: () => {},
  });

  // 模态框状态
  const [showCreateModal, setShowCreateModal] = useState(false);
  const [editingModel, setEditingModel] = useState<UnifiedModel | null>(null);
  const [showAddMappingModal, setShowAddMappingModal] = useState(false);
  const [selectedModel, setSelectedModel] = useState<UnifiedModel | null>(null);

  // 映射管理状态
  const [expandedModelId, setExpandedModelId] = useState<number | null>(null);
  const [modelMappings, setModelMappings] = useState<Record<number, ModelMapping[]>>({});
  const [activeTab, setActiveTab] = useState<'claudecode' | 'all'>('claudecode');
  const [viewMode, setViewMode] = useState<ViewMode>('card');
  const [deletingMappingId, setDeletingMappingId] = useState<number | null>(null);

  // AddMapping 相关状态
  const [selectedProviderId, setSelectedProviderId] = useState<number | null>(null);
  const [availableModels, setAvailableModels] = useState<AvailableModelsResponse | null>(null);
  const [loadingModels, setLoadingModels] = useState(false);
  const [mappingForm, setMappingForm] = useState({
    target_model: '',
  });

  // 编辑映射状态
  const [editingMappingId, setEditingMappingId] = useState<number | null>(null);
  const [editingMappingForm, setEditingMappingForm] = useState<{ weight: number; priority: number }>({ weight: 50, priority: 1 });

  const claudeCodeModelSet = useMemo(() => new Set(CLAUDE_CODE_MODEL_NAMES), []);
  const VIEW_MODE_ICONS: Record<ViewMode, JSX.Element> = useMemo(
    () => ({
      card: (
        <svg
          className="h-4 w-4"
          viewBox="0 0 20 20"
          fill="none"
          stroke="currentColor"
          strokeWidth="1.5"
          aria-hidden="true"
        >
          <rect x="3" y="3" width="5" height="5" rx="1" />
          <rect x="12" y="3" width="5" height="5" rx="1" />
          <rect x="3" y="12" width="5" height="5" rx="1" />
          <rect x="12" y="12" width="5" height="5" rx="1" />
        </svg>
      ),
      list: (
        <svg
          className="h-4 w-4"
          viewBox="0 0 20 20"
          fill="none"
          stroke="currentColor"
          strokeWidth="1.5"
          aria-hidden="true"
        >
          <circle cx="4" cy="5" r="1.5" />
          <circle cx="4" cy="10" r="1.5" />
          <circle cx="4" cy="15" r="1.5" />
          <path d="M8 5h9" />
          <path d="M8 10h9" />
          <path d="M8 15h9" />
        </svg>
      ),
      compact: (
        <svg
          className="h-4 w-4"
          viewBox="0 0 20 20"
          fill="none"
          stroke="currentColor"
          strokeWidth="1.5"
          aria-hidden="true"
        >
          <rect x="3" y="4" width="14" height="4" rx="1" />
          <rect x="3" y="12" width="9" height="4" rx="1" />
          <path d="M15 12h2" />
          <path d="M15 16h2" />
        </svg>
      ),
    }),
    []
  );

  const VIEW_MODE_OPTIONS: { value: ViewMode; label: string }[] = useMemo(
    () => [
      { value: 'card', label: '卡片模式' },
      { value: 'list', label: '列表模式' },
      { value: 'compact', label: '紧凑模式' },
    ],
    []
  );

  const filteredModels = useMemo(() => (
    activeTab === 'claudecode'
      ? models.filter(model => claudeCodeModelSet.has(model.name))
      : models
  ), [activeTab, models, claudeCodeModelSet]);

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

  useEffect(() => {
    if (activeTab === 'claudecode' && expandedModelId !== null) {
      const targetModel = models.find(item => item.id === expandedModelId);
      if (targetModel && !claudeCodeModelSet.has(targetModel.name)) {
        setExpandedModelId(null);
      }
    }
  }, [activeTab, expandedModelId, models, claudeCodeModelSet]);

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
    // 显示自定义确认对话框
    setConfirmDialog({
      show: true,
      title: '确认删除',
      message: `确定要删除统一模型 "${name}" 吗？这将同时删除所有相关映射。`,
      onConfirm: () => confirmDeleteModel(id, name),
    });
  };

  const confirmDeleteModel = async (id: number, name: string) => {
    // 关闭确认对话框
    setConfirmDialog({ show: false, title: '', message: '', onConfirm: () => {} });

    // 设置删除状态
    setDeletingId(id);
    showToast(`正在删除模型 "${name}"...`, 'info');

    try {
      await api.deleteModel(id);
      await fetchModels();
      showToast(`模型 "${name}" 删除成功`, 'success');
    } catch (err) {
      showToast('删除失败: ' + (err instanceof Error ? err.message : '未知错误'), 'error');
    } finally {
      setDeletingId(null);
    }
  };

  const handleAddMapping = (model: UnifiedModel) => {
    setSelectedModel(model);
    setSelectedProviderId(null);
    setAvailableModels(null);
    setMappingForm({ target_model: '' });
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
        weight: 50, // 默认权重
        priority: 1, // 默认优先级
        enabled: true,
      });
      showToast('映射添加成功', 'success');
      setShowAddMappingModal(false);
      await fetchModelMappings(selectedModel.id);
    } catch (err) {
      showToast('添加映射失败: ' + (err instanceof Error ? err.message : '未知错误'), 'error');
    }
  };

  const handleDeleteMapping = async (mappingId: number, modelId: number, providerName: string, targetModel: string) => {
    // 显示自定义确认对话框
    setConfirmDialog({
      show: true,
      title: '确认删除映射',
      message: `确定要删除映射 "${providerName}/${targetModel}" 吗？`,
      onConfirm: () => confirmDeleteMapping(mappingId, modelId, providerName, targetModel),
    });
  };

  const handleEditMapping = (mapping: ModelMapping) => {
    setEditingMappingId(mapping.id);
    setEditingMappingForm({ weight: mapping.weight, priority: mapping.priority });
  };

  const handleCancelEditMapping = () => {
    setEditingMappingId(null);
    setEditingMappingForm({ weight: 50, priority: 1 });
  };

  const handleSaveMapping = async (mappingId: number, modelId: number) => {
    try {
      await api.updateMapping(mappingId, editingMappingForm);
      showToast('映射更新成功', 'success');
      setEditingMappingId(null);
      await fetchModelMappings(modelId);
    } catch (err) {
      showToast('更新映射失败: ' + (err instanceof Error ? err.message : '未知错误'), 'error');
    }
  };

  const confirmDeleteMapping = async (mappingId: number, modelId: number, providerName: string, targetModel: string) => {
    // 关闭确认对话框
    setConfirmDialog({ show: false, title: '', message: '', onConfirm: () => {} });

    // 设置删除状态
    setDeletingMappingId(mappingId);
    showToast(`正在删除映射 "${providerName}/${targetModel}"...`, 'info');

    try {
      await api.deleteMapping(mappingId);
      showToast('映射删除成功', 'success');
      await fetchModelMappings(modelId);
    } catch (err) {
      showToast('删除映射失败: ' + (err instanceof Error ? err.message : '未知错误'), 'error');
    } finally {
      setDeletingMappingId(null);
    }
  };

  const renderMappingSection = (model: UnifiedModel, layout: 'card' | 'list' | 'compact') => {
    if (expandedModelId !== model.id) return null;

    const containerClass =
      layout === 'card'
        ? 'border-t border-gray-200 bg-gray-50 p-6'
        : 'border border-gray-200 bg-gray-50 p-4 rounded-b-lg';

    const mappings = modelMappings[model.id];

    return (
      <div className={containerClass}>
        <h4 className="text-sm font-medium text-gray-900 mb-4">供应商映射</h4>
        {mappings && mappings.length > 0 ? (
          <div className="space-y-3">
            {mappings.map((mapping) => {
              const isEditing = editingMappingId === mapping.id;
              const providerName = providers.find((p) => p.id === mapping.provider_id)?.name || `Provider #${mapping.provider_id}`;
              return (
                <div key={mapping.id} className="bg-white p-4 rounded-lg border border-gray-200">
                  <div className="flex items-start justify-between">
                    <div className="flex-1">
                      <div className="flex items-center gap-3 mb-2">
                        <span className="text-sm font-mono text-gray-900">
                          {providerName}/{mapping.target_model}
                        </span>
                        {mapping.enabled ? (
                          <span className="inline-flex items-center px-2 py-0.5 rounded text-xs font-medium bg-green-100 text-green-800">启用</span>
                        ) : (
                          <span className="inline-flex items-center px-2 py-0.5 rounded text-xs font-medium bg-gray-100 text-gray-800">禁用</span>
                        )}
                      </div>

                      {isEditing ? (
                        <div className="space-y-3 mt-3">
                          <div className="flex gap-4">
                            <div className="flex-1">
                              <label className="block text-xs font-medium text-gray-700 mb-1">
                                <span className="inline-flex items-center gap-1">
                                  权重 (1-100)
                                  <span
                                    className="inline-flex items-center justify-center w-4 h-4 text-xs text-gray-500 border border-gray-300 rounded-full cursor-help hover:bg-gray-100 transition-colors"
                                    title="负载均衡时的权重值，数值越大被选中的概率越高"
                                  >
                                    ?
                                  </span>
                                </span>
                              </label>
                              <input
                                type="number"
                                min="1"
                                max="100"
                                value={editingMappingForm.weight}
                                onChange={(e) =>
                                  setEditingMappingForm({
                                    ...editingMappingForm,
                                    weight: Math.min(100, Math.max(1, parseInt(e.target.value) || 1)),
                                  })
                                }
                                className="w-full px-3 py-1.5 text-sm border border-gray-300 rounded-md focus:outline-none focus:ring-2 focus:ring-blue-500"
                              />
                            </div>
                            <div className="flex-1">
                              <label className="block text-xs font-medium text-gray-700 mb-1">
                                <span className="inline-flex items-center gap-1">
                                  优先级 (≥1)
                                  <span
                                    className="inline-flex items-center justify-center w-4 h-4 text-xs text-gray-500 border border-gray-300 rounded-full cursor-help hover:bg-gray-100 transition-colors"
                                    title="映射优先级，数字越小优先级越高"
                                  >
                                    ?
                                  </span>
                                </span>
                              </label>
                              <input
                                type="number"
                                min="1"
                                value={editingMappingForm.priority}
                                onChange={(e) =>
                                  setEditingMappingForm({
                                    ...editingMappingForm,
                                    priority: Math.max(1, parseInt(e.target.value) || 1),
                                  })
                                }
                                className="w-full px-3 py-1.5 text-sm border border-gray-300 rounded-md focus:outline-none focus:ring-2 focus:ring-blue-500"
                              />
                            </div>
                          </div>
                        </div>
                      ) : (
                        <div className="flex gap-4 text-xs text-gray-500">
                          <span>权重: {mapping.weight}</span>
                          <span>优先级: {mapping.priority}</span>
                        </div>
                      )}
                    </div>

                    <div className="flex gap-2 ml-4">
                      {isEditing ? (
                        <>
                          <button
                            onClick={() => handleSaveMapping(mapping.id, model.id)}
                            className="px-3 py-1.5 text-sm font-medium text-white bg-blue-600 rounded-md hover:bg-blue-700"
                          >
                            保存
                          </button>
                          <button
                            onClick={handleCancelEditMapping}
                            className="px-3 py-1.5 text-sm font-medium text-gray-700 bg-gray-100 rounded-md hover:bg-gray-200"
                          >
                            取消
                          </button>
                        </>
                      ) : (
                        <>
                          <button
                            onClick={() => handleEditMapping(mapping)}
                            className="px-3 py-1.5 text-sm font-medium text-gray-700 bg-gray-100 rounded-md hover:bg-gray-200"
                          >
                            编辑
                          </button>
                          <button
                            onClick={() => handleDeleteMapping(mapping.id, model.id, providerName, mapping.target_model)}
                            disabled={deletingMappingId === mapping.id}
                            className={`px-3 py-1.5 text-sm font-medium text-red-600 bg-red-50 rounded-md hover:bg-red-100 transition-opacity ${
                              deletingMappingId === mapping.id ? 'opacity-50 cursor-not-allowed' : ''
                            }`}
                          >
                            {deletingMappingId === mapping.id ? '删除中...' : '删除'}
                          </button>
                        </>
                      )}
                    </div>
                  </div>
                </div>
              );
            })}
          </div>
        ) : (
          <div className="text-sm text-gray-500">暂无映射，点击“添加映射”按钮进行配置。</div>
        )}
      </div>
    );
  };

  const renderListActions = (model: UnifiedModel) => (
    <div className="flex flex-wrap gap-2">
      <button onClick={() => handleAddMapping(model)} className="px-2.5 py-1 text-xs font-medium text-blue-600 bg-blue-50 rounded hover:bg-blue-100">
        添加映射
      </button>
      <button onClick={() => handleEditModel(model)} className="px-2.5 py-1 text-xs font-medium text-gray-700 bg-gray-100 rounded hover:bg-gray-200">
        编辑
      </button>
      <button
        onClick={() => handleDeleteModel(model.id, model.name)}
        disabled={deletingId === model.id}
        className={`px-2.5 py-1 text-xs font-medium text-red-600 bg-red-50 rounded hover:bg-red-100 transition-opacity ${
          deletingId === model.id ? 'opacity-50 cursor-not-allowed' : ''
        }`}
      >
        {deletingId === model.id ? '删除中...' : '删除'}
      </button>
      <button onClick={() => handleToggleExpand(model.id)} className="px-2.5 py-1 text-xs font-medium text-gray-700 bg-gray-100 rounded hover:bg-gray-200">
        {expandedModelId === model.id ? '收起' : '展开映射'}
      </button>
    </div>
  );

  const renderCompactActions = (model: UnifiedModel) => (
    <div className="flex flex-wrap gap-2 text-xs">
      <button onClick={() => handleAddMapping(model)} className="px-2 py-1 bg-blue-50 text-blue-600 rounded">
        映射
      </button>
      <button onClick={() => handleEditModel(model)} className="px-2 py-1 bg-gray-100 text-gray-700 rounded">
        编辑
      </button>
      <button
        onClick={() => handleDeleteModel(model.id, model.name)}
        disabled={deletingId === model.id}
        className={`px-2 py-1 bg-red-50 text-red-600 rounded transition-opacity ${
          deletingId === model.id ? 'opacity-50 cursor-not-allowed' : ''
        }`}
      >
        {deletingId === model.id ? '删除中...' : '删除'}
      </button>
      <button onClick={() => handleToggleExpand(model.id)} className="px-2 py-1 bg-gray-100 text-gray-700 rounded">
        {expandedModelId === model.id ? '收起' : '明细'}
      </button>
    </div>
  );

  const renderCardActions = (model: UnifiedModel) => (
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
        disabled={deletingId === model.id}
        className={`px-3 py-1.5 text-sm font-medium text-red-600 bg-red-50 rounded-md hover:bg-red-100 transition-opacity ${
          deletingId === model.id ? 'opacity-50 cursor-not-allowed' : ''
        }`}
        title={deletingId === model.id ? '正在删除...' : '删除模型'}
      >
        {deletingId === model.id ? '删除中...' : '删除'}
      </button>
      <button
        onClick={() => handleToggleExpand(model.id)}
        className="px-3 py-1.5 text-sm font-medium text-gray-700 bg-gray-100 rounded-md hover:bg-gray-200"
      >
        {expandedModelId === model.id ? '收起' : '展开映射'}
      </button>
    </div>
  );

  const renderCardContent = (model: UnifiedModel) => {
    const mappingCount = modelMappings[model.id]?.length ?? 0;
    return (
      <div className="p-6">
        <div className="flex items-start justify-between mb-4">
          <div className="flex-1">
            <div className="flex items-center gap-3 mb-2">
              <h3 className="text-xl font-semibold text-gray-900">{model.display_name}</h3>
              <span className="inline-flex items-center px-2.5 py-0.5 rounded-full text-xs font-medium bg-blue-100 text-blue-800">
                {model.name}
              </span>
              {mappingCount > 0 && (
                <span className="inline-flex items-center px-2.5 py-0.5 rounded-full text-xs font-medium bg-green-100 text-green-800">
                  {mappingCount} 个映射
                </span>
              )}
            </div>
            {model.description && <p className="text-sm text-gray-600">{model.description}</p>}
          </div>
          {renderCardActions(model)}
        </div>
      </div>
    );
  };

  const renderModels = () => {
    if (filteredModels.length === 0) {
      return (
        <div className="bg-white rounded-lg border border-dashed border-gray-300 p-12 text-center text-gray-500">
          暂无模型，点击右上角“创建统一模型”按钮进行添加。
        </div>
      );
    }

    if (viewMode === 'list') {
      return (
        <div className="bg-white rounded-lg shadow">
          <table className="min-w-full divide-y divide-gray-200">
            <thead className="bg-gray-50">
              <tr>
                <th className="px-4 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">模型名称</th>
                <th className="px-4 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">别名</th>
                <th className="px-4 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">描述</th>
                <th className="px-4 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">映射数量</th>
                <th className="px-4 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">操作</th>
              </tr>
            </thead>
            <tbody className="divide-y divide-gray-200">
              {filteredModels.map((model) => {
                const mappingCount = modelMappings[model.id]?.length ?? 0;
                return (
                  <Fragment key={model.id}>
                    <tr className="hover:bg-gray-50">
                      <td className="px-4 py-3 text-sm font-medium text-gray-900">{model.display_name}</td>
                      <td className="px-4 py-3 text-sm text-gray-500">{model.name}</td>
                      <td className="px-4 py-3 text-sm text-gray-500 truncate max-w-xs">{model.description || '-'}</td>
                      <td className="px-4 py-3 text-sm text-gray-500">{mappingCount}</td>
                      <td className="px-4 py-3 text-sm text-gray-500">{renderListActions(model)}</td>
                    </tr>
                    {expandedModelId === model.id && (
                      <tr className="bg-gray-50">
                        <td colSpan={5} className="px-4 py-4">
                          {renderMappingSection(model, 'list')}
                        </td>
                      </tr>
                    )}
                  </Fragment>
                );
              })}
            </tbody>
          </table>
        </div>
      );
    }

    if (viewMode === 'compact') {
      return (
        <div className="space-y-3">
          {filteredModels.map((model) => {
            const mappingCount = modelMappings[model.id]?.length ?? 0;
            return (
              <div key={model.id} className="bg-white rounded-lg border border-gray-200 p-4 shadow-sm">
                <div className="flex flex-col sm:flex-row sm:items-center sm:justify-between gap-3">
                  <div>
                    <div className="flex items-center gap-2">
                      <span className="text-sm font-semibold text-gray-900">{model.display_name}</span>
                      <span className="inline-flex items-center px-2 py-0.5 rounded text-xs bg-blue-100 text-blue-700">{model.name}</span>
                      <span className="inline-flex items-center px-2 py-0.5 rounded text-xs bg-green-100 text-green-700">映射 {mappingCount}</span>
                    </div>
                    {model.description && <p className="mt-1 text-xs text-gray-500 line-clamp-2">{model.description}</p>}
                  </div>
                  {renderCompactActions(model)}
                </div>
                {renderMappingSection(model, 'compact')}
              </div>
            );
          })}
        </div>
      );
    }

    return (
      <div className="grid grid-cols-1 gap-6">
        {filteredModels.map((model) => (
          <div key={model.id} className="bg-white rounded-lg shadow hover:shadow-lg transition-shadow">
            {renderCardContent(model)}
            {renderMappingSection(model, 'card')}
          </div>
        ))}
      </div>
    );
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

      {/* 确认删除对话框 */}
      {confirmDialog.show && (
        <ConfirmDialog
          title={confirmDialog.title}
          message={confirmDialog.message}
          onConfirm={confirmDialog.onConfirm}
          onCancel={() => {
            setConfirmDialog({ show: false, title: '', message: '', onConfirm: () => {} });
            showToast('已取消删除', 'info');
          }}
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
            <a
              href="/"
              className="px-4 py-2 text-sm font-medium text-gray-700 bg-white border border-gray-300 rounded-md hover:bg-gray-50"
            >
              返回首页
            </a>
            <button
              onClick={handleCreateModel}
              className="px-4 py-2 text-sm font-medium text-white bg-blue-600 border border-transparent rounded-md hover:bg-blue-700"
            >
              创建统一模型
            </button>
          </div>
        </div>

        {/* Tab 切换 */}
        <div className="mb-6 border-b border-gray-200">
          <nav className="flex space-x-6" aria-label="Tabs">
            <button
              type="button"
              onClick={() => setActiveTab('claudecode')}
              className={`pb-2 text-sm font-medium border-b-2 transition-colors ${
                activeTab === 'claudecode'
                  ? 'border-blue-500 text-blue-600'
                  : 'border-transparent text-gray-500 hover:text-gray-700'
              }`}
            >
              ClaudeCode 管理
            </button>
            <button
              type="button"
              onClick={() => setActiveTab('all')}
              className={`pb-2 text-sm font-medium border-b-2 transition-colors ${
                activeTab === 'all'
                  ? 'border-blue-500 text-blue-600'
                  : 'border-transparent text-gray-500 hover:text-gray-700'
              }`}
            >
              全部模型
            </button>
          </nav>
        </div>

        <div className="flex flex-col gap-6">
          <div className="flex flex-col gap-3 sm:flex-row sm:items-center sm:justify-between">
            <div className="text-sm text-gray-500">
              {filteredModels.length > 0 ? `共 ${filteredModels.length} 个模型` : '暂无模型数据'}
            </div>
            <div className="flex flex-wrap items-center gap-2">
              <span className="text-xs font-medium uppercase tracking-wide text-gray-500">视图模式</span>
              <div className="inline-flex divide-x divide-gray-200 rounded-md border border-gray-200 shadow-sm">
                {VIEW_MODE_OPTIONS.map((option) => {
                  const isActive = viewMode === option.value;
                  return (
                    <button
                      key={option.value}
                      type="button"
                      onClick={() => setViewMode(option.value)}
                      className={`p-2 transition-colors focus:outline-none focus:ring-2 focus:ring-blue-500 focus:ring-offset-2 focus:ring-offset-white ${
                        isActive
                          ? 'bg-blue-600 text-white'
                          : 'bg-white text-gray-600 hover:bg-gray-50'
                      }`}
                      aria-pressed={isActive}
                      aria-label={option.label}
                      title={option.label}
                    >
                      {VIEW_MODE_ICONS[option.value]}
                      <span className="sr-only">{option.label}</span>
                    </button>
                  );
                })}
              </div>
            </div>
          </div>
          {renderModels()}
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
  mappingForm: { target_model: string };
  onSelectProvider: (id: number) => void;
  onChangeMappingForm: (form: { target_model: string }) => void;
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

// 确认对话框组件
function ConfirmDialog({
  title,
  message,
  onConfirm,
  onCancel,
}: {
  title: string;
  message: string;
  onConfirm: () => void;
  onCancel: () => void;
}) {
  return (
    <div className="fixed inset-0 bg-gray-600 bg-opacity-50 flex items-center justify-center z-50">
      <div className="bg-white rounded-lg shadow-xl max-w-md w-full mx-4">
        <div className="px-6 py-4 border-b border-gray-200">
          <h3 className="text-lg font-medium text-gray-900">{title}</h3>
        </div>

        <div className="px-6 py-4">
          <p className="text-sm text-gray-700">{message}</p>
        </div>

        <div className="px-6 py-4 bg-gray-50 flex justify-end space-x-3 rounded-b-lg">
          <button
            type="button"
            onClick={onCancel}
            className="px-4 py-2 text-sm font-medium text-gray-700 bg-white border border-gray-300 rounded-md hover:bg-gray-50 focus:outline-none focus:ring-2 focus:ring-offset-2 focus:ring-blue-500"
          >
            取消
          </button>
          <button
            type="button"
            onClick={onConfirm}
            className="px-4 py-2 text-sm font-medium text-white bg-red-600 rounded-md hover:bg-red-700 focus:outline-none focus:ring-2 focus:ring-offset-2 focus:ring-red-500"
          >
            确认删除
          </button>
        </div>
      </div>
    </div>
  );
}
