package balancer

import (
	"context"
	"errors"
	"net"
	"net/http"
	"strings"
	"sync"
	"time"
)

// ==================== 接口定义 ====================

// FailureDetector 故障检测器接口
type FailureDetector interface {
	// IsFailure 检测错误和响应是否表示故障
	IsFailure(err error, resp *http.Response) bool

	// RecordFailure 记录供应商故障
	RecordFailure(providerID uint, failureType FailureType)

	// RecordSuccess 记录供应商成功
	RecordSuccess(providerID uint)

	// IsAvailable 检查供应商是否可用 (不在冷却期)
	IsAvailable(providerID uint) bool

	// GetFailureStats 获取供应商故障统计
	GetFailureStats(providerID uint) *FailureStats

	// Reset 重置供应商状态
	Reset(providerID uint)

	// GetAllStats 获取所有供应商统计
	GetAllStats() map[uint]*FailureStats

	// Close 关闭故障检测器，清理资源
	Close()
}

// ==================== 类型定义 ====================

// FailureType 故障类型枚举
type FailureType string

const (
	TimeoutFailure    FailureType = "timeout"
	ConnectionFailure FailureType = "connection"
	ServerError       FailureType = "server_error"
	RateLimitFailure  FailureType = "rate_limit"
	UnknownFailure    FailureType = "unknown"
)

// ProviderState 供应商状态
type ProviderState struct {
	ProviderID          uint                  `json:"provider_id"`
	ConsecutiveFailures int                   `json:"consecutive_failures"`
	TotalFailures       int64                 `json:"total_failures"`
	TotalRequests       int64                 `json:"total_requests"`
	LastFailureTime     time.Time             `json:"last_failure_time"`
	LastSuccessTime     time.Time             `json:"last_success_time"`
	CooldownUntil       time.Time             `json:"cooldown_until"`
	IsInCooldown        bool                  `json:"is_in_cooldown"`
	FailureTypes        map[FailureType]int64 `json:"failure_types"`
	mutex               sync.RWMutex          // 状态保护锁
}

// FailureStats 故障统计信息
type FailureStats struct {
	ProviderID          uint                  `json:"provider_id"`
	ConsecutiveFailures int                   `json:"consecutive_failures"`
	TotalFailures       int64                 `json:"total_failures"`
	TotalRequests       int64                 `json:"total_requests"`
	FailureRate         float64               `json:"failure_rate"`
	LastFailureTime     time.Time             `json:"last_failure_time"`
	LastSuccessTime     time.Time             `json:"last_success_time"`
	CooldownUntil       time.Time             `json:"cooldown_until"`
	IsInCooldown        bool                  `json:"is_in_cooldown"`
	FailureTypes        map[FailureType]int64 `json:"failure_types"`
	TimeToRecovery      time.Duration         `json:"time_to_recovery"`
}

// FailureDetectorConfig 故障检测器配置
type FailureDetectorConfig struct {
	FailureThreshold  int           `yaml:"failure_threshold"`  // 故障阈值，默认3次
	CooldownDuration  time.Duration `yaml:"cooldown_duration"`  // 冷却时长，默认5分钟
	TimeoutThreshold  time.Duration `yaml:"timeout_threshold"`  // 超时阈值，默认30秒
	CleanupInterval   time.Duration `yaml:"cleanup_interval"`   // 清理间隔，默认1小时
	MaxFailureHistory int           `yaml:"max_failure_history"` // 最大故障历史记录数，默认1000
}

// ==================== 默认实现 ====================

// DefaultFailureDetector 默认故障检测器实现
type DefaultFailureDetector struct {
	failureStates map[uint]*ProviderState
	config        *FailureDetectorConfig
	mutex         sync.RWMutex
	stopCleanup   chan struct{}
}

// NewFailureDetector 创建新的故障检测器
func NewFailureDetector(config *FailureDetectorConfig) *DefaultFailureDetector {
	if config == nil {
		config = DefaultFailureDetectorConfig()
	}

	// 确保关键配置项有合法的默认值
	if config.FailureThreshold <= 0 {
		config.FailureThreshold = 3
	}
	if config.CooldownDuration <= 0 {
		config.CooldownDuration = 5 * time.Minute
	}
	if config.TimeoutThreshold <= 0 {
		config.TimeoutThreshold = 30 * time.Second
	}
	if config.CleanupInterval <= 0 {
		config.CleanupInterval = 1 * time.Hour
	}
	if config.MaxFailureHistory <= 0 {
		config.MaxFailureHistory = 1000
	}

	detector := &DefaultFailureDetector{
		failureStates: make(map[uint]*ProviderState),
		config:        config,
		stopCleanup:   make(chan struct{}),
	}

	// 启动后台清理任务
	go detector.startCleanup()

	return detector
}

// IsFailure 检测错误和响应是否表示故障
func (d *DefaultFailureDetector) IsFailure(err error, resp *http.Response) bool {
	// 1. 检查错误类型
	if err != nil {
		if d.isTimeoutError(err) {
			return true
		}
		if d.isConnectionError(err) {
			return true
		}
		// 其他错误也可能表示故障
		return true
	}

	// 2. 检查HTTP响应状态
	if resp != nil {
		switch {
		case resp.StatusCode >= 500 && resp.StatusCode < 600:
			// 5xx服务器错误
			return true
		case resp.StatusCode == 429:
			// 429 Too Many Requests (限流)
			return true
		}
	}

	return false
}

// GetFailureType 根据错误和响应确定故障类型
func (d *DefaultFailureDetector) GetFailureType(err error, resp *http.Response) FailureType {
	if err != nil {
		if d.isTimeoutError(err) {
			return TimeoutFailure
		}
		if d.isConnectionError(err) {
			return ConnectionFailure
		}
		return UnknownFailure
	}

	if resp != nil {
		switch {
		case resp.StatusCode >= 500 && resp.StatusCode < 600:
			return ServerError
		case resp.StatusCode == 429:
			return RateLimitFailure
		}
	}

	return UnknownFailure
}

// RecordFailure 记录供应商故障
func (d *DefaultFailureDetector) RecordFailure(providerID uint, failureType FailureType) {
	d.mutex.Lock()
	defer d.mutex.Unlock()

	state := d.getOrCreateProviderState(providerID)

	state.mutex.Lock()
	defer state.mutex.Unlock()

	// 更新故障统计
	state.ConsecutiveFailures++
	state.TotalFailures++
	state.TotalRequests++
	state.LastFailureTime = time.Now()

	// 统计故障类型
	if state.FailureTypes == nil {
		state.FailureTypes = make(map[FailureType]int64)
	}
	state.FailureTypes[failureType]++

	// 检查是否需要进入冷却期
	if state.ConsecutiveFailures >= d.config.FailureThreshold && !state.IsInCooldown {
		state.IsInCooldown = true
		state.CooldownUntil = time.Now().Add(d.config.CooldownDuration)
	}
}

// RecordSuccess 记录供应商成功
func (d *DefaultFailureDetector) RecordSuccess(providerID uint) {
	d.mutex.Lock()
	defer d.mutex.Unlock()

	state := d.getOrCreateProviderState(providerID)

	state.mutex.Lock()
	defer state.mutex.Unlock()

	// 重置连续故障计数
	state.ConsecutiveFailures = 0
	state.TotalRequests++
	state.LastSuccessTime = time.Now()

	// 如果在冷却期且冷却时间已过，则自动恢复
	if state.IsInCooldown && time.Now().After(state.CooldownUntil) {
		state.IsInCooldown = false
		state.CooldownUntil = time.Time{}
	}
}

// IsAvailable 检查供应商是否可用
func (d *DefaultFailureDetector) IsAvailable(providerID uint) bool {
	d.mutex.RLock()
	state, exists := d.failureStates[providerID]
	d.mutex.RUnlock()

	if !exists {
		// 新供应商默认可用
		return true
	}

	state.mutex.Lock()
	defer state.mutex.Unlock()

	// 检查是否在冷却期
	if state.IsInCooldown {
		// 检查冷却期是否已过
		if time.Now().After(state.CooldownUntil) {
			// 直接更新状态，避免异步竞争
			state.IsInCooldown = false
			state.CooldownUntil = time.Time{}
			return true
		}
		return false
	}

	return true
}

// GetFailureStats 获取供应商故障统计
func (d *DefaultFailureDetector) GetFailureStats(providerID uint) *FailureStats {
	d.mutex.RLock()
	state, exists := d.failureStates[providerID]
	d.mutex.RUnlock()

	if !exists {
		return &FailureStats{
			ProviderID:    providerID,
			FailureTypes:  make(map[FailureType]int64),
		}
	}

	state.mutex.RLock()
	defer state.mutex.RUnlock()

	// 计算故障率
	failureRate := 0.0
	if state.TotalRequests > 0 {
		failureRate = float64(state.TotalFailures) / float64(state.TotalRequests) * 100
	}

	// 计算恢复时间
	timeToRecovery := time.Duration(0)
	if state.IsInCooldown {
		remaining := time.Until(state.CooldownUntil)
		if remaining > 0 {
			timeToRecovery = remaining
		}
	}

	// 深拷贝故障类型统计
	failureTypes := make(map[FailureType]int64)
	for ft, count := range state.FailureTypes {
		failureTypes[ft] = count
	}

	return &FailureStats{
		ProviderID:          providerID,
		ConsecutiveFailures: state.ConsecutiveFailures,
		TotalFailures:       state.TotalFailures,
		TotalRequests:       state.TotalRequests,
		FailureRate:         failureRate,
		LastFailureTime:     state.LastFailureTime,
		LastSuccessTime:     state.LastSuccessTime,
		CooldownUntil:       state.CooldownUntil,
		IsInCooldown:        state.IsInCooldown,
		FailureTypes:        failureTypes,
		TimeToRecovery:      timeToRecovery,
	}
}

// Reset 重置供应商状态
func (d *DefaultFailureDetector) Reset(providerID uint) {
	d.mutex.Lock()
	defer d.mutex.Unlock()

	delete(d.failureStates, providerID)
}

// GetAllStats 获取所有供应商统计
func (d *DefaultFailureDetector) GetAllStats() map[uint]*FailureStats {
	d.mutex.RLock()
	providerIDs := make([]uint, 0, len(d.failureStates))
	for providerID := range d.failureStates {
		providerIDs = append(providerIDs, providerID)
	}
	d.mutex.RUnlock()

	stats := make(map[uint]*FailureStats)
	for _, providerID := range providerIDs {
		stats[providerID] = d.GetFailureStats(providerID)
	}

	return stats
}

// Close 关闭故障检测器
func (d *DefaultFailureDetector) Close() {
	close(d.stopCleanup)
}

// ==================== 私有方法 ====================

// getOrCreateProviderState 获取或创建供应商状态
func (d *DefaultFailureDetector) getOrCreateProviderState(providerID uint) *ProviderState {
	state, exists := d.failureStates[providerID]
	if !exists {
		state = &ProviderState{
			ProviderID:   providerID,
			FailureTypes: make(map[FailureType]int64),
		}
		d.failureStates[providerID] = state
	}
	return state
}

// isTimeoutError 检查是否为超时错误
func (d *DefaultFailureDetector) isTimeoutError(err error) bool {
	if err == nil {
		return false
	}

	// 检查context.DeadlineExceeded
	if errors.Is(err, context.DeadlineExceeded) {
		return true
	}

	// 检查网络超时
	var netErr net.Error
	if errors.As(err, &netErr) && netErr.Timeout() {
		return true
	}

	// 检查错误消息中的超时关键词
	errMsg := strings.ToLower(err.Error())
	timeoutKeywords := []string{"timeout", "deadline exceeded", "timed out"}
	for _, keyword := range timeoutKeywords {
		if strings.Contains(errMsg, keyword) {
			return true
		}
	}

	return false
}

// isConnectionError 检查是否为连接错误
func (d *DefaultFailureDetector) isConnectionError(err error) bool {
	if err == nil {
		return false
	}

	// 检查网络连接错误
	var netErr *net.OpError
	if errors.As(err, &netErr) {
		return true
	}

	// 检查DNS错误
	var dnsErr *net.DNSError
	if errors.As(err, &dnsErr) {
		return true
	}

	// 检查错误消息中的连接关键词
	errMsg := strings.ToLower(err.Error())
	connectionKeywords := []string{
		"connection refused", "connection reset", "connection aborted",
		"network is unreachable", "host is unreachable",
		"no route to host", "connection timeout",
		"broken pipe", "socket", "dial",
	}
	for _, keyword := range connectionKeywords {
		if strings.Contains(errMsg, keyword) {
			return true
		}
	}

	return false
}

// startCleanup 启动后台清理任务
func (d *DefaultFailureDetector) startCleanup() {
	ticker := time.NewTicker(d.config.CleanupInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			d.cleanup()
		case <-d.stopCleanup:
			return
		}
	}
}

// cleanup 清理过期状态
func (d *DefaultFailureDetector) cleanup() {
	d.mutex.Lock()
	defer d.mutex.Unlock()

	now := time.Now()
	expireThreshold := now.Add(-24 * time.Hour) // 清理24小时前的数据

	for providerID, state := range d.failureStates {
		state.mutex.RLock()
		shouldCleanup := false

		// 如果长时间没有活动且不在冷却期，则清理
		if !state.IsInCooldown &&
			(state.LastFailureTime.Before(expireThreshold) && state.LastSuccessTime.Before(expireThreshold)) {
			shouldCleanup = true
		}

		state.mutex.RUnlock()

		if shouldCleanup {
			delete(d.failureStates, providerID)
		}
	}
}

// ==================== 配置 ====================

// DefaultFailureDetectorConfig 默认故障检测配置
func DefaultFailureDetectorConfig() *FailureDetectorConfig {
	return &FailureDetectorConfig{
		FailureThreshold:  3,                // 连续3次故障触发冷却
		CooldownDuration:  5 * time.Minute, // 冷却5分钟
		TimeoutThreshold:  30 * time.Second, // 30秒超时
		CleanupInterval:   1 * time.Hour,    // 每小时清理一次
		MaxFailureHistory: 1000,             // 最多记录1000个供应商状态
	}
}

// ==================== 工具函数 ====================

// DetectFailureFromResponse 从HTTP响应检测故障并记录
func (d *DefaultFailureDetector) DetectFailureFromResponse(providerID uint, err error, resp *http.Response) bool {
	if d.IsFailure(err, resp) {
		failureType := d.GetFailureType(err, resp)
		d.RecordFailure(providerID, failureType)
		return true
	}

	d.RecordSuccess(providerID)
	return false
}