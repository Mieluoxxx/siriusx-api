package mapping

import (
	"sync"
	"time"
)

// ==================== 缓存接口 ====================

// Cache 缓存接口
type Cache interface {
	// Get 获取缓存数据
	Get(key string) ([]*ResolvedMapping, bool)

	// Set 设置缓存数据
	Set(key string, data []*ResolvedMapping)

	// Delete 删除指定缓存
	Delete(key string)

	// Clear 清空所有缓存
	Clear()

	// Stats 获取缓存统计
	Stats() *CacheStats

	// Cleanup 清理过期缓存
	Cleanup()
}

// ==================== 内存缓存实现 ====================

// MemoryCache 内存缓存实现
type MemoryCache struct {
	mu          sync.RWMutex
	data        map[string]*CacheEntry
	config      *CacheConfig
	stats       *cacheStats
	stopCleanup chan struct{}
}

// cacheStats 内部缓存统计
type cacheStats struct {
	hitCount  int64
	missCount int64
}

// NewMemoryCache 创建内存缓存
func NewMemoryCache(config *CacheConfig) *MemoryCache {
	if config == nil {
		config = &CacheConfig{
			TTL:         5 * time.Minute,
			MaxSize:     1000,
			CleanupTime: 10 * time.Minute,
		}
	}

	cache := &MemoryCache{
		data:        make(map[string]*CacheEntry),
		config:      config,
		stats:       &cacheStats{},
		stopCleanup: make(chan struct{}),
	}

	// 启动定期清理
	go cache.startCleanup()

	return cache
}

// Get 获取缓存数据
func (c *MemoryCache) Get(key string) ([]*ResolvedMapping, bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	entry, exists := c.data[key]
	if !exists {
		c.stats.missCount++
		return nil, false
	}

	// 检查是否过期
	if time.Now().After(entry.ExpiresAt) {
		c.stats.missCount++
		// 延迟删除，避免在读锁中执行写操作
		go func() {
			c.mu.Lock()
			delete(c.data, key)
			c.mu.Unlock()
		}()
		return nil, false
	}

	// 更新命中次数
	entry.HitCount++
	c.stats.hitCount++

	// 返回数据深拷贝
	result := make([]*ResolvedMapping, len(entry.Data))
	for i, mapping := range entry.Data {
		// 创建新的 ResolvedMapping 对象
		result[i] = &ResolvedMapping{
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

		// 如果有供应商信息，也要深拷贝
		if mapping.Provider != nil {
			result[i].Provider = &ProviderInfo{
				ID:           mapping.Provider.ID,
				Name:         mapping.Provider.Name,
				BaseURL:      mapping.Provider.BaseURL,
				Enabled:      mapping.Provider.Enabled,
				HealthStatus: mapping.Provider.HealthStatus,
			}
		}
	}

	return result, true
}

// Set 设置缓存数据
func (c *MemoryCache) Set(key string, data []*ResolvedMapping) {
	c.mu.Lock()
	defer c.mu.Unlock()

	// 检查缓存大小限制
	if len(c.data) >= c.config.MaxSize {
		c.evictOldest()
	}

	// 创建数据副本
	dataCopy := make([]*ResolvedMapping, len(data))
	copy(dataCopy, data)

	// 设置缓存条目
	c.data[key] = &CacheEntry{
		Data:      dataCopy,
		ExpiresAt: time.Now().Add(c.config.TTL),
		CreatedAt: time.Now(),
		HitCount:  0,
	}
}

// Delete 删除指定缓存
func (c *MemoryCache) Delete(key string) {
	c.mu.Lock()
	defer c.mu.Unlock()

	delete(c.data, key)
}

// Clear 清空所有缓存
func (c *MemoryCache) Clear() {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.data = make(map[string]*CacheEntry)
	c.stats = &cacheStats{}
}

// Stats 获取缓存统计
func (c *MemoryCache) Stats() *CacheStats {
	c.mu.RLock()
	defer c.mu.RUnlock()

	totalRequests := c.stats.hitCount + c.stats.missCount
	hitRate := 0.0
	if totalRequests > 0 {
		hitRate = float64(c.stats.hitCount) / float64(totalRequests)
	}

	// 计算内存使用量（粗略估算）
	memoryUsage := int64(len(c.data) * 1024) // 假设每个条目大约 1KB

	return &CacheStats{
		Size:        len(c.data),
		HitCount:    c.stats.hitCount,
		MissCount:   c.stats.missCount,
		HitRate:     hitRate,
		MemoryUsage: memoryUsage,
		LastCleanup: time.Now(), // 简化实现
		TTL:         c.config.TTL,
	}
}

// Cleanup 清理过期缓存
func (c *MemoryCache) Cleanup() {
	c.mu.Lock()
	defer c.mu.Unlock()

	now := time.Now()
	for key, entry := range c.data {
		if now.After(entry.ExpiresAt) {
			delete(c.data, key)
		}
	}
}

// Close 关闭缓存，停止清理协程
func (c *MemoryCache) Close() {
	close(c.stopCleanup)
}

// ==================== 私有方法 ====================

// startCleanup 启动定期清理
func (c *MemoryCache) startCleanup() {
	ticker := time.NewTicker(c.config.CleanupTime)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			c.Cleanup()
		case <-c.stopCleanup:
			return
		}
	}
}

// evictOldest 淘汰最老的缓存条目（简单的 LRU 实现）
func (c *MemoryCache) evictOldest() {
	if len(c.data) == 0 {
		return
	}

	var oldestKey string
	var oldestTime time.Time

	// 找到最老的条目
	for key, entry := range c.data {
		if oldestKey == "" || entry.CreatedAt.Before(oldestTime) {
			oldestKey = key
			oldestTime = entry.CreatedAt
		}
	}

	// 删除最老的条目
	if oldestKey != "" {
		delete(c.data, oldestKey)
	}
}

// ==================== 默认缓存配置 ====================

// DefaultCacheConfig 默认缓存配置
func DefaultCacheConfig() *CacheConfig {
	return &CacheConfig{
		TTL:         5 * time.Minute,
		MaxSize:     1000,
		CleanupTime: 10 * time.Minute,
	}
}