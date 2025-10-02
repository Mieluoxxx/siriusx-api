package handlers

import (
	"bytes"
	"encoding/json"
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

// setupTokenTestHandler 创建 Token 测试处理器和路由
func setupTokenTestHandler(t *testing.T) (*gin.Engine, *token.Service, *gorm.DB) {
	// 设置 Gin 测试模式
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

	// 创建处理器
	repo := token.NewRepository(db)
	service := token.NewService(repo)
	handler := NewTokenHandler(service)

	// 配置路由
	router := gin.New()
	api := router.Group("/api")
	{
		tokens := api.Group("/tokens")
		{
			tokens.POST("", handler.CreateToken)
			tokens.GET("", handler.ListTokens)
			tokens.DELETE("/:id", handler.DeleteToken)
		}
	}

	return router, service, db
}

// TestTokenHandler_CreateToken_Success 测试成功创建 Token
func TestTokenHandler_CreateToken_Success(t *testing.T) {
	router, _, _ := setupTokenTestHandler(t)

	reqBody := token.CreateTokenRequest{
		Name: "Test Token",
	}
	body, _ := json.Marshal(reqBody)

	req, _ := http.NewRequest("POST", "/api/tokens", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	resp := httptest.NewRecorder()

	router.ServeHTTP(resp, req)

	if resp.Code != http.StatusCreated {
		t.Errorf("Expected status 201, got %d. Body: %s", resp.Code, resp.Body.String())
	}

	var response token.TokenDTO
	json.Unmarshal(resp.Body.Bytes(), &response)

	if response.Name != reqBody.Name {
		t.Errorf("Expected name %s, got %s", reqBody.Name, response.Name)
	}

	// 验证返回了完整 Token
	if response.Token == "" {
		t.Error("Expected token to be included in response")
	}

	// 验证 Token 格式
	if len(response.Token) < 40 || response.Token[:3] != "sk-" {
		t.Errorf("Invalid token format: %s", response.Token)
	}
}

// TestTokenHandler_CreateToken_WithExpiresAt 测试创建带过期时间的 Token
func TestTokenHandler_CreateToken_WithExpiresAt(t *testing.T) {
	router, _, _ := setupTokenTestHandler(t)

	futureTime := time.Now().Add(24 * time.Hour)
	reqBody := token.CreateTokenRequest{
		Name:      "Test Token",
		ExpiresAt: &futureTime,
	}
	body, _ := json.Marshal(reqBody)

	req, _ := http.NewRequest("POST", "/api/tokens", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	resp := httptest.NewRecorder()

	router.ServeHTTP(resp, req)

	if resp.Code != http.StatusCreated {
		t.Errorf("Expected status 201, got %d", resp.Code)
	}

	var response token.TokenDTO
	json.Unmarshal(resp.Body.Bytes(), &response)

	if response.ExpiresAt == nil {
		t.Error("Expected ExpiresAt to be set")
	}
}

// TestTokenHandler_CreateToken_InvalidExpiresAt 测试创建过期时间无效的 Token
func TestTokenHandler_CreateToken_InvalidExpiresAt(t *testing.T) {
	router, _, _ := setupTokenTestHandler(t)

	pastTime := time.Now().Add(-24 * time.Hour)
	reqBody := token.CreateTokenRequest{
		Name:      "Test Token",
		ExpiresAt: &pastTime,
	}
	body, _ := json.Marshal(reqBody)

	req, _ := http.NewRequest("POST", "/api/tokens", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	resp := httptest.NewRecorder()

	router.ServeHTTP(resp, req)

	if resp.Code != http.StatusBadRequest {
		t.Errorf("Expected status 400, got %d", resp.Code)
	}

	var response map[string]interface{}
	json.Unmarshal(resp.Body.Bytes(), &response)

	errorData := response["error"].(map[string]interface{})
	if errorData["code"] != "INVALID_EXPIRES_AT" {
		t.Errorf("Expected error code INVALID_EXPIRES_AT, got %v", errorData["code"])
	}
}

// TestTokenHandler_CreateToken_ValidationError 测试验证错误
func TestTokenHandler_CreateToken_ValidationError(t *testing.T) {
	router, _, _ := setupTokenTestHandler(t)

	// 缺少 name 字段
	reqBody := token.CreateTokenRequest{}
	body, _ := json.Marshal(reqBody)

	req, _ := http.NewRequest("POST", "/api/tokens", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	resp := httptest.NewRecorder()

	router.ServeHTTP(resp, req)

	if resp.Code != http.StatusBadRequest {
		t.Errorf("Expected status 400, got %d", resp.Code)
	}
}

// TestTokenHandler_ListTokens 测试获取 Token 列表
func TestTokenHandler_ListTokens(t *testing.T) {
	router, service, _ := setupTokenTestHandler(t)

	// 创建测试数据
	service.CreateToken("Token 1", nil, "")
	service.CreateToken("Token 2", nil, "")

	req, _ := http.NewRequest("GET", "/api/tokens", nil)
	resp := httptest.NewRecorder()

	router.ServeHTTP(resp, req)

	if resp.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", resp.Code)
	}

	var response []*token.TokenDTO
	json.Unmarshal(resp.Body.Bytes(), &response)

	if len(response) != 2 {
		t.Errorf("Expected 2 tokens, got %d", len(response))
	}

	// 验证 Token 已脱敏
	for _, tok := range response {
		if tok.Token != "" {
			t.Error("Token should not be included in list response")
		}
		if tok.TokenDisplay == "" {
			t.Error("TokenDisplay should be included in list response")
		}
		if tok.TokenDisplay[:7] != "sk-****" {
			t.Errorf("TokenDisplay should be masked, got %s", tok.TokenDisplay)
		}
	}
}

// TestTokenHandler_DeleteToken_Success 测试成功删除 Token
func TestTokenHandler_DeleteToken_Success(t *testing.T) {
	router, service, _ := setupTokenTestHandler(t)

	// 创建测试数据
	tok, _ := service.CreateToken("Test Token", nil, "")

	req, _ := http.NewRequest("DELETE", "/api/tokens/1", nil)
	resp := httptest.NewRecorder()

	router.ServeHTTP(resp, req)

	if resp.Code != http.StatusNoContent {
		t.Errorf("Expected status 204, got %d", resp.Code)
	}

	// 验证已删除
	_, err := service.GetToken(tok.ID)
	if err == nil {
		t.Error("Token should be deleted")
	}
}

// TestTokenHandler_DeleteToken_NotFound 测试删除不存在的 Token
func TestTokenHandler_DeleteToken_NotFound(t *testing.T) {
	router, _, _ := setupTokenTestHandler(t)

	req, _ := http.NewRequest("DELETE", "/api/tokens/9999", nil)
	resp := httptest.NewRecorder()

	router.ServeHTTP(resp, req)

	if resp.Code != http.StatusNotFound {
		t.Errorf("Expected status 404, got %d", resp.Code)
	}

	var response map[string]interface{}
	json.Unmarshal(resp.Body.Bytes(), &response)

	errorData := response["error"].(map[string]interface{})
	if errorData["code"] != "TOKEN_NOT_FOUND" {
		t.Errorf("Expected error code TOKEN_NOT_FOUND, got %v", errorData["code"])
	}
}

// TestTokenHandler_DeleteToken_InvalidID 测试删除无效 ID
func TestTokenHandler_DeleteToken_InvalidID(t *testing.T) {
	router, _, _ := setupTokenTestHandler(t)

	req, _ := http.NewRequest("DELETE", "/api/tokens/invalid", nil)
	resp := httptest.NewRecorder()

	router.ServeHTTP(resp, req)

	if resp.Code != http.StatusBadRequest {
		t.Errorf("Expected status 400, got %d", resp.Code)
	}

	var response map[string]interface{}
	json.Unmarshal(resp.Body.Bytes(), &response)

	errorData := response["error"].(map[string]interface{})
	if errorData["code"] != "INVALID_ID" {
		t.Errorf("Expected error code INVALID_ID, got %v", errorData["code"])
	}
}
