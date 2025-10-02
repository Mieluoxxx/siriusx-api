package balancer

import (
	"math/rand"
	"sync"
	"time"

	"github.com/Mieluoxxx/Siriusx-API/internal/mapping"
)

// ==================== 接口定义 ====================

// LoadBalancer 负载均衡器接口
type LoadBalancer interface {
	// SelectProvider 从映射列表中选择一个供应商
	SelectProvider(mappings []*mapping.ResolvedMapping) *mapping.ResolvedMapping

	// GetStats 获取负载均衡统计信息
	GetStats() *BalancerStats

	// Reset 重置统计信息
	Reset()
}

// ==================== 统计信息 ====================

// BalancerStats 负载均衡统计信息
type BalancerStats struct {
	TotalSelections int64              `json:"total_selections"` // 总选择次数
	ProviderCounts  map[uint]int64     `json:"provider_counts"`  // 各供应商选择次数
	LastSelection   time.Time          `json:"last_selection"`   // 最后选择时间
	AverageTime     time.Duration      `json:"average_time"`     // 平均选择耗时
	mutex           sync.RWMutex       // 保护统计信息的并发安全
}

// GetProviderCount 获取指定供应商的选择次数
func (s *BalancerStats) GetProviderCount(providerID uint) int64 {
	s.mutex.RLock()
	defer s.mutex.RUnlock()
	return s.ProviderCounts[providerID]
}

// GetDistribution 获取权重分布百分比
func (s *BalancerStats) GetDistribution() map[uint]float64 {
	s.mutex.RLock()
	defer s.mutex.RUnlock()

	if s.TotalSelections == 0 {
		return make(map[uint]float64)
	}

	distribution := make(map[uint]float64)
	for providerID, count := range s.ProviderCounts {
		distribution[providerID] = float64(count) / float64(s.TotalSelections) * 100
	}
	return distribution
}

// ==================== 加权随机负载均衡器 ====================

// WeightedRandomBalancer 加权随机负载均衡器
type WeightedRandomBalancer struct {
	random *rand.Rand
	mutex  sync.RWMutex
	stats  *BalancerStats
}

// NewWeightedRandomBalancer 创建新的加权随机负载均衡器
func NewWeightedRandomBalancer() *WeightedRandomBalancer {
	return &WeightedRandomBalancer{
		random: rand.New(rand.NewSource(time.Now().UnixNano())),
		stats: &BalancerStats{
			ProviderCounts: make(map[uint]int64),
		},
	}
}

// NewWeightedRandomBalancerWithSeed 使用指定种子创建负载均衡器
func NewWeightedRandomBalancerWithSeed(seed int64) *WeightedRandomBalancer {
	return &WeightedRandomBalancer{
		random: rand.New(rand.NewSource(seed)),
		stats: &BalancerStats{
			ProviderCounts: make(map[uint]int64),
		},
	}
}

// SelectProvider 从映射列表中选择一个供应商
func (b *WeightedRandomBalancer) SelectProvider(mappings []*mapping.ResolvedMapping) *mapping.ResolvedMapping {
	start := time.Now()
	defer func() {
		// 更新平均选择时间
		duration := time.Since(start)
		b.updateAverageTime(duration)
	}()

	// 边界条件检查
	if len(mappings) == 0 {
		return nil
	}

	// 单个供应商直接返回
	if len(mappings) == 1 {
		b.updateStats(mappings[0])
		return mappings[0]
	}

	// 计算总权重
	totalWeight := b.calculateTotalWeight(mappings)

	var selected *mapping.ResolvedMapping

	if totalWeight == 0 {
		// 所有权重为0，随机选择
		selected = b.selectRandomly(mappings)
	} else {
		// 根据权重选择
		selected = b.selectByWeight(mappings, totalWeight)
	}

	// 更新统计信息
	b.updateStats(selected)

	return selected
}

// calculateTotalWeight 计算总权重
func (b *WeightedRandomBalancer) calculateTotalWeight(mappings []*mapping.ResolvedMapping) int {
	totalWeight := 0
	for _, mapping := range mappings {
		if mapping.Weight > 0 {
			totalWeight += mapping.Weight
		}
	}
	return totalWeight
}

// selectRandomly 随机选择 (当所有权重为0时)
func (b *WeightedRandomBalancer) selectRandomly(mappings []*mapping.ResolvedMapping) *mapping.ResolvedMapping {
	b.mutex.Lock()
	randomIndex := b.random.Intn(len(mappings))
	b.mutex.Unlock()
	return mappings[randomIndex]
}

// selectByWeight 根据权重选择
func (b *WeightedRandomBalancer) selectByWeight(mappings []*mapping.ResolvedMapping, totalWeight int) *mapping.ResolvedMapping {
	// 生成随机数 [0, totalWeight)
	b.mutex.Lock()
	randomValue := b.random.Intn(totalWeight)
	b.mutex.Unlock()

	// 累积权重选择
	currentWeight := 0
	for _, mapping := range mappings {
		if mapping.Weight > 0 {
			currentWeight += mapping.Weight
			if randomValue < currentWeight {
				return mapping
			}
		}
	}

	// 兜底返回最后一个有权重的供应商
	for i := len(mappings) - 1; i >= 0; i-- {
		if mappings[i].Weight > 0 {
			return mappings[i]
		}
	}

	// 如果所有权重都为0，返回第一个
	return mappings[0]
}

// updateStats 更新统计信息
func (b *WeightedRandomBalancer) updateStats(selected *mapping.ResolvedMapping) {
	b.stats.mutex.Lock()
	defer b.stats.mutex.Unlock()

	b.stats.TotalSelections++
	b.stats.ProviderCounts[selected.ProviderID]++
	b.stats.LastSelection = time.Now()
}

// updateAverageTime 更新平均选择时间
func (b *WeightedRandomBalancer) updateAverageTime(duration time.Duration) {
	b.stats.mutex.Lock()
	defer b.stats.mutex.Unlock()

	// 计算移动平均
	if b.stats.AverageTime == 0 {
		b.stats.AverageTime = duration
	} else {
		// 使用加权移动平均，新样本权重为0.1
		b.stats.AverageTime = time.Duration(float64(b.stats.AverageTime)*0.9 + float64(duration)*0.1)
	}
}

// GetStats 获取负载均衡统计信息
func (b *WeightedRandomBalancer) GetStats() *BalancerStats {
	b.stats.mutex.RLock()
	defer b.stats.mutex.RUnlock()

	// 返回统计信息的副本
	statsCopy := &BalancerStats{
		TotalSelections: b.stats.TotalSelections,
		ProviderCounts:  make(map[uint]int64),
		LastSelection:   b.stats.LastSelection,
		AverageTime:     b.stats.AverageTime,
	}

	// 深拷贝供应商计数
	for providerID, count := range b.stats.ProviderCounts {
		statsCopy.ProviderCounts[providerID] = count
	}

	return statsCopy
}

// Reset 重置统计信息
func (b *WeightedRandomBalancer) Reset() {
	b.stats.mutex.Lock()
	defer b.stats.mutex.Unlock()

	b.stats.TotalSelections = 0
	b.stats.ProviderCounts = make(map[uint]int64)
	b.stats.LastSelection = time.Time{}
	b.stats.AverageTime = 0
}

// ==================== 工厂函数 ====================

// BalancerType 负载均衡器类型
type BalancerType string

const (
	WeightedRandom BalancerType = "weighted_random"
	// 未来可扩展其他类型: RoundRobin, LeastConnections 等
)

// NewLoadBalancer 创建指定类型的负载均衡器
func NewLoadBalancer(balancerType BalancerType) LoadBalancer {
	switch balancerType {
	case WeightedRandom:
		return NewWeightedRandomBalancer()
	default:
		// 默认使用加权随机
		return NewWeightedRandomBalancer()
	}
}

// NewLoadBalancerWithSeed 使用种子创建负载均衡器 (主要用于测试)
func NewLoadBalancerWithSeed(balancerType BalancerType, seed int64) LoadBalancer {
	switch balancerType {
	case WeightedRandom:
		return NewWeightedRandomBalancerWithSeed(seed)
	default:
		return NewWeightedRandomBalancerWithSeed(seed)
	}
}