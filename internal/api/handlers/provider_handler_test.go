package handlers

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/Mieluoxxx/Siriusx-API/internal/models"
	"github.com/Mieluoxxx/Siriusx-API/internal/provider"
	"github.com/gin-gonic/gin"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

// setupTestHandler 创建测试处理器和路由
func setupTestHandler(t *testing.T) (*gin.Engine, *gorm.DB) {
	// 设置 Gin 测试模式
	gin.SetMode(gin.TestMode)

	// 创建测试数据库
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("failed to create test database: %v", err)
	}

	// 自动迁移
	if err := db.AutoMigrate(&models.UnifiedModel{}, &models.Provider{}, &models.ModelMapping{}); err != nil {
		t.Fatalf("failed to migrate test database: %v", err)
	}

	// 创建处理器
	repo := provider.NewRepository(db)
	service := provider.NewService(repo)
	handler := NewProviderHandler(service)

	// 配置路由
	router := gin.New()
	api := router.Group("/api")
	{
		providers := api.Group("/providers")
		{
			providers.POST("", handler.CreateProvider)
			providers.GET("", handler.ListProviders)
			providers.GET("/:id", handler.GetProvider)
			providers.PUT("/:id", handler.UpdateProvider)
			providers.DELETE("/:id", handler.DeleteProvider)
		}
	}

	return router, db
}

// TestCreateProvider_Success 测试成功创建供应商
func TestCreateProvider_Success(t *testing.T) {
	router, _ := setupTestHandler(t)

	reqBody := provider.CreateProviderRequest{
		Name:      "Test Provider",
		BaseURL:   "https://api.test.com",
		APIKey:    "sk-test-key-12345",
		TestModel: "claude-sonnet-4",
	}
	body, _ := json.Marshal(reqBody)

	req, _ := http.NewRequest("POST", "/api/providers", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	resp := httptest.NewRecorder()

	router.ServeHTTP(resp, req)

	if resp.Code != http.StatusCreated {
		t.Errorf("Expected status 201, got %d", resp.Code)
	}

	var response provider.ProviderResponse
	json.Unmarshal(resp.Body.Bytes(), &response)

	if response.Name != reqBody.Name {
		t.Errorf("Expected name %s, got %s", reqBody.Name, response.Name)
	}

	// 验证 API Key 脱敏
	if response.APIKey == reqBody.APIKey {
		t.Error("API Key should be masked in response")
	}
	if response.APIKey != "sk-****2345" {
		t.Errorf("API Key masking incorrect, got %s", response.APIKey)
	}
}

// TestCreateProvider_InvalidJSON 测试无效的 JSON
func TestCreateProvider_InvalidJSON(t *testing.T) {
	router, _ := setupTestHandler(t)

	req, _ := http.NewRequest("POST", "/api/providers", bytes.NewBufferString("invalid json"))
	req.Header.Set("Content-Type", "application/json")
	resp := httptest.NewRecorder()

	router.ServeHTTP(resp, req)

	if resp.Code != http.StatusBadRequest {
		t.Errorf("Expected status 400, got %d", resp.Code)
	}
}

// TestCreateProvider_ValidationError 测试验证错误
func TestCreateProvider_ValidationError(t *testing.T) {
	router, _ := setupTestHandler(t)

	testCases := []struct {
		name    string
		reqBody provider.CreateProviderRequest
	}{
		{
			name: "empty name",
			reqBody: provider.CreateProviderRequest{
				Name:      "",
				BaseURL:   "https://api.test.com",
				APIKey:    "sk-test-key",
				TestModel: "claude-sonnet-4",
			},
		},
		{
			name: "invalid URL",
			reqBody: provider.CreateProviderRequest{
				Name:      "Test Provider",
				BaseURL:   "invalid-url",
				APIKey:    "sk-test-key",
				TestModel: "claude-sonnet-4",
			},
		},
		{
			name: "empty API key",
			reqBody: provider.CreateProviderRequest{
				Name:      "Test Provider",
				BaseURL:   "https://api.test.com",
				APIKey:    "",
				TestModel: "claude-sonnet-4",
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			body, _ := json.Marshal(tc.reqBody)
			req, _ := http.NewRequest("POST", "/api/providers", bytes.NewBuffer(body))
			req.Header.Set("Content-Type", "application/json")
			resp := httptest.NewRecorder()

			router.ServeHTTP(resp, req)

			if resp.Code != http.StatusBadRequest {
				t.Errorf("Expected status 400, got %d", resp.Code)
			}
		})
	}
}

// TestCreateProvider_DuplicateName 测试重复名称
func TestCreateProvider_DuplicateName(t *testing.T) {
	router, _ := setupTestHandler(t)

	reqBody := provider.CreateProviderRequest{
		Name:      "Test Provider",
		BaseURL:   "https://api.test.com",
		APIKey:    "sk-test-key",
		TestModel: "claude-sonnet-4",
	}
	body, _ := json.Marshal(reqBody)

	// 创建第一个供应商
	req1, _ := http.NewRequest("POST", "/api/providers", bytes.NewBuffer(body))
	req1.Header.Set("Content-Type", "application/json")
	resp1 := httptest.NewRecorder()
	router.ServeHTTP(resp1, req1)

	// 尝试创建同名供应商
	req2, _ := http.NewRequest("POST", "/api/providers", bytes.NewBuffer(body))
	req2.Header.Set("Content-Type", "application/json")
	resp2 := httptest.NewRecorder()
	router.ServeHTTP(resp2, req2)

	if resp2.Code != http.StatusConflict {
		t.Errorf("Expected status 409, got %d", resp2.Code)
	}
}

// TestGetProvider_Success 测试成功获取供应商
func TestGetProvider_Success(t *testing.T) {
	router, db := setupTestHandler(t)

	// 创建测试数据
	testProvider := &models.Provider{
		Name:      "Test Provider",
		BaseURL:   "https://api.test.com",
		APIKey:    "sk-test-key-12345",
		TestModel: "claude-sonnet-4",
	}
	db.Create(testProvider)

	req, _ := http.NewRequest("GET", "/api/providers/1", nil)
	resp := httptest.NewRecorder()

	router.ServeHTTP(resp, req)

	if resp.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", resp.Code)
	}

	var response provider.ProviderResponse
	json.Unmarshal(resp.Body.Bytes(), &response)

	if response.Name != testProvider.Name {
		t.Errorf("Expected name %s, got %s", testProvider.Name, response.Name)
	}

	// 验证 API Key 脱敏
	if response.APIKey != "sk-****2345" {
		t.Errorf("API Key masking incorrect, got %s", response.APIKey)
	}
}

// TestGetProvider_NotFound 测试获取不存在的供应商
func TestGetProvider_NotFound(t *testing.T) {
	router, _ := setupTestHandler(t)

	req, _ := http.NewRequest("GET", "/api/providers/9999", nil)
	resp := httptest.NewRecorder()

	router.ServeHTTP(resp, req)

	if resp.Code != http.StatusNotFound {
		t.Errorf("Expected status 404, got %d", resp.Code)
	}
}

// TestGetProvider_InvalidID 测试无效的 ID
func TestGetProvider_InvalidID(t *testing.T) {
	router, _ := setupTestHandler(t)

	req, _ := http.NewRequest("GET", "/api/providers/invalid", nil)
	resp := httptest.NewRecorder()

	router.ServeHTTP(resp, req)

	if resp.Code != http.StatusBadRequest {
		t.Errorf("Expected status 400, got %d", resp.Code)
	}
}

// TestListProviders_Success 测试成功获取供应商列表
func TestListProviders_Success(t *testing.T) {
	router, db := setupTestHandler(t)

	// 创建测试数据
	for i := 0; i < 15; i++ {
		p := &models.Provider{
			Name:    "Provider " + string(rune('A'+i)),
			BaseURL: "https://api.test.com",
			APIKey:  "sk-test-key",
		}
		db.Create(p)
	}

	req, _ := http.NewRequest("GET", "/api/providers?page=1&page_size=10", nil)
	resp := httptest.NewRecorder()

	router.ServeHTTP(resp, req)

	if resp.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", resp.Code)
	}

	var response provider.ProviderListResponse
	json.Unmarshal(resp.Body.Bytes(), &response)

	if response.Pagination.Total != 15 {
		t.Errorf("Expected total 15, got %d", response.Pagination.Total)
	}
	if len(response.Data) != 10 {
		t.Errorf("Expected 10 providers, got %d", len(response.Data))
	}
	if response.Pagination.TotalPages != 2 {
		t.Errorf("Expected 2 pages, got %d", response.Pagination.TotalPages)
	}
}

// TestListProviders_DefaultParams 测试默认分页参数
func TestListProviders_DefaultParams(t *testing.T) {
	router, db := setupTestHandler(t)

	// 创建测试数据
	for i := 0; i < 5; i++ {
		p := &models.Provider{
			Name:    "Provider " + string(rune('A'+i)),
			BaseURL: "https://api.test.com",
			APIKey:  "sk-test-key",
		}
		db.Create(p)
	}

	req, _ := http.NewRequest("GET", "/api/providers", nil)
	resp := httptest.NewRecorder()

	router.ServeHTTP(resp, req)

	if resp.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", resp.Code)
	}

	var response provider.ProviderListResponse
	json.Unmarshal(resp.Body.Bytes(), &response)

	if response.Pagination.Page != 1 {
		t.Errorf("Expected default page 1, got %d", response.Pagination.Page)
	}
	if response.Pagination.PageSize != 10 {
		t.Errorf("Expected default page_size 10, got %d", response.Pagination.PageSize)
	}
}

// TestUpdateProvider_Success 测试成功更新供应商
func TestUpdateProvider_Success(t *testing.T) {
	router, db := setupTestHandler(t)

	// 创建测试数据
	testProvider := &models.Provider{
		Name:    "Test Provider",
		BaseURL: "https://api.test.com",
		APIKey:  "sk-test-key",
	}
	db.Create(testProvider)

	newName := "Updated Provider"
	newURL := "https://api.updated.com"
	reqBody := provider.UpdateProviderRequest{
		Name:    &newName,
		BaseURL: &newURL,
	}
	body, _ := json.Marshal(reqBody)

	req, _ := http.NewRequest("PUT", "/api/providers/1", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	resp := httptest.NewRecorder()

	router.ServeHTTP(resp, req)

	if resp.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", resp.Code)
	}

	var response provider.ProviderResponse
	json.Unmarshal(resp.Body.Bytes(), &response)

	if response.Name != newName {
		t.Errorf("Expected name %s, got %s", newName, response.Name)
	}
	if response.BaseURL != newURL {
		t.Errorf("Expected base_url %s, got %s", newURL, response.BaseURL)
	}
}

// TestUpdateProvider_NotFound 测试更新不存在的供应商
func TestUpdateProvider_NotFound(t *testing.T) {
	router, _ := setupTestHandler(t)

	newName := "Updated Provider"
	reqBody := provider.UpdateProviderRequest{
		Name: &newName,
	}
	body, _ := json.Marshal(reqBody)

	req, _ := http.NewRequest("PUT", "/api/providers/9999", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	resp := httptest.NewRecorder()

	router.ServeHTTP(resp, req)

	if resp.Code != http.StatusNotFound {
		t.Errorf("Expected status 404, got %d", resp.Code)
	}
}

// TestUpdateProvider_InvalidID 测试无效的 ID
func TestUpdateProvider_InvalidID(t *testing.T) {
	router, _ := setupTestHandler(t)

	newName := "Updated Provider"
	reqBody := provider.UpdateProviderRequest{
		Name: &newName,
	}
	body, _ := json.Marshal(reqBody)

	req, _ := http.NewRequest("PUT", "/api/providers/invalid", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	resp := httptest.NewRecorder()

	router.ServeHTTP(resp, req)

	if resp.Code != http.StatusBadRequest {
		t.Errorf("Expected status 400, got %d", resp.Code)
	}
}

// TestDeleteProvider_Success 测试成功删除供应商
func TestDeleteProvider_Success(t *testing.T) {
	router, db := setupTestHandler(t)

	// 创建测试数据
	testProvider := &models.Provider{
		Name:      "Test Provider",
		BaseURL:   "https://api.test.com",
		APIKey:    "sk-test-key",
		TestModel: "claude-sonnet-4",
		Enabled:   true,
	}
	db.Create(testProvider)

	req, _ := http.NewRequest("DELETE", "/api/providers/1", nil)
	resp := httptest.NewRecorder()

	router.ServeHTTP(resp, req)

	if resp.Code != http.StatusNoContent {
		t.Errorf("Expected status 204, got %d", resp.Code)
	}

	// 验证软删除
	var deleted models.Provider
	db.First(&deleted, 1)
	if deleted.Enabled {
		t.Error("Provider should be disabled after soft delete")
	}
}

func TestDeleteProvider_WithMappings(t *testing.T) {
	router, db := setupTestHandler(t)

	provider := &models.Provider{
		Name:      "Provider With Mapping",
		BaseURL:   "https://api.mapping.com",
		APIKey:    "sk-mapping",
		TestModel: "claude-sonnet-4",
		Enabled:   true,
	}
	if err := db.Create(provider).Error; err != nil {
		t.Fatalf("failed to create provider: %v", err)
	}

	model := &models.UnifiedModel{
		Name:        "mapping-model",
		DisplayName: "mapping-model",
		Description: "test model",
	}
	if err := db.Create(model).Error; err != nil {
		t.Fatalf("failed to create unified model: %v", err)
	}

	mapping := &models.ModelMapping{
		UnifiedModelID: model.ID,
		ProviderID:     provider.ID,
		TargetModel:    "gpt-4o",
		Weight:         50,
		Priority:       1,
		Enabled:        true,
	}
	if err := db.Create(mapping).Error; err != nil {
		t.Fatalf("failed to create model mapping: %v", err)
	}

	req, _ := http.NewRequest(http.MethodDelete, fmt.Sprintf("/api/providers/%d", provider.ID), nil)
	resp := httptest.NewRecorder()
	router.ServeHTTP(resp, req)

	if resp.Code != http.StatusConflict {
		t.Fatalf("expected status 409, got %d", resp.Code)
	}
}

// TestDeleteProvider_NotFound 测试删除不存在的供应商
func TestDeleteProvider_NotFound(t *testing.T) {
	router, _ := setupTestHandler(t)

	req, _ := http.NewRequest("DELETE", "/api/providers/9999", nil)
	resp := httptest.NewRecorder()

	router.ServeHTTP(resp, req)

	if resp.Code != http.StatusNotFound {
		t.Errorf("Expected status 404, got %d", resp.Code)
	}
}

// TestDeleteProvider_InvalidID 测试无效的 ID
func TestDeleteProvider_InvalidID(t *testing.T) {
	router, _ := setupTestHandler(t)

	req, _ := http.NewRequest("DELETE", "/api/providers/invalid", nil)
	resp := httptest.NewRecorder()

	router.ServeHTTP(resp, req)

	if resp.Code != http.StatusBadRequest {
		t.Errorf("Expected status 400, got %d", resp.Code)
	}
}
