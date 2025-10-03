package db

import (
	"testing"
	"time"

	"github.com/Mieluoxxx/Siriusx-API/internal/config"
	"github.com/Mieluoxxx/Siriusx-API/internal/models"
	"gorm.io/gorm"
)

// setupTestDB 创建测试用内存数据库
func setupTestDB(t *testing.T) *gorm.DB {
	cfg := &config.DatabaseConfig{
		Path:            ":memory:",
		MaxOpenConns:    10,
		MaxIdleConns:    5,
		ConnMaxLifetime: time.Hour,
		AutoMigrate:     true,
	}

	db, err := InitDatabase(cfg)
	if err != nil {
		t.Fatalf("初始化测试数据库失败: %v", err)
	}

	// 自动迁移
	if err := AutoMigrate(db); err != nil {
		t.Fatalf("数据库迁移失败: %v", err)
	}

	return db
}

// TestInitDatabase 测试数据库初始化
func TestInitDatabase(t *testing.T) {
	cfg := &config.DatabaseConfig{
		Path:            ":memory:",
		MaxOpenConns:    10,
		MaxIdleConns:    5,
		ConnMaxLifetime: time.Hour,
	}

	db, err := InitDatabase(cfg)
	if err != nil {
		t.Errorf("初始化数据库失败: %v", err)
	}

	if db == nil {
		t.Error("数据库连接为 nil")
	}

	// 验证连接池配置
	sqlDB, err := db.DB()
	if err != nil {
		t.Errorf("获取 SQL DB 失败: %v", err)
	}

	stats := sqlDB.Stats()
	if stats.MaxOpenConnections != 10 {
		t.Errorf("最大连接数配置错误: got %d, want 10", stats.MaxOpenConnections)
	}
}

// TestAutoMigrate 测试自动迁移
func TestAutoMigrate(t *testing.T) {
	db := setupTestDB(t)

	// 验证表是否存在
	tables := []interface{}{
		&models.Provider{},
		&models.UnifiedModel{},
		&models.ModelMapping{},
		&models.Token{},
	}

	for _, table := range tables {
		if !db.Migrator().HasTable(table) {
			t.Errorf("表 %T 不存在", table)
		}
	}
}

// TestProviderCRUD 测试 Provider CRUD 操作
func TestProviderCRUD(t *testing.T) {
	db := setupTestDB(t)

	// Create
	provider := &models.Provider{
		Name:         "Test Provider",
		BaseURL:      "https://test.example.com",
		APIKey:       "test-api-key",
		Enabled:      true,
		TestModel:    "gpt-3.5-turbo",
		HealthStatus: "unknown",
	}

	result := db.Create(provider)
	if result.Error != nil {
		t.Fatalf("创建 Provider 失败: %v", result.Error)
	}

	if provider.ID == 0 {
		t.Error("Provider ID 未自动生成")
	}

	// Read
	var found models.Provider
	result = db.First(&found, provider.ID)
	if result.Error != nil {
		t.Fatalf("查询 Provider 失败: %v", result.Error)
	}

	if found.Name != "Test Provider" {
		t.Errorf("Provider 名称不匹配: got %s, want Test Provider", found.Name)
	}

	// Update
	found.TestModel = "gpt-4"
	result = db.Save(&found)
	if result.Error != nil {
		t.Fatalf("更新 Provider 失败: %v", result.Error)
	}

	var updated models.Provider
	db.First(&updated, provider.ID)
	if updated.TestModel != "gpt-4" {
		t.Errorf("Provider 测试模型未更新: got %s, want gpt-4", updated.TestModel)
	}

	// Delete
	result = db.Delete(&found)
	if result.Error != nil {
		t.Fatalf("删除 Provider 失败: %v", result.Error)
	}

	var deleted models.Provider
	result = db.First(&deleted, provider.ID)
	if result.Error == nil {
		t.Error("Provider 未被删除")
	}
}

// TestUnifiedModelCRUD 测试 UnifiedModel CRUD 操作
func TestUnifiedModelCRUD(t *testing.T) {
	db := setupTestDB(t)

	// Create
	model := &models.UnifiedModel{
		Name:        "claude-sonnet-4",
		Description: "平衡性能的 Sonnet 模型",
	}

	result := db.Create(model)
	if result.Error != nil {
		t.Fatalf("创建 UnifiedModel 失败: %v", result.Error)
	}

	// Read
	var found models.UnifiedModel
	result = db.First(&found, model.ID)
	if result.Error != nil {
		t.Fatalf("查询 UnifiedModel 失败: %v", result.Error)
	}

	if found.Name != "claude-sonnet-4" {
		t.Errorf("UnifiedModel 名称不匹配: got %s, want claude-sonnet-4", found.Name)
	}
}

// TestModelMappingWithForeignKey 测试 ModelMapping 外键关系
func TestModelMappingWithForeignKey(t *testing.T) {
	db := setupTestDB(t)

	// 创建 Provider
	provider := &models.Provider{
		Name:    "Test Provider",
		BaseURL: "https://test.com",
		APIKey:  "test-key",
	}
	db.Create(provider)

	// 创建 UnifiedModel
	model := &models.UnifiedModel{
		Name:        "test-model",
		Description: "Test Model",
	}
	db.Create(model)

	// 创建 ModelMapping
	mapping := &models.ModelMapping{
		UnifiedModelID: model.ID,
		ProviderID:     provider.ID,
		TargetModel:    "gpt-4",
		Weight:         70,
		Priority:       1,
		Enabled:        true,
	}

	result := db.Create(mapping)
	if result.Error != nil {
		t.Fatalf("创建 ModelMapping 失败: %v", result.Error)
	}

	// 验证外键关系
	var foundMapping models.ModelMapping
	result = db.Preload("UnifiedModel").Preload("Provider").First(&foundMapping, mapping.ID)
	if result.Error != nil {
		t.Fatalf("查询 ModelMapping 失败: %v", result.Error)
	}

	if foundMapping.UnifiedModel.Name != "test-model" {
		t.Errorf("外键关系错误: UnifiedModel 名称不匹配")
	}

	if foundMapping.Provider.Name != "Test Provider" {
		t.Errorf("外键关系错误: Provider 名称不匹配")
	}

	// 测试级联删除
	db.Delete(provider)

	var deletedMapping models.ModelMapping
	result = db.First(&deletedMapping, mapping.ID)
	if result.Error == nil {
		t.Error("级联删除失败: ModelMapping 应该被删除")
	}
}

// TestTokenCRUD 测试 Token CRUD 操作
func TestTokenCRUD(t *testing.T) {
	db := setupTestDB(t)

	// Create
	expiresAt := time.Now().Add(24 * time.Hour)
	token := &models.Token{
		Name:      "Test Token",
		Token:     "sk-test1234567890",
		Enabled:   true,
		ExpiresAt: &expiresAt,
	}

	result := db.Create(token)
	if result.Error != nil {
		t.Fatalf("创建 Token 失败: %v", result.Error)
	}

	// Read
	var found models.Token
	result = db.First(&found, token.ID)
	if result.Error != nil {
		t.Fatalf("查询 Token 失败: %v", result.Error)
	}

	if found.Token != "sk-test1234567890" {
		t.Errorf("Token 不匹配: got %s, want sk-test1234567890", found.Token)
	}

	// 测试唯一约束
	duplicate := &models.Token{
		Name:    "Duplicate Token",
		Token:   "sk-test1234567890", // 相同的 token
		Enabled: true,
	}

	result = db.Create(duplicate)
	if result.Error == nil {
		t.Error("唯一约束未生效: 允许创建重复的 Token")
	}
}
