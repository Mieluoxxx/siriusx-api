package balancer

import (
	"math"
	"testing"
	"time"

	"github.com/Mieluoxxx/Siriusx-API/internal/mapping"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ==================== 辅助函数 ====================

// createTestMappings 创建测试用的映射列表
func createTestMappings(weights []int) []*mapping.ResolvedMapping {
	mappings := make([]*mapping.ResolvedMapping, len(weights))
	for i, weight := range weights {
		mappings[i] = &mapping.ResolvedMapping{
			ID:         uint(i + 1),
			ProviderID: uint(i + 1),
			Weight:     weight,
			Enabled:    true,
		}
	}
	return mappings
}

// calculateExpectedDistribution 计算期望分布
func calculateExpectedDistribution(weights []int) map[uint]float64 {
	totalWeight := 0
	for _, weight := range weights {
		totalWeight += weight
	}

	distribution := make(map[uint]float64)
	for i, weight := range weights {
		providerID := uint(i + 1)
		if totalWeight > 0 {
			distribution[providerID] = float64(weight) / float64(totalWeight) * 100
		} else {
			distribution[providerID] = 100.0 / float64(len(weights))
		}
	}
	return distribution
}

// ==================== 基础功能测试 ====================

func TestWeightedRandomBalancer_SelectProvider_EmptyList(t *testing.T) {
	balancer := NewWeightedRandomBalancer()

	// 测试空列表
	selected := balancer.SelectProvider(nil)
	assert.Nil(t, selected)

	selected = balancer.SelectProvider([]*mapping.ResolvedMapping{})
	assert.Nil(t, selected)
}

func TestWeightedRandomBalancer_SelectProvider_SingleProvider(t *testing.T) {
	balancer := NewWeightedRandomBalancer()
	mappings := createTestMappings([]int{50})

	// 单个供应商应该总是被选中
	for i := 0; i < 10; i++ {
		selected := balancer.SelectProvider(mappings)
		assert.NotNil(t, selected)
		assert.Equal(t, uint(1), selected.ProviderID)
	}
}

func TestWeightedRandomBalancer_SelectProvider_ZeroWeights(t *testing.T) {
	balancer := NewWeightedRandomBalancerWithSeed(42) // 固定种子确保可重复
	mappings := createTestMappings([]int{0, 0, 0})

	// 所有权重为0时，应该随机选择
	selections := make(map[uint]int)
	for i := 0; i < 300; i++ {
		selected := balancer.SelectProvider(mappings)
		assert.NotNil(t, selected)
		selections[selected.ProviderID]++
	}

	// 每个供应商都应该被选中
	assert.Equal(t, 3, len(selections))
	for providerID := uint(1); providerID <= 3; providerID++ {
		assert.Greater(t, selections[providerID], 0, "Provider %d should be selected", providerID)
	}
}

func TestWeightedRandomBalancer_SelectProvider_NormalWeights(t *testing.T) {
	balancer := NewWeightedRandomBalancerWithSeed(42)
	mappings := createTestMappings([]int{70, 20, 10}) // 7:2:1 的权重比例

	selected := balancer.SelectProvider(mappings)
	assert.NotNil(t, selected)
	assert.Contains(t, []uint{1, 2, 3}, selected.ProviderID)
}

func TestWeightedRandomBalancer_SelectProvider_MixedWeights(t *testing.T) {
	balancer := NewWeightedRandomBalancer()
	mappings := createTestMappings([]int{50, 0, 30, 20}) // 包含零权重

	selections := make(map[uint]int)
	for i := 0; i < 1000; i++ {
		selected := balancer.SelectProvider(mappings)
		assert.NotNil(t, selected)
		selections[selected.ProviderID]++
	}

	// 权重为0的供应商不应该被选中
	assert.Equal(t, 0, selections[2], "Provider with zero weight should not be selected")

	// 其他供应商都应该被选中
	assert.Greater(t, selections[1], 0)
	assert.Greater(t, selections[3], 0)
	assert.Greater(t, selections[4], 0)
}

// ==================== 权重分布验证测试 ====================

func TestWeightedRandomBalancer_WeightDistribution_Accuracy(t *testing.T) {
	balancer := NewWeightedRandomBalancerWithSeed(42)
	weights := []int{70, 20, 10} // 期望比例 70%, 20%, 10%
	mappings := createTestMappings(weights)

	// 运行足够多次以获得统计意义
	iterations := 10000
	selections := make(map[uint]int)

	for i := 0; i < iterations; i++ {
		selected := balancer.SelectProvider(mappings)
		require.NotNil(t, selected)
		selections[selected.ProviderID]++
	}

	// 计算实际分布
	actualDistribution := make(map[uint]float64)
	for providerID, count := range selections {
		actualDistribution[providerID] = float64(count) / float64(iterations) * 100
	}

	// 计算期望分布
	expectedDistribution := calculateExpectedDistribution(weights)

	// 验证分布误差 < 5%
	for providerID := uint(1); providerID <= 3; providerID++ {
		expected := expectedDistribution[providerID]
		actual := actualDistribution[providerID]
		errorRate := math.Abs(actual-expected) / expected * 100

		t.Logf("Provider %d: Expected %.2f%%, Actual %.2f%%, Error %.2f%%",
			providerID, expected, actual, errorRate)

		assert.Less(t, errorRate, 5.0,
			"Provider %d distribution error %.2f%% exceeds 5%% threshold",
			providerID, errorRate)
	}
}

func TestWeightedRandomBalancer_WeightDistribution_EqualWeights(t *testing.T) {
	balancer := NewWeightedRandomBalancerWithSeed(42)
	weights := []int{25, 25, 25, 25} // 均等权重
	mappings := createTestMappings(weights)

	iterations := 8000
	selections := make(map[uint]int)

	for i := 0; i < iterations; i++ {
		selected := balancer.SelectProvider(mappings)
		require.NotNil(t, selected)
		selections[selected.ProviderID]++
	}

	// 每个供应商应该获得约25%的请求
	expectedPercentage := 25.0
	for providerID := uint(1); providerID <= 4; providerID++ {
		actualPercentage := float64(selections[providerID]) / float64(iterations) * 100
		errorRate := math.Abs(actualPercentage-expectedPercentage) / expectedPercentage * 100

		t.Logf("Provider %d: Expected %.2f%%, Actual %.2f%%, Error %.2f%%",
			providerID, expectedPercentage, actualPercentage, errorRate)

		assert.Less(t, errorRate, 5.0,
			"Provider %d distribution error %.2f%% exceeds 5%% threshold",
			providerID, errorRate)
	}
}

// ==================== 统计功能测试 ====================

func TestWeightedRandomBalancer_Stats(t *testing.T) {
	balancer := NewWeightedRandomBalancer()
	mappings := createTestMappings([]int{50, 30, 20})

	// 初始统计应该为空
	stats := balancer.GetStats()
	assert.Equal(t, int64(0), stats.TotalSelections)
	assert.Equal(t, 0, len(stats.ProviderCounts))
	assert.True(t, stats.LastSelection.IsZero())

	// 执行一些选择
	for i := 0; i < 100; i++ {
		selected := balancer.SelectProvider(mappings)
		assert.NotNil(t, selected)
	}

	// 检查统计更新
	stats = balancer.GetStats()
	assert.Equal(t, int64(100), stats.TotalSelections)
	assert.Greater(t, len(stats.ProviderCounts), 0)
	assert.False(t, stats.LastSelection.IsZero())
	assert.Greater(t, stats.AverageTime, time.Duration(0))

	// 验证统计分布
	distribution := stats.GetDistribution()
	totalPercentage := 0.0
	for _, percentage := range distribution {
		totalPercentage += percentage
	}
	assert.InDelta(t, 100.0, totalPercentage, 0.01) // 总和应该接近100%
}

func TestWeightedRandomBalancer_Reset(t *testing.T) {
	balancer := NewWeightedRandomBalancer()
	mappings := createTestMappings([]int{50, 50})

	// 执行一些选择
	for i := 0; i < 10; i++ {
		balancer.SelectProvider(mappings)
	}

	// 验证统计不为空
	stats := balancer.GetStats()
	assert.Greater(t, stats.TotalSelections, int64(0))

	// 重置统计
	balancer.Reset()

	// 验证统计被重置
	stats = balancer.GetStats()
	assert.Equal(t, int64(0), stats.TotalSelections)
	assert.Equal(t, 0, len(stats.ProviderCounts))
	assert.True(t, stats.LastSelection.IsZero())
	assert.Equal(t, time.Duration(0), stats.AverageTime)
}

// ==================== 并发安全测试 ====================

func TestWeightedRandomBalancer_ConcurrentAccess(t *testing.T) {
	balancer := NewWeightedRandomBalancer()
	mappings := createTestMappings([]int{30, 30, 40})

	// 并发选择
	numGoroutines := 50
	selectionsPerGoroutine := 100
	done := make(chan bool, numGoroutines)

	for i := 0; i < numGoroutines; i++ {
		go func() {
			defer func() { done <- true }()
			for j := 0; j < selectionsPerGoroutine; j++ {
				selected := balancer.SelectProvider(mappings)
				assert.NotNil(t, selected)
			}
		}()
	}

	// 等待所有协程完成
	for i := 0; i < numGoroutines; i++ {
		select {
		case <-done:
		case <-time.After(5 * time.Second):
			t.Fatal("并发测试超时")
		}
	}

	// 验证统计一致性
	stats := balancer.GetStats()
	expectedTotal := int64(numGoroutines * selectionsPerGoroutine)
	assert.Equal(t, expectedTotal, stats.TotalSelections)

	// 验证所有计数的总和等于总选择次数
	var totalCounts int64
	for _, count := range stats.ProviderCounts {
		totalCounts += count
	}
	assert.Equal(t, expectedTotal, totalCounts)
}

// ==================== 工厂函数测试 ====================

func TestNewLoadBalancer(t *testing.T) {
	balancer := NewLoadBalancer(WeightedRandom)
	assert.NotNil(t, balancer)

	// 测试默认类型
	defaultBalancer := NewLoadBalancer("unknown")
	assert.NotNil(t, defaultBalancer)
}

func TestNewLoadBalancerWithSeed(t *testing.T) {
	seed := int64(12345)
	balancer1 := NewLoadBalancerWithSeed(WeightedRandom, seed)
	balancer2 := NewLoadBalancerWithSeed(WeightedRandom, seed)

	mappings := createTestMappings([]int{50, 50})

	// 相同种子应该产生相同的选择序列
	for i := 0; i < 10; i++ {
		selected1 := balancer1.SelectProvider(mappings)
		selected2 := balancer2.SelectProvider(mappings)
		assert.Equal(t, selected1.ProviderID, selected2.ProviderID,
			"Same seed should produce same selection at iteration %d", i)
	}
}

// ==================== 性能基准测试 ====================

func BenchmarkWeightedRandomBalancer_SelectProvider(b *testing.B) {
	balancer := NewWeightedRandomBalancer()
	mappings := createTestMappings([]int{40, 30, 20, 10})

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		selected := balancer.SelectProvider(mappings)
		if selected == nil {
			b.Fatal("Selection failed")
		}
	}
}

func BenchmarkWeightedRandomBalancer_SelectProvider_LargeList(b *testing.B) {
	balancer := NewWeightedRandomBalancer()

	// 创建大量供应商 (50个)
	weights := make([]int, 50)
	for i := range weights {
		weights[i] = i + 1 // 权重从1到50
	}
	mappings := createTestMappings(weights)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		selected := balancer.SelectProvider(mappings)
		if selected == nil {
			b.Fatal("Selection failed")
		}
	}
}

func BenchmarkWeightedRandomBalancer_ConcurrentSelect(b *testing.B) {
	balancer := NewWeightedRandomBalancer()
	mappings := createTestMappings([]int{40, 30, 20, 10})

	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			selected := balancer.SelectProvider(mappings)
			if selected == nil {
				b.Fatal("Selection failed")
			}
		}
	})
}

// ==================== 边界条件测试 ====================

func TestWeightedRandomBalancer_EdgeCases(t *testing.T) {
	balancer := NewWeightedRandomBalancer()

	// 测试极大权重
	mappings := createTestMappings([]int{1000000, 1})
	selected := balancer.SelectProvider(mappings)
	assert.NotNil(t, selected)

	// 测试单个非零权重
	mappings = createTestMappings([]int{0, 0, 100, 0})
	for i := 0; i < 10; i++ {
		selected = balancer.SelectProvider(mappings)
		assert.Equal(t, uint(3), selected.ProviderID)
	}

	// 测试负权重 (应该被忽略)
	mappings = []*mapping.ResolvedMapping{
		{ID: 1, ProviderID: 1, Weight: -10, Enabled: true},
		{ID: 2, ProviderID: 2, Weight: 50, Enabled: true},
	}

	for i := 0; i < 10; i++ {
		selected = balancer.SelectProvider(mappings)
		assert.Equal(t, uint(2), selected.ProviderID, "Only positive weight provider should be selected")
	}
}