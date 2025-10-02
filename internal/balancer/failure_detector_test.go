package balancer

import (
	"context"
	"errors"
	"net"
	"net/http"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

// ==================== 测试辅助函数 ====================

// createTestDetector 创建测试用的故障检测器
func createTestDetector() *DefaultFailureDetector {
	config := &FailureDetectorConfig{
		FailureThreshold:  3,
		CooldownDuration:  100 * time.Millisecond, // 短冷却期用于测试
		TimeoutThreshold:  30 * time.Second,
		CleanupInterval:   10 * time.Minute,       // 避免0值导致panic
		MaxFailureHistory: 100,
	}
	return NewFailureDetector(config)
}

// createHTTPResponse 创建测试用的HTTP响应
func createHTTPResponse(statusCode int) *http.Response {
	return &http.Response{
		StatusCode: statusCode,
		Header:     make(http.Header),
	}
}

// ==================== IsFailure 功能测试 ====================

func TestFailureDetector_IsFailure_TimeoutError(t *testing.T) {
	detector := createTestDetector()
	defer detector.Close()

	// 测试context.DeadlineExceeded
	err := context.DeadlineExceeded
	assert.True(t, detector.IsFailure(err, nil), "Should detect context.DeadlineExceeded as failure")

	// 测试网络超时错误
	timeoutErr := &net.OpError{
		Op:  "read",
		Err: &timeoutError{},
	}
	assert.True(t, detector.IsFailure(timeoutErr, nil), "Should detect network timeout as failure")

	// 测试超时错误消息
	timeoutMsgErr := errors.New("connection timeout")
	assert.True(t, detector.IsFailure(timeoutMsgErr, nil), "Should detect timeout message as failure")
}

// timeoutError 模拟超时错误
type timeoutError struct{}

func (e *timeoutError) Error() string   { return "timeout" }
func (e *timeoutError) Timeout() bool   { return true }
func (e *timeoutError) Temporary() bool { return false }

func TestFailureDetector_IsFailure_ConnectionError(t *testing.T) {
	detector := createTestDetector()
	defer detector.Close()

	// 测试连接拒绝错误
	connErr := &net.OpError{
		Op:  "dial",
		Err: errors.New("connection refused"),
	}
	assert.True(t, detector.IsFailure(connErr, nil), "Should detect connection error as failure")

	// 测试DNS错误
	dnsErr := &net.DNSError{
		Err:  "no such host",
		Name: "example.com",
	}
	assert.True(t, detector.IsFailure(dnsErr, nil), "Should detect DNS error as failure")

	// 测试各种连接错误消息
	connectionErrors := []string{
		"connection refused",
		"connection reset by peer",
		"network is unreachable",
		"no route to host",
		"broken pipe",
	}

	for _, errMsg := range connectionErrors {
		err := errors.New(errMsg)
		assert.True(t, detector.IsFailure(err, nil), "Should detect '%s' as failure", errMsg)
	}
}

func TestFailureDetector_IsFailure_HTTPErrors(t *testing.T) {
	detector := createTestDetector()
	defer detector.Close()

	// 测试5xx服务器错误
	serverErrors := []int{500, 501, 502, 503, 504, 505}
	for _, statusCode := range serverErrors {
		resp := createHTTPResponse(statusCode)
		assert.True(t, detector.IsFailure(nil, resp), "Should detect %d as failure", statusCode)
	}

	// 测试429限流错误
	resp429 := createHTTPResponse(429)
	assert.True(t, detector.IsFailure(nil, resp429), "Should detect 429 as failure")

	// 测试正常响应
	normalResponses := []int{200, 201, 204, 400, 401, 403, 404}
	for _, statusCode := range normalResponses {
		resp := createHTTPResponse(statusCode)
		assert.False(t, detector.IsFailure(nil, resp), "Should not detect %d as failure", statusCode)
	}
}

func TestFailureDetector_IsFailure_NoError(t *testing.T) {
	detector := createTestDetector()
	defer detector.Close()

	// 测试无错误和成功响应
	resp := createHTTPResponse(200)
	assert.False(t, detector.IsFailure(nil, resp), "Should not detect success as failure")

	// 测试无错误无响应
	assert.False(t, detector.IsFailure(nil, nil), "Should not detect nil as failure")
}

// ==================== 故障类型检测测试 ====================

func TestFailureDetector_GetFailureType(t *testing.T) {
	detector := createTestDetector()
	defer detector.Close()

	testCases := []struct {
		name         string
		err          error
		resp         *http.Response
		expectedType FailureType
	}{
		{
			name:         "timeout error",
			err:          context.DeadlineExceeded,
			resp:         nil,
			expectedType: TimeoutFailure,
		},
		{
			name:         "connection error",
			err:          &net.OpError{Op: "dial", Err: errors.New("connection refused")},
			resp:         nil,
			expectedType: ConnectionFailure,
		},
		{
			name:         "server error",
			err:          nil,
			resp:         createHTTPResponse(500),
			expectedType: ServerError,
		},
		{
			name:         "rate limit error",
			err:          nil,
			resp:         createHTTPResponse(429),
			expectedType: RateLimitFailure,
		},
		{
			name:         "unknown error",
			err:          errors.New("unknown error"),
			resp:         nil,
			expectedType: UnknownFailure,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			failureType := detector.GetFailureType(tc.err, tc.resp)
			assert.Equal(t, tc.expectedType, failureType)
		})
	}
}

// ==================== 故障记录和统计测试 ====================

func TestFailureDetector_RecordFailure(t *testing.T) {
	detector := createTestDetector()
	defer detector.Close()

	providerID := uint(1)

	// 初始状态检查
	stats := detector.GetFailureStats(providerID)
	assert.Equal(t, 0, stats.ConsecutiveFailures)
	assert.Equal(t, int64(0), stats.TotalFailures)
	assert.False(t, stats.IsInCooldown)

	// 记录第一次故障
	detector.RecordFailure(providerID, TimeoutFailure)
	stats = detector.GetFailureStats(providerID)
	assert.Equal(t, 1, stats.ConsecutiveFailures)
	assert.Equal(t, int64(1), stats.TotalFailures)
	assert.Equal(t, int64(1), stats.TotalRequests)
	assert.False(t, stats.IsInCooldown) // 未达到阈值

	// 记录第二次故障
	detector.RecordFailure(providerID, ServerError)
	stats = detector.GetFailureStats(providerID)
	assert.Equal(t, 2, stats.ConsecutiveFailures)
	assert.Equal(t, int64(2), stats.TotalFailures)
	assert.False(t, stats.IsInCooldown) // 未达到阈值

	// 记录第三次故障，触发冷却期
	detector.RecordFailure(providerID, ConnectionFailure)
	stats = detector.GetFailureStats(providerID)
	assert.Equal(t, 3, stats.ConsecutiveFailures)
	assert.Equal(t, int64(3), stats.TotalFailures)
	assert.True(t, stats.IsInCooldown) // 达到阈值，进入冷却期
	assert.True(t, stats.CooldownUntil.After(time.Now()))

	// 验证故障类型统计
	assert.Equal(t, int64(1), stats.FailureTypes[TimeoutFailure])
	assert.Equal(t, int64(1), stats.FailureTypes[ServerError])
	assert.Equal(t, int64(1), stats.FailureTypes[ConnectionFailure])
}

func TestFailureDetector_RecordSuccess(t *testing.T) {
	detector := createTestDetector()
	defer detector.Close()

	providerID := uint(1)

	// 先记录一些故障
	detector.RecordFailure(providerID, TimeoutFailure)
	detector.RecordFailure(providerID, ServerError)

	stats := detector.GetFailureStats(providerID)
	assert.Equal(t, 2, stats.ConsecutiveFailures)

	// 记录成功，应该重置连续故障计数
	detector.RecordSuccess(providerID)
	stats = detector.GetFailureStats(providerID)
	assert.Equal(t, 0, stats.ConsecutiveFailures) // 连续故障重置
	assert.Equal(t, int64(2), stats.TotalFailures) // 总故障数不变
	assert.Equal(t, int64(3), stats.TotalRequests) // 总请求数增加
	assert.False(t, stats.IsInCooldown)
}

func TestFailureDetector_IsAvailable(t *testing.T) {
	detector := createTestDetector()
	defer detector.Close()

	providerID := uint(1)

	// 新供应商默认可用
	assert.True(t, detector.IsAvailable(providerID))

	// 记录故障直到进入冷却期
	detector.RecordFailure(providerID, TimeoutFailure)
	detector.RecordFailure(providerID, TimeoutFailure)
	detector.RecordFailure(providerID, TimeoutFailure) // 第3次故障，进入冷却期

	// 验证进入冷却期
	stats := detector.GetFailureStats(providerID)
	assert.True(t, stats.IsInCooldown, "Should be in cooldown after 3 failures")

	// 应该不可用
	assert.False(t, detector.IsAvailable(providerID))

	// 等待冷却期结束
	time.Sleep(150 * time.Millisecond) // 等待超过冷却期

	// 应该自动恢复可用
	assert.True(t, detector.IsAvailable(providerID))

	// 验证状态已更新
	stats = detector.GetFailureStats(providerID)
	assert.False(t, stats.IsInCooldown)
}

// ==================== 冷却期和恢复机制测试 ====================

func TestFailureDetector_CooldownMechanism(t *testing.T) {
	detector := createTestDetector()
	defer detector.Close()

	providerID := uint(1)

	// 记录3次故障触发冷却期
	for i := 0; i < 3; i++ {
		detector.RecordFailure(providerID, TimeoutFailure)
	}

	stats := detector.GetFailureStats(providerID)
	assert.True(t, stats.IsInCooldown)
	assert.True(t, stats.TimeToRecovery > 0)

	// 在冷却期内应该不可用
	assert.False(t, detector.IsAvailable(providerID))

	// 等待冷却期结束
	time.Sleep(150 * time.Millisecond)

	// 冷却期后应该自动恢复
	assert.True(t, detector.IsAvailable(providerID))

	// 记录成功后应该正常工作
	detector.RecordSuccess(providerID)
	stats = detector.GetFailureStats(providerID)
	assert.Equal(t, 0, stats.ConsecutiveFailures)
	assert.False(t, stats.IsInCooldown)
}

func TestFailureDetector_CooldownRecoveryBySuccess(t *testing.T) {
	detector := createTestDetector()
	defer detector.Close()

	providerID := uint(1)

	// 记录3次故障触发冷却期
	for i := 0; i < 3; i++ {
		detector.RecordFailure(providerID, TimeoutFailure)
	}

	assert.True(t, detector.GetFailureStats(providerID).IsInCooldown)

	// 等待冷却期结束
	time.Sleep(150 * time.Millisecond)

	// 记录成功应该自动清除冷却状态
	detector.RecordSuccess(providerID)
	stats := detector.GetFailureStats(providerID)
	assert.False(t, stats.IsInCooldown)
	assert.Equal(t, 0, stats.ConsecutiveFailures)
}

// ==================== 统计功能测试 ====================

func TestFailureDetector_FailureStats(t *testing.T) {
	detector := createTestDetector()
	defer detector.Close()

	providerID := uint(1)

	// 记录多种类型的故障
	detector.RecordFailure(providerID, TimeoutFailure)
	detector.RecordFailure(providerID, TimeoutFailure)
	detector.RecordFailure(providerID, ServerError)
	detector.RecordSuccess(providerID)
	detector.RecordSuccess(providerID)

	stats := detector.GetFailureStats(providerID)

	// 验证基本统计
	assert.Equal(t, providerID, stats.ProviderID)
	assert.Equal(t, int64(3), stats.TotalFailures)
	assert.Equal(t, int64(5), stats.TotalRequests)
	assert.InDelta(t, 60.0, stats.FailureRate, 0.1) // 3/5 = 60%

	// 验证故障类型统计
	assert.Equal(t, int64(2), stats.FailureTypes[TimeoutFailure])
	assert.Equal(t, int64(1), stats.FailureTypes[ServerError])

	// 验证时间戳
	assert.False(t, stats.LastFailureTime.IsZero())
	assert.False(t, stats.LastSuccessTime.IsZero())
	assert.True(t, stats.LastSuccessTime.After(stats.LastFailureTime))
}

func TestFailureDetector_GetAllStats(t *testing.T) {
	detector := createTestDetector()
	defer detector.Close()

	// 为多个供应商记录故障
	detector.RecordFailure(1, TimeoutFailure)
	detector.RecordFailure(2, ServerError)
	detector.RecordSuccess(3)

	allStats := detector.GetAllStats()

	assert.Len(t, allStats, 3)
	assert.Contains(t, allStats, uint(1))
	assert.Contains(t, allStats, uint(2))
	assert.Contains(t, allStats, uint(3))

	// 验证各供应商的统计
	assert.Equal(t, int64(1), allStats[1].TotalFailures)
	assert.Equal(t, int64(1), allStats[2].TotalFailures)
	assert.Equal(t, int64(0), allStats[3].TotalFailures)
}

// ==================== 并发安全测试 ====================

func TestFailureDetector_ConcurrentAccess(t *testing.T) {
	detector := createTestDetector()
	defer detector.Close()

	providerID := uint(1)
	numGoroutines := 50
	operationsPerGoroutine := 100

	done := make(chan bool, numGoroutines*2)

	// 并发记录故障
	for i := 0; i < numGoroutines; i++ {
		go func() {
			defer func() { done <- true }()
			for j := 0; j < operationsPerGoroutine; j++ {
				detector.RecordFailure(providerID, TimeoutFailure)
			}
		}()
	}

	// 并发记录成功
	for i := 0; i < numGoroutines; i++ {
		go func() {
			defer func() { done <- true }()
			for j := 0; j < operationsPerGoroutine; j++ {
				detector.RecordSuccess(providerID)
			}
		}()
	}

	// 等待所有协程完成
	for i := 0; i < numGoroutines*2; i++ {
		select {
		case <-done:
		case <-time.After(10 * time.Second):
			t.Fatal("并发测试超时")
		}
	}

	// 验证最终状态一致性
	stats := detector.GetFailureStats(providerID)
	expectedTotal := int64(numGoroutines * operationsPerGoroutine * 2)
	assert.Equal(t, expectedTotal, stats.TotalRequests)
	assert.Equal(t, int64(numGoroutines*operationsPerGoroutine), stats.TotalFailures)
}

// ==================== 边界条件和错误处理测试 ====================

func TestFailureDetector_Reset(t *testing.T) {
	detector := createTestDetector()
	defer detector.Close()

	providerID := uint(1)

	// 记录一些故障
	detector.RecordFailure(providerID, TimeoutFailure)
	detector.RecordFailure(providerID, ServerError)

	stats := detector.GetFailureStats(providerID)
	assert.Greater(t, stats.TotalFailures, int64(0))

	// 重置状态
	detector.Reset(providerID)

	// 验证状态已重置
	stats = detector.GetFailureStats(providerID)
	assert.Equal(t, int64(0), stats.TotalFailures)
	assert.Equal(t, int64(0), stats.TotalRequests)
	assert.Equal(t, 0, len(stats.FailureTypes))
}

func TestFailureDetector_NonExistentProvider(t *testing.T) {
	detector := createTestDetector()
	defer detector.Close()

	nonExistentID := uint(999)

	// 查询不存在的供应商应该返回默认值
	assert.True(t, detector.IsAvailable(nonExistentID))

	stats := detector.GetFailureStats(nonExistentID)
	assert.Equal(t, nonExistentID, stats.ProviderID)
	assert.Equal(t, int64(0), stats.TotalFailures)
	assert.Equal(t, int64(0), stats.TotalRequests)
	assert.False(t, stats.IsInCooldown)
}

// ==================== 集成测试 ====================

func TestFailureDetector_DetectFailureFromResponse(t *testing.T) {
	detector := createTestDetector()
	defer detector.Close()

	providerID := uint(1)

	testCases := []struct {
		name        string
		err         error
		resp        *http.Response
		expectFailure bool
	}{
		{
			name:          "success response",
			err:           nil,
			resp:          createHTTPResponse(200),
			expectFailure: false,
		},
		{
			name:          "server error response",
			err:           nil,
			resp:          createHTTPResponse(500),
			expectFailure: true,
		},
		{
			name:          "timeout error",
			err:           context.DeadlineExceeded,
			resp:          nil,
			expectFailure: true,
		},
		{
			name:          "rate limit response",
			err:           nil,
			resp:          createHTTPResponse(429),
			expectFailure: true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			initialStats := detector.GetFailureStats(providerID)
			initialFailures := initialStats.TotalFailures

			isFailure := detector.DetectFailureFromResponse(providerID, tc.err, tc.resp)
			assert.Equal(t, tc.expectFailure, isFailure)

			newStats := detector.GetFailureStats(providerID)
			if tc.expectFailure {
				assert.Greater(t, newStats.TotalFailures, initialFailures)
			} else {
				assert.Equal(t, initialFailures, newStats.TotalFailures)
			}
		})
	}
}

// ==================== 配置测试 ====================

func TestDefaultFailureDetectorConfig(t *testing.T) {
	config := DefaultFailureDetectorConfig()

	assert.Equal(t, 3, config.FailureThreshold)
	assert.Equal(t, 5*time.Minute, config.CooldownDuration)
	assert.Equal(t, 30*time.Second, config.TimeoutThreshold)
	assert.Equal(t, 1*time.Hour, config.CleanupInterval)
	assert.Equal(t, 1000, config.MaxFailureHistory)
}

func TestFailureDetector_CustomConfig(t *testing.T) {
	config := &FailureDetectorConfig{
		FailureThreshold: 5,           // 自定义阈值
		CooldownDuration: 10 * time.Minute, // 自定义冷却期
	}

	detector := NewFailureDetector(config)
	defer detector.Close()

	providerID := uint(1)

	// 记录4次故障，不应触发冷却期
	for i := 0; i < 4; i++ {
		detector.RecordFailure(providerID, TimeoutFailure)
	}

	stats := detector.GetFailureStats(providerID)
	assert.False(t, stats.IsInCooldown, "Should not be in cooldown with custom threshold")

	// 记录第5次故障，应该触发冷却期
	detector.RecordFailure(providerID, TimeoutFailure)
	stats = detector.GetFailureStats(providerID)
	assert.True(t, stats.IsInCooldown, "Should be in cooldown after reaching custom threshold")
}