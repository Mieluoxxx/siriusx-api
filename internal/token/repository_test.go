package token

import (
	"testing"
	"time"

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
	if err := db.AutoMigrate(&models.Token{}); err != nil {
		t.Fatalf("failed to migrate test database: %v", err)
	}

	return db
}

// TestRepository_Create 测试创建 Token
func TestRepository_Create(t *testing.T) {
	db := setupTestDB(t)
	repo := NewRepository(db)

	token := &models.Token{
		Name:    "Test Token",
		Token:   "sk-test123456789",
		Enabled: true,
	}

	err := repo.Create(token)
	if err != nil {
		t.Errorf("Create() failed: %v", err)
	}

	if token.ID == 0 {
		t.Error("Create() did not set token ID")
	}
}

// TestRepository_FindByID 测试根据 ID 查找 Token
func TestRepository_FindByID(t *testing.T) {
	db := setupTestDB(t)
	repo := NewRepository(db)

	// 创建测试数据
	token := &models.Token{
		Name:    "Test Token",
		Token:   "sk-test123456789",
		Enabled: true,
	}
	repo.Create(token)

	// 测试查找存在的 Token
	found, err := repo.FindByID(token.ID)
	if err != nil {
		t.Errorf("FindByID() failed: %v", err)
	}
	if found.Name != token.Name {
		t.Errorf("FindByID() got name = %v, want %v", found.Name, token.Name)
	}

	// 测试查找不存在的 Token
	_, err = repo.FindByID(9999)
	if err != ErrTokenNotFound {
		t.Errorf("FindByID() with non-existent ID should return ErrTokenNotFound, got %v", err)
	}
}

// TestRepository_FindByValue 测试根据 Token 值查找 Token
func TestRepository_FindByValue(t *testing.T) {
	db := setupTestDB(t)
	repo := NewRepository(db)

	// 创建测试数据
	token := &models.Token{
		Name:    "Test Token",
		Token:   "sk-test123456789",
		Enabled: true,
	}
	repo.Create(token)

	// 测试查找存在的 Token
	found, err := repo.FindByValue("sk-test123456789")
	if err != nil {
		t.Errorf("FindByValue() failed: %v", err)
	}
	if found.Name != token.Name {
		t.Errorf("FindByValue() got name = %v, want %v", found.Name, token.Name)
	}

	// 测试查找不存在的 Token
	_, err = repo.FindByValue("sk-nonexistent")
	if err != ErrTokenNotFound {
		t.Errorf("FindByValue() with non-existent value should return ErrTokenNotFound, got %v", err)
	}
}

// TestRepository_FindAll 测试查找所有 Token
func TestRepository_FindAll(t *testing.T) {
	db := setupTestDB(t)
	repo := NewRepository(db)

	// 创建测试数据
	token1 := &models.Token{Name: "Token 1", Token: "sk-token1", Enabled: true}
	token2 := &models.Token{Name: "Token 2", Token: "sk-token2", Enabled: true}
	repo.Create(token1)
	repo.Create(token2)

	// 测试查找所有 Token
	tokens, err := repo.FindAll()
	if err != nil {
		t.Errorf("FindAll() failed: %v", err)
	}

	if len(tokens) != 2 {
		t.Errorf("FindAll() got %d tokens, want 2", len(tokens))
	}
}

// TestRepository_Delete 测试删除 Token
func TestRepository_Delete(t *testing.T) {
	db := setupTestDB(t)
	repo := NewRepository(db)

	// 创建测试数据
	token := &models.Token{
		Name:    "Test Token",
		Token:   "sk-test123456789",
		Enabled: true,
	}
	repo.Create(token)

	// 测试删除存在的 Token
	err := repo.Delete(token.ID)
	if err != nil {
		t.Errorf("Delete() failed: %v", err)
	}

	// 验证已删除
	_, err = repo.FindByID(token.ID)
	if err != ErrTokenNotFound {
		t.Error("Delete() did not delete the token")
	}

	// 测试删除不存在的 Token
	err = repo.Delete(9999)
	if err != ErrTokenNotFound {
		t.Errorf("Delete() with non-existent ID should return ErrTokenNotFound, got %v", err)
	}
}

// TestRepository_CheckValueExists 测试检查 Token 值是否存在
func TestRepository_CheckValueExists(t *testing.T) {
	db := setupTestDB(t)
	repo := NewRepository(db)

	// 创建测试数据
	token := &models.Token{
		Name:    "Test Token",
		Token:   "sk-test123456789",
		Enabled: true,
	}
	repo.Create(token)

	// 测试已存在的 Token 值
	exists, err := repo.CheckValueExists("sk-test123456789")
	if err != nil {
		t.Errorf("CheckValueExists() failed: %v", err)
	}
	if !exists {
		t.Error("CheckValueExists() should return true for existing token")
	}

	// 测试不存在的 Token 值
	exists, err = repo.CheckValueExists("sk-nonexistent")
	if err != nil {
		t.Errorf("CheckValueExists() failed: %v", err)
	}
	if exists {
		t.Error("CheckValueExists() should return false for non-existent token")
	}
}

// TestRepository_UniqueConstraint 测试唯一约束
func TestRepository_UniqueConstraint(t *testing.T) {
	db := setupTestDB(t)
	repo := NewRepository(db)

	// 创建第一个 Token
	token1 := &models.Token{
		Name:    "Token 1",
		Token:   "sk-duplicate",
		Enabled: true,
	}
	err := repo.Create(token1)
	if err != nil {
		t.Fatalf("Create() failed for first token: %v", err)
	}

	// 尝试创建重复的 Token
	token2 := &models.Token{
		Name:    "Token 2",
		Token:   "sk-duplicate",
		Enabled: true,
	}
	err = repo.Create(token2)
	if err == nil {
		t.Error("Create() should fail for duplicate token value")
	}
}

// TestRepository_ExpiresAt 测试过期时间字段
func TestRepository_ExpiresAt(t *testing.T) {
	db := setupTestDB(t)
	repo := NewRepository(db)

	futureTime := time.Now().Add(24 * time.Hour)
	token := &models.Token{
		Name:      "Test Token",
		Token:     "sk-test123456789",
		Enabled:   true,
		ExpiresAt: &futureTime,
	}

	err := repo.Create(token)
	if err != nil {
		t.Errorf("Create() failed: %v", err)
	}

	found, err := repo.FindByID(token.ID)
	if err != nil {
		t.Errorf("FindByID() failed: %v", err)
	}

	if found.ExpiresAt == nil {
		t.Error("ExpiresAt should not be nil")
	}

	if !found.ExpiresAt.Equal(futureTime) {
		t.Errorf("ExpiresAt got = %v, want %v", found.ExpiresAt, futureTime)
	}
}
