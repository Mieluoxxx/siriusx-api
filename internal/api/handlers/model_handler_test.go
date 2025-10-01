package handlers

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/Mieluoxxx/Siriusx-API/internal/mapping"
	"github.com/Mieluoxxx/Siriusx-API/internal/models"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

func setupModelTestHandler(t *testing.T) (*ModelHandler, *gin.Engine) {
	// 直接创建内存数据库
	database, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent),
	})
	require.NoError(t, err)

	// 手动迁移只需要的模型
	err = database.AutoMigrate(&models.UnifiedModel{})
	require.NoError(t, err)

	// 创建依赖
	repo := mapping.NewRepository(database)
	service := mapping.NewService(repo)
	handler := NewModelHandler(service)

	// 设置 Gin 路由
	gin.SetMode(gin.TestMode)
	router := gin.New()

	models := router.Group("/api/models")
	{
		models.POST("", handler.CreateModel)
		models.GET("", handler.ListModels)
		models.GET("/:id", handler.GetModel)
		models.PUT("/:id", handler.UpdateModel)
		models.DELETE("/:id", handler.DeleteModel)
	}

	return handler, router
}

func TestModelHandler_CreateModel_Success(t *testing.T) {
	_, router := setupModelTestHandler(t)

	reqBody := mapping.CreateModelRequest{
		Name:        "claude-sonnet-4",
		Description: "Claude Sonnet 4 model",
	}

	body, _ := json.Marshal(reqBody)
	req := httptest.NewRequest(http.MethodPost, "/api/models", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")

	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusCreated, w.Code)

	var response mapping.ModelResponse
	err := json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err)

	assert.Equal(t, "claude-sonnet-4", response.Name)
	assert.Equal(t, "Claude Sonnet 4 model", response.Description)
	assert.NotZero(t, response.ID)
}

func TestModelHandler_CreateModel_ValidationError(t *testing.T) {
	_, router := setupModelTestHandler(t)

	reqBody := mapping.CreateModelRequest{
		Name: "", // 空名称应该失败
	}

	body, _ := json.Marshal(reqBody)
	req := httptest.NewRequest(http.MethodPost, "/api/models", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")

	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)

	var response ErrorResponse
	err := json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err)

	// Gin 的验证错误包含 "required" 关键词
	assert.Contains(t, response.Error, "required")
}

func TestModelHandler_CreateModel_InvalidJSON(t *testing.T) {
	_, router := setupModelTestHandler(t)

	req := httptest.NewRequest(http.MethodPost, "/api/models", bytes.NewBufferString("invalid json"))
	req.Header.Set("Content-Type", "application/json")

	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestModelHandler_CreateModel_DuplicateName(t *testing.T) {
	_, router := setupModelTestHandler(t)

	reqBody := mapping.CreateModelRequest{
		Name:        "claude-sonnet-4",
		Description: "Claude Sonnet 4 model",
	}

	body, _ := json.Marshal(reqBody)

	// 创建第一个模型
	req := httptest.NewRequest(http.MethodPost, "/api/models", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")

	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)
	assert.Equal(t, http.StatusCreated, w.Code)

	// 尝试创建同名模型
	req = httptest.NewRequest(http.MethodPost, "/api/models", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")

	w = httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusConflict, w.Code)

	var response ErrorResponse
	err := json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err)

	assert.Contains(t, response.Error, "model name already exists")
}

func TestModelHandler_ListModels_Success(t *testing.T) {
	_, router := setupModelTestHandler(t)

	// 创建测试数据
	models := []mapping.CreateModelRequest{
		{Name: "claude-sonnet-4", Description: "Claude Sonnet 4"},
		{Name: "gpt-4o", Description: "GPT-4o"},
	}

	for _, model := range models {
		body, _ := json.Marshal(model)
		req := httptest.NewRequest(http.MethodPost, "/api/models", bytes.NewBuffer(body))
		req.Header.Set("Content-Type", "application/json")

		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)
		require.Equal(t, http.StatusCreated, w.Code)
	}

	// 测试列表查询
	req := httptest.NewRequest(http.MethodGet, "/api/models", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var response mapping.ListModelsResponse
	err := json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err)

	assert.Len(t, response.Models, 2)
	assert.Equal(t, int64(2), response.Pagination.Total)
	assert.Equal(t, 1, response.Pagination.Page)
	assert.Equal(t, 20, response.Pagination.PageSize)
}

func TestModelHandler_ListModels_WithPagination(t *testing.T) {
	_, router := setupModelTestHandler(t)

	// 创建 3 个测试模型
	for i := 1; i <= 3; i++ {
		model := mapping.CreateModelRequest{
			Name:        fmt.Sprintf("model-%d", i),
			Description: fmt.Sprintf("Model %d", i),
		}

		body, _ := json.Marshal(model)
		req := httptest.NewRequest(http.MethodPost, "/api/models", bytes.NewBuffer(body))
		req.Header.Set("Content-Type", "application/json")

		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)
		require.Equal(t, http.StatusCreated, w.Code)
	}

	// 测试分页
	req := httptest.NewRequest(http.MethodGet, "/api/models?page=1&page_size=2", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var response mapping.ListModelsResponse
	err := json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err)

	assert.Len(t, response.Models, 2)
	assert.Equal(t, int64(3), response.Pagination.Total)
	assert.Equal(t, 1, response.Pagination.Page)
	assert.Equal(t, 2, response.Pagination.PageSize)
	assert.Equal(t, 2, response.Pagination.TotalPages)
}

func TestModelHandler_ListModels_WithSearch(t *testing.T) {
	_, router := setupModelTestHandler(t)

	// 创建测试数据
	models := []mapping.CreateModelRequest{
		{Name: "claude-sonnet-4", Description: "Claude Sonnet 4"},
		{Name: "claude-haiku", Description: "Claude Haiku"},
		{Name: "gpt-4o", Description: "GPT-4o"},
	}

	for _, model := range models {
		body, _ := json.Marshal(model)
		req := httptest.NewRequest(http.MethodPost, "/api/models", bytes.NewBuffer(body))
		req.Header.Set("Content-Type", "application/json")

		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)
		require.Equal(t, http.StatusCreated, w.Code)
	}

	// 测试搜索
	req := httptest.NewRequest(http.MethodGet, "/api/models?search=claude", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var response mapping.ListModelsResponse
	err := json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err)

	assert.Len(t, response.Models, 2)
	assert.Equal(t, int64(2), response.Pagination.Total)
}

func TestModelHandler_GetModel_Success(t *testing.T) {
	_, router := setupModelTestHandler(t)

	// 创建测试模型
	reqBody := mapping.CreateModelRequest{
		Name:        "test-model",
		Description: "Test model",
	}

	body, _ := json.Marshal(reqBody)
	req := httptest.NewRequest(http.MethodPost, "/api/models", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")

	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)
	require.Equal(t, http.StatusCreated, w.Code)

	var created mapping.ModelResponse
	err := json.Unmarshal(w.Body.Bytes(), &created)
	require.NoError(t, err)

	// 测试获取模型
	req = httptest.NewRequest(http.MethodGet, fmt.Sprintf("/api/models/%d", created.ID), nil)
	w = httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var response mapping.ModelResponse
	err = json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err)

	assert.Equal(t, created.ID, response.ID)
	assert.Equal(t, "test-model", response.Name)
	assert.Equal(t, "Test model", response.Description)
}

func TestModelHandler_GetModel_NotFound(t *testing.T) {
	_, router := setupModelTestHandler(t)

	req := httptest.NewRequest(http.MethodGet, "/api/models/9999", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusNotFound, w.Code)

	var response ErrorResponse
	err := json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err)

	assert.Contains(t, response.Error, "model not found")
}

func TestModelHandler_GetModel_InvalidID(t *testing.T) {
	_, router := setupModelTestHandler(t)

	req := httptest.NewRequest(http.MethodGet, "/api/models/invalid", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)

	var response ErrorResponse
	err := json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err)

	assert.Contains(t, response.Error, "无效的模型ID")
}

func TestModelHandler_UpdateModel_Success(t *testing.T) {
	_, router := setupModelTestHandler(t)

	// 创建测试模型
	createReq := mapping.CreateModelRequest{
		Name:        "test-model",
		Description: "Original description",
	}

	body, _ := json.Marshal(createReq)
	req := httptest.NewRequest(http.MethodPost, "/api/models", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")

	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)
	require.Equal(t, http.StatusCreated, w.Code)

	var created mapping.ModelResponse
	err := json.Unmarshal(w.Body.Bytes(), &created)
	require.NoError(t, err)

	// 更新模型
	newName := "updated-model"
	newDesc := "Updated description"
	updateReq := mapping.UpdateModelRequest{
		Name:        &newName,
		Description: &newDesc,
	}

	body, _ = json.Marshal(updateReq)
	req = httptest.NewRequest(http.MethodPut, fmt.Sprintf("/api/models/%d", created.ID), bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")

	w = httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var response mapping.ModelResponse
	err = json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err)

	assert.Equal(t, created.ID, response.ID)
	assert.Equal(t, "updated-model", response.Name)
	assert.Equal(t, "Updated description", response.Description)
}

func TestModelHandler_UpdateModel_NotFound(t *testing.T) {
	_, router := setupModelTestHandler(t)

	newName := "updated-model"
	updateReq := mapping.UpdateModelRequest{
		Name: &newName,
	}

	body, _ := json.Marshal(updateReq)
	req := httptest.NewRequest(http.MethodPut, "/api/models/9999", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")

	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusNotFound, w.Code)
}

func TestModelHandler_DeleteModel_Success(t *testing.T) {
	_, router := setupModelTestHandler(t)

	// 创建测试模型
	createReq := mapping.CreateModelRequest{
		Name:        "test-model",
		Description: "Test model",
	}

	body, _ := json.Marshal(createReq)
	req := httptest.NewRequest(http.MethodPost, "/api/models", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")

	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)
	require.Equal(t, http.StatusCreated, w.Code)

	var created mapping.ModelResponse
	err := json.Unmarshal(w.Body.Bytes(), &created)
	require.NoError(t, err)

	// 删除模型
	req = httptest.NewRequest(http.MethodDelete, fmt.Sprintf("/api/models/%d", created.ID), nil)
	w = httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusNoContent, w.Code)

	// 验证模型已删除
	req = httptest.NewRequest(http.MethodGet, fmt.Sprintf("/api/models/%d", created.ID), nil)
	w = httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusNotFound, w.Code)
}

func TestModelHandler_DeleteModel_NotFound(t *testing.T) {
	_, router := setupModelTestHandler(t)

	req := httptest.NewRequest(http.MethodDelete, "/api/models/9999", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusNotFound, w.Code)
}