package mapping

import (
	"context"
	"fmt"
	"sort"
	"strings"
	"sync"
	"time"
)

// ==================== 路由接口 ====================

// Router 路由解析接口
type Router interface {
	// ResolveModel 根据模型名称解析映射列表
	ResolveModel(ctx context.Context, modelName string) ([]*ResolvedMapping, error)

	// InvalidateCache 清理指定模型的缓存
	InvalidateCache(modelName string)

	// ClearCache 清理所有缓存
	ClearCache()

	// GetCacheStats 获取缓存统计信息
	GetCacheStats() *CacheStats

	// Close 关闭路由器，释放资源
	Close() error
}

// ==================== 默认路由实现 ====================

// DefaultRouter 默认路由实现
type DefaultRouter struct {
	mu         sync.RWMutex
	repository *Repository
	cache      Cache
	config     *RouterConfig
}

// RouterConfig 路由配置
type RouterConfig struct {
	Cache        *CacheConfig `yaml:"cache"`
	HealthCheck  bool         `yaml:"health_check"`
	EnableWeight bool         `yaml:"enable_weight"`
}

// NewRouter 创建新的路由器
func NewRouter(repository *Repository, config *RouterConfig) *DefaultRouter {
	if config == nil {
		config = DefaultRouterConfig()
	}

	if config.Cache == nil {
		config.Cache = DefaultCacheConfig()
	}

	return &DefaultRouter{
		repository: repository,
		cache:      NewMemoryCache(config.Cache),
		config:     config,
	}
}

// ResolveModel 根据模型名称解析映射列表
func (r *DefaultRouter) ResolveModel(ctx context.Context, modelName string) ([]*ResolvedMapping, error) {
	// 参数验证
	if strings.TrimSpace(modelName) == "" {
		return nil, NewModelNotFoundError(modelName)
	}

	// 尝试从缓存获取
	if cached, found := r.cache.Get(modelName); found {
		return cached, nil
	}

	// 从数据库查询
	mappings, err := r.resolveMappingsFromDB(ctx, modelName)
	if err != nil {
		return nil, err
	}

	// 存入缓存
	r.cache.Set(modelName, mappings)

	return mappings, nil
}

// InvalidateCache 清理指定模型的缓存
func (r *DefaultRouter) InvalidateCache(modelName string) {
	r.cache.Delete(modelName)
}

// ClearCache 清理所有缓存
func (r *DefaultRouter) ClearCache() {
	r.cache.Clear()
}

// GetCacheStats 获取缓存统计信息
func (r *DefaultRouter) GetCacheStats() *CacheStats {
	return r.cache.Stats()
}

// Close 关闭路由器，释放资源
func (r *DefaultRouter) Close() error {
	if memCache, ok := r.cache.(*MemoryCache); ok {
		memCache.Close()
	}
	return nil
}

// ==================== 私有方法 ====================

// resolveMappingsFromDB 从数据库解析映射
func (r *DefaultRouter) resolveMappingsFromDB(ctx context.Context, modelName string) ([]*ResolvedMapping, error) {
	// 查找统一模型
	unifiedModel, err := r.repository.FindByName(modelName)
	if err != nil {
		if err == ErrModelNotFound {
			return nil, NewModelNotFoundError(modelName)
		}
		return nil, fmt.Errorf("failed to find unified model: %w", err)
	}

	// 查询模型的所有映射（包含供应商信息）
	modelMappings, err := r.repository.FindMappingsByModelIDWithAll(unifiedModel.ID, true)
	if err != nil {
		return nil, fmt.Errorf("failed to find mappings: %w", err)
	}

	if len(modelMappings) == 0 {
		return nil, NewNoAvailableProvidersError(modelName)
	}

	// 转换为解析后的映射
	resolvedMappings := ToResolvedMappingList(modelMappings)

	// 过滤健康的供应商
	if r.config.HealthCheck {
		resolvedMappings = r.filterHealthyProviders(resolvedMappings)
	}

	// 过滤启用的映射
	resolvedMappings = r.filterEnabledMappings(resolvedMappings)

	if len(resolvedMappings) == 0 {
		return nil, NewNoAvailableProvidersError(modelName)
	}

	// 排序映射
	r.sortMappings(resolvedMappings)

	return resolvedMappings, nil
}

// filterHealthyProviders 过滤健康的供应商
func (r *DefaultRouter) filterHealthyProviders(mappings []*ResolvedMapping) []*ResolvedMapping {
	var healthy []*ResolvedMapping

	for _, mapping := range mappings {
		if mapping.Provider != nil &&
			mapping.Provider.Enabled &&
			r.isProviderHealthy(mapping.Provider) {
			healthy = append(healthy, mapping)
		}
	}

	return healthy
}

// filterEnabledMappings 过滤启用的映射
func (r *DefaultRouter) filterEnabledMappings(mappings []*ResolvedMapping) []*ResolvedMapping {
	var enabled []*ResolvedMapping

	for _, mapping := range mappings {
		if mapping.Enabled {
			enabled = append(enabled, mapping)
		}
	}

	return enabled
}

// sortMappings 排序映射（按优先级升序，权重降序）
func (r *DefaultRouter) sortMappings(mappings []*ResolvedMapping) {
	sort.Slice(mappings, func(i, j int) bool {
		// 优先级升序（1 = 最高优先级）
		if mappings[i].Priority != mappings[j].Priority {
			return mappings[i].Priority < mappings[j].Priority
		}

		// 相同优先级时，按权重降序
		if r.config.EnableWeight {
			return mappings[i].Weight > mappings[j].Weight
		}

		// 如果不考虑权重，按 ID 升序保证稳定排序
		return mappings[i].ID < mappings[j].ID
	})
}

// isProviderHealthy 检查供应商是否健康
func (r *DefaultRouter) isProviderHealthy(provider *ProviderInfo) bool {
	// 简化实现：基于健康状态字符串判断
	switch strings.ToLower(provider.HealthStatus) {
	case "healthy", "ok", "active", "":
		return true
	case "unhealthy", "error", "failed", "timeout":
		return false
	default:
		// 未知状态默认为健康
		return true
	}
}

// ==================== 路由选择算法 ====================

// SelectProvider 选择供应商（负载均衡）
func (r *DefaultRouter) SelectProvider(mappings []*ResolvedMapping) *ResolvedMapping {
	if len(mappings) == 0 {
		return nil
	}

	if !r.config.EnableWeight {
		// 不考虑权重，返回第一个（优先级最高）
		return mappings[0]
	}

	// 加权轮询算法
	return r.selectByWeight(mappings)
}

// selectByWeight 基于权重选择供应商
func (r *DefaultRouter) selectByWeight(mappings []*ResolvedMapping) *ResolvedMapping {
	// 简化实现：基于权重随机选择
	totalWeight := 0
	for _, mapping := range mappings {
		totalWeight += mapping.Weight
	}

	if totalWeight == 0 {
		// 如果总权重为0，返回第一个
		return mappings[0]
	}

	// 使用时间戳作为简单的随机数源
	rand := int(time.Now().UnixNano()) % totalWeight
	currentWeight := 0

	for _, mapping := range mappings {
		currentWeight += mapping.Weight
		if rand < currentWeight {
			return mapping
		}
	}

	// 兜底返回第一个
	return mappings[0]
}

// ==================== 批量操作 ====================

// ResolveBatch 批量解析模型
func (r *DefaultRouter) ResolveBatch(ctx context.Context, modelNames []string) (map[string][]*ResolvedMapping, error) {
	results := make(map[string][]*ResolvedMapping, len(modelNames))
	errors := make(map[string]error)

	for _, modelName := range modelNames {
		mappings, err := r.ResolveModel(ctx, modelName)
		if err != nil {
			errors[modelName] = err
		} else {
			results[modelName] = mappings
		}
	}

	if len(errors) > 0 {
		// 如果有错误，返回第一个错误
		for modelName, err := range errors {
			return results, fmt.Errorf("failed to resolve model %s: %w", modelName, err)
		}
	}

	return results, nil
}

// ==================== 配置 ====================

// DefaultRouterConfig 默认路由配置
func DefaultRouterConfig() *RouterConfig {
	return &RouterConfig{
		Cache:        DefaultCacheConfig(),
		HealthCheck:  true,
		EnableWeight: true,
	}
}

// ==================== 路由器工厂 ====================

// RouterFactory 路由器工厂
type RouterFactory struct {
	routers map[string]Router
	mu      sync.RWMutex
}

// NewRouterFactory 创建路由器工厂
func NewRouterFactory() *RouterFactory {
	return &RouterFactory{
		routers: make(map[string]Router),
	}
}

// GetRouter 获取或创建路由器
func (f *RouterFactory) GetRouter(name string, repository *Repository, config *RouterConfig) Router {
	f.mu.RLock()
	if router, exists := f.routers[name]; exists {
		f.mu.RUnlock()
		return router
	}
	f.mu.RUnlock()

	f.mu.Lock()
	defer f.mu.Unlock()

	// 双重检查
	if router, exists := f.routers[name]; exists {
		return router
	}

	// 创建新路由器
	router := NewRouter(repository, config)
	f.routers[name] = router
	return router
}

// CloseAll 关闭所有路由器
func (f *RouterFactory) CloseAll() error {
	f.mu.Lock()
	defer f.mu.Unlock()

	for name, router := range f.routers {
		if err := router.Close(); err != nil {
			return fmt.Errorf("failed to close router %s: %w", name, err)
		}
	}

	f.routers = make(map[string]Router)
	return nil
}