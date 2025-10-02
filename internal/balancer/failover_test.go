package balancer

import (
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/Mieluoxxx/Siriusx-API/internal/mapping"
)

// Mock FailureDetector for testing
type mockFailureDetectorForFailover struct {
	unavailableProviders map[uint]bool
}

func (m *mockFailureDetectorForFailover) IsFailure(err error, resp *http.Response) bool {
	return false
}

func (m *mockFailureDetectorForFailover) RecordFailure(providerID uint, failureType FailureType) {
}

func (m *mockFailureDetectorForFailover) RecordSuccess(providerID uint) {
}

func (m *mockFailureDetectorForFailover) IsAvailable(providerID uint) bool {
	if m.unavailableProviders == nil {
		return true
	}
	return !m.unavailableProviders[providerID]
}

func (m *mockFailureDetectorForFailover) GetFailureStats(providerID uint) *FailureStats {
	return nil
}

func (m *mockFailureDetectorForFailover) Reset(providerID uint) {
}

func (m *mockFailureDetectorForFailover) GetAllStats() map[uint]*FailureStats {
	return nil
}

func (m *mockFailureDetectorForFailover) Close() {
}

// createTestMappingsForFailover 创建测试用的映射列表
func createTestMappingsForFailover(priorities []int) []*mapping.ResolvedMapping {
	mappings := make([]*mapping.ResolvedMapping, len(priorities))
	for i, priority := range priorities {
		mappings[i] = &mapping.ResolvedMapping{
			ProviderID:  uint(i + 1),
			TargetModel: "model-" + string(rune('a'+i)),
			Weight:      50,
			Priority:    priority,
			Provider: &mapping.ProviderInfo{
				ID:   uint(i + 1),
				Name: "Provider " + string(rune('A'+i)),
			},
		}
	}
	return mappings
}

// TestFailoverExecutor_FirstProviderAvailable 测试第一个供应商可用的情况
func TestFailoverExecutor_FirstProviderAvailable(t *testing.T) {
	balancer := NewWeightedRandomBalancer()
	detector := &mockFailureDetectorForFailover{}
	config := &FailoverConfig{
		MaxRetries:     3,
		EnableFailover: true,
	}
	executor := NewFailoverExecutor(balancer, detector, config)

	// 3 个供应商,优先级 1, 2, 3,全部可用
	mappings := createTestMappingsForFailover([]int{1, 2, 3})

	result, err := executor.SelectProviderWithFailover(mappings)

	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.Equal(t, uint(1), result.SelectedProvider.ProviderID, "Should select first provider (Priority 1)")
	assert.Equal(t, 1, result.AttemptCount, "Should attempt only once")
	assert.Equal(t, 0, len(result.FailedProviders), "Should have no failed providers")
}

// TestFailoverExecutor_FirstProviderUnavailable 测试第一个供应商不可用的情况
func TestFailoverExecutor_FirstProviderUnavailable(t *testing.T) {
	balancer := NewWeightedRandomBalancer()
	detector := &mockFailureDetectorForFailover{
		unavailableProviders: map[uint]bool{
			1: true, // Provider 1 不可用
		},
	}
	config := &FailoverConfig{
		MaxRetries:     3,
		EnableFailover: true,
	}
	executor := NewFailoverExecutor(balancer, detector, config)

	// 3 个供应商,优先级 1, 2, 3
	mappings := createTestMappingsForFailover([]int{1, 2, 3})

	result, err := executor.SelectProviderWithFailover(mappings)

	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.Equal(t, uint(2), result.SelectedProvider.ProviderID, "Should failover to second provider")
	assert.Equal(t, 2, result.AttemptCount, "Should attempt twice")
	assert.Equal(t, 1, len(result.FailedProviders), "Should have one failed provider")
	assert.Equal(t, uint(1), result.FailedProviders[0].ProviderID)
	assert.Equal(t, FailureTypeCooldown, result.FailedProviders[0].FailureType)
}

// TestFailoverExecutor_MultipleFailovers 测试多次故障转移
func TestFailoverExecutor_MultipleFailovers(t *testing.T) {
	balancer := NewWeightedRandomBalancer()
	detector := &mockFailureDetectorForFailover{
		unavailableProviders: map[uint]bool{
			1: true, // Provider 1 不可用
			2: true, // Provider 2 不可用
		},
	}
	config := &FailoverConfig{
		MaxRetries:     3,
		EnableFailover: true,
	}
	executor := NewFailoverExecutor(balancer, detector, config)

	// 3 个供应商,优先级 1, 2, 3
	mappings := createTestMappingsForFailover([]int{1, 2, 3})

	result, err := executor.SelectProviderWithFailover(mappings)

	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.Equal(t, uint(3), result.SelectedProvider.ProviderID, "Should failover to third provider")
	assert.Equal(t, 3, result.AttemptCount, "Should attempt three times")
	assert.Equal(t, 2, len(result.FailedProviders), "Should have two failed providers")
}

// TestFailoverExecutor_MaxRetriesLimit 测试重试次数限制
func TestFailoverExecutor_MaxRetriesLimit(t *testing.T) {
	balancer := NewWeightedRandomBalancer()
	detector := &mockFailureDetectorForFailover{
		unavailableProviders: map[uint]bool{
			1: true,
			2: true,
			3: true,
			4: true,
			5: true,
		},
	}
	config := &FailoverConfig{
		MaxRetries:     3, // 最多尝试 3 次
		EnableFailover: true,
	}
	executor := NewFailoverExecutor(balancer, detector, config)

	// 5 个供应商,全部不可用
	mappings := createTestMappingsForFailover([]int{1, 2, 3, 4, 5})

	result, err := executor.SelectProviderWithFailover(mappings)

	assert.Error(t, err)
	assert.NotNil(t, result)
	assert.Nil(t, result.SelectedProvider)
	assert.Equal(t, 3, result.AttemptCount, "Should attempt only MaxRetries times")
	assert.Equal(t, 3, len(result.FailedProviders), "Should record only 3 failures")
}

// TestFailoverExecutor_AllProvidersUnavailable 测试所有供应商都不可用
func TestFailoverExecutor_AllProvidersUnavailable(t *testing.T) {
	balancer := NewWeightedRandomBalancer()
	detector := &mockFailureDetectorForFailover{
		unavailableProviders: map[uint]bool{
			1: true,
			2: true,
			3: true,
		},
	}
	config := &FailoverConfig{
		MaxRetries:     3,
		EnableFailover: true,
	}
	executor := NewFailoverExecutor(balancer, detector, config)

	// 3 个供应商,全部不可用
	mappings := createTestMappingsForFailover([]int{1, 2, 3})

	result, err := executor.SelectProviderWithFailover(mappings)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "all providers unavailable")
	assert.NotNil(t, result)
	assert.Nil(t, result.SelectedProvider)
	assert.Equal(t, 3, result.AttemptCount)
	assert.Equal(t, 3, len(result.FailedProviders))
}

// TestFailoverExecutor_PrioritySorting 测试优先级排序
func TestFailoverExecutor_PrioritySorting(t *testing.T) {
	balancer := NewWeightedRandomBalancer()
	detector := &mockFailureDetectorForFailover{}
	config := &FailoverConfig{
		MaxRetries:     3,
		EnableFailover: true,
	}
	executor := NewFailoverExecutor(balancer, detector, config)

	// 提供乱序的映射列表 (Priority: 3, 1, 2)
	mappings := []*mapping.ResolvedMapping{
		{ProviderID: 3, TargetModel: "model-c", Priority: 3, Provider: &mapping.ProviderInfo{ID: 3, Name: "Provider C"}},
		{ProviderID: 1, TargetModel: "model-a", Priority: 1, Provider: &mapping.ProviderInfo{ID: 1, Name: "Provider A"}},
		{ProviderID: 2, TargetModel: "model-b", Priority: 2, Provider: &mapping.ProviderInfo{ID: 2, Name: "Provider B"}},
	}

	result, err := executor.SelectProviderWithFailover(mappings)

	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.Equal(t, uint(1), result.SelectedProvider.ProviderID, "Should select provider with Priority 1")
	assert.Equal(t, 1, result.SelectedProvider.Priority, "Selected provider should have Priority 1")
}

// TestFailoverExecutor_NoProvidersAvailable 测试没有可用供应商
func TestFailoverExecutor_NoProvidersAvailable(t *testing.T) {
	balancer := NewWeightedRandomBalancer()
	detector := &mockFailureDetectorForFailover{}
	config := &FailoverConfig{
		MaxRetries:     3,
		EnableFailover: true,
	}
	executor := NewFailoverExecutor(balancer, detector, config)

	// 空映射列表
	mappings := []*mapping.ResolvedMapping{}

	result, err := executor.SelectProviderWithFailover(mappings)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "no available providers")
	assert.Nil(t, result)
}

// TestFailoverExecutor_FailoverDisabled 测试禁用故障转移
func TestFailoverExecutor_FailoverDisabled(t *testing.T) {
	balancer := NewWeightedRandomBalancer()
	detector := &mockFailureDetectorForFailover{}
	config := &FailoverConfig{
		MaxRetries:     3,
		EnableFailover: false, // 禁用故障转移
	}
	executor := NewFailoverExecutor(balancer, detector, config)

	mappings := createTestMappingsForFailover([]int{1, 2, 3})

	result, err := executor.SelectProviderWithFailover(mappings)

	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.NotNil(t, result.SelectedProvider, "Should use load balancer selection")
	assert.Equal(t, 1, result.AttemptCount)
}

// TestFailoverExecutor_IntelligentSelection 测试智能选择功能
func TestFailoverExecutor_IntelligentSelection(t *testing.T) {
	balancer := NewWeightedRandomBalancerWithSeed(42)
	detector := &mockFailureDetectorForFailover{
		unavailableProviders: map[uint]bool{
			1: true, // 假设负载均衡选中了 Provider 1,但它不可用
		},
	}
	config := &FailoverConfig{
		MaxRetries:     3,
		EnableFailover: true,
	}
	executor := NewFailoverExecutor(balancer, detector, config)

	mappings := createTestMappingsForFailover([]int{1, 2, 3})

	selected, err := executor.SelectProviderIntelligent(mappings)

	assert.NoError(t, err)
	assert.NotNil(t, selected)
	// 应该选择可用的供应商
	assert.True(t, detector.IsAvailable(selected.ProviderID), "Selected provider should be available")
}

// TestFailoverExecutor_IntelligentSelection_AllUnavailable 测试智能选择但全部不可用
func TestFailoverExecutor_IntelligentSelection_AllUnavailable(t *testing.T) {
	balancer := NewWeightedRandomBalancer()
	detector := &mockFailureDetectorForFailover{
		unavailableProviders: map[uint]bool{
			1: true,
			2: true,
			3: true,
		},
	}
	config := &FailoverConfig{
		MaxRetries:     3,
		EnableFailover: true,
	}
	executor := NewFailoverExecutor(balancer, detector, config)

	mappings := createTestMappingsForFailover([]int{1, 2, 3})

	selected, err := executor.SelectProviderIntelligent(mappings)

	assert.Error(t, err)
	assert.Nil(t, selected)
	assert.Contains(t, err.Error(), "all providers unavailable")
}

// TestFailoverExecutor_ConfigUpdate 测试配置更新
func TestFailoverExecutor_ConfigUpdate(t *testing.T) {
	balancer := NewWeightedRandomBalancer()
	detector := &mockFailureDetectorForFailover{}
	config := &FailoverConfig{
		MaxRetries:     3,
		EnableFailover: true,
	}
	executor := NewFailoverExecutor(balancer, detector, config)

	// 更新配置
	newConfig := &FailoverConfig{
		MaxRetries:     5,
		EnableFailover: false,
	}
	executor.UpdateConfig(newConfig)

	// 验证配置已更新
	currentConfig := executor.GetConfig()
	assert.Equal(t, 5, currentConfig.MaxRetries)
	assert.False(t, currentConfig.EnableFailover)
}

// TestFailoverExecutor_NilDetector 测试无故障检测器的情况
func TestFailoverExecutor_NilDetector(t *testing.T) {
	balancer := NewWeightedRandomBalancer()
	config := &FailoverConfig{
		MaxRetries:     3,
		EnableFailover: true,
	}
	executor := NewFailoverExecutor(balancer, nil, config) // nil detector

	mappings := createTestMappingsForFailover([]int{1, 2, 3})

	result, err := executor.SelectProviderWithFailover(mappings)

	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.NotNil(t, result.SelectedProvider, "Should select first provider when no detector")
	assert.Equal(t, uint(1), result.SelectedProvider.ProviderID)
}

// TestNewFailoverExecutor_DefaultConfig 测试默认配置
func TestNewFailoverExecutor_DefaultConfig(t *testing.T) {
	balancer := NewWeightedRandomBalancer()
	detector := &mockFailureDetectorForFailover{}

	executor := NewFailoverExecutor(balancer, detector, nil) // nil config

	config := executor.GetConfig()
	assert.Equal(t, 3, config.MaxRetries, "Default MaxRetries should be 3")
	assert.True(t, config.EnableFailover, "Default EnableFailover should be true")
}
