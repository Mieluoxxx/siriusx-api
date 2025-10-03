package provider

import (
	"fmt"
	"testing"

	"github.com/Mieluoxxx/Siriusx-API/internal/models"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

// setupTestService 创建测试服务
func setupTestService(t *testing.T) *Service {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("failed to create test database: %v", err)
	}

	// 自动迁移
	if err := db.AutoMigrate(&models.Provider{}); err != nil {
		t.Fatalf("failed to migrate test database: %v", err)
	}

	repo := NewRepository(db)
	return NewService(repo)
}

// TestService_CreateProvider_Success 测试成功创建供应商
func TestService_CreateProvider_Success(t *testing.T) {
	service := setupTestService(t)

	req := CreateProviderRequest{
		Name:      "Test Provider",
		BaseURL:   "https://api.test.com",
		APIKey:    "sk-test-key",
		TestModel: "gpt-3.5-turbo",
	}

	provider, err := service.CreateProvider(req)
	if err != nil {
		t.Errorf("CreateProvider() failed: %v", err)
	}
	if provider.Name != req.Name {
		t.Errorf("CreateProvider() got name = %v, want %v", provider.Name, req.Name)
	}
	if provider.TestModel != "gpt-3.5-turbo" {
		t.Errorf("CreateProvider() test model should be gpt-3.5-turbo, got %v", provider.TestModel)
	}
}

// TestService_CreateProvider_WithOptionalFields 测试创建供应商（带可选字段）
func TestService_CreateProvider_WithOptionalFields(t *testing.T) {
	service := setupTestService(t)

	enabled := false
	req := CreateProviderRequest{
		Name:      "Test Provider",
		BaseURL:   "https://api.test.com",
		APIKey:    "sk-test-key",
		TestModel: "gpt-3.5-turbo",
		Enabled:   &enabled,
	}

	provider, err := service.CreateProvider(req)
	if err != nil {
		t.Errorf("CreateProvider() failed: %v", err)
	}
	if provider.Enabled != false {
		t.Errorf("CreateProvider() enabled should be false, got %v", provider.Enabled)
	}
}

// TestService_CreateProvider_EmptyName 测试创建供应商（空名称）
func TestService_CreateProvider_EmptyName(t *testing.T) {
	service := setupTestService(t)

	req := CreateProviderRequest{
		Name:    "",
		BaseURL: "https://api.test.com",
		APIKey:  "sk-test-key",
	}

	_, err := service.CreateProvider(req)
	if err == nil {
		t.Error("CreateProvider() with empty name should fail")
	}
	if err != nil && err.Error() != "invalid input: name is required" {
		t.Errorf("CreateProvider() error should be 'invalid input: name is required', got %v", err)
	}
}

// TestService_CreateProvider_InvalidURL 测试创建供应商（无效 URL）
func TestService_CreateProvider_InvalidURL(t *testing.T) {
	service := setupTestService(t)

	testCases := []struct {
		name    string
		baseURL string
	}{
		{"missing scheme", "api.test.com"},
		{"invalid scheme", "ftp://api.test.com"},
		{"missing host", "https://"},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			req := CreateProviderRequest{
				Name:    "Test Provider",
				BaseURL: tc.baseURL,
				APIKey:  "sk-test-key",
			}

			_, err := service.CreateProvider(req)
			if err == nil {
				t.Errorf("CreateProvider() with %s should fail", tc.name)
			}
		})
	}
}

// TestService_CreateProvider_EmptyAPIKey 测试创建供应商（空 API Key）
func TestService_CreateProvider_EmptyAPIKey(t *testing.T) {
	service := setupTestService(t)

	req := CreateProviderRequest{
		Name:    "Test Provider",
		BaseURL: "https://api.test.com",
		APIKey:  "",
	}

	_, err := service.CreateProvider(req)
	if err == nil {
		t.Error("CreateProvider() with empty api_key should fail")
	}
}


// TestService_CreateProvider_DuplicateName 测试创建重复名称的供应商
func TestService_CreateProvider_DuplicateName(t *testing.T) {
	service := setupTestService(t)

	// 创建第一个供应商
	req := CreateProviderRequest{
		Name:      "Test Provider",
		BaseURL:   "https://api.test.com",
		APIKey:    "sk-test-key",
		TestModel: "gpt-3.5-turbo",
	}
	service.CreateProvider(req)

	// 尝试创建同名供应商
	_, err := service.CreateProvider(req)
	if err != ErrProviderNameExists {
		t.Errorf("CreateProvider() with duplicate name should return ErrProviderNameExists, got %v", err)
	}
}

// TestService_GetProvider 测试获取单个供应商
func TestService_GetProvider(t *testing.T) {
	service := setupTestService(t)

	// 创建测试数据
	req := CreateProviderRequest{
		Name:      "Test Provider",
		BaseURL:   "https://api.test.com",
		APIKey:    "sk-test-key",
		TestModel: "gpt-3.5-turbo",
	}
	created, _ := service.CreateProvider(req)

	// 测试获取存在的供应商
	provider, err := service.GetProvider(created.ID)
	if err != nil {
		t.Errorf("GetProvider() failed: %v", err)
	}
	if provider.Name != created.Name {
		t.Errorf("GetProvider() got name = %v, want %v", provider.Name, created.Name)
	}

	// 测试获取不存在的供应商
	_, err = service.GetProvider(9999)
	if err != ErrProviderNotFound {
		t.Errorf("GetProvider() with non-existent ID should return ErrProviderNotFound, got %v", err)
	}
}

// TestService_ListProviders 测试获取供应商列表
func TestService_ListProviders(t *testing.T) {
	service := setupTestService(t)

	// 创建测试数据
	for i := 0; i < 15; i++ {
		req := CreateProviderRequest{
			Name:      fmt.Sprintf("Provider %c", 'A'+i),
			BaseURL:   "https://api.test.com",
			APIKey:    "sk-test-key",
			TestModel: "gpt-3.5-turbo",
		}
		service.CreateProvider(req)
	}

	// 测试默认分页
	providers, total, err := service.ListProviders(1, 10)
	if err != nil {
		t.Errorf("ListProviders() failed: %v", err)
	}
	if total != 15 {
		t.Errorf("ListProviders() got total = %v, want 15", total)
	}
	if len(providers) != 10 {
		t.Errorf("ListProviders() got %v providers, want 10", len(providers))
	}

	// 测试第二页
	providers, _, err = service.ListProviders(2, 10)
	if err != nil {
		t.Errorf("ListProviders() failed: %v", err)
	}
	if len(providers) != 5 {
		t.Errorf("ListProviders() page 2 got %v providers, want 5", len(providers))
	}
}

// TestService_ListProviders_InvalidParams 测试无效分页参数
func TestService_ListProviders_InvalidParams(t *testing.T) {
	service := setupTestService(t)

	testCases := []struct {
		name     string
		page     int
		pageSize int
		wantPage int
		wantSize int
	}{
		{"negative page", -1, 10, 1, 10},
		{"zero page", 0, 10, 1, 10},
		{"negative page_size", 1, -1, 1, 10},
		{"zero page_size", 1, 0, 1, 10},
		{"page_size too large", 1, 200, 1, 100},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			_, _, err := service.ListProviders(tc.page, tc.pageSize)
			if err != nil {
				t.Errorf("ListProviders() should handle invalid params gracefully, got error: %v", err)
			}
		})
	}
}

// TestService_UpdateProvider_Success 测试成功更新供应商
func TestService_UpdateProvider_Success(t *testing.T) {
	service := setupTestService(t)

	// 创建测试数据
	req := CreateProviderRequest{
		Name:      "Test Provider",
		BaseURL:   "https://api.test.com",
		APIKey:    "sk-test-key",
		TestModel: "gpt-3.5-turbo",
	}
	created, _ := service.CreateProvider(req)

	// 更新数据
	newName := "Updated Provider"
	newURL := "https://api.updated.com"
	newTestModel := "gpt-4"
	updateReq := UpdateProviderRequest{
		Name:      &newName,
		BaseURL:   &newURL,
		TestModel: &newTestModel,
	}

	updated, err := service.UpdateProvider(created.ID, updateReq)
	if err != nil {
		t.Errorf("UpdateProvider() failed: %v", err)
	}
	if updated.Name != newName {
		t.Errorf("UpdateProvider() name not updated, got %v", updated.Name)
	}
	if updated.BaseURL != newURL {
		t.Errorf("UpdateProvider() base_url not updated, got %v", updated.BaseURL)
	}
	if updated.TestModel != newTestModel {
		t.Errorf("UpdateProvider() test model not updated, got %v", updated.TestModel)
	}
}

// TestService_UpdateProvider_NotFound 测试更新不存在的供应商
func TestService_UpdateProvider_NotFound(t *testing.T) {
	service := setupTestService(t)

	newName := "Updated Provider"
	updateReq := UpdateProviderRequest{
		Name: &newName,
	}

	_, err := service.UpdateProvider(9999, updateReq)
	if err != ErrProviderNotFound {
		t.Errorf("UpdateProvider() with non-existent ID should return ErrProviderNotFound, got %v", err)
	}
}

// TestService_UpdateProvider_DuplicateName 测试更新为重复名称
func TestService_UpdateProvider_DuplicateName(t *testing.T) {
	service := setupTestService(t)

	// 创建两个供应商
	req1 := CreateProviderRequest{
		Name:      "Provider 1",
		BaseURL:   "https://api1.test.com",
		APIKey:    "sk-test-key-1",
		TestModel: "gpt-3.5-turbo",
	}
	provider1, _ := service.CreateProvider(req1)

	req2 := CreateProviderRequest{
		Name:      "Provider 2",
		BaseURL:   "https://api2.test.com",
		APIKey:    "sk-test-key-2",
		TestModel: "gpt-3.5-turbo",
	}
	provider2, _ := service.CreateProvider(req2)

	// 尝试将 Provider 2 改名为 Provider 1
	name := "Provider 1"
	updateReq := UpdateProviderRequest{
		Name: &name,
	}

	_, err := service.UpdateProvider(provider2.ID, updateReq)
	if err != ErrProviderNameExists {
		t.Errorf("UpdateProvider() with duplicate name should return ErrProviderNameExists, got %v", err)
	}

	// 验证可以更新为自己的名称
	name = "Provider 1"
	updateReq = UpdateProviderRequest{
		Name: &name,
	}

	_, err = service.UpdateProvider(provider1.ID, updateReq)
	if err != nil {
		t.Errorf("UpdateProvider() with same name should succeed, got error: %v", err)
	}
}

// TestService_UpdateProvider_EmptyName 测试更新为空名称
func TestService_UpdateProvider_EmptyName(t *testing.T) {
	service := setupTestService(t)

	// 创建测试数据
	req := CreateProviderRequest{
		Name:      "Test Provider",
		BaseURL:   "https://api.test.com",
		APIKey:    "sk-test-key",
		TestModel: "gpt-3.5-turbo",
	}
	created, _ := service.CreateProvider(req)

	// 尝试更新为空名称
	emptyName := ""
	updateReq := UpdateProviderRequest{
		Name: &emptyName,
	}

	_, err := service.UpdateProvider(created.ID, updateReq)
	if err == nil {
		t.Error("UpdateProvider() with empty name should fail")
	}
}

// TestService_DeleteProvider 测试删除供应商
func TestService_DeleteProvider(t *testing.T) {
	service := setupTestService(t)

	// 创建测试数据
	req := CreateProviderRequest{
		Name:      "Test Provider",
		BaseURL:   "https://api.test.com",
		APIKey:    "sk-test-key",
		TestModel: "gpt-3.5-turbo",
	}
	created, _ := service.CreateProvider(req)

	// 删除供应商
	err := service.DeleteProvider(created.ID)
	if err != nil {
		t.Errorf("DeleteProvider() failed: %v", err)
	}

	// 验证硬删除（记录应该不存在）
	_, err = service.GetProvider(created.ID)
	if err != ErrProviderNotFound {
		t.Error("DeleteProvider() should remove the record from database")
	}

	// 测试删除不存在的供应商
	err = service.DeleteProvider(9999)
	if err != ErrProviderNotFound {
		t.Errorf("DeleteProvider() with non-existent ID should return ErrProviderNotFound, got %v", err)
	}
}
