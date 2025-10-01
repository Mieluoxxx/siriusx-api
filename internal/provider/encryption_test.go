package provider

import (
	"encoding/base64"
	"testing"

	"github.com/Mieluoxxx/Siriusx-API/internal/crypto"
	"github.com/Mieluoxxx/Siriusx-API/internal/models"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

// setupTestServiceWithEncryption 创建带加密密钥的测试服务
func setupTestServiceWithEncryption(t *testing.T, encryptionKey []byte) *Service {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("failed to create test database: %v", err)
	}

	// 自动迁移
	if err := db.AutoMigrate(&models.Provider{}); err != nil {
		t.Fatalf("failed to migrate test database: %v", err)
	}

	repo := NewRepository(db)
	return NewServiceWithEncryption(repo, encryptionKey)
}

// generateTestEncryptionKey 生成测试加密密钥
func generateTestEncryptionKey(t *testing.T) []byte {
	keyStr, err := crypto.GenerateEncryptionKey()
	if err != nil {
		t.Fatalf("failed to generate encryption key: %v", err)
	}

	key, err := base64.StdEncoding.DecodeString(keyStr)
	if err != nil {
		t.Fatalf("failed to decode encryption key: %v", err)
	}

	return key
}

// TestService_CreateProvider_WithEncryption 测试创建供应商时加密 API Key
func TestService_CreateProvider_WithEncryption(t *testing.T) {
	encryptionKey := generateTestEncryptionKey(t)
	service := setupTestServiceWithEncryption(t, encryptionKey)

	req := CreateProviderRequest{
		Name:    "Test Provider",
		BaseURL: "https://api.test.com",
		APIKey:  "sk-test-key-12345",
	}

	// 创建供应商
	provider, err := service.CreateProvider(req)
	if err != nil {
		t.Fatalf("CreateProvider() failed: %v", err)
	}

	// 验证返回的 Provider 包含明文 API Key
	if provider.APIKey != "sk-test-key-12345" {
		t.Errorf("CreateProvider() returned encrypted API key, want plaintext")
	}

	// 直接从数据库读取，验证存储的是加密后的 API Key
	var dbProvider models.Provider
	service.repo.db.First(&dbProvider, provider.ID)

	// 验证数据库中的 API Key 已加密
	if dbProvider.APIKey == "sk-test-key-12345" {
		t.Error("API Key stored in plaintext in database")
	}

	// 验证可以解密
	decrypted, err := crypto.DecryptString(dbProvider.APIKey, encryptionKey)
	if err != nil {
		t.Fatalf("Failed to decrypt API key: %v", err)
	}

	if decrypted != "sk-test-key-12345" {
		t.Errorf("Decrypted API key = %s, want sk-test-key-12345", decrypted)
	}
}

// TestService_GetProvider_WithEncryption 测试获取供应商时解密 API Key
func TestService_GetProvider_WithEncryption(t *testing.T) {
	encryptionKey := generateTestEncryptionKey(t)
	service := setupTestServiceWithEncryption(t, encryptionKey)

	// 创建供应商
	req := CreateProviderRequest{
		Name:    "Test Provider",
		BaseURL: "https://api.test.com",
		APIKey:  "sk-test-key-12345",
	}
	created, _ := service.CreateProvider(req)

	// 获取供应商
	provider, err := service.GetProvider(created.ID)
	if err != nil {
		t.Fatalf("GetProvider() failed: %v", err)
	}

	// 验证返回的 API Key 已解密
	if provider.APIKey != "sk-test-key-12345" {
		t.Errorf("GetProvider() API Key = %s, want sk-test-key-12345", provider.APIKey)
	}
}

// TestService_UpdateProvider_WithEncryption 测试更新供应商时加密新的 API Key
func TestService_UpdateProvider_WithEncryption(t *testing.T) {
	encryptionKey := generateTestEncryptionKey(t)
	service := setupTestServiceWithEncryption(t, encryptionKey)

	// 创建供应商
	req := CreateProviderRequest{
		Name:    "Test Provider",
		BaseURL: "https://api.test.com",
		APIKey:  "sk-test-key-12345",
	}
	created, _ := service.CreateProvider(req)

	// 更新 API Key
	newAPIKey := "sk-new-key-67890"
	updateReq := UpdateProviderRequest{
		APIKey: &newAPIKey,
	}

	updated, err := service.UpdateProvider(created.ID, updateReq)
	if err != nil {
		t.Fatalf("UpdateProvider() failed: %v", err)
	}

	// 验证返回的 Provider 包含新的明文 API Key
	if updated.APIKey != "sk-new-key-67890" {
		t.Errorf("UpdateProvider() returned wrong API key, got %s", updated.APIKey)
	}

	// 直接从数据库读取，验证存储的是加密后的 API Key
	var dbProvider models.Provider
	service.repo.db.First(&dbProvider, created.ID)

	// 验证数据库中的 API Key 已加密
	if dbProvider.APIKey == "sk-new-key-67890" {
		t.Error("Updated API Key stored in plaintext in database")
	}

	// 验证可以解密为新的 API Key
	decrypted, err := crypto.DecryptString(dbProvider.APIKey, encryptionKey)
	if err != nil {
		t.Fatalf("Failed to decrypt updated API key: %v", err)
	}

	if decrypted != "sk-new-key-67890" {
		t.Errorf("Decrypted updated API key = %s, want sk-new-key-67890", decrypted)
	}
}

// TestService_UpdateProvider_WithoutAPIKeyChange 测试更新其他字段时保持 API Key 解密
func TestService_UpdateProvider_WithoutAPIKeyChange(t *testing.T) {
	encryptionKey := generateTestEncryptionKey(t)
	service := setupTestServiceWithEncryption(t, encryptionKey)

	// 创建供应商
	req := CreateProviderRequest{
		Name:    "Test Provider",
		BaseURL: "https://api.test.com",
		APIKey:  "sk-test-key-12345",
	}
	created, _ := service.CreateProvider(req)

	// 更新名称（不更新 API Key）
	newName := "Updated Provider"
	updateReq := UpdateProviderRequest{
		Name: &newName,
	}

	updated, err := service.UpdateProvider(created.ID, updateReq)
	if err != nil {
		t.Fatalf("UpdateProvider() failed: %v", err)
	}

	// 验证返回的 Provider 包含解密后的原 API Key
	if updated.APIKey != "sk-test-key-12345" {
		t.Errorf("UpdateProvider() returned wrong API key, got %s", updated.APIKey)
	}
}

// TestService_Encryption_NoKey 测试没有加密密钥时不加密
func TestService_Encryption_NoKey(t *testing.T) {
	// 创建没有加密密钥的服务
	db, _ := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	db.AutoMigrate(&models.Provider{})
	repo := NewRepository(db)
	service := NewService(repo) // 不传加密密钥

	req := CreateProviderRequest{
		Name:    "Test Provider",
		BaseURL: "https://api.test.com",
		APIKey:  "sk-test-key-12345",
	}

	// 创建供应商
	provider, err := service.CreateProvider(req)
	if err != nil {
		t.Fatalf("CreateProvider() failed: %v", err)
	}

	// 直接从数据库读取
	var dbProvider models.Provider
	service.repo.db.First(&dbProvider, provider.ID)

	// 验证数据库中的 API Key 是明文
	if dbProvider.APIKey != "sk-test-key-12345" {
		t.Error("API Key should be stored in plaintext when no encryption key")
	}
}

// TestService_Encryption_DifferentKeys 测试使用不同密钥解密失败
func TestService_Encryption_DifferentKeys(t *testing.T) {
	encryptionKey1 := generateTestEncryptionKey(t)
	encryptionKey2 := generateTestEncryptionKey(t)

	service1 := setupTestServiceWithEncryption(t, encryptionKey1)

	// 使用 key1 创建供应商
	req := CreateProviderRequest{
		Name:    "Test Provider",
		BaseURL: "https://api.test.com",
		APIKey:  "sk-test-key-12345",
	}
	created, _ := service1.CreateProvider(req)

	// 使用 key2 创建新的 service
	service2 := NewServiceWithEncryption(service1.repo, encryptionKey2)

	// 尝试获取供应商（应该解密失败）
	_, err := service2.GetProvider(created.ID)
	if err == nil {
		t.Error("GetProvider() with wrong key should fail")
	}
}
