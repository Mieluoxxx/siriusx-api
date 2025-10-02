package provider

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
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
		timeout = 15 * time.Second // 默认 15 秒超时
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
// 通过发送一个简单的聊天请求来测试指定模型是否可用
// 使用 OpenAI 兼容的 API 格式
func (hc *HealthChecker) CheckHealth(ctx context.Context, baseURL, apiKey, testModel string) (*HealthCheckResult, error) {
	startTime := time.Now()
	result := &HealthCheckResult{
		CheckedAt: startTime,
	}

	// 构建 OpenAI 兼容的聊天完成请求
	// 标准化 baseURL，移除末尾斜杠以避免双斜杠问题
	checkURL := strings.TrimRight(baseURL, "/") + "/v1/chat/completions"

	// 构建请求体
	requestBody := map[string]interface{}{
		"model": testModel,
		"messages": []map[string]string{
			{
				"role":    "user",
				"content": "Hi",
			},
		},
		"max_tokens": 1,
	}

	jsonData, err := json.Marshal(requestBody)
	if err != nil {
		result.Error = fmt.Sprintf("构建请求失败: %v", err)
		return result, nil
	}

	req, err := http.NewRequestWithContext(ctx, "POST", checkURL, bytes.NewBuffer(jsonData))
	if err != nil {
		result.Error = fmt.Sprintf("创建请求失败: %v", err)
		return result, nil
	}

	// 设置 OpenAI 兼容的认证头
	req.Header.Set("Authorization", "Bearer "+apiKey)
	req.Header.Set("Content-Type", "application/json")
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
func (hc *HealthChecker) CheckHealthSimple(baseURL, apiKey, testModel string) (*HealthCheckResult, error) {
	ctx, cancel := context.WithTimeout(context.Background(), hc.timeout)
	defer cancel()

	return hc.CheckHealth(ctx, baseURL, apiKey, testModel)
}
