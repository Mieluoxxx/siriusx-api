package balancer

import (
	"errors"
	"sort"
	"sync"

	"github.com/Mieluoxxx/Siriusx-API/internal/mapping"
)

// FailoverExecutor 故障转移执行器
type FailoverExecutor struct {
	balancer        LoadBalancer
	failureDetector FailureDetector
	config          *FailoverConfig
	mutex           sync.RWMutex
}

// FailoverConfig 故障转移配置
type FailoverConfig struct {
	MaxRetries     int  // 最大重试次数,默认 3
	EnableFailover bool // 是否启用故障转移,默认 true
}

// FailoverResult 故障转移结果
type FailoverResult struct {
	SelectedProvider *mapping.ResolvedMapping // 成功的供应商映射
	AttemptCount     int                      // 尝试次数
	FailedProviders  []FailureAttempt         // 失败的供应商列表
}

// FailureAttempt 故障转移尝试记录
type FailureAttempt struct {
	ProviderID  uint        // 供应商 ID
	TargetModel string      // 目标模型
	FailureType FailureType // 故障类型
	Error       error       // 错误信息
}

// NewFailoverExecutor 创建故障转移执行器
func NewFailoverExecutor(
	balancer LoadBalancer,
	failureDetector FailureDetector,
	config *FailoverConfig,
) *FailoverExecutor {
	if config == nil {
		config = &FailoverConfig{
			MaxRetries:     3,
			EnableFailover: true,
		}
	}

	return &FailoverExecutor{
		balancer:        balancer,
		failureDetector: failureDetector,
		config:          config,
	}
}

// SelectProviderWithFailover 选择供应商并支持故障转移
func (f *FailoverExecutor) SelectProviderWithFailover(
	mappings []*mapping.ResolvedMapping,
) (*FailoverResult, error) {
	if len(mappings) == 0 {
		return nil, errors.New("no available providers")
	}

	// 如果未启用故障转移,使用负载均衡器直接选择
	if !f.config.EnableFailover {
		selected := f.balancer.SelectProvider(mappings)
		if selected == nil {
			return nil, errors.New("balancer returned nil provider")
		}
		return &FailoverResult{
			SelectedProvider: selected,
			AttemptCount:     1,
			FailedProviders:  []FailureAttempt{},
		}, nil
	}

	// 1. 按优先级排序 (Priority 从小到大)
	sortedMappings := f.sortByPriority(mappings)

	result := &FailoverResult{
		FailedProviders: []FailureAttempt{},
	}

	// 2. 按优先级顺序尝试每个供应商
	for i, mapping := range sortedMappings {
		// 检查是否超过重试次数
		if i >= f.config.MaxRetries {
			break
		}

		// 检查供应商是否可用 (未在冷却期)
		if f.failureDetector != nil && !f.failureDetector.IsAvailable(mapping.ProviderID) {
			result.FailedProviders = append(result.FailedProviders, FailureAttempt{
				ProviderID:  mapping.ProviderID,
				TargetModel: mapping.TargetModel,
				FailureType: FailureTypeCooldown,
				Error:       errors.New("provider in cooldown period"),
			})
			result.AttemptCount++
			continue
		}

		// 3. 选择该供应商
		result.SelectedProvider = mapping
		result.AttemptCount++
		return result, nil
	}

	// 4. 所有供应商都不可用
	return result, errors.New("all providers unavailable or in cooldown")
}

// SelectProviderIntelligent 智能选择供应商 (负载均衡 + 故障转移)
func (f *FailoverExecutor) SelectProviderIntelligent(
	mappings []*mapping.ResolvedMapping,
) (*mapping.ResolvedMapping, error) {
	if len(mappings) == 0 {
		return nil, errors.New("no available providers")
	}

	// 1. 先使用负载均衡器选择
	selected := f.balancer.SelectProvider(mappings)
	if selected != nil {
		// 检查选中的供应商是否可用
		if f.failureDetector == nil || f.failureDetector.IsAvailable(selected.ProviderID) {
			return selected, nil
		}
	}

	// 2. 如果负载均衡选择的供应商不可用,使用故障转移
	if !f.config.EnableFailover {
		return nil, errors.New("selected provider unavailable and failover disabled")
	}

	result, err := f.SelectProviderWithFailover(mappings)
	if err != nil {
		return nil, err
	}

	return result.SelectedProvider, nil
}

// sortByPriority 按优先级排序映射列表
func (f *FailoverExecutor) sortByPriority(
	mappings []*mapping.ResolvedMapping,
) []*mapping.ResolvedMapping {
	// 创建副本避免修改原始切片
	sorted := make([]*mapping.ResolvedMapping, len(mappings))
	copy(sorted, mappings)

	// 按 Priority 从小到大排序 (Priority 越小优先级越高)
	sort.Slice(sorted, func(i, j int) bool {
		return sorted[i].Priority < sorted[j].Priority
	})

	return sorted
}

// GetConfig 获取配置
func (f *FailoverExecutor) GetConfig() *FailoverConfig {
	f.mutex.RLock()
	defer f.mutex.RUnlock()

	return &FailoverConfig{
		MaxRetries:     f.config.MaxRetries,
		EnableFailover: f.config.EnableFailover,
	}
}

// UpdateConfig 更新配置
func (f *FailoverExecutor) UpdateConfig(config *FailoverConfig) {
	f.mutex.Lock()
	defer f.mutex.Unlock()

	if config != nil {
		f.config = config
	}
}

// FailureTypeCooldown 冷却期故障类型
const FailureTypeCooldown FailureType = "cooldown"
