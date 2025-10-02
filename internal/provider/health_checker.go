package provider

import (
	"context"
	"fmt"
	"net/http"
	"time"
)

// HealthChecker 供应商健康检查器
type HealthChecker struct {
	client  *http.Client
	timeout time.Duration
}

// NewHealthChecker 创建健康检查器
func NewHealthChecker(timeout time.Duration) *HealthChecker {
	if timeout == 0 {
		timeout = 5 * time.Second // 默认 5 秒超时
	}

	return &HealthChecker{
		client: &http.Client{
			Timeout: timeout,
		},
		timeout: timeout,
	}
}

// HealthCheckResult 健康检查结果
type HealthCheckResult struct {
	Healthy        bool          `json:"healthy"`
	ResponseTimeMs int64         `json:"response_time_ms"`
	StatusCode     int           `json:"status_code,omitempty"`
	Error          string        `json:"error,omitempty"`
	CheckedAt      time.Time     `json:"checked_at"`
}

// CheckHealth 执行健康检查
// 通过调用供应商的健康端点或测试端点来验证可用性
func (hc *HealthChecker) CheckHealth(ctx context.Context, baseURL, apiKey string) (*HealthCheckResult, error) {
	startTime := time.Now()
	result := &HealthCheckResult{
		CheckedAt: startTime,
	}

	// 构建健康检查请求
	// 对于 Claude API，我们可以尝试调用 /v1/models 端点来验证
	checkURL := baseURL + "/v1/models"

	req, err := http.NewRequestWithContext(ctx, "GET", checkURL, nil)
	if err != nil {
		result.Error = fmt.Sprintf("创建请求失败: %v", err)
		return result, nil
	}

	// 添加认证头
	req.Header.Set("x-api-key", apiKey)
	req.Header.Set("anthropic-version", "2023-06-01")
	req.Header.Set("User-Agent", "Siriusx-API/1.0")

	// 执行请求
	resp, err := hc.client.Do(req)
	if err != nil {
		result.Error = fmt.Sprintf("请求失败: %v", err)
		result.ResponseTimeMs = time.Since(startTime).Milliseconds()
		return result, nil
	}
	defer resp.Body.Close()

	// 计算响应时间
	result.ResponseTimeMs = time.Since(startTime).Milliseconds()
	result.StatusCode = resp.StatusCode

	// 判断健康状态
	// 2xx 状态码视为健康
	if resp.StatusCode >= 200 && resp.StatusCode < 300 {
		result.Healthy = true
	} else {
		result.Error = fmt.Sprintf("HTTP %d", resp.StatusCode)
	}

	return result, nil
}

// CheckHealthSimple 简化的健康检查（不需要 context）
func (hc *HealthChecker) CheckHealthSimple(baseURL, apiKey string) (*HealthCheckResult, error) {
	ctx, cancel := context.WithTimeout(context.Background(), hc.timeout)
	defer cancel()

	return hc.CheckHealth(ctx, baseURL, apiKey)
}
