package token

import (
	"testing"
	"time"

	"github.com/Mieluoxxx/Siriusx-API/internal/models"
	"github.com/stretchr/testify/assert"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

// setupCustomTokenTestDB 创建测试数据库
func setupCustomTokenTestDB(t *testing.T) *gorm.DB {
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

// TestValidateCustomToken 测试自定义 Token 验证
func TestValidateCustomToken(t *testing.T) {
	tests := []struct {
		name    string
		token   string
		wantErr bool
	}{
		{
			name:    "有效的自定义 Token",
			token:   "sk-123456",
			wantErr: false,
		},
		{
			name:    "有效的长自定义 Token",
			token:   "sk-my-custom-production-key-2024",
			wantErr: false,
		},
		{
			name:    "太短的 Token",
			token:   "sk-123",
			wantErr: true,
		},
		{
			name:    "不以 sk- 开头",
			token:   "abc-12345678",
			wantErr: true,
		},
		{
			name:    "完全不符合格式",
			token:   "invalidtoken",
			wantErr: true,
		},
		{
			name:    "空字符串",
			token:   "",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateCustomToken(tt.token)
			if tt.wantErr {
				assert.Error(t, err, "ValidateCustomToken() should return error")
				assert.Equal(t, ErrInvalidCustomToken, err)
			} else {
				assert.NoError(t, err, "ValidateCustomToken() should not return error")
			}
		})
	}
}

// TestCreateToken_WithCustomToken 测试创建自定义 Token
func TestCreateToken_WithCustomToken(t *testing.T) {
	database := setupCustomTokenTestDB(t)
	repo := NewRepository(database)
	service := NewService(repo)

	t.Run("创建有效的自定义 Token", func(t *testing.T) {
		customToken := "sk-test-custom-token-001"
		token, err := service.CreateToken("测试自定义Token", nil, customToken)

		assert.NoError(t, err)
		assert.NotNil(t, token)
		assert.Equal(t, customToken, token.Token)
		assert.Equal(t, "测试自定义Token", token.Name)
		assert.True(t, token.Enabled)
	})

	t.Run("创建重复的自定义 Token", func(t *testing.T) {
		customToken := "sk-duplicate-token-001"

		// 第一次创建
		_, err := service.CreateToken("第一个Token", nil, customToken)
		assert.NoError(t, err)

		// 第二次创建相同的 Token
		_, err = service.CreateToken("第二个Token", nil, customToken)
		assert.Error(t, err)
		assert.Equal(t, ErrTokenValueExists, err)
	})

	t.Run("创建无效格式的自定义 Token", func(t *testing.T) {
		invalidToken := "invalid-token"
		_, err := service.CreateToken("测试无效Token", nil, invalidToken)

		assert.Error(t, err)
		assert.Equal(t, ErrInvalidCustomToken, err)
	})

	t.Run("自定义 Token 为空时自动生成", func(t *testing.T) {
		token, err := service.CreateToken("自动生成Token", nil, "")

		assert.NoError(t, err)
		assert.NotNil(t, token)
		assert.NotEmpty(t, token.Token)
		assert.True(t, len(token.Token) > 40) // 自动生成的 Token 很长
		assert.Equal(t, "sk-", token.Token[:3])
	})
}

// TestCreateToken_WithCustomTokenAndExpiry 测试自定义 Token 和过期时间
func TestCreateToken_WithCustomTokenAndExpiry(t *testing.T) {
	database := setupCustomTokenTestDB(t)
	repo := NewRepository(database)
	service := NewService(repo)

	expiresAt := time.Now().Add(24 * time.Hour)
	customToken := "sk-expiring-custom-token"

	token, err := service.CreateToken("带过期时间的自定义Token", &expiresAt, customToken)

	assert.NoError(t, err)
	assert.NotNil(t, token)
	assert.Equal(t, customToken, token.Token)
	assert.NotNil(t, token.ExpiresAt)
	assert.WithinDuration(t, expiresAt, *token.ExpiresAt, time.Second)
}

// TestGetToken 测试获取 Token
func TestGetToken(t *testing.T) {
	database := setupCustomTokenTestDB(t)
	repo := NewRepository(database)
	service := NewService(repo)

	// 创建一个 Token
	customToken := "sk-get-token-test-001"
	created, err := service.CreateToken("测试获取Token", nil, customToken)
	assert.NoError(t, err)

	// 获取 Token
	token, err := service.GetToken(created.ID)
	assert.NoError(t, err)
	assert.NotNil(t, token)
	assert.Equal(t, created.ID, token.ID)
	assert.Equal(t, customToken, token.Token)
	assert.Equal(t, "测试获取Token", token.Name)
}

// TestGetToken_NotFound 测试获取不存在的 Token
func TestGetToken_NotFound(t *testing.T) {
	database := setupCustomTokenTestDB(t)
	repo := NewRepository(database)
	service := NewService(repo)

	token, err := service.GetToken(99999)
	assert.Error(t, err)
	assert.Nil(t, token)
	assert.Equal(t, ErrTokenNotFound, err)
}
