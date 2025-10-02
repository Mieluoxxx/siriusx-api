package stats

import (
	"sync"
	"sync/atomic"
	"time"
)

// RequestCounter 请求计数器
// 使用内存计数器 + 时间窗口滑动统计实现
type RequestCounter struct {
	totalRequests int64 // 总请求数（原子操作）

	// 时间窗口统计（用于 QPS 计算）
	windowMutex    sync.RWMutex
	currentWindow  *timeWindow
	previousWindow *timeWindow
	windowDuration time.Duration
}

// timeWindow 时间窗口
type timeWindow struct {
	count     int64
	startTime time.Time
}

// NewRequestCounter 创建请求计数器
func NewRequestCounter(windowDuration time.Duration) *RequestCounter {
	if windowDuration == 0 {
		windowDuration = 60 * time.Second // 默认 60 秒窗口
	}

	counter := &RequestCounter{
		windowDuration: windowDuration,
		currentWindow: &timeWindow{
			startTime: time.Now(),
		},
		previousWindow: &timeWindow{
			startTime: time.Now().Add(-windowDuration),
		},
	}

	// 启动后台协程，定期滚动时间窗口
	go counter.rotateWindows()

	return counter
}

// Increment 增加请求计数
func (rc *RequestCounter) Increment() {
	// 增加总计数（原子操作）
	atomic.AddInt64(&rc.totalRequests, 1)

	// 增加当前窗口计数
	rc.windowMutex.Lock()
	rc.currentWindow.count++
	rc.windowMutex.Unlock()
}

// GetTotal 获取总请求数
func (rc *RequestCounter) GetTotal() int64 {
	return atomic.LoadInt64(&rc.totalRequests)
}

// GetQPS 获取当前 QPS（每秒请求数）
// 基于滑动时间窗口计算
func (rc *RequestCounter) GetQPS() float64 {
	rc.windowMutex.RLock()
	defer rc.windowMutex.RUnlock()

	now := time.Now()

	// 计算当前窗口已经过去的时间
	currentElapsed := now.Sub(rc.currentWindow.startTime).Seconds()
	if currentElapsed == 0 {
		currentElapsed = 1 // 避免除零
	}

	// 当前窗口的 QPS
	currentQPS := float64(rc.currentWindow.count) / currentElapsed

	// 如果当前窗口时间很短，结合上一个窗口的数据
	if currentElapsed < rc.windowDuration.Seconds() {
		prevWeight := (rc.windowDuration.Seconds() - currentElapsed) / rc.windowDuration.Seconds()
		prevQPS := float64(rc.previousWindow.count) / rc.windowDuration.Seconds()

		// 加权平均
		return currentQPS*(1-prevWeight) + prevQPS*prevWeight
	}

	return currentQPS
}

// rotateWindows 定期滚动时间窗口
func (rc *RequestCounter) rotateWindows() {
	ticker := time.NewTicker(rc.windowDuration)
	defer ticker.Stop()

	for range ticker.C {
		rc.windowMutex.Lock()

		// 将当前窗口变为前一个窗口
		rc.previousWindow = rc.currentWindow

		// 创建新的当前窗口
		rc.currentWindow = &timeWindow{
			startTime: time.Now(),
			count:     0,
		}

		rc.windowMutex.Unlock()
	}
}

// GetStats 获取统计信息
func (rc *RequestCounter) GetStats() RequestStats {
	return RequestStats{
		Total:      rc.GetTotal(),
		CurrentQPS: rc.GetQPS(),
	}
}

// RequestStats 请求统计信息
type RequestStats struct {
	Total      int64   `json:"total"`
	CurrentQPS float64 `json:"current_qps"`
}
