package token

import (
	"regexp"
	"testing"
	"time"

	"github.com/Mieluoxxx/Siriusx-API/internal/models"
)

// TestGenerateTokenValue 测试 Token 生成
func TestGenerateTokenValue(t *testing.T) {
	// 测试 Token 格式
	token, err := GenerateTokenValue()
	if err != nil {
		t.Errorf("GenerateTokenValue() failed: %v", err)
	}

	// 验证格式: sk- + base64 字符串 (包含 =, -, _, 字母和数字)
	pattern := `^sk-[a-zA-Z0-9_\-=]{43,44}$`
	matched, err := regexp.MatchString(pattern, token)
	if err != nil {
		t.Fatalf("regexp.MatchString() failed: %v", err)
	}
	if !matched {
		t.Errorf("GenerateTokenValue() = %v, does not match pattern %v", token, pattern)
	}
}

// TestGenerateTokenValue_Uniqueness 测试 Token 唯一性
func TestGenerateTokenValue_Uniqueness(t *testing.T) {
	tokens := make(map[string]bool)
	count := 1000

	for i := 0; i < count; i++ {
		token, err := GenerateTokenValue()
		if err != nil {
			t.Fatalf("GenerateTokenValue() failed at iteration %d: %v", i, err)
		}

		if tokens[token] {
			t.Errorf("GenerateTokenValue() generated duplicate token: %v", token)
		}
		tokens[token] = true
	}

	if len(tokens) != count {
		t.Errorf("GenerateTokenValue() generated %d unique tokens, want %d", len(tokens), count)
	}
}

// TestService_CreateToken 测试创建 Token
func TestService_CreateToken(t *testing.T) {
	db := setupTestDB(t)
	repo := NewRepository(db)
	service := NewService(repo)

	token, err := service.CreateToken("Test Token", nil)
	if err != nil {
		t.Errorf("CreateToken() failed: %v", err)
	}

	if token.ID == 0 {
		t.Error("CreateToken() did not set token ID")
	}

	if token.Name != "Test Token" {
		t.Errorf("CreateToken() got name = %v, want 'Test Token'", token.Name)
	}

	if !token.Enabled {
		t.Error("CreateToken() should set Enabled to true by default")
	}
}

// TestService_CreateToken_WithExpiresAt 测试创建带过期时间的 Token
func TestService_CreateToken_WithExpiresAt(t *testing.T) {
	db := setupTestDB(t)
	repo := NewRepository(db)
	service := NewService(repo)

	futureTime := time.Now().Add(24 * time.Hour)
	token, err := service.CreateToken("Test Token", &futureTime)
	if err != nil {
		t.Errorf("CreateToken() failed: %v", err)
	}

	if token.ExpiresAt == nil {
		t.Error("CreateToken() should set ExpiresAt")
	}
}

// TestService_CreateToken_InvalidExpiresAt 测试创建过期时间无效的 Token
func TestService_CreateToken_InvalidExpiresAt(t *testing.T) {
	db := setupTestDB(t)
	repo := NewRepository(db)
	service := NewService(repo)

	pastTime := time.Now().Add(-24 * time.Hour)
	_, err := service.CreateToken("Test Token", &pastTime)
	if err != ErrInvalidExpiresAt {
		t.Errorf("CreateToken() with past expiresAt should return ErrInvalidExpiresAt, got %v", err)
	}
}

// TestService_ListTokens 测试获取 Token 列表
func TestService_ListTokens(t *testing.T) {
	db := setupTestDB(t)
	repo := NewRepository(db)
	service := NewService(repo)

	// 创建测试数据
	service.CreateToken("Token 1", nil)
	service.CreateToken("Token 2", nil)

	tokens, err := service.ListTokens()
	if err != nil {
		t.Errorf("ListTokens() failed: %v", err)
	}

	if len(tokens) != 2 {
		t.Errorf("ListTokens() got %d tokens, want 2", len(tokens))
	}
}

// TestService_GetToken 测试获取单个 Token
func TestService_GetToken(t *testing.T) {
	db := setupTestDB(t)
	repo := NewRepository(db)
	service := NewService(repo)

	// 创建测试数据
	created, _ := service.CreateToken("Test Token", nil)

	// 测试获取存在的 Token
	found, err := service.GetToken(created.ID)
	if err != nil {
		t.Errorf("GetToken() failed: %v", err)
	}
	if found.Name != created.Name {
		t.Errorf("GetToken() got name = %v, want %v", found.Name, created.Name)
	}

	// 测试获取不存在的 Token
	_, err = service.GetToken(9999)
	if err != ErrTokenNotFound {
		t.Errorf("GetToken() with non-existent ID should return ErrTokenNotFound, got %v", err)
	}
}

// TestService_DeleteToken 测试删除 Token
func TestService_DeleteToken(t *testing.T) {
	db := setupTestDB(t)
	repo := NewRepository(db)
	service := NewService(repo)

	// 创建测试数据
	token, _ := service.CreateToken("Test Token", nil)

	// 测试删除存在的 Token
	err := service.DeleteToken(token.ID)
	if err != nil {
		t.Errorf("DeleteToken() failed: %v", err)
	}

	// 验证已删除
	_, err = service.GetToken(token.ID)
	if err != ErrTokenNotFound {
		t.Error("DeleteToken() did not delete the token")
	}

	// 测试删除不存在的 Token
	err = service.DeleteToken(9999)
	if err != ErrTokenNotFound {
		t.Errorf("DeleteToken() with non-existent ID should return ErrTokenNotFound, got %v", err)
	}
}

// TestService_ValidateToken 测试验证 Token
func TestService_ValidateToken(t *testing.T) {
	db := setupTestDB(t)
	repo := NewRepository(db)
	service := NewService(repo)

	// 创建测试数据
	token, _ := service.CreateToken("Test Token", nil)

	// 测试有效 Token
	valid, err := service.ValidateToken(token.Token)
	if err != nil {
		t.Errorf("ValidateToken() failed for valid token: %v", err)
	}
	if valid.ID != token.ID {
		t.Errorf("ValidateToken() got ID = %v, want %v", valid.ID, token.ID)
	}

	// 测试无效 Token
	_, err = service.ValidateToken("sk-invalid")
	if err != ErrInvalidToken {
		t.Errorf("ValidateToken() with invalid token should return ErrInvalidToken, got %v", err)
	}
}

// TestService_ValidateToken_Disabled 测试验证已禁用的 Token
func TestService_ValidateToken_Disabled(t *testing.T) {
	db := setupTestDB(t)
	repo := NewRepository(db)
	service := NewService(repo)

	// 先创建一个启用的 Token
	token, _ := service.CreateToken("Disabled Token", nil)

	// 然后禁用它（直接使用 DB 更新，绕过默认值问题）
	db.Model(&models.Token{}).Where("id = ?", token.ID).Update("enabled", false)

	// 测试已禁用的 Token
	_, err := service.ValidateToken(token.Token)
	if err != ErrTokenDisabled {
		t.Errorf("ValidateToken() with disabled token should return ErrTokenDisabled, got %v", err)
	}
}

// TestService_ValidateToken_Expired 测试验证已过期的 Token
func TestService_ValidateToken_Expired(t *testing.T) {
	db := setupTestDB(t)
	repo := NewRepository(db)
	service := NewService(repo)

	// 创建已过期的 Token
	pastTime := time.Now().Add(-1 * time.Hour)
	token := &models.Token{
		Name:      "Expired Token",
		Token:     "sk-expired123",
		Enabled:   true,
		ExpiresAt: &pastTime,
	}
	repo.Create(token)

	// 测试已过期的 Token
	_, err := service.ValidateToken(token.Token)
	if err != ErrTokenExpired {
		t.Errorf("ValidateToken() with expired token should return ErrTokenExpired, got %v", err)
	}
}

// TestMaskToken 测试 Token 脱敏
func TestMaskToken(t *testing.T) {
	tests := []struct {
		name   string
		token  string
		want   string
	}{
		{
			name:  "正常 Token",
			token: "sk-abc123xyz789",
			want:  "sk-****z789",
		},
		{
			name:  "短 Token",
			token: "sk-123",
			want:  "****",
		},
		{
			name:  "空 Token",
			token: "",
			want:  "****",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := MaskToken(tt.token)
			if got != tt.want {
				t.Errorf("MaskToken() = %v, want %v", got, tt.want)
			}
		})
	}
}
