package balancer

import (
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/Mieluoxxx/Siriusx-API/internal/mapping"
)

// TestFailoverScenario_RealFailoverFlow 测试真实故障转移流程
func TestFailoverScenario_RealFailoverFlow(t *testing.T) {
	// 1. 创建故障检测器,模拟主供应商故障
	detectorConfig := &FailureDetectorConfig{
		FailureThreshold:  3,
		CooldownDuration:  100 * time.Millisecond,
		TimeoutThreshold:  30 * time.Second,
		CleanupInterval:   10 * time.Minute,
		MaxFailureHistory: 100,
	}
	detector := NewFailureDetector(detectorConfig)

	// 2. 创建负载均衡器
	balancer := NewWeightedRandomBalancerWithSeed(42)

	// 3. 创建故障转移执行器
	failoverConfig := &FailoverConfig{
		MaxRetries:     3,
		EnableFailover: true,
	}
	executor := NewFailoverExecutor(balancer, detector, failoverConfig)

	// 4. 创建映射列表
	mappings := []*mapping.ResolvedMapping{
		{
			ProviderID:  1,
			TargetModel: "model-a",
			Weight:      70,
			Priority:    1,
			Provider: &mapping.ProviderInfo{
				ID:   1,
				Name: "Provider A",
			},
		},
		{
			ProviderID:  2,
			TargetModel: "model-b",
			Weight:      30,
			Priority:    2,
			Provider: &mapping.ProviderInfo{
				ID:   2,
				Name: "Provider B",
			},
		},
		{
			ProviderID:  3,
			TargetModel: "model-c",
			Weight:      0,
			Priority:    3,
			Provider: &mapping.ProviderInfo{
				ID:   3,
				Name: "Provider C",
			},
		},
	}

	// 5. 模拟 Provider 1 连续 3 次故障,进入冷却期
	for i := 0; i < 3; i++ {
		detector.RecordFailure(1, TimeoutFailure)
	}

	// 验证 Provider 1 不可用
	assert.False(t, detector.IsAvailable(1), "Provider 1 should be in cooldown")

	// 6. 执行故障转移
	result, err := executor.SelectProviderWithFailover(mappings)

	// 验证故障转移成功
	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.Equal(t, uint(2), result.SelectedProvider.ProviderID, "Should failover to Provider 2")
	assert.Equal(t, 2, result.AttemptCount)
	assert.Equal(t, 1, len(result.FailedProviders))

	// 7. 等待冷却期结束
	time.Sleep(150 * time.Millisecond)

	// 验证 Provider 1 恢复
	assert.True(t, detector.IsAvailable(1), "Provider 1 should recover after cooldown")

	// 8. 再次执行选择,应该能选到 Provider 1
	result2, err := executor.SelectProviderWithFailover(mappings)
	assert.NoError(t, err)
	assert.Equal(t, uint(1), result2.SelectedProvider.ProviderID, "Should select Provider 1 after recovery")
}

// TestFailoverScenario_LoadBalancerIntegration 测试与负载均衡器协同
func TestFailoverScenario_LoadBalancerIntegration(t *testing.T) {
	detector := &mockFailureDetectorForFailover{
		unavailableProviders: map[uint]bool{
			1: true, // Provider 1 不可用
		},
	}

	// 使用固定种子的负载均衡器
	balancer := NewWeightedRandomBalancerWithSeed(42)

	failoverConfig := &FailoverConfig{
		MaxRetries:     3,
		EnableFailover: true,
	}
	executor := NewFailoverExecutor(balancer, detector, failoverConfig)

	mappings := []*mapping.ResolvedMapping{
		{ProviderID: 1, TargetModel: "model-a", Weight: 100, Priority: 1, Provider: &mapping.ProviderInfo{ID: 1, Name: "Provider A"}},
		{ProviderID: 2, TargetModel: "model-b", Weight: 0, Priority: 2, Provider: &mapping.ProviderInfo{ID: 2, Name: "Provider B"}},
	}

	// 1. 负载均衡器会选择 Provider 1 (权重 100)
	// 2. 但 Provider 1 不可用
	// 3. 故障转移到 Provider 2
	selected, err := executor.SelectProviderIntelligent(mappings)

	assert.NoError(t, err)
	assert.NotNil(t, selected)
	assert.Equal(t, uint(2), selected.ProviderID, "Should failover to Provider 2")
}

// TestFailoverScenario_HighConcurrency 测试高并发场景
func TestFailoverScenario_HighConcurrency(t *testing.T) {
	detectorConfig := &FailureDetectorConfig{
		FailureThreshold: 3,
		CooldownDuration: 5 * time.Minute,
		TimeoutThreshold: 30 * time.Second,
		CleanupInterval:  10 * time.Minute,
		MaxFailureHistory: 1000,
	}
	detector := NewFailureDetector(detectorConfig)

	balancer := NewWeightedRandomBalancer()

	failoverConfig := &FailoverConfig{
		MaxRetries:     3,
		EnableFailover: true,
	}
	executor := NewFailoverExecutor(balancer, detector, failoverConfig)

	mappings := []*mapping.ResolvedMapping{
		{ProviderID: 1, TargetModel: "model-a", Weight: 50, Priority: 1, Provider: &mapping.ProviderInfo{ID: 1, Name: "Provider A"}},
		{ProviderID: 2, TargetModel: "model-b", Weight: 50, Priority: 2, Provider: &mapping.ProviderInfo{ID: 2, Name: "Provider B"}},
	}

	// 使用 WaitGroup 等待所有 goroutine 完成
	var wg sync.WaitGroup
	concurrency := 50
	iterationsPerGoroutine := 100

	// 启动多个 goroutine 并发请求
	for i := 0; i < concurrency; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := 0; j < iterationsPerGoroutine; j++ {
				selected, err := executor.SelectProviderIntelligent(mappings)
				assert.NoError(t, err)
				assert.NotNil(t, selected)
			}
		}()
	}

	wg.Wait()

	// 验证负载均衡器统计正确
	stats := balancer.GetStats()
	expectedSelections := concurrency * iterationsPerGoroutine
	assert.Equal(t, int64(expectedSelections), stats.TotalSelections, "Total selections should match")
}

// TestFailoverScenario_CascadeFailure 测试级联故障场景
func TestFailoverScenario_CascadeFailure(t *testing.T) {
	detectorConfig := &FailureDetectorConfig{
		FailureThreshold: 3,
		CooldownDuration: 100 * time.Millisecond,
		TimeoutThreshold: 30 * time.Second,
		CleanupInterval:  10 * time.Minute,
		MaxFailureHistory: 100,
	}
	detector := NewFailureDetector(detectorConfig)

	balancer := NewWeightedRandomBalancer()

	failoverConfig := &FailoverConfig{
		MaxRetries:     3,
		EnableFailover: true,
	}
	executor := NewFailoverExecutor(balancer, detector, failoverConfig)

	mappings := []*mapping.ResolvedMapping{
		{ProviderID: 1, TargetModel: "model-a", Weight: 50, Priority: 1, Provider: &mapping.ProviderInfo{ID: 1, Name: "Provider A"}},
		{ProviderID: 2, TargetModel: "model-b", Weight: 50, Priority: 2, Provider: &mapping.ProviderInfo{ID: 2, Name: "Provider B"}},
		{ProviderID: 3, TargetModel: "model-c", Weight: 0, Priority: 3, Provider: &mapping.ProviderInfo{ID: 3, Name: "Provider C"}},
	}

	// 1. Provider 1 故障
	for i := 0; i < 3; i++ {
		detector.RecordFailure(1, ServerError)
	}
	assert.False(t, detector.IsAvailable(1))

	// 2. 第一次故障转移到 Provider 2
	result1, err := executor.SelectProviderWithFailover(mappings)
	assert.NoError(t, err)
	assert.Equal(t, uint(2), result1.SelectedProvider.ProviderID)

	// 3. Provider 2 也故障
	for i := 0; i < 3; i++ {
		detector.RecordFailure(2, ServerError)
	}
	assert.False(t, detector.IsAvailable(2))

	// 4. 第二次故障转移到 Provider 3
	result2, err := executor.SelectProviderWithFailover(mappings)
	assert.NoError(t, err)
	assert.Equal(t, uint(3), result2.SelectedProvider.ProviderID)

	// 5. Provider 3 也故障,所有供应商都不可用
	for i := 0; i < 3; i++ {
		detector.RecordFailure(3, ServerError)
	}
	assert.False(t, detector.IsAvailable(3))

	// 6. 所有供应商都故障,应该返回错误
	result3, err := executor.SelectProviderWithFailover(mappings)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "all providers unavailable")
	assert.Nil(t, result3.SelectedProvider)
	assert.Equal(t, 3, result3.AttemptCount)
}

// TestFailoverScenario_SuccessAfterFailure 测试故障后成功恢复
func TestFailoverScenario_SuccessAfterFailure(t *testing.T) {
	detectorConfig := &FailureDetectorConfig{
		FailureThreshold: 3,
		CooldownDuration: 5 * time.Minute,
		TimeoutThreshold: 30 * time.Second,
		CleanupInterval:  10 * time.Minute,
		MaxFailureHistory: 100,
	}
	detector := NewFailureDetector(detectorConfig)

	balancer := NewWeightedRandomBalancer()

	failoverConfig := &FailoverConfig{
		MaxRetries:     3,
		EnableFailover: true,
	}
	executor := NewFailoverExecutor(balancer, detector, failoverConfig)

	mappings := []*mapping.ResolvedMapping{
		{ProviderID: 1, TargetModel: "model-a", Weight: 50, Priority: 1, Provider: &mapping.ProviderInfo{ID: 1, Name: "Provider A"}},
		{ProviderID: 2, TargetModel: "model-b", Weight: 50, Priority: 2, Provider: &mapping.ProviderInfo{ID: 2, Name: "Provider B"}},
	}

	// 1. Provider 1 连续 2 次故障 (未达到阈值)
	detector.RecordFailure(1, TimeoutFailure)
	detector.RecordFailure(1, TimeoutFailure)

	// 验证 Provider 1 仍然可用
	assert.True(t, detector.IsAvailable(1), "Provider 1 should still be available (only 2 failures)")

	// 2. Provider 1 成功,计数器重置
	detector.RecordSuccess(1)

	// 3. 再次故障 2 次
	detector.RecordFailure(1, TimeoutFailure)
	detector.RecordFailure(1, TimeoutFailure)

	// 验证 Provider 1 仍然可用 (因为计数器被重置了)
	assert.True(t, detector.IsAvailable(1), "Provider 1 should be available after success reset")

	// 4. 验证执行器可以正常选择
	result, err := executor.SelectProviderWithFailover(mappings)
	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.NotNil(t, result.SelectedProvider)
}

// BenchmarkFailover_FirstProviderSuccess 基准测试 - 第一个供应商成功
func BenchmarkFailover_FirstProviderSuccess(b *testing.B) {
	detector := &mockFailureDetectorForFailover{}
	balancer := NewWeightedRandomBalancer()
	config := &FailoverConfig{
		MaxRetries:     3,
		EnableFailover: true,
	}
	executor := NewFailoverExecutor(balancer, detector, config)

	mappings := createTestMappingsForFailover([]int{1, 2, 3})

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = executor.SelectProviderWithFailover(mappings)
	}
}

// BenchmarkFailover_SecondProviderSuccess 基准测试 - 故障转移到第二个供应商
func BenchmarkFailover_SecondProviderSuccess(b *testing.B) {
	detector := &mockFailureDetectorForFailover{
		unavailableProviders: map[uint]bool{
			1: true,
		},
	}
	balancer := NewWeightedRandomBalancer()
	config := &FailoverConfig{
		MaxRetries:     3,
		EnableFailover: true,
	}
	executor := NewFailoverExecutor(balancer, detector, config)

	mappings := createTestMappingsForFailover([]int{1, 2, 3})

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = executor.SelectProviderWithFailover(mappings)
	}
}

// BenchmarkFailover_IntelligentSelection 基准测试 - 智能选择
func BenchmarkFailover_IntelligentSelection(b *testing.B) {
	detector := &mockFailureDetectorForFailover{}
	balancer := NewWeightedRandomBalancer()
	config := &FailoverConfig{
		MaxRetries:     3,
		EnableFailover: true,
	}
	executor := NewFailoverExecutor(balancer, detector, config)

	mappings := createTestMappingsForFailover([]int{1, 2, 3})

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = executor.SelectProviderIntelligent(mappings)
	}
}
