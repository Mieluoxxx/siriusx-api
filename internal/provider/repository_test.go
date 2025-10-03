package provider

import (
	"fmt"
	"testing"

	"github.com/Mieluoxxx/Siriusx-API/internal/models"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

// setupTestDB 创建测试数据库
func setupTestDB(t *testing.T) *gorm.DB {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("failed to create test database: %v", err)
	}

	// 自动迁移
	if err := db.AutoMigrate(&models.Provider{}); err != nil {
		t.Fatalf("failed to migrate test database: %v", err)
	}

	return db
}

// TestRepository_Create 测试创建供应商
func TestRepository_Create(t *testing.T) {
	db := setupTestDB(t)
	repo := NewRepository(db)

	provider := &models.Provider{
		Name:      "Test Provider",
		BaseURL:   "https://api.test.com",
		APIKey:    "sk-test-key",
		Enabled:   true,
		TestModel: "gpt-3.5-turbo",
	}

	err := repo.Create(provider)
	if err != nil {
		t.Errorf("Create() failed: %v", err)
	}

	if provider.ID == 0 {
		t.Error("Create() did not set provider ID")
	}
}

// TestRepository_FindByID 测试根据 ID 查找供应商
func TestRepository_FindByID(t *testing.T) {
	db := setupTestDB(t)
	repo := NewRepository(db)

	// 创建测试数据
	provider := &models.Provider{
		Name:    "Test Provider",
		BaseURL: "https://api.test.com",
		APIKey:  "sk-test-key",
	}
	repo.Create(provider)

	// 测试查找存在的供应商
	found, err := repo.FindByID(provider.ID)
	if err != nil {
		t.Errorf("FindByID() failed: %v", err)
	}
	if found.Name != provider.Name {
		t.Errorf("FindByID() got name = %v, want %v", found.Name, provider.Name)
	}

	// 测试查找不存在的供应商
	_, err = repo.FindByID(9999)
	if err != ErrProviderNotFound {
		t.Errorf("FindByID() with non-existent ID should return ErrProviderNotFound, got %v", err)
	}
}

// TestRepository_FindByName 测试根据名称查找供应商
func TestRepository_FindByName(t *testing.T) {
	db := setupTestDB(t)
	repo := NewRepository(db)

	// 创建测试数据
	provider := &models.Provider{
		Name:    "Test Provider",
		BaseURL: "https://api.test.com",
		APIKey:  "sk-test-key",
	}
	repo.Create(provider)

	// 测试查找存在的供应商
	found, err := repo.FindByName("Test Provider")
	if err != nil {
		t.Errorf("FindByName() failed: %v", err)
	}
	if found.ID != provider.ID {
		t.Errorf("FindByName() got ID = %v, want %v", found.ID, provider.ID)
	}

	// 测试查找不存在的供应商
	_, err = repo.FindByName("Non-existent")
	if err != ErrProviderNotFound {
		t.Errorf("FindByName() with non-existent name should return ErrProviderNotFound, got %v", err)
	}
}

// TestRepository_FindAll 测试分页查询
func TestRepository_FindAll(t *testing.T) {
	db := setupTestDB(t)
	repo := NewRepository(db)

	// 创建测试数据
	for i := 0; i < 25; i++ {
		provider := &models.Provider{
			Name:    fmt.Sprintf("Provider %c", 'A'+i),
			BaseURL: "https://api.test.com",
			APIKey:  "sk-test-key",
		}
		repo.Create(provider)
	}

	// 测试第一页
	providers, total, err := repo.FindAll(1, 10)
	if err != nil {
		t.Errorf("FindAll() failed: %v", err)
	}
	if total != 25 {
		t.Errorf("FindAll() got total = %v, want 25", total)
	}
	if len(providers) != 10 {
		t.Errorf("FindAll() got %v providers, want 10", len(providers))
	}

	// 测试第三页
	providers, total, err = repo.FindAll(3, 10)
	if err != nil {
		t.Errorf("FindAll() failed: %v", err)
	}
	if len(providers) != 5 {
		t.Errorf("FindAll() page 3 got %v providers, want 5", len(providers))
	}
}

// TestRepository_Update 测试更新供应商
func TestRepository_Update(t *testing.T) {
	db := setupTestDB(t)
	repo := NewRepository(db)

	// 创建测试数据
	provider := &models.Provider{
		Name:      "Test Provider",
		BaseURL:   "https://api.test.com",
		APIKey:    "sk-test-key",
		TestModel: "gpt-3.5-turbo",
	}
	repo.Create(provider)

	// 更新数据
	provider.Name = "Updated Provider"
	provider.TestModel = "gpt-4"
	err := repo.Update(provider)
	if err != nil {
		t.Errorf("Update() failed: %v", err)
	}

	// 验证更新
	updated, _ := repo.FindByID(provider.ID)
	if updated.Name != "Updated Provider" {
		t.Errorf("Update() name not updated, got %v", updated.Name)
	}
	if updated.TestModel != "gpt-4" {
		t.Errorf("Update() test model not updated, got %v", updated.TestModel)
	}
}

// TestRepository_Delete 测试硬删除
func TestRepository_Delete(t *testing.T) {
	db := setupTestDB(t)
	repo := NewRepository(db)

	// 创建测试数据
	provider := &models.Provider{
		Name:    "Test Provider",
		BaseURL: "https://api.test.com",
		APIKey:  "sk-test-key",
		Enabled: true,
	}
	repo.Create(provider)

	// 硬删除
	err := repo.Delete(provider.ID)
	if err != nil {
		t.Errorf("Delete() failed: %v", err)
	}

	// 验证硬删除（记录应该不存在）
	_, err = repo.FindByID(provider.ID)
	if err != ErrProviderNotFound {
		t.Error("Delete() did not remove the record from database")
	}

	// 测试删除不存在的供应商
	err = repo.Delete(9999)
	if err != ErrProviderNotFound {
		t.Errorf("Delete() with non-existent ID should return ErrProviderNotFound, got %v", err)
	}
}

// TestRepository_CheckNameExists 测试名称唯一性检查
func TestRepository_CheckNameExists(t *testing.T) {
	db := setupTestDB(t)
	repo := NewRepository(db)

	// 创建测试数据
	provider := &models.Provider{
		Name:    "Test Provider",
		BaseURL: "https://api.test.com",
		APIKey:  "sk-test-key",
	}
	repo.Create(provider)

	// 测试已存在的名称
	exists, err := repo.CheckNameExists("Test Provider", 0)
	if err != nil {
		t.Errorf("CheckNameExists() failed: %v", err)
	}
	if !exists {
		t.Error("CheckNameExists() should return true for existing name")
	}

	// 测试不存在的名称
	exists, err = repo.CheckNameExists("Non-existent", 0)
	if err != nil {
		t.Errorf("CheckNameExists() failed: %v", err)
	}
	if exists {
		t.Error("CheckNameExists() should return false for non-existent name")
	}

	// 测试排除当前 ID
	exists, err = repo.CheckNameExists("Test Provider", provider.ID)
	if err != nil {
		t.Errorf("CheckNameExists() failed: %v", err)
	}
	if exists {
		t.Error("CheckNameExists() should return false when excluding current ID")
	}
}
