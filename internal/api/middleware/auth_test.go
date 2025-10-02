package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/Mieluoxxx/Siriusx-API/internal/models"
	"github.com/Mieluoxxx/Siriusx-API/internal/token"
	"github.com/gin-gonic/gin"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

// setupAuthTestEnv 创建测试环境
func setupAuthTestEnv(t *testing.T) (*gin.Engine, *token.Service, *gorm.DB) {
	gin.SetMode(gin.TestMode)

	// 创建测试数据库
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("failed to create test database: %v", err)
	}

	// 自动迁移
	if err := db.AutoMigrate(&models.Token{}); err != nil {
		t.Fatalf("failed to migrate test database: %v", err)
	}

	// 创建 Service
	repo := token.NewRepository(db)
	service := token.NewService(repo)

	// 配置路由
	router := gin.New()

	// 受保护的端点
	protected := router.Group("/protected")
	protected.Use(TokenAuthMiddleware(service))
	{
		protected.GET("/resource", func(c *gin.Context) {
			// 从 Context 获取 Token 信息
			tokenID, _ := c.Get("token_id")
			tok, _ := c.Get("token")

			c.JSON(http.StatusOK, gin.H{
				"message":  "Success",
				"token_id": tokenID,
				"token":    tok,
			})
		})
	}

	return router, service, db
}

// TestTokenAuthMiddleware_Success 测试成功验证
func TestTokenAuthMiddleware_Success(t *testing.T) {
	router, service, _ := setupAuthTestEnv(t)

	// 创建有效 Token
	tok, _ := service.CreateToken("Test Token", nil)

	// 发送请求
	req, _ := http.NewRequest("GET", "/protected/resource", nil)
	req.Header.Set("Authorization", "Bearer "+tok.Token)
	resp := httptest.NewRecorder()

	router.ServeHTTP(resp, req)

	if resp.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d. Body: %s", resp.Code, resp.Body.String())
	}
}

// TestTokenAuthMiddleware_MissingAuthHeader 测试缺少 Authorization 头
func TestTokenAuthMiddleware_MissingAuthHeader(t *testing.T) {
	router, _, _ := setupAuthTestEnv(t)

	req, _ := http.NewRequest("GET", "/protected/resource", nil)
	// 不设置 Authorization 头
	resp := httptest.NewRecorder()

	router.ServeHTTP(resp, req)

	if resp.Code != http.StatusUnauthorized {
		t.Errorf("Expected status 401, got %d", resp.Code)
	}

	// 验证错误码
	if !contains(resp.Body.String(), "MISSING_AUTH_HEADER") {
		t.Errorf("Expected error code MISSING_AUTH_HEADER, got %s", resp.Body.String())
	}
}

// TestTokenAuthMiddleware_InvalidAuthFormat 测试 Authorization 格式错误
func TestTokenAuthMiddleware_InvalidAuthFormat(t *testing.T) {
	router, _, _ := setupAuthTestEnv(t)

	tests := []struct {
		name   string
		header string
	}{
		{"缺少 Bearer", "sk-token123"},
		{"空 Bearer", "Bearer "},
		{"错误前缀", "Token sk-token123"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req, _ := http.NewRequest("GET", "/protected/resource", nil)
			req.Header.Set("Authorization", tt.header)
			resp := httptest.NewRecorder()

			router.ServeHTTP(resp, req)

			if resp.Code != http.StatusUnauthorized {
				t.Errorf("Expected status 401, got %d", resp.Code)
			}

			if !contains(resp.Body.String(), "INVALID_AUTH_FORMAT") {
				t.Errorf("Expected error code INVALID_AUTH_FORMAT, got %s", resp.Body.String())
			}
		})
	}
}

// TestTokenAuthMiddleware_InvalidToken 测试无效 Token
func TestTokenAuthMiddleware_InvalidToken(t *testing.T) {
	router, _, _ := setupAuthTestEnv(t)

	req, _ := http.NewRequest("GET", "/protected/resource", nil)
	req.Header.Set("Authorization", "Bearer sk-invalid-token-123")
	resp := httptest.NewRecorder()

	router.ServeHTTP(resp, req)

	if resp.Code != http.StatusUnauthorized {
		t.Errorf("Expected status 401, got %d", resp.Code)
	}

	if !contains(resp.Body.String(), "INVALID_TOKEN") {
		t.Errorf("Expected error code INVALID_TOKEN, got %s", resp.Body.String())
	}
}

// TestTokenAuthMiddleware_DisabledToken 测试已禁用 Token
func TestTokenAuthMiddleware_DisabledToken(t *testing.T) {
	router, service, db := setupAuthTestEnv(t)

	// 创建 Token 并禁用
	tok, _ := service.CreateToken("Disabled Token", nil)
	db.Model(&models.Token{}).Where("id = ?", tok.ID).Update("enabled", false)

	req, _ := http.NewRequest("GET", "/protected/resource", nil)
	req.Header.Set("Authorization", "Bearer "+tok.Token)
	resp := httptest.NewRecorder()

	router.ServeHTTP(resp, req)

	if resp.Code != http.StatusUnauthorized {
		t.Errorf("Expected status 401, got %d", resp.Code)
	}

	if !contains(resp.Body.String(), "TOKEN_DISABLED") {
		t.Errorf("Expected error code TOKEN_DISABLED, got %s", resp.Body.String())
	}
}

// TestTokenAuthMiddleware_ExpiredToken 测试已过期 Token
func TestTokenAuthMiddleware_ExpiredToken(t *testing.T) {
	router, _, db := setupAuthTestEnv(t)

	// 创建已过期的 Token
	pastTime := time.Now().Add(-1 * time.Hour)
	expiredToken := &models.Token{
		Name:      "Expired Token",
		Token:     "sk-expired123",
		Enabled:   true,
		ExpiresAt: &pastTime,
	}
	db.Select("Name", "Token", "Enabled", "ExpiresAt").Create(expiredToken)

	req, _ := http.NewRequest("GET", "/protected/resource", nil)
	req.Header.Set("Authorization", "Bearer "+expiredToken.Token)
	resp := httptest.NewRecorder()

	router.ServeHTTP(resp, req)

	if resp.Code != http.StatusUnauthorized {
		t.Errorf("Expected status 401, got %d", resp.Code)
	}

	if !contains(resp.Body.String(), "TOKEN_EXPIRED") {
		t.Errorf("Expected error code TOKEN_EXPIRED, got %s", resp.Body.String())
	}
}

// TestTokenAuthMiddleware_ContextData 测试 Context 数据存储
func TestTokenAuthMiddleware_ContextData(t *testing.T) {
	router, service, _ := setupAuthTestEnv(t)

	// 创建有效 Token
	tok, _ := service.CreateToken("Test Token", nil)

	// 发送请求
	req, _ := http.NewRequest("GET", "/protected/resource", nil)
	req.Header.Set("Authorization", "Bearer "+tok.Token)
	resp := httptest.NewRecorder()

	router.ServeHTTP(resp, req)

	if resp.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", resp.Code)
	}

	// 验证响应中包含 Token 信息
	body := resp.Body.String()
	if !contains(body, "token_id") {
		t.Error("Response should contain token_id")
	}
}

// contains 检查字符串是否包含子串
func contains(s, substr string) bool {
	return len(s) > 0 && len(substr) > 0 && (s == substr || len(s) >= len(substr) && (s[:len(substr)] == substr || s[len(s)-len(substr):] == substr || containsSubstring(s, substr)))
}

func containsSubstring(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
