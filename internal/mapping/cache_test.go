package mapping

import (
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestMemoryCache_SetAndGet(t *testing.T) {
	cache := NewMemoryCache(DefaultCacheConfig())
	defer cache.Close()

	// 测试数据
	key := "test-model"
	data := []*ResolvedMapping{
		{
			ID:          1,
			TargetModel: "gpt-4o",
			Weight:      70,
			Priority:    1,
			Enabled:     true,
		},
	}

	// 设置缓存
	cache.Set(key, data)

	// 获取缓存
	result, found := cache.Get(key)
	assert.True(t, found)
	assert.Len(t, result, 1)
	assert.Equal(t, "gpt-4o", result[0].TargetModel)
	assert.Equal(t, 70, result[0].Weight)

	// 验证是否为副本（修改返回的数据不应影响缓存）
	originalWeight := result[0].Weight
	result[0].Weight = 100
	result2, _ := cache.Get(key)
	assert.Equal(t, originalWeight, result2[0].Weight) // 原始数据应该未被修改
}

func TestMemoryCache_GetNotFound(t *testing.T) {
	cache := NewMemoryCache(DefaultCacheConfig())
	defer cache.Close()

	// 获取不存在的缓存
	result, found := cache.Get("non-existent")
	assert.False(t, found)
	assert.Nil(t, result)
}

func TestMemoryCache_TTLExpiration(t *testing.T) {
	config := &CacheConfig{
		TTL:         100 * time.Millisecond,
		MaxSize:     100,
		CleanupTime: 50 * time.Millisecond,
	}
	cache := NewMemoryCache(config)
	defer cache.Close()

	key := "test-model"
	data := []*ResolvedMapping{{ID: 1, TargetModel: "gpt-4o"}}

	// 设置缓存
	cache.Set(key, data)

	// 立即获取应该成功
	_, found := cache.Get(key)
	assert.True(t, found)

	// 等待过期
	time.Sleep(150 * time.Millisecond)

	// 过期后获取应该失败
	_, found = cache.Get(key)
	assert.False(t, found)
}

func TestMemoryCache_MaxSizeEviction(t *testing.T) {
	config := &CacheConfig{
		TTL:         time.Hour,
		MaxSize:     2,
		CleanupTime: time.Hour,
	}
	cache := NewMemoryCache(config)
	defer cache.Close()

	data := []*ResolvedMapping{{ID: 1, TargetModel: "gpt-4o"}}

	// 设置超过最大容量的缓存
	cache.Set("key1", data)
	cache.Set("key2", data)
	cache.Set("key3", data) // 这应该触发淘汰

	// 验证缓存大小不超过限制
	stats := cache.Stats()
	assert.LessOrEqual(t, stats.Size, 2)
}

func TestMemoryCache_Delete(t *testing.T) {
	cache := NewMemoryCache(DefaultCacheConfig())
	defer cache.Close()

	key := "test-model"
	data := []*ResolvedMapping{{ID: 1, TargetModel: "gpt-4o"}}

	// 设置缓存
	cache.Set(key, data)
	_, found := cache.Get(key)
	assert.True(t, found)

	// 删除缓存
	cache.Delete(key)
	_, found = cache.Get(key)
	assert.False(t, found)
}

func TestMemoryCache_Clear(t *testing.T) {
	cache := NewMemoryCache(DefaultCacheConfig())
	defer cache.Close()

	data := []*ResolvedMapping{{ID: 1, TargetModel: "gpt-4o"}}

	// 设置多个缓存
	cache.Set("key1", data)
	cache.Set("key2", data)
	cache.Set("key3", data)

	// 验证缓存存在
	assert.Equal(t, 3, cache.Stats().Size)

	// 清空缓存
	cache.Clear()

	// 验证缓存被清空
	stats := cache.Stats()
	assert.Equal(t, 0, stats.Size)
	assert.Equal(t, int64(0), stats.HitCount)
	assert.Equal(t, int64(0), stats.MissCount)
}

func TestMemoryCache_Stats(t *testing.T) {
	cache := NewMemoryCache(DefaultCacheConfig())
	defer cache.Close()

	key := "test-model"
	data := []*ResolvedMapping{{ID: 1, TargetModel: "gpt-4o"}}

	// 初始统计
	stats := cache.Stats()
	assert.Equal(t, 0, stats.Size)
	assert.Equal(t, int64(0), stats.HitCount)
	assert.Equal(t, int64(0), stats.MissCount)
	assert.Equal(t, 0.0, stats.HitRate)

	// 设置缓存
	cache.Set(key, data)

	// 命中缓存
	cache.Get(key)
	cache.Get(key)

	// 未命中缓存
	cache.Get("non-existent")

	// 验证统计
	stats = cache.Stats()
	assert.Equal(t, 1, stats.Size)
	assert.Equal(t, int64(2), stats.HitCount)
	assert.Equal(t, int64(1), stats.MissCount)
	assert.InDelta(t, 0.67, stats.HitRate, 0.01) // 2/3 ≈ 0.67
	assert.Greater(t, stats.MemoryUsage, int64(0))
}

func TestMemoryCache_ConcurrentAccess(t *testing.T) {
	cache := NewMemoryCache(DefaultCacheConfig())
	defer cache.Close()

	data := []*ResolvedMapping{{ID: 1, TargetModel: "gpt-4o"}}
	numRoutines := 10
	numOperations := 100

	// 并发读写测试
	done := make(chan bool, numRoutines*2)

	// 启动写协程
	for i := 0; i < numRoutines; i++ {
		go func(id int) {
			defer func() { done <- true }()
			for j := 0; j < numOperations; j++ {
				key := fmt.Sprintf("key-%d-%d", id, j)
				cache.Set(key, data)
			}
		}(i)
	}

	// 启动读协程
	for i := 0; i < numRoutines; i++ {
		go func(id int) {
			defer func() { done <- true }()
			for j := 0; j < numOperations; j++ {
				key := fmt.Sprintf("key-%d-%d", id, j)
				cache.Get(key)
			}
		}(i)
	}

	// 等待所有协程完成
	for i := 0; i < numRoutines*2; i++ {
		select {
		case <-done:
		case <-time.After(5 * time.Second):
			t.Fatal("并发测试超时")
		}
	}

	// 验证缓存状态正常
	stats := cache.Stats()
	assert.GreaterOrEqual(t, stats.Size, 0)
	assert.GreaterOrEqual(t, stats.HitCount, int64(0))
	assert.GreaterOrEqual(t, stats.MissCount, int64(0))
}

func TestMemoryCache_Cleanup(t *testing.T) {
	config := &CacheConfig{
		TTL:         50 * time.Millisecond,
		MaxSize:     100,
		CleanupTime: 30 * time.Millisecond,
	}
	cache := NewMemoryCache(config)
	defer cache.Close()

	data := []*ResolvedMapping{{ID: 1, TargetModel: "gpt-4o"}}

	// 设置缓存
	cache.Set("key1", data)
	cache.Set("key2", data)

	// 验证缓存存在
	assert.Equal(t, 2, cache.Stats().Size)

	// 等待自动清理
	time.Sleep(100 * time.Millisecond)

	// 手动触发清理
	cache.Cleanup()

	// 验证过期缓存被清理
	assert.Equal(t, 0, cache.Stats().Size)
}

func TestMemoryCache_DefaultConfig(t *testing.T) {
	config := DefaultCacheConfig()

	assert.Equal(t, 5*time.Minute, config.TTL)
	assert.Equal(t, 1000, config.MaxSize)
	assert.Equal(t, 10*time.Minute, config.CleanupTime)
}

func BenchmarkMemoryCache_Set(b *testing.B) {
	cache := NewMemoryCache(DefaultCacheConfig())
	defer cache.Close()

	data := []*ResolvedMapping{{ID: 1, TargetModel: "gpt-4o"}}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		key := fmt.Sprintf("key-%d", i)
		cache.Set(key, data)
	}
}

func BenchmarkMemoryCache_Get(b *testing.B) {
	cache := NewMemoryCache(DefaultCacheConfig())
	defer cache.Close()

	data := []*ResolvedMapping{{ID: 1, TargetModel: "gpt-4o"}}

	// 预设置一些数据
	for i := 0; i < 1000; i++ {
		key := fmt.Sprintf("key-%d", i)
		cache.Set(key, data)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		key := fmt.Sprintf("key-%d", i%1000)
		cache.Get(key)
	}
}

func BenchmarkMemoryCache_GetMiss(b *testing.B) {
	cache := NewMemoryCache(DefaultCacheConfig())
	defer cache.Close()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		key := fmt.Sprintf("non-existent-%d", i)
		cache.Get(key)
	}
}