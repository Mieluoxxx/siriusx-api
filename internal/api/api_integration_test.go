package api_test

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strconv"
	"testing"

	"github.com/Mieluoxxx/Siriusx-API/internal/api"
	"github.com/Mieluoxxx/Siriusx-API/internal/db"
	"github.com/Mieluoxxx/Siriusx-API/internal/provider"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

// setupAPITestEnv 创建 API 集成测试环境
func setupAPITestEnv(t *testing.T) (*gin.Engine, *gorm.DB) {
	gin.SetMode(gin.TestMode)

	// 创建测试数据库
	database, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)

	// 自动迁移
	err = db.AutoMigrate(database)
	require.NoError(t, err)

	// 创建路由
	router := api.SetupRouter(database, nil)

	return router, database
}

// TestAPI_Stats 测试统计 API
func TestAPI_Stats(t *testing.T) {
	router, database := setupAPITestEnv(t)
	_ = database // 使用 database 创建测试数据

	// 创建测试供应商数据
	database.Exec("INSERT INTO providers (name, base_url, api_key, enabled, priority, health_status) VALUES (?, ?, ?, ?, ?, ?)",
		"Test Provider 1", "https://api.test1.com", "sk-test1", true, 80, "healthy")
	database.Exec("INSERT INTO providers (name, base_url, api_key, enabled, priority, health_status) VALUES (?, ?, ?, ?, ?, ?)",
		"Test Provider 2", "https://api.test2.com", "sk-test2", true, 70, "healthy")
	database.Exec("INSERT INTO providers (name, base_url, api_key, enabled, priority, health_status) VALUES (?, ?, ?, ?, ?, ?)",
		"Test Provider 3", "https://api.test3.com", "sk-test3", false, 60, "unhealthy")

	// 发送 GET /api/stats 请求
	req := httptest.NewRequest("GET", "/api/stats", nil)
	resp := httptest.NewRecorder()
	router.ServeHTTP(resp, req)

	// 验证响应
	assert.Equal(t, http.StatusOK, resp.Code)

	var stats map[string]interface{}
	err := json.Unmarshal(resp.Body.Bytes(), &stats)
	require.NoError(t, err)

	// 验证供应商统计
	providers := stats["providers"].(map[string]interface{})
	assert.Equal(t, float64(3), providers["total"])
	assert.Equal(t, float64(2), providers["healthy"])
	assert.Equal(t, float64(1), providers["unhealthy"])

	t.Log("✅ Stats API 返回正确的供应商统计")
}

// TestAPI_HealthCheck 测试健康检查 API
func TestAPI_HealthCheck(t *testing.T) {
	router, _ := setupAPITestEnv(t)

	// 创建测试供应商
	priority := 80
	createReq := provider.CreateProviderRequest{
		Name:     "Health Test Provider",
		BaseURL:  "https://api.health-test.com",
		APIKey:   "sk-health-test",
		Priority: &priority,
	}
	body, _ := json.Marshal(createReq)

	req := httptest.NewRequest("POST", "/api/providers", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	resp := httptest.NewRecorder()
	router.ServeHTTP(resp, req)

	var createResp provider.ProviderResponse
	json.Unmarshal(resp.Body.Bytes(), &createResp)
	providerID := createResp.ID

	// 发送健康检查请求
	req = httptest.NewRequest("POST", "/api/providers/"+strconv.Itoa(int(providerID))+"/health-check", nil)
	resp = httptest.NewRecorder()
	router.ServeHTTP(resp, req)

	// 验证响应
	assert.Equal(t, http.StatusOK, resp.Code)

	var healthResp map[string]interface{}
	err := json.Unmarshal(resp.Body.Bytes(), &healthResp)
	require.NoError(t, err)

	assert.Equal(t, float64(providerID), healthResp["provider_id"])
	assert.NotNil(t, healthResp["healthy"])
	assert.NotNil(t, healthResp["checked_at"])

	t.Log("✅ Health Check API 返回正确的健康状态")
}

// TestAPI_ToggleEnabled 测试启用/禁用 API
func TestAPI_ToggleEnabled(t *testing.T) {
	router, _ := setupAPITestEnv(t)

	// 创建测试供应商
	priority := 80
	createReq := provider.CreateProviderRequest{
		Name:     "Toggle Test Provider",
		BaseURL:  "https://api.toggle-test.com",
		APIKey:   "sk-toggle-test",
		Priority: &priority,
	}
	body, _ := json.Marshal(createReq)

	req := httptest.NewRequest("POST", "/api/providers", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	resp := httptest.NewRecorder()
	router.ServeHTTP(resp, req)

	var createResp provider.ProviderResponse
	json.Unmarshal(resp.Body.Bytes(), &createResp)
	providerID := createResp.ID

	// 初始状态应该是启用的
	assert.True(t, createResp.Enabled)

	// 禁用供应商
	toggleReq := map[string]bool{"enabled": false}
	body, _ = json.Marshal(toggleReq)

	req = httptest.NewRequest("PATCH", "/api/providers/"+strconv.Itoa(int(providerID))+"/enabled", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	resp = httptest.NewRecorder()
	router.ServeHTTP(resp, req)

	// 验证响应
	if resp.Code != http.StatusOK {
		t.Logf("Toggle enabled failed with status %d, body: %s", resp.Code, resp.Body.String())
	}
	assert.Equal(t, http.StatusOK, resp.Code)

	var toggleResp provider.ProviderResponse
	json.Unmarshal(resp.Body.Bytes(), &toggleResp)
	assert.False(t, toggleResp.Enabled)

	t.Log("✅ Toggle Enabled API 成功禁用供应商")

	// 重新启用供应商
	toggleReq = map[string]bool{"enabled": true}
	body, _ = json.Marshal(toggleReq)

	req = httptest.NewRequest("PATCH", "/api/providers/"+strconv.Itoa(int(providerID))+"/enabled", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	resp = httptest.NewRecorder()
	router.ServeHTTP(resp, req)

	assert.Equal(t, http.StatusOK, resp.Code)

	json.Unmarshal(resp.Body.Bytes(), &toggleResp)
	assert.True(t, toggleResp.Enabled)

	t.Log("✅ Toggle Enabled API 成功重新启用供应商")
}

// TestAPI_CORS 测试 CORS 配置
func TestAPI_CORS(t *testing.T) {
	router, _ := setupAPITestEnv(t)

	// 发送 OPTIONS 预检请求
	req := httptest.NewRequest("OPTIONS", "/api/stats", nil)
	req.Header.Set("Origin", "http://localhost:4321")
	req.Header.Set("Access-Control-Request-Method", "GET")
	resp := httptest.NewRecorder()
	router.ServeHTTP(resp, req)

	// 验证 CORS 头
	assert.Equal(t, "http://localhost:4321", resp.Header().Get("Access-Control-Allow-Origin"))
	assert.Contains(t, resp.Header().Get("Access-Control-Allow-Methods"), "GET")
	assert.Contains(t, resp.Header().Get("Access-Control-Allow-Headers"), "Content-Type")

	t.Log("✅ CORS 中间件配置正确")
}
