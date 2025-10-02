package provider

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestHealthChecker_CheckHealth_Success(t *testing.T) {
	// 创建模拟服务器
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// 验证请求头
		assert.Equal(t, "GET", r.Method)
		assert.NotEmpty(t, r.Header.Get("x-api-key"))

		// 返回成功响应
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"models": []}`))
	}))
	defer server.Close()

	// 创建健康检查器
	checker := NewHealthChecker(5 * time.Second)

	// 执行健康检查
	result, err := checker.CheckHealthSimple(server.URL, "test-api-key")
	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.True(t, result.Healthy)
	assert.Equal(t, http.StatusOK, result.StatusCode)
	assert.Greater(t, result.ResponseTimeMs, int64(0))
}

func TestHealthChecker_CheckHealth_Failure(t *testing.T) {
	// 创建返回错误的模拟服务器
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
	}))
	defer server.Close()

	checker := NewHealthChecker(5 * time.Second)
	result, err := checker.CheckHealthSimple(server.URL, "invalid-key")

	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.False(t, result.Healthy)
	assert.Equal(t, http.StatusUnauthorized, result.StatusCode)
}

func TestHealthChecker_CheckHealth_Timeout(t *testing.T) {
	// 创建慢响应的模拟服务器
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(2 * time.Second)
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	// 使用短超时
	checker := NewHealthChecker(500 * time.Millisecond)
	result, err := checker.CheckHealthSimple(server.URL, "test-key")

	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.False(t, result.Healthy)
	assert.NotEmpty(t, result.Error)
}

func TestHealthChecker_CheckHealth_WithContext(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	checker := NewHealthChecker(5 * time.Second)

	// 使用带超时的 context
	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
	defer cancel()

	result, err := checker.CheckHealth(ctx, server.URL, "test-key")
	assert.NoError(t, err)
	assert.True(t, result.Healthy)
}

func TestHealthChecker_CheckHealth_InvalidURL(t *testing.T) {
	checker := NewHealthChecker(5 * time.Second)
	result, err := checker.CheckHealthSimple("http://invalid-url-that-does-not-exist-12345.com", "test-key")

	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.False(t, result.Healthy)
	assert.NotEmpty(t, result.Error)
}
