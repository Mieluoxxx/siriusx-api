package mapping

import (
	"time"

	"github.com/Mieluoxxx/Siriusx-API/internal/models"
)

// ==================== 路由解析相关类型 ====================

// ResolvedMapping 解析后的映射信息
type ResolvedMapping struct {
	ID             uint          `json:"id"`
	UnifiedModelID uint          `json:"unified_model_id"`
	ProviderID     uint          `json:"provider_id"`
	TargetModel    string        `json:"target_model"`
	Weight         int           `json:"weight"`
	Priority       int           `json:"priority"`
	Enabled        bool          `json:"enabled"`
	Provider       *ProviderInfo `json:"provider"`
	CreatedAt      time.Time     `json:"created_at"`
	UpdatedAt      time.Time     `json:"updated_at"`
}

// ProviderInfo 供应商信息
type ProviderInfo struct {
	ID           uint   `json:"id"`
	Name         string `json:"name"`
	BaseURL      string `json:"base_url"`
	Enabled      bool   `json:"enabled"`
	HealthStatus string `json:"health_status"`
}

// ==================== 缓存相关类型 ====================

// CacheEntry 缓存条目
type CacheEntry struct {
	Data      []*ResolvedMapping `json:"data"`
	ExpiresAt time.Time          `json:"expires_at"`
	CreatedAt time.Time          `json:"created_at"`
	HitCount  int64              `json:"hit_count"`
}

// CacheStats 缓存统计信息
type CacheStats struct {
	Size        int           `json:"size"`         // 当前缓存条目数
	HitCount    int64         `json:"hit_count"`    // 缓存命中次数
	MissCount   int64         `json:"miss_count"`   // 缓存未命中次数
	HitRate     float64       `json:"hit_rate"`     // 缓存命中率
	MemoryUsage int64         `json:"memory_usage"` // 内存使用量(字节)
	LastCleanup time.Time     `json:"last_cleanup"` // 最后清理时间
	TTL         time.Duration `json:"ttl"`          // TTL 设置
}

// CacheConfig 缓存配置
type CacheConfig struct {
	TTL         time.Duration `yaml:"ttl"`          // 默认: 5分钟
	MaxSize     int           `yaml:"max_size"`     // 默认: 1000
	CleanupTime time.Duration `yaml:"cleanup_time"` // 默认: 10分钟
}

// ==================== 路由错误类型 ====================

// RouterError 路由错误
type RouterError struct {
	Code    string `json:"code"`
	Message string `json:"message"`
	Model   string `json:"model,omitempty"`
}

func (e *RouterError) Error() string {
	return e.Message
}

// 预定义路由错误
var (
	ErrRouterModelNotFound      = &RouterError{Code: "MODEL_NOT_FOUND", Message: "模型未找到"}
	ErrRouterNoAvailableProviders = &RouterError{Code: "NO_AVAILABLE_PROVIDERS", Message: "暂无可用供应商"}
	ErrRouterMappingDisabled    = &RouterError{Code: "MAPPING_DISABLED", Message: "映射已被禁用"}
	ErrRouterInternalError      = &RouterError{Code: "ROUTER_ERROR", Message: "路由解析失败"}
)

// NewModelNotFoundError 创建模型未找到错误
func NewModelNotFoundError(modelName string) *RouterError {
	return &RouterError{
		Code:    "MODEL_NOT_FOUND",
		Message: "模型 '" + modelName + "' 未找到",
		Model:   modelName,
	}
}

// NewNoAvailableProvidersError 创建无可用供应商错误
func NewNoAvailableProvidersError(modelName string) *RouterError {
	return &RouterError{
		Code:    "NO_AVAILABLE_PROVIDERS",
		Message: "模型 '" + modelName + "' 暂无可用供应商",
		Model:   modelName,
	}
}

// ==================== 转换函数 ====================

// ToResolvedMapping 将模型映射转换为解析后的映射
func ToResolvedMapping(mapping *models.ModelMapping) *ResolvedMapping {
	resolved := &ResolvedMapping{
		ID:             mapping.ID,
		UnifiedModelID: mapping.UnifiedModelID,
		ProviderID:     mapping.ProviderID,
		TargetModel:    mapping.TargetModel,
		Weight:         mapping.Weight,
		Priority:       mapping.Priority,
		Enabled:        mapping.Enabled,
		CreatedAt:      mapping.CreatedAt,
		UpdatedAt:      mapping.UpdatedAt,
	}

	// 转换供应商信息
	if mapping.Provider.ID > 0 {
		resolved.Provider = &ProviderInfo{
			ID:           mapping.Provider.ID,
			Name:         mapping.Provider.Name,
			BaseURL:      mapping.Provider.BaseURL,
			Enabled:      mapping.Provider.Enabled,
			HealthStatus: mapping.Provider.HealthStatus,
		}
	}

	return resolved
}

// ToResolvedMappingList 将模型映射列表转换为解析后的映射列表
func ToResolvedMappingList(mappings []*models.ModelMapping) []*ResolvedMapping {
	resolved := make([]*ResolvedMapping, len(mappings))
	for i, mapping := range mappings {
		resolved[i] = ToResolvedMapping(mapping)
	}
	return resolved
}