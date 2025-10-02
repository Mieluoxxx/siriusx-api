package token_test

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/Mieluoxxx/Siriusx-API/internal/api"
	"github.com/Mieluoxxx/Siriusx-API/internal/api/middleware"
	"github.com/Mieluoxxx/Siriusx-API/internal/db"
	"github.com/Mieluoxxx/Siriusx-API/internal/models"
	"github.com/Mieluoxxx/Siriusx-API/internal/token"
	"github.com/gin-gonic/gin"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

// setupEpic6IntegrationTestEnv 创建集成测试环境
func setupEpic6IntegrationTestEnv(t *testing.T) (*gin.Engine, *gorm.DB) {
	gin.SetMode(gin.TestMode)

	// 创建测试数据库
	database, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("failed to create test database: %v", err)
	}

	// 自动迁移
	if err := db.AutoMigrate(database); err != nil {
		t.Fatalf("failed to migrate test database: %v", err)
	}

	// 创建 Token Service
	tokenRepo := token.NewRepository(database)
	tokenService := token.NewService(tokenRepo)

	// 配置路由（使用实际的路由配置）
	router := api.SetupRouter(database, nil)

	// 添加受保护的测试端点
	v1 := router.Group("/v1")
	v1.Use(middleware.TokenAuthMiddleware(tokenService))
	{
		v1.GET("/test", func(c *gin.Context) {
			tokenID, _ := c.Get("token_id")
			c.JSON(http.StatusOK, gin.H{
				"message":  "Protected resource accessed",
				"token_id": tokenID,
			})
		})
	}

	return router, database
}

// TestEpic6_TokenLifecycle 测试完整的 Token 生命周期
func TestEpic6_TokenLifecycle(t *testing.T) {
	router, _ := setupEpic6IntegrationTestEnv(t)

	// 1. 创建 Token
	t.Log("步骤 1: 创建 Token")
	createReq := map[string]interface{}{
		"name": "Integration Test Token",
	}
	body, _ := json.Marshal(createReq)

	req, _ := http.NewRequest("POST", "/api/tokens", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	resp := httptest.NewRecorder()
	router.ServeHTTP(resp, req)

	if resp.Code != http.StatusCreated {
		t.Fatalf("Failed to create token: status %d, body: %s", resp.Code, resp.Body.String())
	}

	var createResp token.TokenDTO
	json.Unmarshal(resp.Body.Bytes(), &createResp)
	createdToken := createResp.Token
	tokenID := createResp.ID

	if createdToken == "" {
		t.Fatal("Token should be returned on creation")
	}
	t.Logf("✅ Token 创建成功: %s", token.MaskToken(createdToken))

	// 2. 使用 Token 访问受保护的端点
	t.Log("步骤 2: 使用 Token 访问受保护的端点")
	req, _ = http.NewRequest("GET", "/v1/test", nil)
	req.Header.Set("Authorization", "Bearer "+createdToken)
	resp = httptest.NewRecorder()
	router.ServeHTTP(resp, req)

	if resp.Code != http.StatusOK {
		t.Fatalf("Failed to access protected endpoint: status %d, body: %s", resp.Code, resp.Body.String())
	}
	t.Logf("✅ 成功访问受保护端点")

	// 3. 获取 Token 列表（验证脱敏）
	t.Log("步骤 3: 获取 Token 列表")
	req, _ = http.NewRequest("GET", "/api/tokens", nil)
	resp = httptest.NewRecorder()
	router.ServeHTTP(resp, req)

	if resp.Code != http.StatusOK {
		t.Fatalf("Failed to list tokens: status %d", resp.Code)
	}

	var tokens []token.TokenDTO
	json.Unmarshal(resp.Body.Bytes(), &tokens)

	if len(tokens) != 1 {
		t.Fatalf("Expected 1 token, got %d", len(tokens))
	}

	if tokens[0].Token != "" {
		t.Error("Token should be masked in list response")
	}

	if tokens[0].TokenDisplay == "" || tokens[0].TokenDisplay[:7] != "sk-****" {
		t.Errorf("TokenDisplay should be masked, got %s", tokens[0].TokenDisplay)
	}
	t.Logf("✅ Token 列表返回正确，已脱敏: %s", tokens[0].TokenDisplay)

	// 4. 删除 Token
	t.Log("步骤 4: 删除 Token")
	req, _ = http.NewRequest("DELETE", "/api/tokens/"+string(rune(tokenID+48)), nil)
	resp = httptest.NewRecorder()
	router.ServeHTTP(resp, req)

	if resp.Code != http.StatusNoContent {
		t.Fatalf("Failed to delete token: status %d", resp.Code)
	}
	t.Logf("✅ Token 删除成功")

	// 5. 验证 Token 已失效
	t.Log("步骤 5: 验证 Token 已失效")
	req, _ = http.NewRequest("GET", "/v1/test", nil)
	req.Header.Set("Authorization", "Bearer "+createdToken)
	resp = httptest.NewRecorder()
	router.ServeHTTP(resp, req)

	if resp.Code != http.StatusUnauthorized {
		t.Fatalf("Deleted token should be invalid: status %d", resp.Code)
	}
	t.Logf("✅ 已删除的 Token 无法访问受保护端点")
}

// TestEpic6_TokenDisableEnable 测试 Token 禁用/启用
func TestEpic6_TokenDisableEnable(t *testing.T) {
	router, database := setupEpic6IntegrationTestEnv(t)

	// 1. 创建 Token
	createReq := map[string]interface{}{"name": "Disable Test Token"}
	body, _ := json.Marshal(createReq)
	req, _ := http.NewRequest("POST", "/api/tokens", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	resp := httptest.NewRecorder()
	router.ServeHTTP(resp, req)

	var createResp token.TokenDTO
	json.Unmarshal(resp.Body.Bytes(), &createResp)
	createdToken := createResp.Token
	tokenID := createResp.ID

	// 2. 验证 Token 可用
	req, _ = http.NewRequest("GET", "/v1/test", nil)
	req.Header.Set("Authorization", "Bearer "+createdToken)
	resp = httptest.NewRecorder()
	router.ServeHTTP(resp, req)

	if resp.Code != http.StatusOK {
		t.Fatalf("Token should be valid initially: status %d", resp.Code)
	}
	t.Logf("✅ Token 初始状态可用")

	// 3. 禁用 Token（直接操作数据库模拟禁用）
	database.Model(&models.Token{}).Where("id = ?", tokenID).Update("enabled", false)
	t.Logf("🔒 Token 已禁用")

	// 4. 验证禁用后无法访问
	req, _ = http.NewRequest("GET", "/v1/test", nil)
	req.Header.Set("Authorization", "Bearer "+createdToken)
	resp = httptest.NewRecorder()
	router.ServeHTTP(resp, req)

	if resp.Code != http.StatusUnauthorized {
		t.Fatalf("Disabled token should be invalid: status %d", resp.Code)
	}

	var errorResp map[string]interface{}
	json.Unmarshal(resp.Body.Bytes(), &errorResp)
	errorData := errorResp["error"].(map[string]interface{})
	if errorData["code"] != "TOKEN_DISABLED" {
		t.Errorf("Expected TOKEN_DISABLED error, got %v", errorData["code"])
	}
	t.Logf("✅ 禁用的 Token 返回 TOKEN_DISABLED 错误")

	// 5. 启用 Token
	database.Model(&models.Token{}).Where("id = ?", tokenID).Update("enabled", true)
	t.Logf("🔓 Token 已重新启用")

	// 6. 验证启用后可以访问
	req, _ = http.NewRequest("GET", "/v1/test", nil)
	req.Header.Set("Authorization", "Bearer "+createdToken)
	resp = httptest.NewRecorder()
	router.ServeHTTP(resp, req)

	if resp.Code != http.StatusOK {
		t.Fatalf("Enabled token should be valid: status %d", resp.Code)
	}
	t.Logf("✅ 重新启用的 Token 可以访问")
}

// TestEpic6_TokenExpiration 测试 Token 过期
func TestEpic6_TokenExpiration(t *testing.T) {
	router, _ := setupEpic6IntegrationTestEnv(t)

	// 1. 创建带过期时间的 Token（1秒后过期）
	futureTime := time.Now().Add(1 * time.Second)
	createReq := map[string]interface{}{
		"name":       "Expiring Test Token",
		"expires_at": futureTime.Format(time.RFC3339),
	}
	body, _ := json.Marshal(createReq)
	req, _ := http.NewRequest("POST", "/api/tokens", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	resp := httptest.NewRecorder()
	router.ServeHTTP(resp, req)

	var createResp token.TokenDTO
	json.Unmarshal(resp.Body.Bytes(), &createResp)
	createdToken := createResp.Token

	// 2. 验证 Token 当前可用
	req, _ = http.NewRequest("GET", "/v1/test", nil)
	req.Header.Set("Authorization", "Bearer "+createdToken)
	resp = httptest.NewRecorder()
	router.ServeHTTP(resp, req)

	if resp.Code != http.StatusOK {
		t.Fatalf("Token should be valid before expiration: status %d", resp.Code)
	}
	t.Logf("✅ Token 过期前可用")

	// 3. 等待 Token 过期
	t.Logf("⏳ 等待 Token 过期...")
	time.Sleep(2 * time.Second)

	// 4. 验证过期后无法访问
	req, _ = http.NewRequest("GET", "/v1/test", nil)
	req.Header.Set("Authorization", "Bearer "+createdToken)
	resp = httptest.NewRecorder()
	router.ServeHTTP(resp, req)

	if resp.Code != http.StatusUnauthorized {
		t.Fatalf("Expired token should be invalid: status %d", resp.Code)
	}

	var errorResp map[string]interface{}
	json.Unmarshal(resp.Body.Bytes(), &errorResp)
	errorData := errorResp["error"].(map[string]interface{})
	if errorData["code"] != "TOKEN_EXPIRED" {
		t.Errorf("Expected TOKEN_EXPIRED error, got %v", errorData["code"])
	}
	t.Logf("✅ 过期的 Token 返回 TOKEN_EXPIRED 错误")
}

// TestEpic6_MultipleTokens 测试多个 Token 管理
func TestEpic6_MultipleTokens(t *testing.T) {
	router, _ := setupEpic6IntegrationTestEnv(t)

	var tokens []string

	// 1. 创建多个 Token
	for i := 1; i <= 3; i++ {
		createReq := map[string]interface{}{
			"name": "Multi Test Token " + string(rune(i+48)),
		}
		body, _ := json.Marshal(createReq)
		req, _ := http.NewRequest("POST", "/api/tokens", bytes.NewBuffer(body))
		req.Header.Set("Content-Type", "application/json")
		resp := httptest.NewRecorder()
		router.ServeHTTP(resp, req)

		var createResp token.TokenDTO
		json.Unmarshal(resp.Body.Bytes(), &createResp)
		tokens = append(tokens, createResp.Token)
	}
	t.Logf("✅ 创建了 %d 个 Token", len(tokens))

	// 2. 验证每个 Token 都可以独立工作
	for i, tok := range tokens {
		req, _ := http.NewRequest("GET", "/v1/test", nil)
		req.Header.Set("Authorization", "Bearer "+tok)
		resp := httptest.NewRecorder()
		router.ServeHTTP(resp, req)

		if resp.Code != http.StatusOK {
			t.Fatalf("Token %d should be valid: status %d", i+1, resp.Code)
		}
	}
	t.Logf("✅ 所有 Token 都可以独立访问受保护端点")

	// 3. 获取 Token 列表
	req, _ := http.NewRequest("GET", "/api/tokens", nil)
	resp := httptest.NewRecorder()
	router.ServeHTTP(resp, req)

	var tokenList []token.TokenDTO
	json.Unmarshal(resp.Body.Bytes(), &tokenList)

	if len(tokenList) != 3 {
		t.Fatalf("Expected 3 tokens in list, got %d", len(tokenList))
	}
	t.Logf("✅ Token 列表返回了所有 3 个 Token")
}

// TestEpic6_InvalidTokenScenarios 测试各种无效 Token 场景
func TestEpic6_InvalidTokenScenarios(t *testing.T) {
	router, _ := setupEpic6IntegrationTestEnv(t)

	scenarios := []struct {
		name           string
		authHeader     string
		expectedStatus int
		expectedCode   string
	}{
		{
			name:           "缺少 Authorization 头",
			authHeader:     "",
			expectedStatus: http.StatusUnauthorized,
			expectedCode:   "MISSING_AUTH_HEADER",
		},
		{
			name:           "错误的 Authorization 格式",
			authHeader:     "sk-invalid-format",
			expectedStatus: http.StatusUnauthorized,
			expectedCode:   "INVALID_AUTH_FORMAT",
		},
		{
			name:           "无效的 Token",
			authHeader:     "Bearer sk-completely-invalid-token-123456",
			expectedStatus: http.StatusUnauthorized,
			expectedCode:   "INVALID_TOKEN",
		},
		{
			name:           "空 Bearer Token",
			authHeader:     "Bearer ",
			expectedStatus: http.StatusUnauthorized,
			expectedCode:   "INVALID_AUTH_FORMAT",
		},
	}

	for _, scenario := range scenarios {
		t.Run(scenario.name, func(t *testing.T) {
			req, _ := http.NewRequest("GET", "/v1/test", nil)
			if scenario.authHeader != "" {
				req.Header.Set("Authorization", scenario.authHeader)
			}
			resp := httptest.NewRecorder()
			router.ServeHTTP(resp, req)

			if resp.Code != scenario.expectedStatus {
				t.Errorf("Expected status %d, got %d", scenario.expectedStatus, resp.Code)
			}

			var errorResp map[string]interface{}
			json.Unmarshal(resp.Body.Bytes(), &errorResp)
			if errorData, ok := errorResp["error"].(map[string]interface{}); ok {
				if errorData["code"] != scenario.expectedCode {
					t.Errorf("Expected error code %s, got %v", scenario.expectedCode, errorData["code"])
				}
			}
			t.Logf("✅ %s - 返回正确的错误: %s", scenario.name, scenario.expectedCode)
		})
	}
}
