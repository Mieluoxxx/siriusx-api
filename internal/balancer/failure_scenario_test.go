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

// ==================== 故障场景模拟测试 ====================

// FailureScenario 故障场景定义
type FailureScenario struct {
	Name        string
	Err         error
	Resp        *http.Response
	IsFailure   bool
	FailureType FailureType
}

// createFailureScenarios 创建各种故障场景
func createFailureScenarios() []FailureScenario {
	return []FailureScenario{
		// 成功场景
		{
			Name:        "success_200",
			Err:         nil,
			Resp:        createHTTPResponse(200),
			IsFailure:   false,
			FailureType: "",
		},
		{
			Name:        "success_201",
			Err:         nil,
			Resp:        createHTTPResponse(201),
			IsFailure:   false,
			FailureType: "",
		},
		{
			Name:        "client_error_400",
			Err:         nil,
			Resp:        createHTTPResponse(400),
			IsFailure:   false,
			FailureType: "",
		},
		{
			Name:        "client_error_404",
			Err:         nil,
			Resp:        createHTTPResponse(404),
			IsFailure:   false,
			FailureType: "",
		},

		// 超时故障场景
		{
			Name:        "context_timeout",
			Err:         context.DeadlineExceeded,
			Resp:        nil,
			IsFailure:   true,
			FailureType: TimeoutFailure,
		},
		{
			Name:        "network_timeout",
			Err:         &net.OpError{Op: "read", Err: &timeoutError{}},
			Resp:        nil,
			IsFailure:   true,
			FailureType: TimeoutFailure,
		},
		{
			Name:        "timeout_message",
			Err:         errors.New("connection timed out"),
			Resp:        nil,
			IsFailure:   true,
			FailureType: TimeoutFailure,
		},

		// 连接故障场景
		{
			Name:        "connection_refused",
			Err:         &net.OpError{Op: "dial", Err: errors.New("connection refused")},
			Resp:        nil,
			IsFailure:   true,
			FailureType: ConnectionFailure,
		},
		{
			Name:        "dns_error",
			Err:         &net.DNSError{Err: "no such host", Name: "example.com"},
			Resp:        nil,
			IsFailure:   true,
			FailureType: ConnectionFailure,
		},
		{
			Name:        "network_unreachable",
			Err:         errors.New("network is unreachable"),
			Resp:        nil,
			IsFailure:   true,
			FailureType: ConnectionFailure,
		},
		{
			Name:        "broken_pipe",
			Err:         errors.New("broken pipe"),
			Resp:        nil,
			IsFailure:   true,
			FailureType: ConnectionFailure,
		},

		// 服务器错误场景
		{
			Name:        "internal_server_error",
			Err:         nil,
			Resp:        createHTTPResponse(500),
			IsFailure:   true,
			FailureType: ServerError,
		},
		{
			Name:        "bad_gateway",
			Err:         nil,
			Resp:        createHTTPResponse(502),
			IsFailure:   true,
			FailureType: ServerError,
		},
		{
			Name:        "service_unavailable",
			Err:         nil,
			Resp:        createHTTPResponse(503),
			IsFailure:   true,
			FailureType: ServerError,
		},
		{
			Name:        "gateway_timeout",
			Err:         nil,
			Resp:        createHTTPResponse(504),
			IsFailure:   true,
			FailureType: ServerError,
		},

		// 限流故障场景
		{
			Name:        "rate_limit",
			Err:         nil,
			Resp:        createHTTPResponse(429),
			IsFailure:   true,
			FailureType: RateLimitFailure,
		},

		// 未知错误场景
		{
			Name:        "unknown_error",
			Err:         errors.New("some unknown error"),
			Resp:        nil,
			IsFailure:   true,
			FailureType: UnknownFailure,
		},
	}
}

// TestFailureDetector_AccuracyValidation 验证故障检测准确率
func TestFailureDetector_AccuracyValidation(t *testing.T) {
	detector := createTestDetector()
	defer detector.Close()

	scenarios := createFailureScenarios()
	totalTests := len(scenarios)
	correctDetections := 0

	for _, scenario := range scenarios {
		t.Run(scenario.Name, func(t *testing.T) {
			// 测试故障检测
			isFailure := detector.IsFailure(scenario.Err, scenario.Resp)
			if isFailure == scenario.IsFailure {
				correctDetections++
			}

			assert.Equal(t, scenario.IsFailure, isFailure,
				"Scenario %s: expected IsFailure=%v, got %v",
				scenario.Name, scenario.IsFailure, isFailure)

			// 如果是故障，验证故障类型
			if scenario.IsFailure {
				failureType := detector.GetFailureType(scenario.Err, scenario.Resp)
				assert.Equal(t, scenario.FailureType, failureType,
					"Scenario %s: expected FailureType=%v, got %v",
					scenario.Name, scenario.FailureType, failureType)
			}
		})
	}

	// 计算准确率
	accuracy := float64(correctDetections) / float64(totalTests) * 100
	t.Logf("故障检测准确率: %.2f%% (%d/%d)", accuracy, correctDetections, totalTests)

	// 验证准确率 > 95%
	assert.Greater(t, accuracy, 95.0, "故障检测准确率应该大于95%%")
}

// TestFailureDetector_RealWorldScenarios 真实世界故障场景测试
func TestFailureDetector_RealWorldScenarios(t *testing.T) {
	detector := createTestDetector()
	defer detector.Close()

	// 模拟真实世界的故障模式
	realWorldScenarios := []struct {
		name     string
		pattern  []FailureScenario
		expected string // 期望的最终状态
	}{
		{
			name: "间歇性网络故障",
			pattern: []FailureScenario{
				{Err: nil, Resp: createHTTPResponse(200), IsFailure: false},
				{Err: errors.New("connection timeout"), Resp: nil, IsFailure: true},
				{Err: nil, Resp: createHTTPResponse(200), IsFailure: false},
				{Err: errors.New("connection timeout"), Resp: nil, IsFailure: true},
				{Err: nil, Resp: createHTTPResponse(200), IsFailure: false},
			},
			expected: "available", // 成功恢复
		},
		{
			name: "服务器连续故障",
			pattern: []FailureScenario{
				{Err: nil, Resp: createHTTPResponse(500), IsFailure: true},
				{Err: nil, Resp: createHTTPResponse(502), IsFailure: true},
				{Err: nil, Resp: createHTTPResponse(503), IsFailure: true},
			},
			expected: "cooldown", // 进入冷却期
		},
		{
			name: "限流后恢复",
			pattern: []FailureScenario{
				{Err: nil, Resp: createHTTPResponse(429), IsFailure: true},
				{Err: nil, Resp: createHTTPResponse(429), IsFailure: true},
				{Err: nil, Resp: createHTTPResponse(200), IsFailure: false},
				{Err: nil, Resp: createHTTPResponse(200), IsFailure: false},
			},
			expected: "available", // 恢复正常
		},
	}

	for _, scenario := range realWorldScenarios {
		t.Run(scenario.name, func(t *testing.T) {
			providerID := uint(1)
			detector.Reset(providerID) // 重置状态

			// 模拟故障模式
			for i, pattern := range scenario.pattern {
				if pattern.IsFailure {
					failureType := detector.GetFailureType(pattern.Err, pattern.Resp)
					detector.RecordFailure(providerID, failureType)
				} else {
					detector.RecordSuccess(providerID)
				}

				t.Logf("Step %d: IsFailure=%v, Available=%v",
					i+1, pattern.IsFailure, detector.IsAvailable(providerID))
			}

			// 验证最终状态
			stats := detector.GetFailureStats(providerID)
			isAvailable := detector.IsAvailable(providerID)

			switch scenario.expected {
			case "available":
				assert.True(t, isAvailable, "Provider should be available")
				assert.False(t, stats.IsInCooldown, "Provider should not be in cooldown")
			case "cooldown":
				assert.False(t, isAvailable, "Provider should be in cooldown")
				assert.True(t, stats.IsInCooldown, "Provider should be in cooldown state")
			}

			t.Logf("Final state - Available: %v, Cooldown: %v, ConsecutiveFailures: %d",
				isAvailable, stats.IsInCooldown, stats.ConsecutiveFailures)
		})
	}
}

// TestFailureDetector_PerformanceUnderLoad 高并发下的故障检测性能
func TestFailureDetector_PerformanceUnderLoad(t *testing.T) {
	detector := createTestDetector()
	defer detector.Close()

	numProviders := 100
	numOperationsPerProvider := 1000
	numGoroutines := 20

	start := time.Now()

	done := make(chan bool, numGoroutines)

	// 并发执行故障检测
	for i := 0; i < numGoroutines; i++ {
		go func(workerID int) {
			defer func() { done <- true }()

			scenarios := createFailureScenarios()
			for j := 0; j < numOperationsPerProvider; j++ {
				providerID := uint(workerID*numOperationsPerProvider + j%numProviders + 1)
				scenario := scenarios[j%len(scenarios)]

				// 执行故障检测
				isFailure := detector.IsFailure(scenario.Err, scenario.Resp)

				// 记录结果
				if isFailure {
					failureType := detector.GetFailureType(scenario.Err, scenario.Resp)
					detector.RecordFailure(providerID, failureType)
				} else {
					detector.RecordSuccess(providerID)
				}

				// 检查可用性
				detector.IsAvailable(providerID)
			}
		}(i)
	}

	// 等待所有协程完成
	for i := 0; i < numGoroutines; i++ {
		select {
		case <-done:
		case <-time.After(30 * time.Second):
			t.Fatal("性能测试超时")
		}
	}

	duration := time.Since(start)
	totalOperations := numGoroutines * numOperationsPerProvider
	operationsPerSecond := float64(totalOperations) / duration.Seconds()

	t.Logf("性能测试结果:")
	t.Logf("- 总操作数: %d", totalOperations)
	t.Logf("- 总耗时: %v", duration)
	t.Logf("- 操作/秒: %.2f", operationsPerSecond)
	t.Logf("- 平均延迟: %v", duration/time.Duration(totalOperations))

	// 验证性能要求 (至少1000 ops/sec)
	assert.Greater(t, operationsPerSecond, 1000.0, "故障检测性能应该 > 1000 ops/sec")

	// 验证最终状态一致性
	allStats := detector.GetAllStats()
	t.Logf("最终状态: %d个供应商有记录", len(allStats))

	for providerID, stats := range allStats {
		assert.GreaterOrEqual(t, stats.TotalRequests, int64(0))
		assert.GreaterOrEqual(t, stats.TotalFailures, int64(0))
		assert.LessOrEqual(t, stats.TotalFailures, stats.TotalRequests)

		if stats.TotalRequests > 0 {
			assert.GreaterOrEqual(t, stats.FailureRate, 0.0)
			assert.LessOrEqual(t, stats.FailureRate, 100.0)
		}

		// 记录一些统计信息
		if providerID <= 5 { // 只记录前5个供应商的详细信息
			t.Logf("Provider %d: Requests=%d, Failures=%d, FailureRate=%.2f%%, Cooldown=%v",
				providerID, stats.TotalRequests, stats.TotalFailures, stats.FailureRate, stats.IsInCooldown)
		}
	}
}

// TestFailureDetector_EdgeCasesValidation 边界条件验证
func TestFailureDetector_EdgeCasesValidation(t *testing.T) {
	detector := createTestDetector()
	defer detector.Close()

	// 测试各种边界条件
	edgeCases := []struct {
		name     string
		err      error
		resp     *http.Response
		expected bool
	}{
		{
			name:     "nil_error_nil_response",
			err:      nil,
			resp:     nil,
			expected: false,
		},
		{
			name:     "context_canceled",
			err:      context.Canceled,
			resp:     nil,
			expected: true, // 上下文取消也算故障
		},
		{
			name:     "empty_error_message",
			err:      errors.New(""),
			resp:     nil,
			expected: true, // 任何错误都算故障
		},
		{
			name:     "malformed_response",
			err:      nil,
			resp:     &http.Response{StatusCode: 0}, // 异常状态码
			expected: false,
		},
		{
			name:     "edge_status_codes",
			err:      nil,
			resp:     createHTTPResponse(499), // 边界状态码
			expected: false,
		},
		{
			name:     "edge_status_codes_599",
			err:      nil,
			resp:     createHTTPResponse(599), // 边界状态码
			expected: true, // 5xx错误
		},
	}

	correctCount := 0
	totalCount := len(edgeCases)

	for _, tc := range edgeCases {
		t.Run(tc.name, func(t *testing.T) {
			result := detector.IsFailure(tc.err, tc.resp)
			if result == tc.expected {
				correctCount++
			}
			assert.Equal(t, tc.expected, result, "边界条件 %s 检测结果不符合预期", tc.name)
		})
	}

	// 验证边界条件处理准确率
	accuracy := float64(correctCount) / float64(totalCount) * 100
	t.Logf("边界条件处理准确率: %.2f%% (%d/%d)", accuracy, correctCount, totalCount)
	assert.Equal(t, 100.0, accuracy, "边界条件处理应该100%%准确")
}

// TestFailureDetector_ComprehensiveAccuracy 综合准确率测试
func TestFailureDetector_ComprehensiveAccuracy(t *testing.T) {
	detector := createTestDetector()
	defer detector.Close()

	// 创建大量随机化的测试场景
	testCount := 1000
	correctDetections := 0

	// 定义各类场景的分布
	scenarios := createFailureScenarios()

	for i := 0; i < testCount; i++ {
		scenario := scenarios[i%len(scenarios)]

		// 执行检测
		result := detector.IsFailure(scenario.Err, scenario.Resp)

		if result == scenario.IsFailure {
			correctDetections++
		}

		// 详细记录前10个测试的结果
		if i < 10 {
			t.Logf("Test %d (%s): Expected=%v, Got=%v, Correct=%v",
				i+1, scenario.Name, scenario.IsFailure, result, result == scenario.IsFailure)
		}
	}

	// 计算总体准确率
	overallAccuracy := float64(correctDetections) / float64(testCount) * 100

	t.Logf("=== 综合准确率测试结果 ===")
	t.Logf("总测试数: %d", testCount)
	t.Logf("正确检测数: %d", correctDetections)
	t.Logf("综合准确率: %.2f%%", overallAccuracy)

	// 验证超过95%的准确率要求
	assert.Greater(t, overallAccuracy, 95.0,
		"故障检测综合准确率应该大于95%%, 实际为%.2f%%", overallAccuracy)

	// 如果达到99%以上，给予额外表扬
	if overallAccuracy >= 99.0 {
		t.Logf("🎉 优秀！故障检测准确率达到%.2f%%, 超越期望！", overallAccuracy)
	}
}

// ==================== 基准测试 ====================

// createBenchDetector 创建基准测试专用的检测器 (不启动后台任务)
func createBenchDetector() *DefaultFailureDetector {
	return &DefaultFailureDetector{
		failureStates: make(map[uint]*ProviderState),
		config: &FailureDetectorConfig{
			FailureThreshold:  3,
			CooldownDuration:  5 * time.Minute,
			TimeoutThreshold:  30 * time.Second,
			CleanupInterval:   1 * time.Hour,
			MaxFailureHistory: 1000,
		},
		stopCleanup: make(chan struct{}),
	}
}

func BenchmarkFailureDetector_IsFailure(b *testing.B) {
	detector := createBenchDetector()
	defer detector.Close()

	scenarios := createFailureScenarios()
	scenarioCount := len(scenarios)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		scenario := scenarios[i%scenarioCount]
		detector.IsFailure(scenario.Err, scenario.Resp)
	}
}

func BenchmarkFailureDetector_RecordFailure(b *testing.B) {
	detector := createBenchDetector()
	defer detector.Close()

	providerID := uint(1)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		detector.RecordFailure(providerID, TimeoutFailure)
	}
}

func BenchmarkFailureDetector_IsAvailable(b *testing.B) {
	detector := createBenchDetector()
	defer detector.Close()

	providerID := uint(1)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		detector.IsAvailable(providerID)
	}
}

func BenchmarkFailureDetector_ConcurrentOperations(b *testing.B) {
	detector := createBenchDetector()
	defer detector.Close()

	b.RunParallel(func(pb *testing.PB) {
		providerID := uint(1)
		scenarios := createFailureScenarios()
		i := 0

		for pb.Next() {
			scenario := scenarios[i%len(scenarios)]
			isFailure := detector.IsFailure(scenario.Err, scenario.Resp)

			if isFailure {
				failureType := detector.GetFailureType(scenario.Err, scenario.Resp)
				detector.RecordFailure(providerID, failureType)
			} else {
				detector.RecordSuccess(providerID)
			}

			detector.IsAvailable(providerID)
			i++
		}
	})
}